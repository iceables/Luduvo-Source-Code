package handlers

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
)

var errorPages = map[int]struct {
	Template string
	Title    string
}{
	fiber.StatusNotFound:            {"pages/errors/404", "404 - Page Not Found - Buildoverse"},
	fiber.StatusForbidden:           {"pages/errors/403", "403 - Access Denied - Buildoverse"},
	fiber.StatusInternalServerError: {"pages/errors/500", "500 - Internal Server Error - Buildoverse"},
}

func serveErrorPage(c fiber.Ctx, code int) error {
	page, ok := errorPages[code]
	if !ok {
		page = errorPages[fiber.StatusInternalServerError]
	}
	c.Status(code)
	if err := Render(c, page.Template, fiber.Map{"Title": page.Title}); err != nil {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return c.Status(code).SendString(page.Title)
	}
	return nil
}

func ErrorHandler(c fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) && e != nil && e.Code > 0 {
		code = e.Code
	}

	if code >= fiber.StatusInternalServerError {
		log.Printf("ERROR [%d] %s: %v", code, c.Path(), err)
	}

	return serveErrorPage(c, code)
}

func NotFoundHandler(c fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), "/api/") || strings.HasPrefix(c.Path(), "/static/") {
		return serveErrorPage(c, fiber.StatusNotFound)
	}
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return serveErrorPage(c, fiber.StatusNotFound)
	}
	return Render(c, "pages/stub", fiber.Map{
		"Title": "Buildoverse",
		"Path":  c.Path(),
		"Blurb": "This page is on the site. Use the nav to keep moving.",
	}, "layouts/main")
}
