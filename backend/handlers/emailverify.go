package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

func issueEmailVerification(user *models.User) {
	if user == nil || service.User == nil {
		return
	}
	token := generateToken()
	expiry := time.Now().Add(48 * time.Hour).Unix()
	_ = service.User.SetEmailVerification(user.ID, token, expiry)
	// Automatic: the system verifies the address as soon as the account exists.
	_ = service.User.MarkEmailVerified(user.ID, user.Mail)
	recordMailbox(user.Mail, user.Username, token)
}

func recordMailbox(email, username, token string) {
	dir := filepath.Join("data")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "mailbox.json")
	var rows []map[string]string
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &rows)
	}
	rows = append(rows, map[string]string{
		"to":       email,
		"username": username,
		"token":    token,
		"link":     "/auth/verify-email?token=" + token,
		"status":   "verified-automatically",
		"at":       time.Now().UTC().Format(time.RFC3339),
	})
	b, _ := json.MarshalIndent(rows, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
}

func AuthVerifyEmail(c fiber.Ctx) error {
	token := strings.TrimSpace(c.FormValue("token"))
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token == "" {
		var body map[string]any
		_ = json.Unmarshal(c.Body(), &body)
		if body != nil {
			token, _ = body["token"].(string)
		}
	}
	if token == "" {
		if c.Method() == fiber.MethodGet {
			return Render(c, "pages/verify_email", fiber.Map{
				"Title":   "Verify email - Buildoverse",
				"Message": "This address was verified automatically when the account was created.",
			}, "layouts/main")
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing token"})
	}
	user, err := service.User.GetUserByEmailToken(token)
	if err != nil || user == nil {
		if c.Method() == fiber.MethodGet {
			return Render(c, "pages/verify_email", fiber.Map{
				"Title":   "Verify email - Buildoverse",
				"Message": "This link is already used or the address was verified automatically.",
			}, "layouts/main")
		}
		return c.JSON(fiber.Map{"success": true, "message": "already verified"})
	}
	if user.EmailVerifyExpiry.Valid && time.Now().Unix() > user.EmailVerifyExpiry.Int64 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token expired"})
	}
	_ = service.User.MarkEmailVerified(user.ID, user.Mail)
	if c.Method() == fiber.MethodGet {
		return Render(c, "pages/verify_email", fiber.Map{
			"Title":   "Verify email - Buildoverse",
			"Message": "Email verified. You can sign in.",
		}, "layouts/main")
	}
	return c.JSON(fiber.Map{"success": true})
}

func AuthResendVerification(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "sign in first"})
	}
	issueEmailVerification(user)
	return c.JSON(fiber.Map{"success": true, "message": "verified automatically"})
}
