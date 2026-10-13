package handlers

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/config"
	"buildoverse/backend/service"
)

func registerErrorField(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "captcha"):
		return "captcha"
	case strings.Contains(lower, "username"):
		return "username"
	case strings.Contains(lower, "email"):
		return "email"
	case strings.Contains(lower, "match"):
		return "passwordconfirm"
	case strings.Contains(lower, "password"):
		return "password"
	default:
		return "form"
	}
}

func loginErrorField(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "captcha"):
		return "captcha"
	case strings.Contains(lower, "password") || strings.Contains(lower, "invalid") || strings.Contains(lower, "required"):
		return "credentials"
	default:
		return "identifier"
	}
}

func SetHashedCookie(c fiber.Ctx, userID int) {
	if service.Auth == nil {
		return
	}
	encodedToken, err := service.Auth.CreateRememberToken(userID, 30*24*time.Hour)
	if err != nil {
		return
	}

	cookie := &fiber.Cookie{
		Name:     "Buildoverse_remember",
		Value:    encodedToken,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		MaxAge:   30 * 24 * 60 * 60,
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	}
	c.Cookie(cookie)
}

func AltchaGet(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")

	challenge, err := service.Auth.GenerateAltchaChallenge()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(challenge)
}

func LoginGet(c fiber.Ctx) error {
	if username := GetActiveUser(c); username != "" {
		return c.Redirect().To("/")
	}
	return Render(c, "pages/login", fiber.Map{
		"Title": "Log In - Buildoverse",
	}, "layouts/main")
}

func LoginPost(c fiber.Ctx) error {
	if username := GetActiveUser(c); username != "" {
		return c.Redirect().To("/")
	}

	altchaPayload := c.FormValue("altcha")
	if err := service.Auth.VerifyAltcha(altchaPayload); err != nil {
		return Render(c, "pages/login", fiber.Map{
			"Title":      "Log In - Buildoverse",
			"Error":      err.Error(),
			"ErrorField": loginErrorField(err.Error()),
		}, "layouts/main")
	}

	identifier := c.FormValue("identifier")
	password := c.FormValue("password")

	if identifier == "" || password == "" {
		return Render(c, "pages/login", fiber.Map{
			"Title":      "Log In - Buildoverse",
			"Error":      "Username/Email and Password are required",
			"ErrorField": "credentials",
		}, "layouts/main")
	}

	user, err := service.Auth.AuthenticateUser(identifier, password)
	if err != nil {
		return Render(c, "pages/login", fiber.Map{
			"Title":      "Log In - Buildoverse",
			"Error":      err.Error(),
			"ErrorField": loginErrorField(err.Error()),
		}, "layouts/main")
	}

	sess := session.FromContext(c)
	if sess != nil {
		sess.Set("username", user.Username)
	}

	SetHashedCookie(c, user.ID)

	return c.Redirect().To("/")
}

func RegisterGet(c fiber.Ctx) error {
	if username := GetActiveUser(c); username != "" {
		return c.Redirect().To("/")
	}
	userCount := 0
	if service.User != nil {
		userCount, _ = service.User.GetUserCount()
	}
	return Render(c, "pages/register", fiber.Map{
		"Title":     "Register - Buildoverse",
		"UserCount": userCount,
	}, "layouts/main")
}

func RegisterPost(c fiber.Ctx) error {
	if username := GetActiveUser(c); username != "" {
		return c.Redirect().To("/")
	}

	userCount := 0
	if service.User != nil {
		userCount, _ = service.User.GetUserCount()
	}

	altchaPayload := c.FormValue("altcha")
	if err := service.Auth.VerifyAltcha(altchaPayload); err != nil {
		return Render(c, "pages/register", fiber.Map{
			"Title":      "Register - Buildoverse",
			"Error":      err.Error(),
			"ErrorField": "captcha",
			"UserCount":  userCount,
		}, "layouts/main")
	}

	username := c.FormValue("username")
	displayname := c.FormValue("displayname")
	email := c.FormValue("email")
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	user, err := service.Auth.RegisterUser(username, displayname, email, password, passwordConfirm)
	if err != nil {
		return Render(c, "pages/register", fiber.Map{
			"Title":      "Register - Buildoverse",
			"Error":      err.Error(),
			"ErrorField": registerErrorField(err.Error()),
			"UserCount":  userCount,
		}, "layouts/main")
	}

	sess := session.FromContext(c)
	if sess != nil {
		sess.Set("username", user.Username)
	}

	SetHashedCookie(c, user.ID)
	issueEmailVerification(user)
	setSignupFlash(c, user.Username)

	return c.Redirect().To("/")
}

func Logout(c fiber.Ctx) error {
	sess := session.FromContext(c)
	if sess != nil {
		_ = sess.Destroy()
	}

	cookieVal := c.Cookies("Buildoverse_remember")
	if cookieVal != "" && service.Auth != nil {
		service.Auth.RevokeRememberToken(cookieVal)
	}

	cookie := &fiber.Cookie{
		Name:     "Buildoverse_remember",
		Value:    "",
		Expires:  time.Now().Add(-24 * time.Hour),
		HTTPOnly: true,
		Secure:   config.Global.SessionSecure,
		SameSite: config.Global.SessionSameSite,
	}
	c.Cookie(cookie)

	return c.Redirect().To("/")
}

func ValidateUkey(c fiber.Ctx) error {
	apiKey := c.Get("X-Gameserver-Key")
	ukey := c.Query("ukey")

	user, err := service.Auth.ValidateUkey(apiKey, ukey)
	if err != nil {
		if err.Error() == "unauthorized" {
			log.Printf("GAMESERVER_AUTH: Key mismatch. Unauthorized request.")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized gameserver request",
			})
		}
		if err.Error() == "missing ukey" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing ukey parameter",
			})
		}
		if err.Error() == "not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"uid":      user.ID,
		"username": user.Username,
	})
}