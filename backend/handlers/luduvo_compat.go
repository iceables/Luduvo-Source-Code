package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/config"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

// ============================================================
// LUDUVO-COMPAT API HANDLERS
// These map luduvo API paths to proper JSON responses
// ============================================================

// Helper: get current user from session/cookie
func spaRole(u *models.User) string {
	if u != nil && u.Power >= models.PowerOwner {
		return "admin"
	}
	if u != nil && u.HasAdminAccess() {
		return "staff"
	}
	return "player"
}

func spaUser(u *models.User) fiber.Map {
	if u == nil {
		return fiber.Map{}
	}
	return fiber.Map{
		"id":             u.ID,
		"username":       u.Username,
		"display_name":   u.DisplayName,
		"email":          u.Mail,
		"role":           spaRole(u),
		"tier":           "free",
		"balance":        u.Bits,
		"bucks":          u.Bucks,
		"created_at":     u.CreationDate.Unix(),
		"email_verified": true,
		"headshot_url":   fmt.Sprintf("/users/%d/avatar/headshot", u.ID),
		"status":         1,
		"mfa_enabled":    false,
		"has_password":   true,
	}
}

func defaultAvatarRules() fiber.Map {
	var rules fiber.Map
	if raw, err := os.ReadFile("data/avatar-rules.json"); err == nil && json.Unmarshal(raw, &rules) == nil {
		return rules
	}
	return fiber.Map{
		"default_bundle_item_id": models.DefaultBundleItemID,
		"default_face_item_id":   models.DefaultFaceItemID,
		"default_body_colors": fiber.Map{
			"head": "#" + models.DefaultHeadColor, "torso": "#" + models.DefaultTorsoColor,
			"left_arm": "#" + models.DefaultArmColor, "right_arm": "#" + models.DefaultArmColor,
			"left_leg": "#" + models.DefaultLegColor, "right_leg": "#" + models.DefaultLegColor,
		},
		"accessory_slots": []fiber.Map{
			{"name": "hat", "index": 0, "display_name": "Hats"},
			{"name": "hair", "index": 1, "display_name": "Hair"},
			{"name": "face", "index": 2, "display_name": "Face"},
			{"name": "back", "index": 3, "display_name": "Back"},
			{"name": "front", "index": 4, "display_name": "Front"},
			{"name": "waist", "index": 5, "display_name": "Waist"},
			{"name": "neck", "index": 6, "display_name": "Neck"},
		},
		"clothing_target_parts": fiber.Map{
			"shirt": []string{"torso", "left_arm", "right_arm"},
			"pants": []string{"left_leg", "right_leg"},
		},
	}
}

func getCurrentUser(c fiber.Ctx) *models.User {
	if service.User == nil {
		return nil
	}
	if id := userIDFromAuthHeader(c.Get("Authorization")); id > 0 {
		if u, err := service.User.GetUserByID(id); err == nil && u != nil {
			return u
		}
	}
	if id := userIDFromAuthHeader(c.Cookies("Buildoverse_access_token")); id > 0 {
		if u, err := service.User.GetUserByID(id); err == nil && u != nil {
			return u
		}
	}
	username := GetActiveUser(c)
	if username == "" {
		return nil
	}
	user, _ := service.User.GetUserByUsername(username)
	return user
}

// Helper: generate random token
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// ============================================================
// AUTH HANDLERS (JSON API)
// ============================================================

func AuthIndex(c fiber.Ctx) error { return c.Redirect().To("/auth/login") }

func LoginPostAPI(c fiber.Ctx) error {
	identifier := c.FormValue("username")
	if identifier == "" {
		identifier = c.FormValue("identifier")
	}
	password := c.FormValue("password")

	if identifier == "" || password == "" {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err == nil && body != nil {
			identifier, _ = body["username"].(string)
			if identifier == "" {
				identifier, _ = body["identifier"].(string)
			}
			password, _ = body["password"].(string)
		}
	}

	if identifier == "" || password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password are required"})
	}

	user, err := service.Auth.AuthenticateUser(identifier, password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password"})
	}

	// Generate access and refresh tokens
	accessToken := generateToken()
	refreshToken := generateToken()

	// Store tokens in session and set cookies
	sess := session.FromContext(c)
	if sess != nil {
		sess.Set("username", user.Username)
		sess.Set("access_token", accessToken)
		sess.Set("refresh_token", refreshToken)
	}

	SetHashedCookie(c, user.ID)

	// Set localStorage-style tokens as cookies for the SPA
	c.Cookie(&fiber.Cookie{
		Name:     "Buildoverse_access_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "Buildoverse_refresh_token",
		Value:    refreshToken,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	})

	rememberToken(accessToken, user.ID, "access")
	rememberToken(refreshToken, user.ID, "refresh")

	return c.JSON(fiber.Map{
		"user":          spaUser(user),
		"token":         accessToken,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func LogoutAPI(c fiber.Ctx) error {
	sess := session.FromContext(c)
	if sess != nil {
		_ = sess.Destroy()
	}

	// Clear all auth cookies
	for _, name := range []string{"Buildoverse_remember", "Buildoverse_access_token", "Buildoverse_refresh_token"} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-24 * time.Hour),
			HTTPOnly: true,
			Secure:   config.Global.SessionSecure,
			SameSite: config.Global.SessionSameSite,
		})
	}

	return c.JSON(fiber.Map{"ok": true})
}

func RegisterPostAPI(c fiber.Ctx) error {
	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")
	displayname := c.FormValue("display_name")
	if username == "" || email == "" || password == "" {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err == nil && body != nil {
			if username == "" {
				username, _ = body["username"].(string)
			}
			if email == "" {
				email, _ = body["email"].(string)
			}
			if password == "" {
				password, _ = body["password"].(string)
			}
			if displayname == "" {
				displayname, _ = body["display_name"].(string)
			}
		}
	}

	if displayname == "" {
		displayname = username
	}

	if username == "" || email == "" || password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "All fields are required"})
	}

	// Use Auth service to register user
	var newUser *models.User
	var err error
	if service.Auth != nil {
		newUser, err = service.Auth.RegisterUser(username, displayname, email, password, password)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
	}

	if newUser == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
	}

	// Generate tokens
	accessToken := generateToken()
	refreshToken := generateToken()

	sess := session.FromContext(c)
	if sess != nil {
		sess.Set("username", newUser.Username)
		sess.Set("access_token", accessToken)
		sess.Set("refresh_token", refreshToken)
	}
	setSignupFlash(c, newUser.Username)
	issueEmailVerification(newUser)

	SetHashedCookie(c, newUser.ID)

	c.Cookie(&fiber.Cookie{
		Name:     "Buildoverse_access_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "Buildoverse_refresh_token",
		Value:    refreshToken,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	})

	rememberToken(accessToken, newUser.ID, "access")
	rememberToken(refreshToken, newUser.ID, "refresh")

	return c.JSON(fiber.Map{
		"user":          spaUser(newUser),
		"token":         accessToken,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"message":       "Welcome to Buildoverse. Your avatar is ready in the editor.",
	})
}

func AuthRefresh(c fiber.Ctx) error {
	refreshToken := c.FormValue("refresh_token")
	if refreshToken == "" {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err == nil && body != nil {
			refreshToken, _ = body["refresh_token"].(string)
		}
	}
	if refreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Refresh token required"})
	}

	rec, ok := lookupToken(refreshToken)
	if !ok || rec.UserID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
	}

	accessToken := generateToken()
	newRefreshToken := generateToken()
	rememberToken(accessToken, rec.UserID, "access")
	rememberToken(newRefreshToken, rec.UserID, "refresh")
	if sess := session.FromContext(c); sess != nil {
		sess.Set("access_token", accessToken)
		sess.Set("refresh_token", newRefreshToken)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "Buildoverse_access_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "Buildoverse_refresh_token",
		Value:    newRefreshToken,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	})

	return c.JSON(fiber.Map{
		"token":         accessToken,
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

func AuthForgotPassword(c fiber.Ctx) error {
	return notReady(c)
}
func AuthResetPassword(c fiber.Ctx) error {
	return notReady(c)
}

func AuthTicket(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	ticket := generateToken()
	rememberToken(ticket, user.ID, "ticket")
	return c.JSON(fiber.Map{"ticket": ticket})
}
func AuthConfirmEmailChange(c fiber.Ctx) error { return notReady(c) }
func AuthStudio(c fiber.Ctx) error             { return c.Redirect().To("https://buildoverse.com/studio") }

// ============================================================
// ME / CURRENT USER HANDLERS
// ============================================================

func MeHandler(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return c.JSON(fiber.Map{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"email":        user.Mail,
		"bits":         user.Bits,
		"bucks":        user.Bucks,
		"created_at":   user.CreationDate.Unix(),
	})
}

func MeProfile(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return renderUserProfile(c, user)
}

func MeBootstrap(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	rules := fiber.Map{}
	if raw, err := os.ReadFile("data/avatar-rules.json"); err == nil {
		_ = json.Unmarshal(raw, &rules)
	}
	return c.JSON(fiber.Map{
		"id":                       user.ID,
		"username":                 user.Username,
		"display_name":             user.DisplayName,
		"email":                    user.Mail,
		"role":                     spaRole(user),
		"status":                   1,
		"balance":                  user.Bits,
		"tier":                     "free",
		"created_at":               user.CreationDate.Unix(),
		"email_verified":           true,
		"mfa_enabled":              false,
		"has_password":             true,
		"creator_blacklisted":      false,
		"active_creator_blacklist": false,
		"grants":                   0,
		"headshot_url":             fmt.Sprintf("https://api.buildoverse.com/avatar/headshots/%d", user.ID),
		"permissions":              fiber.Map{},
		"counters": fiber.Map{
			"trades_pending_received": 0,
			"trades_pending_sent":     0,
			"friends_pending":         0,
			"appeals_pending":         0,
			"unread_messages":         0,
			"unread_official":         0,
			"unread_notifications":    0,
		},
		"moderation": fiber.Map{
			"active_game_moderation":    false,
			"active_content_moderation": false,
		},
		"avatar_rules": defaultAvatarRules(),
	})
}

func MeBalance(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return c.JSON(fiber.Map{
		"balance": user.Bits,
		"bits":    user.Bits,
		"bucks":   user.Bucks,
	})
}

func MeDeletions(c fiber.Ctx) error             { return c.JSON([]interface{}{}) }
func MeEnforcement(c fiber.Ctx) error           { return c.JSON(fiber.Map{}) }
func MeAppeals(c fiber.Ctx) error               { return c.JSON([]interface{}{}) }
func MeAppealDetail(c fiber.Ctx) error          { return notReady(c) }
func MeAppealMessages(c fiber.Ctx) error        { return c.JSON([]interface{}{}) }
func MeSuspensionAcknowledge(c fiber.Ctx) error { return notReady(c) }
func MeSuspensionPending(c fiber.Ctx) error     { return notReady(c) }
func MeGameModerationAck(c fiber.Ctx) error     { return notReady(c) }
func MeGroupModerationAck(c fiber.Ctx) error    { return notReady(c) }
func MeItemModerationAck(c fiber.Ctx) error     { return notReady(c) }
func MFAStatus(c fiber.Ctx) error               { return c.JSON(fiber.Map{"enabled": false}) }
func MFASetup(c fiber.Ctx) error                { return notReady(c) }
func MFAVerify(c fiber.Ctx) error               { return notReady(c) }
func MFADisable(c fiber.Ctx) error              { return notReady(c) }
func MFARecoveryCodes(c fiber.Ctx) error        { return notReady(c) }
func MeOfficialMessagesRead(c fiber.Ctx) error {
	return notReady(c)
}
func MeDailyRewards(c fiber.Ctx) error {
	return notReady(c)
}
func MeTransfers(c fiber.Ctx) error         { return c.JSON([]interface{}{}) }
func MeTradesCounts(c fiber.Ctx) error      { return c.JSON(fiber.Map{"incoming": 0, "outgoing": 0}) }
func MeProfileBackground(c fiber.Ctx) error { return c.JSON(fiber.Map{"background_url": ""}) }
func MeProfileBanner(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"banner_url": "", "pending_url": ""})
}
func MeProfilePicture(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.JSON(fiber.Map{"profile_picture": ""})
	}
	return c.JSON(fiber.Map{"profile_picture": fmt.Sprintf("/avatar/headshots/%d", user.ID)})
}
func MePublish(c fiber.Ctx) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Publishing not implemented"})
}
func MeIgnores(c fiber.Ctx) error { return c.JSON([]interface{}{}) }
func MeWarningsAcknowledge(c fiber.Ctx) error {
	return notReady(c)
}
func MeReportsDetail(c fiber.Ctx) error { return MyReportDetail(c) }
func MeTicketsDetail(c fiber.Ctx) error { return notReady(c) }
func MeCreatorBlacklistAck(c fiber.Ctx) error {
	return notReady(c)
}
func MePlacesAgeQuestionnaire(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "unrated", "review_id": nil, "pending_url": nil})
}
func MePlacesRatingReview(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "unrated"}) }
func SettingsPost(c fiber.Ctx) error         { return SettingsPutCompat(c) }

// ============================================================
// USER HANDLERS
// ============================================================

func SearchPage(c fiber.Ctx) error { return SearchRedirect(c) }

func UserProfileAPI(c fiber.Ctx) error {
	idStr := c.Params("id")

	var user *models.User
	if service.User != nil {
		// Try numeric ID first
		if id, err := strconv.Atoi(idStr); err == nil {
			user, _ = service.User.GetUserByID(id)
		}
		// Try username lookup
		if user == nil {
			user, _ = service.User.GetUserByUsername(idStr)
		}
	}

	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return renderUserProfile(c, user)
}

func UsernamesHistory(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil || userID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if service.User == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User service unavailable"})
	}
	user, err := service.User.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user"})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	items, total, err := service.User.GetOldUsernames(userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load username history"})
	}
	return c.JSON(fiber.Map{"items": items, "total": total, "limit": limit, "offset": offset})
}

func renderUserProfile(c fiber.Ctx, user *models.User) error {
	currentUser := getCurrentUser(c)
	isOwner := currentUser != nil && currentUser.ID == user.ID
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Avatar service unavailable"})
	}
	avatarData, err := service.Avatar.GetAvatar(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user avatar"})
	}
	equipped, err := service.Avatar.GetEquippedItems(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load equipped avatar items"})
	}
	equippedItems := make([]fiber.Map, 0, len(equipped))
	for _, item := range equipped {
		entry := inventoryItemResponse(item)
		assetType, slot := profileAssetType(item.Type)
		entry["asset_type"] = assetType
		entry["item_id"] = item.ID
		entry["slots"] = []string{}
		if slot != "" {
			entry["slots"] = []string{slot}
		}
		entry["thumbnail_url"] = fmt.Sprintf("https://api.buildoverse.com/static/renders/shop/%s/%d", item.Type, item.ID)
		equippedItems = append(equippedItems, entry)
	}
	hasAvatarBundle := false
	for _, item := range equippedItems {
		if item["asset_type"] == "avatar_bundle" {
			hasAvatarBundle = true
			break
		}
	}
	if !hasAvatarBundle {
		defaultBundle, ok := findCatalogItem(models.DefaultBundleItemID)
		if !ok {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Default avatar bundle unavailable"})
		}
		bundle := catalogItemResponse(defaultBundle)
		equippedItems = append([]fiber.Map{{
			"item_id":       models.DefaultBundleItemID,
			"name":          bundle["name"],
			"asset_type":    "avatar_bundle",
			"thumbnail_url": bundle["thumbnail_url"],
			"slots":         []string{"torso", "head", "left_arm", "right_arm", "left_leg", "right_leg"},
			"variant_index": 0,
		}}, equippedItems...)
	}

	friendCount := 0
	if service.User != nil {
		_, _, friendCount, _, _, err = service.User.GetProfileData(user.ID, "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user profile"})
		}
	}
	placeCount := 0
	if service.Game != nil {
		_, placeCount, err = service.Game.GetGamesByCreatorID(user.ID, 1, 0)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user places"})
		}
	}
	items, err := service.Avatar.GetInventory(user.ID, "", "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user inventory"})
	}
	inventoryAllowed, err := profileListAllowed(c, user.ID, "inventory_visibility")
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Could not verify profile privacy"})
	}
	itemCount := len(items)
	if !inventoryAllowed {
		itemCount = 0
	}
	lastActive := any(nil)
	if !user.LastOnline.IsZero() {
		lastActive = user.LastOnline.Unix()
	}
	bio := strings.TrimSpace(user.Description)
	if bio == "" || strings.EqualFold(bio, "I am new to Buildoverse!") {
		bio = "I am new to Buildoverse!"
	}
	version, err := avatarRenderVersion(avatarData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not version avatar render"})
	}
	headshotURL := fmt.Sprintf("https://api.buildoverse.com/avatar/headshots/%d?v=%s", user.ID, version)

	return c.JSON(fiber.Map{
		"user_id":          user.ID,
		"username":         user.Username,
		"display_name":     user.DisplayName,
		"visibility":       "public",
		"bio":              bio,
		"status":           nil,
		"avatar":           profileAvatarColors(avatarData),
		"avatar_url":       headshotURL,
		"headshot_url":     headshotURL,
		"equipped_items":   equippedItems,
		"badges":           []any{},
		"friend_count":     friendCount,
		"place_count":      placeCount,
		"item_count":       itemCount,
		"join_policy":      "everyone",
		"is_owner":         isOwner,
		"accent_color":     nil,
		"banner_url":       "",
		"background_url":   "",
		"member_since":     user.CreationDate.Unix(),
		"last_active":      lastActive,
		"inventory_hidden": !inventoryAllowed,
	})
}

func avatarRenderVersion(avatar *models.Avatar) (string, error) {
	data, err := json.Marshal(avatar)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte("avatar-render-v2:"), data...))
	return hex.EncodeToString(sum[:]), nil
}

func profileAvatarColors(avatar *models.Avatar) fiber.Map {
	if avatar == nil {
		return getUserAvatar(0)
	}
	color := func(value string) string {
		return "#" + strings.TrimPrefix(value, "#")
	}
	return fiber.Map{
		"head_color":      color(avatar.HeadColor),
		"torso_color":     color(avatar.TorsoColor),
		"left_arm_color":  color(avatar.LArmColor),
		"right_arm_color": color(avatar.RArmColor),
		"left_leg_color":  color(avatar.LLegColor),
		"right_leg_color": color(avatar.RLegColor),
	}
}

func profileAssetType(itemType string) (string, string) {
	switch itemType {
	case "hat":
		return "accessory", "hat"
	case "shirts":
		return "flat_shirt", ""
	case "pants":
		return "flat_pant", ""
	case "tshirts":
		return "tshirt", ""
	case "faces":
		return "face", ""
	case "gear":
		return "mesh", ""
	default:
		return itemType, ""
	}
}
func UserProfilePage(c fiber.Ctx) error { return UserProfileAPI(c) }
func UserAvatarFullBody(c fiber.Ctx) error {
	return AvatarGet(c)
}
func UserPresence(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "offline"})
}
func UserJoinPlace(c fiber.Ctx) error { return notReady(c) }

func getUserAvatar(userID int) fiber.Map {
	return fiber.Map{
		"head_color":      models.DefaultHeadColor,
		"torso_color":     models.DefaultTorsoColor,
		"left_arm_color":  models.DefaultArmColor,
		"right_arm_color": models.DefaultArmColor,
		"left_leg_color":  models.DefaultLegColor,
		"right_leg_color": models.DefaultLegColor,
	}
}

// ============================================================
// PLACES / GAMES HANDLERS
// ============================================================

func PlacesList(c fiber.Ctx) error {
	limitStr := c.Query("limit", "20")
	offsetStr := c.Query("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var games []*models.Game
	if service.Game != nil {
		games, _ = service.Game.GetPopularGames(limit + offset)
	}

	places := make([]fiber.Map, 0)
	for i, g := range games {
		if i < offset {
			continue
		}
		if len(places) >= limit {
			break
		}
		creatorID := 0
		if g.CreatorID.Valid {
			creatorID = int(g.CreatorID.Int64)
		}
		places = append(places, fiber.Map{
			"id":             g.ID,
			"title":          g.Name,
			"description":    "",
			"owner_id":       creatorID,
			"owner_username": getUsernameByID(creatorID),
			"max_players":    50,
			"visit_count":    g.Visits,
			"thumbs_up":      0,
			"thumbs_down":    0,
			"active_players": g.Playing,
			"created_at":     g.CreatedAt.Unix(),
			"updated_at":     g.EditedAt.Unix(),
			"thumbnail_url":  fmt.Sprintf("https://api.buildoverse.com/thumbnails/%d.png", g.ID),
			"access":         "public",
		})
	}

	return c.JSON(fiber.Map{
		"places": places,
		"total":  len(places),
		"limit":  limit,
		"offset": offset,
	})
}

func PlacesDiscovery(c fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	surface := c.Query("surface", "discover")
	if surface != "home" && surface != "discover" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid discovery surface"})
	}

	var games []*models.Game
	if service.Game != nil {
		games, _ = service.Game.GetPopularGames(limit)
	}

	places := make([]fiber.Map, 0, len(games))
	for _, g := range games {
		creatorID := 0
		if g.CreatorID.Valid {
			creatorID = int(g.CreatorID.Int64)
		}
		places = append(places, fiber.Map{
			"id":             g.ID,
			"title":          g.Name,
			"description":    "",
			"owner_id":       creatorID,
			"owner_username": getUsernameByID(creatorID),
			"max_players":    50,
			"visit_count":    g.Visits,
			"thumbs_up":      0,
			"thumbs_down":    0,
			"active_players": g.Playing,
			"created_at":     g.CreatedAt.Unix(),
			"updated_at":     g.EditedAt.Unix(),
			"thumbnail_url":  fmt.Sprintf("https://api.buildoverse.com/thumbnails/%d.png", g.ID),
			"access":         "public",
		})
	}

	return c.JSON(fiber.Map{
		"surface": surface,
		"shelves": []fiber.Map{
			{
				"id":       "popular",
				"title":    "Popular Games",
				"subtitle": "Most played right now",
				"places":   places,
			},
		},
	})
}

func PlaceDetail(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Place not found"})
	}

	if service.Game == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Game service unavailable"})
	}
	game, err := service.Game.GetGameByID(id)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Place not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load place"})
	}

	creatorID := 0
	if game.CreatorID.Valid {
		creatorID = int(game.CreatorID.Int64)
	}

	return c.JSON(fiber.Map{
		"id":             game.ID,
		"title":          game.Name,
		"description":    game.Description.String,
		"tags":           placeTagMaps(game.Tags),
		"owner_id":       creatorID,
		"owner_username": getUsernameByID(creatorID),
		"max_players":    50,
		"visit_count":    game.Visits,
		"thumbs_up":      0,
		"thumbs_down":    0,
		"active_players": game.Playing,
		"created_at":     game.CreatedAt.Unix(),
		"updated_at":     game.EditedAt.Unix(),
		"thumbnail_url":  fmt.Sprintf("https://api.buildoverse.com/thumbnails/%d.png", game.ID),
		"access":         "public",
		"can_edit":       false,
	})
}

func placeTagMaps(raw sql.NullString) []fiber.Map {
	result := make([]fiber.Map, 0)
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return result
	}

	var objects []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw.String), &objects); err == nil && objects != nil {
		for _, tag := range objects {
			if name := strings.TrimSpace(tag.Name); name != "" {
				result = append(result, fiber.Map{"name": name})
			}
		}
		return result
	}

	var names []string
	if err := json.Unmarshal([]byte(raw.String), &names); err != nil {
		names = strings.Split(raw.String, ",")
	}
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			result = append(result, fiber.Map{"name": name})
		}
	}
	return result
}

func PlaceServers(c fiber.Ctx) error {
	return c.JSON([]fiber.Map{})
}

func PlaceJoin(c fiber.Ctx) error {
	return notReady(c)
}

func PlaceRating(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"thumbs_up": 0, "thumbs_down": 0, "user_rating": nil})
}
func PlaceRate(c fiber.Ctx) error  { return notReady(c) }
func PlaceWorld(c fiber.Ctx) error { return notReady(c) }
func GameThumbnail(c fiber.Ctx) error {
	return sendGameThumbnail(c, c.Params("id"))
}

func PlaceThumbnail(c fiber.Ctx) error {
	return sendGameThumbnail(c, c.Params("id"))
}

func sendGameThumbnail(c fiber.Ctx, rawID string) error {
	id, err := strconv.Atoi(rawID)
	if err != nil || id < 1 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	c.Set("Access-Control-Allow-Origin", "*")
	thumbnailPath := fmt.Sprintf("static/thumbnails/%d.png", id)
	if _, err := os.Stat(thumbnailPath); err == nil {
		return c.SendFile(thumbnailPath)
	} else if !os.IsNotExist(err) {
		return err
	}

	return c.SendFile("static/useful/icons/gm.png")
}
func PlaceUpNext(c fiber.Ctx) error             { return c.JSON(fiber.Map{"places": []interface{}{}}) }
func PlaceBans(c fiber.Ctx) error               { return c.JSON([]interface{}{}) }
func PlaceProblems(c fiber.Ctx) error           { return c.JSON([]interface{}{}) }
func PlaceProductPurchase(c fiber.Ctx) error    { return notReady(c) }
func PlacePaidAccessPurchase(c fiber.Ctx) error { return notReady(c) }
func PlaceTransfers(c fiber.Ctx) error          { return c.JSON([]interface{}{}) }
func PlaceTransferAccept(c fiber.Ctx) error     { return notReady(c) }
func PlaceTransferReject(c fiber.Ctx) error     { return notReady(c) }

func ApiDiscoverRows(c fiber.Ctx) error {
	id := c.Params("id")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 || limit > 60 {
		limit = 20
	}

	var games []*models.Game
	if service.Game != nil {
		games, _ = service.Game.GetPopularGames(limit)
	}

	places := make([]fiber.Map, 0, len(games))
	for _, g := range games {
		creatorID := 0
		if g.CreatorID.Valid {
			creatorID = int(g.CreatorID.Int64)
		}
		places = append(places, fiber.Map{
			"id":             g.ID,
			"title":          g.Name,
			"description":    "",
			"owner_id":       creatorID,
			"owner_username": getUsernameByID(creatorID),
			"max_players":    50,
			"visit_count":    g.Visits,
			"thumbs_up":      0,
			"thumbs_down":    0,
			"active_players": g.Playing,
			"created_at":     g.CreatedAt.Unix(),
			"updated_at":     g.EditedAt.Unix(),
			"thumbnail_url":  fmt.Sprintf("https://api.buildoverse.com/thumbnails/%d.png", g.ID),
		})
	}

	return c.JSON(fiber.Map{
		"row": fiber.Map{
			"id":          id,
			"title":       "Popular Games",
			"subtitle":    "Most played",
			"description": "",
		},
		"places":      places,
		"next_cursor": "",
	})
}

func CreatePlacesPage(c fiber.Ctx) error { return c.Redirect().To("/create?type=game") }

func GamesPage(c fiber.Ctx) error { return PlayPage(c) }

// ============================================================
// GROUPS HANDLERS
// ============================================================

func GroupsPage(c fiber.Ctx) error {
	if wantsJSON(c) || c.Query("q") != "" || c.Query("sort") != "" || c.Query("limit") != "" || c.Query("offset") != "" {
		return GroupsAPI(c)
	}
	return c.Redirect().To("https://buildoverse.com/groups")
}
func GroupsBrowse(c fiber.Ctx) error       { return GroupsAPI(c) }
func GroupAnnouncements(c fiber.Ctx) error { return c.JSON([]interface{}{}) }
func GroupAnnouncementDetail(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupAnnouncementReaction(c fiber.Ctx) error {
	return notReady(c)
}
func GroupAnnouncementReplies(c fiber.Ctx) error { return c.JSON([]interface{}{}) }
func GroupAnnouncementReplyDetail(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupRoles(c fiber.Ctx) error { return c.JSON([]interface{}{}) }
func GroupRoleDetail(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupRolesReorder(c fiber.Ctx) error { return notReady(c) }
func GroupMemberRole(c fiber.Ctx) error   { return notReady(c) }
func GroupInvites(c fiber.Ctx) error      { return c.JSON([]interface{}{}) }
func GroupKick(c fiber.Ctx) error         { return notReady(c) }
func GroupBans(c fiber.Ctx) error         { return c.JSON([]interface{}{}) }
func GroupBan(c fiber.Ctx) error          { return notReady(c) }
func GroupWall(c fiber.Ctx) error         { return c.JSON([]interface{}{}) }
func GroupWallPost(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupPayouts(c fiber.Ctx) error { return c.JSON([]interface{}{}) }
func GroupStoreItem(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupStoreItemTransfer(c fiber.Ctx) error { return notReady(c) }
func GroupTransfer(c fiber.Ctx) error          { return c.JSON([]interface{}{}) }
func GroupTransferPending(c fiber.Ctx) error   { return c.JSON([]interface{}{}) }
func GroupTransferPost(c fiber.Ctx) error      { return notReady(c) }
func GroupBanner(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupIcon(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GroupTransferAccept(c fiber.Ctx) error { return notReady(c) }
func GroupTransferReject(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// FRIENDS HANDLERS
// ============================================================

func FriendDetail(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func FriendSend(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// AVATAR HANDLERS
// ============================================================

func AvatarFullBody(c fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		user := getCurrentUser(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		idStr = fmt.Sprintf("%d", user.ID)
	}
	id, _ := strconv.Atoi(idStr)
	return c.JSON(fiber.Map{"id": id, "url": fmt.Sprintf("/avatar/full-body/%d", id)})
}

func AvatarHeadshotPage(c fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		user := getCurrentUser(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		idStr = fmt.Sprintf("%d", user.ID)
	}
	id, _ := strconv.Atoi(idStr)
	return c.JSON(fiber.Map{"id": id, "url": fmt.Sprintf("/avatar/headshots/%d", id)})
}

// ============================================================
// INVENTORY / SHOP HANDLERS
// ============================================================

func InventoryPage(c fiber.Ctx) error { return notReady(c) }

func InventoryAPI(c fiber.Ctx) error {
	return MeInventory(c)
}

func ShopGetCompat(c fiber.Ctx) error {
	return ShopItemsAPI(c)
}

func ShopCategories(c fiber.Ctx) error {
	raw, err := os.ReadFile("data/store-categories.json")
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Store categories unavailable"})
	}
	c.Set("Content-Type", "application/json")
	return c.Send(raw)
}

func StoreAvatarRules(c fiber.Ctx) error {
	return c.JSON(defaultAvatarRules())
}
func StoreItemDetail(c fiber.Ctx) error       { return CatalogItem(c) }
func StoreBuy(c fiber.Ctx) error              { return notReady(c) }
func StoreBuyRequests(c fiber.Ctx) error      { return c.JSON([]interface{}{}) }
func StoreBuyRequestCancel(c fiber.Ctx) error { return notReady(c) }
func StoreBuyRequestFill(c fiber.Ctx) error   { return notReady(c) }
func StoreTradeDetail(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func StoreRating(c fiber.Ctx) error       { return c.JSON(fiber.Map{"thumbs_up": 0, "thumbs_down": 0}) }
func StoreRate(c fiber.Ctx) error         { return notReady(c) }
func StoreResells(c fiber.Ctx) error      { return c.JSON([]interface{}{}) }
func StoreResellBuy(c fiber.Ctx) error    { return notReady(c) }
func StoreResellCancel(c fiber.Ctx) error { return notReady(c) }
func StoreResellGift(c fiber.Ctx) error   { return notReady(c) }
func StoreGift(c fiber.Ctx) error         { return notReady(c) }

// ============================================================
// MESSAGES / CONVERSATIONS
// ============================================================

func MessagesPage(c fiber.Ctx) error { return notReady(c) }
func ConversationsList(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"conversations": []any{}, "total": 0})
}
func ConversationMembers(c fiber.Ctx) error       { return c.JSON([]interface{}{}) }
func ConversationMembersMe(c fiber.Ctx) error     { return c.JSON(fiber.Map{}) }
func ConversationMessages(c fiber.Ctx) error      { return c.JSON([]interface{}{}) }
func ConversationRead(c fiber.Ctx) error          { return notReady(c) }
func ConversationTyping(c fiber.Ctx) error        { return notReady(c) }
func ConversationTitle(c fiber.Ctx) error         { return notReady(c) }
func ConversationTransferAdmin(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// TRADES
// ============================================================

func TradePage(c fiber.Ctx) error  { return c.Redirect().To("https://buildoverse.com/trade") }
func TradesList(c fiber.Ctx) error { return c.JSON(fiber.Map{"trades": []any{}, "total": 0}) }
func TradeDetail(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func TradeAccept(c fiber.Ctx) error  { return notReady(c) }
func TradeDecline(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// SETTINGS
// ============================================================

func SettingsParentalPin(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// ITEMS / MARKETPLACE
// ============================================================

func ItemsPage(c fiber.Ctx) error          { return notReady(c) }
func ItemsFavorite(c fiber.Ctx) error      { return notReady(c) }
func ItemsWishlist(c fiber.Ctx) error      { return notReady(c) }
func MarketplacePage(c fiber.Ctx) error    { return notReady(c) }
func DeveloperStorePage(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// NOTIFICATIONS
// ============================================================

func NotificationsPage(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// MISC PAGES
// ============================================================

func AnnouncementsPage(c fiber.Ctx) error  { return notReady(c) }
func EventsPage(c fiber.Ctx) error         { return notReady(c) }
func FeaturedEvents(c fiber.Ctx) error     { return c.JSON([]interface{}{}) }
func FeaturedEventClick(c fiber.Ctx) error { return notReady(c) }
func LeaderboardRAP(c fiber.Ctx) error     { return notReady(c) }
func TransactionsPage(c fiber.Ctx) error   { return notReady(c) }
func TicketsPage(c fiber.Ctx) error        { return notReady(c) }
func TicketAttachments(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func AppealsPage(c fiber.Ctx) error { return notReady(c) }
func AppealAttachments(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func ReportsPage(c fiber.Ctx) error       { return notReady(c) }
func BansPage(c fiber.Ctx) error          { return notReady(c) }
func BlockPage(c fiber.Ctx) error         { return notReady(c) }
func LegalPage(c fiber.Ctx) error         { return notReady(c) }
func SafetyPage(c fiber.Ctx) error        { return notReady(c) }
func GuidelineSections(c fiber.Ctx) error { return GuidelineSectionsAPI(c) }
func EmojiWhitelist(c fiber.Ctx) error    { return c.JSON([]string{}) }
func PromocodesPage(c fiber.Ctx) error    { return notReady(c) }

func SiteBanners(c fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{
			"id":        1,
			"title":     "Welcome to Buildoverse",
			"subtitle":  "Create, play, and explore",
			"image_url": "/banners/welcome.png",
			"link":      "/",
		},
	})
}

func SiteBannerClick(c fiber.Ctx) error { return notReady(c) }
func SitePromos(c fiber.Ctx) error      { return c.JSON([]interface{}{}) }

func CDNResolve(c fiber.Ctx) error {
	var body []json.RawMessage
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	result := make(map[string]string)
	for _, rawID := range body {
		var id string
		if err := json.Unmarshal(rawID, &id); err != nil {
			id = string(rawID)
		}
		numericID, err := strconv.ParseUint(id, 10, 64)
		if err != nil || numericID == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid asset ID"})
		}
		result[id] = fmt.Sprintf("/assets/%s", id)
	}

	return c.JSON(result)
}

func LegalCurrent(c fiber.Ctx) error {
	accepted := false
	if user := getCurrentUser(c); user != nil {
		settings, err := readUserSettings(user.ID)
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Legal documents unavailable"})
		}
		version, _ := settings["accepted_legal_version"].(string)
		accepted = version == currentLegalVersion
	}

	return c.JSON(fiber.Map{
		"documents": []fiber.Map{
			legalDocument("terms-of-service", "terms_of_service", "/terms", "2026-10-06", accepted),
			legalDocument("privacy-policy", "privacy_policy", "/privacy", "2026-07-29", accepted),
			legalDocument("community-guidelines", "guidelines", "/community-guidelines", "2026-09-21", accepted),
			legalDocument("creator-guidelines", "guidelines", "/creator-guidelines", "2026-10-06", accepted),
		},
	})
}

func legalDocument(id, kind, url, publishedDate string, accepted bool) fiber.Map {
	publishedAt, _ := time.Parse("2006-01-02", publishedDate)
	return fiber.Map{
		"id":           id,
		"kind":         kind,
		"url":          url,
		"published_at": publishedAt.Unix(),
		"effective_at": publishedAt.Unix(),
		"accepted":     accepted,
	}
}

func TicketCategories(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"categories": []fiber.Map{
			{"code": "account", "label": "Account"},
			{"code": "billing", "label": "Billing and purchases"},
			{"code": "moderation", "label": "Safety and moderation"},
			{"code": "technical", "label": "Technical issue"},
			{"code": "other", "label": "Other"},
		},
	})
}

func EnforcementEvidence(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
func GiftAccept(c fiber.Ctx) error       { return notReady(c) }
func GiftReject(c fiber.Ctx) error       { return notReady(c) }
func BuyRequestCancel(c fiber.Ctx) error { return notReady(c) }

// ============================================================
// USER SEARCH & SPECIFIC ENDPOINTS
// ============================================================

func UsersSearch(c fiber.Ctx) error {
	query := c.Query("q", "")
	limitStr := c.Query("limit", "20")
	offsetStr := c.Query("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var users []*models.User
	if service.User != nil && query != "" {
		users, _ = service.User.SearchUsers(query, limit+offset)
	}

	result := make([]fiber.Map, 0)
	for i, u := range users {
		if i < offset {
			continue
		}
		if len(result) >= limit {
			break
		}
		result = append(result, fiber.Map{
			"id":           u.ID,
			"username":     u.Username,
			"display_name": u.DisplayName,
			"head_color":   models.DefaultHeadColor,
			"torso_color":  models.DefaultTorsoColor,
		})
	}

	total := len(users)

	return c.JSON(fiber.Map{
		"users":  result,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func UserByUsernameProfile(c fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username required"})
	}

	var user *models.User
	if service.User != nil {
		var err error
		user, err = service.User.GetUserByUsername(username)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
	}

	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return renderUserProfile(c, user)
}

func UserInventory(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil || userID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if service.User == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User service unavailable"})
	}
	target, err := service.User.GetUserByID(userID)
	if errors.Is(err, sql.ErrNoRows) || target == nil && err == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user inventory"})
	}
	allowed, err := profileListAllowed(c, userID, "inventory_visibility")
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Could not verify profile privacy"})
	}
	if !allowed {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "This user's inventory is private"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Inventory service unavailable"})
	}

	limit := parsePageSize(c.Query("limit"), 100)
	offset := parsePageOffset(c.Query("offset"))
	items, err := service.Avatar.GetInventory(userID, inventoryCategory(c), "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user inventory"})
	}
	page := itemsOnPage(items, offset, limit)
	result := make([]fiber.Map, 0, len(page))
	for _, item := range page {
		result = append(result, inventoryItemResponse(item))
	}
	return c.JSON(fiber.Map{"items": result, "total": len(items), "limit": limit, "offset": offset})
}

func UserPlaces(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil || userID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	return creatorPlaces(c, userID)
}

// ============================================================
// PRESENCE
// ============================================================

func UsersPresence(c fiber.Ctx) error {
	idsStr := c.Query("ids", "")
	if idsStr == "" {
		return c.JSON(fiber.Map{"presence": []interface{}{}})
	}

	ids := make([]fiber.Map, 0)
	return c.JSON(fiber.Map{"presence": ids})
}

func RecentlyPlayed(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"places": []interface{}{}, "total": 0})
}

func MePlaces(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return creatorPlaces(c, user.ID)
}

func creatorPlaces(c fiber.Ctx, creatorID int) error {
	if service.Game == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Game service unavailable"})
	}
	limit := parsePageSize(c.Query("limit"), 20)
	offset := parsePageOffset(c.Query("offset"))
	games, total, err := service.Game.GetGamesByCreatorID(creatorID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load creator's games"})
	}
	places := make([]fiber.Map, 0, len(games))
	for _, game := range games {
		places = append(places, fiber.Map{
			"id":            game.ID,
			"title":         game.Name,
			"description":   game.Description.String,
			"visit_count":   game.Visits,
			"thumbnail_url": fmt.Sprintf("https://api.buildoverse.com/thumbnails/%d.png", game.ID),
		})
	}
	return c.JSON(fiber.Map{"places": places, "total": total, "limit": limit, "offset": offset})
}

func GroupPlaces(c fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var games []*models.Game
	if service.Game != nil {
		games, _ = service.Game.GetPopularGames(limit + offset)
	}

	places := make([]fiber.Map, 0)
	for i, g := range games {
		if i < offset {
			continue
		}
		if len(places) >= limit {
			break
		}
		places = append(places, fiber.Map{
			"id":            g.ID,
			"title":         g.Name,
			"description":   "",
			"visit_count":   g.Visits,
			"thumbnail_url": fmt.Sprintf("https://api.buildoverse.com/thumbnails/%d.png", g.ID),
		})
	}

	return c.JSON(fiber.Map{
		"places": places,
		"total":  len(places),
		"limit":  limit,
		"offset": offset,
	})
}

// ============================================================
// SSE EVENTS STREAM
// ============================================================

func EventsStream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Return a simple ping immediately, then let the client reconnect
	c.SendString("data: {\"type\":\"ping\"}\n\n")
	return nil
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func getUsernameByID(id int) string {
	if service.User == nil || id == 0 {
		return "Unknown"
	}
	user, err := service.User.GetUserByID(id)
	if err != nil || user == nil {
		return "Unknown"
	}
	return user.Username
}

func NotFoundHandlerCompat(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Endpoint not found"})
}

// ============================================================
// API V1 HANDLERS
// ============================================================

func SearchAPICompat(c fiber.Ctx) error {
	query := c.Query("q", "")
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}

	var users []*models.User
	if service.User != nil && query != "" {
		users, _ = service.User.SearchUsers(query, limit)
	}

	result := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		result = append(result, fiber.Map{
			"id":           u.ID,
			"username":     u.Username,
			"display_name": u.DisplayName,
		})
	}

	return c.JSON(fiber.Map{
		"results": result,
		"total":   len(result),
	})
}

func ProfileGetCompat(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var user *models.User
	if service.User != nil {
		user, _ = service.User.GetUserByID(id)
	}

	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return renderUserProfile(c, user)
}

func ProfileRedirectCompat(c fiber.Ctx) error { return c.Redirect().To("/") }

// Feed page handler
func FeedPageHandler(c fiber.Ctx) error { return Home(c) }

func PostFeedHandler(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	content := c.FormValue("content")
	if content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content required"})
	}

	if service.Feed != nil {
		postID, _, err := service.Feed.PostToFeed(username, content, "worldwide")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"id": postID, "success": true})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Feed service unavailable"})
}

// Groups API
func GroupDetailAPI(c fiber.Ctx) error {
	return GroupDetail(c)
}
