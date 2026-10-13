package handlers

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"buildoverse/backend/config"
	"buildoverse/backend/database"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

var startTime = time.Now()

type AdminUserView struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Mail         string `json:"mail"`
	Unikey       string `json:"unikey"`
	Power        int    `json:"power"`
	RoleName     string `json:"role_name"`
	RoleDisplay  string `json:"role_display"`
	Bits         string `json:"bits"`
	Bucks        string `json:"bucks"`
	LastOnline   string `json:"last_online"`
	CreationDate string `json:"creation_date"`
	Views        int    `json:"views"`
	Pronouns     string `json:"pronouns"`
	Description  string `json:"description"`
}

type ModHistoryView struct {
	*models.ModHistory
	CanRetract bool
}

type AdminLogView struct {
	ID          int    `json:"id"`
	AdminName   string `json:"admin_name"`
	TargetID    int    `json:"target_id"`
	TargetName  string `json:"target_name"`
	ActionLabel string `json:"action_label"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Date        string `json:"date"`
}

type AvatarItems struct {
	HeadColor  string `json:"head_color"`
	TorsoColor string `json:"torso_color"`
	LArmColor  string `json:"larm_color"`
	RArmColor  string `json:"rarm_color"`
	LLegColor  string `json:"lleg_color"`
	RLegColor  string `json:"rleg_color"`
	Hat1       int    `json:"hat1"`
	Hat2       int    `json:"hat2"`
	Hat3       int    `json:"hat3"`
	Hat4       int    `json:"hat4"`
	Hat5       int    `json:"hat5"`
	Tool       int    `json:"tool"`
	Shirt      int    `json:"shirt"`
	TShirt     int    `json:"tshirt"`
	Pants      int    `json:"pants"`
	Face       int    `json:"face"`
}

func getAdminUser(c fiber.Ctx) (*models.User, error) {
	username := GetActiveUser(c)
	if username == "" {
		return nil, fiber.ErrUnauthorized
	}
	if service.User == nil {
		return nil, fiber.ErrInternalServerError
	}
	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		return nil, fiber.ErrUnauthorized
	}
	if !user.HasAdminAccess() {
		return nil, fiber.ErrForbidden
	}
	return user, nil
}

func adminAuthError(c fiber.Ctx, err error) error {
	switch err {
	case fiber.ErrUnauthorized:
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", "/login")
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Redirect().To("/login")
	case fiber.ErrInternalServerError:
		return c.Status(fiber.StatusInternalServerError).SendString("Database offline")
	default:
		return adminRedirectHome(c)
	}
}

func adminAPIError(c fiber.Ctx, err error) error {
	if err == fiber.ErrForbidden {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
}

func adminRedirectHome(c fiber.Ctx) error {
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect().To("/")
}

func getCSRFToken(c fiber.Ctx) string {
	sess := session.FromContext(c)
	if sess == nil {
		return ""
	}
	if token, _ := sess.Get("csrf").(string); token != "" {
		return token
	}
	token, err := models.GenerateRandomString(32)
	if err != nil {
		return ""
	}
	sess.Set("csrf", token)
	return token
}

func csrfValid(c fiber.Ctx) bool {
	sess := session.FromContext(c)
	if sess == nil {
		return false
	}
	token, _ := sess.Get("csrf").(string)
	return token != "" && token == c.FormValue("csrf")
}

func getSystemStats() (fiber.Map, runtime.MemStats) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cpuPct := 0.0
	if percentages, err := cpu.Percent(0, false); err == nil && len(percentages) > 0 {
		cpuPct = percentages[0]
	}

	sysMemTotal, sysMemUsed, sysMemPct := uint64(0), uint64(0), 0.0
	if v, err := mem.VirtualMemory(); err == nil {
		sysMemTotal = v.Total
		sysMemUsed = v.Used
		sysMemPct = v.UsedPercent
	}

	return fiber.Map{
		"uptime":        formatDuration(time.Since(startTime)),
		"goroutines":    runtime.NumGoroutine(),
		"cpu_usage":     fmt.Sprintf("%.1f%%", cpuPct),
		"mem_alloc":     formatBytes(m.Alloc),
		"mem_sys":       formatBytes(m.Sys),
		"sys_mem_total": formatBytes(sysMemTotal),
		"sys_mem_used":  formatBytes(sysMemUsed),
		"sys_mem_pct":   fmt.Sprintf("%.1f%%", sysMemPct),
		"gc_cycles":     m.NumGC,
		"cpuPct":        cpuPct,
		"sysMemTotal":   sysMemTotal,
		"sysMemUsed":    sysMemUsed,
		"sysMemPct":     sysMemPct,
	}, m
}

func getAvatarItems(userID int) *AvatarItems {
	if service.Avatar == nil {
		return nil
	}
	av, err := service.Avatar.GetAvatar(userID)
	if err != nil || av == nil {
		return nil
	}
	return &AvatarItems{
		HeadColor:  av.HeadColor,
		TorsoColor: av.TorsoColor,
		LArmColor:  av.LArmColor,
		RArmColor:  av.RArmColor,
		LLegColor:  av.LLegColor,
		RLegColor:  av.RLegColor,
		Hat1:       av.Hat1,
		Hat2:       av.Hat2,
		Hat3:       av.Hat3,
		Hat4:       av.Hat4,
		Hat5:       av.Hat5,
		Tool:       av.Tool,
		Shirt:      av.Shirt,
		TShirt:     av.TShirt,
		Pants:      av.Pants,
		Face:       av.Face,
	}
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func getCompilationDate() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, setting := range info.Settings {
			if setting.Key == "-time" {
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					return t.Format("Jan 02, 2006 15:04")
				}
			}
		}
	}
	return startTime.Format("Jan 02, 2006 15:04")
}

func formatUserForAdmin(u *models.User, adminPower int) AdminUserView {
	dispName := u.DisplayName
	if dispName == "" {
		dispName = u.Username
	}

	view := AdminUserView{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  dispName,
		Power:        u.Power,
		RoleName:     u.RoleName(),
		RoleDisplay:  u.RoleDisplayName(),
		CreationDate: u.CreationDate.Format("Jan 02, 2006"),
		Views:        u.Views,
		Pronouns:     u.Pronouns,
		Description:  u.Description,
	}

	if adminPower >= models.PowerModerator {
		view.Mail = u.Mail
		view.Bits = fmt.Sprintf("%d", u.Bits)
		view.Bucks = fmt.Sprintf("%d", u.Bucks)
		view.LastOnline = u.LastOnline.Format("Jan 02, 2006 15:04")
	} else {
		view.Mail = "[Protected]"
		view.Bits = "---"
		view.Bucks = "---"
		view.LastOnline = "---"
	}

	if adminPower >= models.PowerBuildoverseTeam {
		view.Unikey = u.Unikey
	} else {
		view.Unikey = "[Protected]"
	}

	return view
}

func canModerateTarget(adminUser, targetUser *models.User) bool {
	return adminUser.HasPower(models.PowerModerator) && adminUser.Power > targetUser.Power && adminUser.ID != targetUser.ID
}

func AdminIndex(c fiber.Ctx) error {
	user, err := getAdminUser(c)
	if err != nil {
		return adminAuthError(c, err)
	}

	stats, m := getSystemStats()

	dbConnected := false
	dbMaxOpen, dbOpenConns, dbInUse, dbIdle := 0, 0, 0, 0
	dbWaitCount := int64(0)
	dbWaitDuration := "0s"

	if database.DB != nil {
		dbConnected = true
		dbStats := database.DB.Stats()
		dbMaxOpen = dbStats.MaxOpenConnections
		dbOpenConns = dbStats.OpenConnections
		dbInUse = dbStats.InUse
		dbIdle = dbStats.Idle
		dbWaitCount = dbStats.WaitCount
		dbWaitDuration = dbStats.WaitDuration.String()
	}

	userCount := 0
	if service.User != nil {
		userCount, _ = service.User.GetUserCount()
	}

	gameCount := 0
	if service.Game != nil {
		games, err := service.Game.GetPopularGames(1000)
		if err == nil {
			gameCount = len(games)
		}
	}

	liveTimerActive := false
	if config.Global != nil {
		liveTimerActive = config.Global.IsLiveTimerActive()
	}

	initialUsers, totalUsers, _ := service.User.SearchAdminUsers("", -1, 15, 0)
	userViews := make([]AdminUserView, len(initialUsers))
	for i, u := range initialUsers {
		userViews[i] = formatUserForAdmin(u, user.Power)
	}

	initialLogs, totalLogs, _ := service.ModHistory.GetAllLogs(15, 0)
	totalLogPages := 1
	if totalLogs > 0 {
		totalLogPages = (totalLogs + 15 - 1) / 15
	}

	var initialAssets []*models.Asset
	totalAssets := 0
	totalAssetPages := 1
	if service.Asset != nil {
		initialAssets, totalAssets, _ = service.Asset.GetQueue("", 1, 12)
		if totalAssets > 0 {
			totalAssetPages = (totalAssets + 12 - 1) / 12
		}
	}
	if initialAssets == nil {
		initialAssets = []*models.Asset{}
	}

	activeTab := c.Query("tab", "status")
	switch activeTab {
	case "status", "users", "reports", "assets", "logs":
	default:
		activeTab = "status"
	}

	return Render(c, "pages/admin", fiber.Map{
		"Title":           "Admin - Buildoverse",
		"AdminUser":       user,
		"RoleName":        user.RoleName(),
		"RoleDisplayName": user.RoleDisplayName(),
		"ActiveTab":       activeTab,
		"StartTimeMs":     startTime.UnixMilli(),
		"Uptime":          stats["uptime"],
		"StartTime":       startTime.Format("Jan 02, 2006 15:04"),
		"CompilationDate": getCompilationDate(),
		"GoVersion":       runtime.Version(),
		"OSArch":          fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		"NumGoroutines":   stats["goroutines"],
		"NumCPU":          runtime.NumCPU(),
		"CPUUsage":        stats["cpu_usage"],
		"MemAlloc":        stats["mem_alloc"],
		"MemSys":          stats["mem_sys"],
		"SysMemTotal":     stats["sys_mem_total"],
		"SysMemUsed":      stats["sys_mem_used"],
		"SysMemPct":       stats["sys_mem_pct"],
		"NumGC":           m.NumGC,
		"DBConnected":     dbConnected,
		"DBMaxOpen":       dbMaxOpen,
		"DBOpenConns":     dbOpenConns,
		"DBInUse":         dbInUse,
		"DBIdle":          dbIdle,
		"DBWaitCount":     dbWaitCount,
		"DBWaitDuration":  dbWaitDuration,
		"UserCount":       userCount,
		"GameCount":       gameCount,
		"LiveTimerActive": liveTimerActive,
		"Users":           userViews,
		"TotalUsers":      totalUsers,
		"Logs":            initialLogs,
		"TotalLogPages":   totalLogPages,
		"Assets":          initialAssets,
		"TotalAssets":     totalAssets,
		"TotalAssetPages": totalAssetPages,
		"CSRF":            getCSRFToken(c),
	}, "layouts/main")
}

func AdminUserViewPage(c fiber.Ctx) error {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return adminAuthError(c, err)
	}

	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	targetUser, err := service.User.GetUserByID(targetID)
	if err != nil || targetUser == nil {
		return c.Status(fiber.StatusNotFound).SendString("User not found")
	}

	var modHistory []*models.ModHistory
	if service.ModHistory != nil {
		modHistory, _ = service.ModHistory.GetByUserID(targetID)
	}

	userView := formatUserForAdmin(targetUser, adminUser.Power)
	avatarData := getAvatarItems(targetID)
	canModerate := canModerateTarget(adminUser, targetUser)

	var outfits []*models.Outfit
	if service.Avatar != nil {
		outfits, _ = service.Avatar.GetOutfits(targetID)
	}

	historyViews := make([]ModHistoryView, len(modHistory))
	for i, entry := range modHistory {
		historyViews[i] = ModHistoryView{
			ModHistory: entry,
			CanRetract: canModerate && entry.Status == models.StatusActive && (adminUser.Power >= entry.AdminPower || adminUser.ID == entry.AdminID),
		}
	}

	return Render(c, "pages/admin_user", fiber.Map{
		"Title":       fmt.Sprintf("Admin - User #%d - Buildoverse", targetUser.ID),
		"AdminUser":   adminUser,
		"TargetUser":  userView,
		"ModHistory":  historyViews,
		"Avatar":      avatarData,
		"Outfits":     outfits,
		"CanModerate": canModerate,
		"CSRF":        getCSRFToken(c),
	}, "layouts/main")
}

func AdminStatusAPI(c fiber.Ctx) error {
	_, err := getAdminUser(c)
	if err != nil {
		return adminAPIError(c, err)
	}

	stats, _ := getSystemStats()
	delete(stats, "cpuPct")
	delete(stats, "sysMemTotal")
	delete(stats, "sysMemUsed")
	delete(stats, "sysMemPct")

	return c.JSON(stats)
}

func AdminUsersAPI(c fiber.Ctx) error {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return adminAPIError(c, err)
	}

	search := c.Query("q")
	roleFilter, _ := strconv.Atoi(c.Query("role", "-1"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "15"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 15
	}
	offset := (page - 1) * limit

	users, total, err := service.User.SearchAdminUsers(search, roleFilter, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	views := make([]AdminUserView, len(users))
	for i, u := range users {
		views[i] = formatUserForAdmin(u, adminUser.Power)
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(fiber.Map{
		"users":       views,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func AdminLogsAPI(c fiber.Ctx) error {
	_, err := getAdminUser(c)
	if err != nil {
		return adminAPIError(c, err)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "15"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 15
	}
	offset := (page - 1) * limit

	logs, total, err := service.ModHistory.GetAllLogs(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	views := make([]AdminLogView, len(logs))
	for i, entry := range logs {
		views[i] = AdminLogView{
			ID:          entry.ID,
			AdminName:   entry.AdminName,
			TargetID:    entry.UID,
			TargetName:  entry.TargetName,
			ActionLabel: entry.ActionLabel(),
			Reason:      entry.Reason,
			Status:      entry.Status,
			StatusLabel: entry.StatusLabel(),
			Date:        entry.CreationDate.Format("Jan 02, 2006 15:04"),
		}
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(fiber.Map{
		"logs":        views,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func AdminUserDetailAPI(c fiber.Ctx) error {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return adminAPIError(c, err)
	}

	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	targetUser, err := service.User.GetUserByID(targetID)
	if err != nil || targetUser == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(formatUserForAdmin(targetUser, adminUser.Power))
}

func adminRedirectToUser(c fiber.Ctx, targetID int) error {
	path := fmt.Sprintf("/admin/users/%d", targetID)
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", path)
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect().To(path)
}

func adminModerationError(c fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).SendString(msg)
}

func AdminScrubPost(c fiber.Ctx) error {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return adminAuthError(c, err)
	}

	if !csrfValid(c) {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	if service.ModHistory == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Moderation service unavailable")
	}

	action := c.Params("action")
	reason := c.FormValue("reason")

	switch action {
	case "description":
		err = service.ModHistory.ScrubDescription(adminUser, targetID, reason)
	case "username":
		err = service.ModHistory.ScrubUsername(adminUser, targetID, reason)
	case "displayname":
		err = service.ModHistory.ScrubDisplayName(adminUser, targetID, reason)
	case "pronouns":
		err = service.ModHistory.ScrubPronouns(adminUser, targetID, reason)
	default:
		return adminModerationError(c, "Unknown scrub action")
	}

	if err != nil {
		return adminModerationError(c, err.Error())
	}

	return adminRedirectToUser(c, targetID)
}

func AdminModhistRetractPost(c fiber.Ctx) error {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return adminAuthError(c, err)
	}

	if !csrfValid(c) {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	modHistID, err := strconv.Atoi(c.Params("mid"))
	if err != nil || modHistID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid moderation record ID")
	}

	if service.ModHistory == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Moderation service unavailable")
	}

	if err := service.ModHistory.Retract(adminUser, modHistID); err != nil {
		return adminModerationError(c, err.Error())
	}

	return adminRedirectToUser(c, targetID)
}

func adminTargetForAction(c fiber.Ctx) (*models.User, *models.User, error) {
	adminUser, err := getAdminUser(c)
	if err != nil {
		return nil, nil, adminAuthError(c, err)
	}

	if !csrfValid(c) {
		return nil, nil, c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return nil, nil, c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	targetUser, err := service.User.GetUserByID(targetID)
	if err != nil || targetUser == nil {
		return nil, nil, c.Status(fiber.StatusNotFound).SendString("User not found")
	}

	return adminUser, targetUser, nil
}

func AdminResetAvatarPost(c fiber.Ctx) error {
	adminUser, targetUser, err := adminTargetForAction(c)
	if err != nil || adminUser == nil || targetUser == nil {
		return err
	}

	if !canModerateTarget(adminUser, targetUser) {
		return adminModerationError(c, "You do not have permission to reset this user's avatar")
	}

	if service.Avatar == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Avatar service unavailable")
	}

	if err := service.Avatar.ResetAvatar(targetUser.ID); err != nil {
		return adminModerationError(c, err.Error())
	}

	if service.ModHistory != nil {
		if err := service.ModHistory.Record(adminUser, targetUser.ID, models.ActionResetAvatar, "Avatar reset", ""); err != nil {
			return adminModerationError(c, err.Error())
		}
	}

	return adminRedirectToUser(c, targetUser.ID)
}

func AdminOutfitDeletePost(c fiber.Ctx) error {
	adminUser, targetUser, err := adminTargetForAction(c)
	if err != nil || adminUser == nil || targetUser == nil {
		return err
	}

	if !canModerateTarget(adminUser, targetUser) {
		return adminModerationError(c, "You do not have permission to delete this user's outfits")
	}

	outfitID, err := strconv.Atoi(c.Params("oid"))
	if err != nil || outfitID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid outfit ID")
	}

	if service.Avatar == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Avatar service unavailable")
	}

	if err := service.Avatar.DeleteOutfit(targetUser.ID, outfitID); err != nil {
		return adminModerationError(c, err.Error())
	}

	if service.ModHistory != nil {
		if err := service.ModHistory.Record(adminUser, targetUser.ID, models.ActionDeleteOutfit, fmt.Sprintf("Outfit #%d deleted", outfitID), ""); err != nil {
			return adminModerationError(c, err.Error())
		}
	}

	return adminRedirectToUser(c, targetUser.ID)
}