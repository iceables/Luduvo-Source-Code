package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/database"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

type reportSection struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Corpus string `json:"corpus"`
	Sort   int    `json:"sort"`
}

var reportSections = buildReportSections()

var (
	reportsSchemaMu    sync.Mutex
	reportsSchemaReady bool
)

func buildReportSections() []reportSection {
	categories := map[string][]string{
		"community": {
			"underage", "account_integrity", "impersonation", "harassment", "threats",
			"hate_speech", "child_exploitation", "suicide_self_harm", "off_platform_behavior",
			"personal_information", "off_platform_links", "sensitive_topics", "violent_media",
			"sexual_content", "erotic_roleplay", "illegal_goods", "account_trading", "profanity",
			"politics", "vigilantism", "exploits", "scams", "account_theft", "misinformation",
			"spam", "ip_infringement", "ai_content", "deceptive_ads", "gambling", "ban_evasion",
			"botting", "harmful_content",
		},
		"creator": {
			"ip_infringement", "unmoderated_content", "excessive_gore", "mass_violence",
			"child_exploitation", "suicide_self_harm", "illegal_goods", "sexual_content",
			"dating_mechanics", "profanity", "scams", "malicious_code", "gambling",
			"random_rewards", "politics", "pii_collection", "ai_content", "monetization_abuse",
			"off_platform_links", "discriminatory_content", "misleading_quality",
			"content_ratings", "harmful_content",
		},
	}

	var sections []reportSection
	for _, corpus := range []string{"community", "creator"} {
		for _, suffix := range categories[corpus] {
			words := strings.Split(suffix, "_")
			for i, word := range words {
				if word == "ai" || word == "pii" {
					words[i] = strings.ToUpper(word)
				} else if word != "" {
					words[i] = strings.ToUpper(word[:1]) + word[1:]
				}
			}
			code := corpus + "." + suffix
			sections = append(sections, reportSection{
				Code:   code,
				Label:  strings.Join(words, " "),
				Corpus: corpus,
				Sort:   len(sections) + 1,
			})
		}
	}
	return sections
}

func GuidelineSectionsAPI(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"sections": reportSections})
}

func ensureReportsSchema() error {
	if database.DB == nil {
		return errors.New("database unavailable")
	}

	reportsSchemaMu.Lock()
	defer reportsSchemaMu.Unlock()
	if reportsSchemaReady {
		return nil
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS bv_reports (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			reporter_id INT NOT NULL,
			target_type VARCHAR(16) NOT NULL,
			target_id BIGINT NOT NULL,
			reason VARCHAR(100) NOT NULL,
			description VARCHAR(2000) NOT NULL,
			surface VARCHAR(32) NOT NULL DEFAULT 'web',
			state VARCHAR(20) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX bv_reports_reporter_created (reporter_id, created_at),
			INDEX bv_reports_state_created (state, created_at),
			INDEX bv_reports_target (target_type, target_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS bv_report_attachments (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			report_id BIGINT NOT NULL,
			uploaded_by INT NOT NULL,
			storage_key CHAR(32) NOT NULL UNIQUE,
			content_type VARCHAR(128) NOT NULL,
			byte_size BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX bv_report_attachments_report (report_id, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, statement := range statements {
		if _, err := database.DB.Exec(statement); err != nil {
			return err
		}
	}
	reportsSchemaReady = true
	return nil
}

func CreateReport(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var request struct {
		TargetType  string `json:"target_type"`
		TargetID    int64  `json:"target_id"`
		Reason      string `json:"reason"`
		Description string `json:"description"`
		Surface     string `json:"surface"`
	}
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid report"})
	}
	request.TargetType = strings.ToLower(strings.TrimSpace(request.TargetType))
	request.Reason = strings.ToLower(strings.TrimSpace(request.Reason))
	request.Description = strings.TrimSpace(request.Description)
	request.Surface = strings.TrimSpace(request.Surface)
	if request.Surface == "" {
		request.Surface = "web"
	}
	if request.TargetID < 1 || !validReportTargetType(request.TargetType) ||
		!validReportReason(request.Reason) ||
		utf8.RuneCountInString(request.Description) < 10 ||
		utf8.RuneCountInString(request.Description) > 2000 ||
		utf8.RuneCountInString(request.Surface) > 32 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Report details are invalid"})
	}
	if err := validateReportTarget(request.TargetType, request.TargetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report target not found"})
		}
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}
	if err := ensureReportsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}

	result, err := database.DB.Exec(
		`INSERT INTO bv_reports (reporter_id, target_type, target_id, reason, description, surface)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, request.TargetType, request.TargetID, request.Reason, request.Description, request.Surface,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not submit report"})
	}
	id, err := result.LastInsertId()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not submit report"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         id,
		"status":     "pending",
		"created_at": time.Now().Unix(),
	})
}

func validReportTargetType(targetType string) bool {
	switch targetType {
	case "user", "place", "item", "group":
		return true
	default:
		return false
	}
}

func validReportReason(reason string) bool {
	for _, section := range reportSections {
		if section.Code == reason {
			return true
		}
	}
	return false
}

func validateReportTarget(targetType string, targetID int64) error {
	switch targetType {
	case "user":
		if service.User == nil {
			return errors.New("user service unavailable")
		}
		user, err := service.User.GetUserByID(int(targetID))
		if err != nil {
			return err
		}
		if user == nil {
			return sql.ErrNoRows
		}
	case "place":
		if service.Game == nil {
			return errors.New("game service unavailable")
		}
		game, err := service.Game.GetGameByID(int(targetID))
		if err != nil {
			return err
		}
		if game == nil {
			return sql.ErrNoRows
		}
	case "item":
		if _, ok := findCatalogItem(int(targetID)); !ok {
			return sql.ErrNoRows
		}
	case "group":
		if err := ensureGroupsSchema(); err != nil {
			return err
		}
		var exists bool
		if err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM bv_groups WHERE id = ?)", targetID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return sql.ErrNoRows
		}
	}
	return nil
}

func MyReports(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if err := ensureReportsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}

	limit := parsePageSize(c.Query("limit"), 50)
	offset := parsePageOffset(c.Query("offset"))
	var total int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM bv_reports WHERE reporter_id = ?", user.ID).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load reports"})
	}
	rows, err := database.DB.Query(
		`SELECT r.id, r.target_type, r.target_id, r.reason, r.description, r.state, r.created_at,
		        (SELECT COUNT(*) FROM bv_report_attachments a WHERE a.report_id = r.id)
		 FROM bv_reports r WHERE r.reporter_id = ? ORDER BY r.created_at DESC, r.id DESC LIMIT ? OFFSET ?`,
		user.ID, limit, offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load reports"})
	}
	defer rows.Close()

	reports := make([]fiber.Map, 0)
	for rows.Next() {
		var id, targetID, attachmentCount int64
		var targetType, reason, description, state string
		var createdAt time.Time
		if err := rows.Scan(&id, &targetType, &targetID, &reason, &description, &state, &createdAt, &attachmentCount); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load reports"})
		}
		reports = append(reports, fiber.Map{
			"id":             id,
			"target_type":    targetType,
			"target_id":      targetID,
			"reason":         reason,
			"description":    description,
			"status":         state,
			"created_at":     createdAt.Unix(),
			"evidence_count": attachmentCount,
			"evidence_items": []any{},
		})
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load reports"})
	}
	return c.JSON(fiber.Map{"reports": reports, "total": total, "limit": limit, "offset": offset})
}

func MyReportDetail(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	id, err := parseReportID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid report ID"})
	}
	if err := ensureReportsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}
	var report fiber.Map
	var targetType, reason, description, state string
	var targetID int64
	var createdAt time.Time
	err = database.DB.QueryRow(
		`SELECT id, target_type, target_id, reason, description, state, created_at
		 FROM bv_reports WHERE id = ? AND reporter_id = ?`, id, user.ID,
	).Scan(&id, &targetType, &targetID, &reason, &description, &state, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load report"})
	}
	report = fiber.Map{
		"id": id, "target_type": targetType, "target_id": targetID,
		"reason": reason, "description": description, "status": state,
		"created_at": createdAt.Unix(), "evidence_items": []any{},
	}
	return c.JSON(fiber.Map{"report": report})
}

func ReportAttachmentUpload(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	reportID, err := parseReportID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid report ID"})
	}
	body := c.Body()
	if len(body) == 0 || len(body) > 10*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Attachment must be between 1 byte and 10 MB"})
	}
	if err := ensureReportsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}
	if err := authorizeReport(c, reportID, user); err != nil {
		return err
	}

	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}
	key := hex.EncodeToString(random)
	dir := filepath.Join("data", "report-evidence")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}
	path := filepath.Join(dir, key+".bin")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}
	if _, err := file.Write(body); err != nil {
		file.Close()
		_ = os.Remove(path)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}

	contentType, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || contentType == "" {
		contentType = "application/octet-stream"
	}
	result, err := database.DB.Exec(
		`INSERT INTO bv_report_attachments (report_id, uploaded_by, storage_key, content_type, byte_size)
		 VALUES (?, ?, ?, ?, ?)`,
		reportID, user.ID, key, contentType, len(body),
	)
	if err != nil {
		_ = os.Remove(path)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}
	id, err := result.LastInsertId()
	if err != nil {
		_ = os.Remove(path)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not store attachment"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":           id,
		"name":         "Evidence attachment",
		"content_type": contentType,
		"size":         len(body),
		"created_at":   time.Now().Unix(),
	})
}

func ReportAttachments(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	reportID, err := parseReportID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid report ID"})
	}
	if err := ensureReportsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}
	if err := authorizeReport(c, reportID, user); err != nil {
		return err
	}
	rows, err := database.DB.Query(
		`SELECT id, content_type, byte_size, created_at
		 FROM bv_report_attachments WHERE report_id = ? ORDER BY created_at ASC, id ASC`,
		reportID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load report attachments"})
	}
	defer rows.Close()

	attachments := make([]fiber.Map, 0)
	for rows.Next() {
		var id, size int64
		var contentType string
		var createdAt time.Time
		if err := rows.Scan(&id, &contentType, &size, &createdAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load report attachments"})
		}
		attachments = append(attachments, fiber.Map{
			"id": id, "name": "Evidence attachment", "content_type": contentType,
			"size": size, "created_at": createdAt.Unix(),
			"download_url": fmt.Sprintf("/reports/%d/attachments/%d", reportID, id),
		})
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load report attachments"})
	}
	return c.JSON(fiber.Map{"attachments": attachments})
}

func ReportAttachmentDownload(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	reportID, err := parseReportID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid report ID"})
	}
	attachmentID, err := strconv.ParseInt(c.Params("attachmentID"), 10, 64)
	if err != nil || attachmentID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid attachment ID"})
	}
	if err := ensureReportsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Report service unavailable"})
	}
	if err := authorizeReport(c, reportID, user); err != nil {
		return err
	}
	var key string
	err = database.DB.QueryRow(
		"SELECT storage_key FROM bv_report_attachments WHERE id = ? AND report_id = ?",
		attachmentID, reportID,
	).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Attachment not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load attachment"})
	}
	if len(key) != 32 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load attachment"})
	}
	if _, err := hex.DecodeString(key); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load attachment"})
	}
	c.Set(fiber.HeaderContentType, "application/octet-stream")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="evidence.bin"`)
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	return c.SendFile(filepath.Join("data", "report-evidence", key+".bin"))
}

func authorizeReport(c fiber.Ctx, reportID int64, user *models.User) error {
	var reporterID int
	err := database.DB.QueryRow("SELECT reporter_id FROM bv_reports WHERE id = ?", reportID).Scan(&reporterID)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not authorize report"})
	}
	if reporterID != user.ID && user.Power < models.PowerTrainee {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	return nil
}

func parseReportID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid report ID")
	}
	return id, nil
}
