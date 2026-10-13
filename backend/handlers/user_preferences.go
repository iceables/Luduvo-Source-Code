package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/database"
	"buildoverse/backend/service"
)

const currentLegalVersion = "2026-10-06"

var defaultUserSettings = fiber.Map{
	"chat_disabled":            false,
	"ai_training_consent":      false,
	"message_privacy":          "everyone",
	"profile_visibility":       "everyone",
	"inventory_visibility":     "everyone",
	"online_status_visibility": "everyone",
	"friends_visibility":       "everyone",
	"wishlist_visibility":      "everyone",
}

func MeLegalAccept(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	settings, err := readUserSettings(user.ID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Could not save legal acceptance"})
	}
	settings["accepted_legal_version"] = currentLegalVersion
	if err := writeUserSettings(user.ID, settings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not save legal acceptance"})
	}
	return c.JSON(fiber.Map{"ok": true, "accepted_legal_version": currentLegalVersion})
}

func readUserSettings(userID int) (fiber.Map, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	if _, err := database.DB.Exec(`CREATE TABLE IF NOT EXISTS bv_user_settings (
		user_id INT NOT NULL PRIMARY KEY,
		settings_json LONGTEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`); err != nil {
		return nil, err
	}

	settings := fiber.Map{}
	for key, value := range defaultUserSettings {
		settings[key] = value
	}
	var raw string
	err := database.DB.QueryRow("SELECT settings_json FROM bv_user_settings WHERE user_id = ?", userID).Scan(&raw)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		stored := make(map[string]any)
		if err := json.Unmarshal([]byte(raw), &stored); err != nil {
			return nil, err
		}
		for key, value := range stored {
			settings[key] = value
		}
	}
	return settings, nil
}

func writeUserSettings(userID int, settings fiber.Map) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(
		`INSERT INTO bv_user_settings (user_id, settings_json) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE settings_json = VALUES(settings_json)`,
		userID, string(raw),
	)
	return err
}

func SettingsGetCompat(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	settings, err := readUserSettings(user.ID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Settings service unavailable"})
	}
	settings["username"] = user.Username
	settings["displayname"] = user.DisplayName
	settings["email"] = user.Mail
	settings["bio"] = user.Description
	return c.JSON(settings)
}

func SettingsPutCompat(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if database.DB == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Settings service unavailable"})
	}
	updates := make(map[string]any)
	if err := json.Unmarshal(c.Body(), &updates); err != nil || len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid settings"})
	}
	settings, err := readUserSettings(user.ID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Settings service unavailable"})
	}
	for key, value := range updates {
		switch key {
		case "chat_disabled", "ai_training_consent":
			if _, ok := value.(bool); !ok {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid settings"})
			}
		case "message_privacy":
			if !oneOf(value, "everyone", "friends", "disabled") {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid settings"})
			}
		case "profile_visibility", "inventory_visibility", "online_status_visibility", "friends_visibility", "wishlist_visibility":
			if !oneOf(value, "everyone", "friends", "private") {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid settings"})
			}
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Unsupported setting"})
		}
		settings[key] = value
	}
	if err := writeUserSettings(user.ID, settings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not save settings"})
	}
	return c.JSON(settings)
}

func oneOf(value any, allowed ...string) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	for _, candidate := range allowed {
		if s == candidate {
			return true
		}
	}
	return false
}

func MeIgnoredList(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return listUserRelations(c, user.ID, "ignore")
}

func MeBlockedList(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return listUserRelations(c, user.ID, "block")
}

func listUserRelations(c fiber.Ctx, userID int, kind string) error {
	if database.DB == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User list service unavailable"})
	}
	if err := ensureUserRelationsTable(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User list service unavailable"})
	}
	limit := parsePageSize(c.Query("limit"), 100)
	offset := parsePageOffset(c.Query("offset"))
	var total int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM bv_user_relations WHERE user_id = ? AND kind = ?", userID, kind).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user list"})
	}
	rows, err := database.DB.Query(
		`SELECT r.target_user_id, u.username, u.displayname
		 FROM bv_user_relations r
		 JOIN users u ON u.id = r.target_user_id
		 WHERE r.user_id = ? AND r.kind = ?
		 ORDER BY r.created_at DESC
		 LIMIT ? OFFSET ?`,
		userID, kind, limit, offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user list"})
	}
	defer rows.Close()
	items := make([]fiber.Map, 0)
	for rows.Next() {
		item := fiber.Map{}
		var targetID int
		var username, displayName string
		if err := rows.Scan(&targetID, &username, &displayName); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user list"})
		}
		if displayName == "" {
			displayName = username
		}
		item["user_id"] = targetID
		item["username"] = username
		item["display_name"] = displayName
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user list"})
	}
	return c.JSON(fiber.Map{"items": items, "total": total, "limit": limit, "offset": offset})
}

func AddIgnoredUser(c fiber.Ctx) error {
	return addUserRelation(c, "ignore")
}

func RemoveIgnoredUser(c fiber.Ctx) error {
	return removeUserRelation(c, "ignore")
}

func AddBlockedUser(c fiber.Ctx) error {
	return addUserRelation(c, "block")
}

func RemoveBlockedUser(c fiber.Ctx) error {
	return removeUserRelation(c, "block")
}

func addUserRelation(c fiber.Ctx, kind string) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 || targetID == user.ID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if service.User == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User service unavailable"})
	}
	target, err := service.User.GetUserByID(targetID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not find user"})
	}
	if target == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	if err := ensureUserRelationsTable(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User list service unavailable"})
	}
	_, err = database.DB.Exec(
		"INSERT IGNORE INTO bv_user_relations (user_id, target_user_id, kind) VALUES (?, ?, ?)",
		user.ID, targetID, kind,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update user list"})
	}
	status := "ignored"
	if kind == "block" {
		status = "blocked"
	}
	return c.JSON(fiber.Map{"status": status})
}

func removeUserRelation(c fiber.Ctx, kind string) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if err := ensureUserRelationsTable(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User list service unavailable"})
	}
	if _, err := database.DB.Exec(
		"DELETE FROM bv_user_relations WHERE user_id = ? AND target_user_id = ? AND kind = ?",
		user.ID, targetID, kind,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update user list"})
	}
	return c.JSON(fiber.Map{"status": "removed"})
}

func ensureUserRelationsTable() error {
	if database.DB == nil {
		return fmt.Errorf("database unavailable")
	}
	_, err := database.DB.Exec(`CREATE TABLE IF NOT EXISTS bv_user_relations (
		user_id INT NOT NULL,
		target_user_id INT NOT NULL,
		kind VARCHAR(16) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, target_user_id, kind),
		INDEX idx_bv_user_relations_lookup (user_id, kind, created_at)
	)`)
	return err
}

func parsePageSize(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		return fallback
	}
	return value
}

func parsePageOffset(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
