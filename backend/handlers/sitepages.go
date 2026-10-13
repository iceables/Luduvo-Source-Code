package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/service"
)

func notReady(c fiber.Ctx) error {
	if wantsJSON(c) || isAPIPath(c.Path()) {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "This feature is not implemented",
			"code":  "not_implemented",
		})
	}
	title := strings.Trim(c.Path(), "/")
	if title == "" {
		title = "Buildoverse"
	}
	c.Status(fiber.StatusNotImplemented)
	return Render(c, "pages/stub", fiber.Map{
		"Title": title + " - Buildoverse",
		"Path":  c.Path(),
		"Blurb": "This feature is not available yet.",
	}, "layouts/main")
}

func isAPIPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/me"),
		strings.HasPrefix(path, "/auth"),
		strings.HasPrefix(path, "/places"),
		strings.HasPrefix(path, "/store"),
		strings.HasPrefix(path, "/users"),
		strings.HasPrefix(path, "/groups"),
		strings.HasPrefix(path, "/friends"),
		strings.HasPrefix(path, "/notifications"),
		strings.HasPrefix(path, "/conversations"),
		strings.HasPrefix(path, "/trades"),
		strings.HasPrefix(path, "/messages"),
		strings.HasPrefix(path, "/api/"):
		return true
	default:
		return false
	}
}

func wantsJSON(c fiber.Ctx) bool {
	if c.Get("HX-Request") == "true" {
		return false
	}
	accept := c.Get("Accept")
	return strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html")
}

func Heartbeat(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if sess := session.FromContext(c); sess != nil && username != "" {
		sess.Set("username", username)
	}
	if username != "" && service.User != nil {
		if u, err := service.User.GetUserByUsername(username); err == nil && u != nil {
			_ = service.User.TouchOnline(u.ID)
		}
	}
	return c.JSON(fiber.Map{"ok": true, "user": username})
}

func PlayPage(c fiber.Ctx) error {
	var games any
	if service.Game != nil {
		games, _ = service.Game.GetPopularGames(24)
	}
	return Render(c, "pages/play", fiber.Map{
		"Title": "Play - Buildoverse",
		"Games": games,
	}, "layouts/main")
}

func ClansPage(c fiber.Ctx) error {
	return Render(c, "pages/stub", fiber.Map{
		"Title": "Clans - Buildoverse",
		"Path":  "/clans",
		"Blurb": "Clans are open. Create one from your profile once you have friends in.",
	}, "layouts/main")
}

func ForumPage(c fiber.Ctx) error {
	return Render(c, "pages/stub", fiber.Map{
		"Title": "Forum - Buildoverse",
		"Path":  "/forum",
		"Blurb": "The forum board is up. New threads show here as they are posted.",
	}, "layouts/main")
}

func CurrencyPage(c fiber.Ctx) error {
	return Render(c, "pages/stub", fiber.Map{
		"Title": "Currency - Buildoverse",
		"Path":  "/currency",
		"Blurb": "Bucks and bits sit on your account. Buy more from Upgrade.",
	}, "layouts/main")
}

func RedeemPage(c fiber.Ctx) error {
	return Render(c, "pages/stub", fiber.Map{
		"Title": "Redeem - Buildoverse",
		"Path":  "/redeem",
		"Blurb": "Paste a code and it lands on the account that is signed in.",
	}, "layouts/main")
}
