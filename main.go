package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
	"buildoverse/backend/config"
	"buildoverse/backend/database"
	"buildoverse/backend/handlers"
	"buildoverse/backend/renderer"
	"buildoverse/backend/routes"
	"buildoverse/backend/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error! ########## %v", err)
	}

	if err := database.Connect(cfg); err != nil {
		log.Fatalf("DB error! ########## %v", err)
	}

	cachedAvatarIDs, err := readCachedAvatarIDs()
	if err != nil {
		log.Printf("Could not enumerate cached avatars for refresh: %v", err)
	}
	service.Init(database.DB)
	migratedAvatarIDs, err := service.Avatar.MigrateDefaultAvatarPalette()
	if err != nil {
		log.Fatalf("Avatar default migration failed: %v", err)
	}
	if len(migratedAvatarIDs) > 0 {
		migratedAvatarIDs = append(migratedAvatarIDs, cachedAvatarIDs...)
		migratedAvatarIDs = uniqueSortedIDs(migratedAvatarIDs)
	}

	renderer.Init()
	for _, userID := range migratedAvatarIDs {
		full, err := renderer.RenderUser(database.DB, userID)
		if err != nil {
			log.Printf("Failed to rerender avatar %d: %v", userID, err)
			continue
		}
		headshot, err := renderer.RenderUserHeadshot(database.DB, userID)
		if err != nil {
			log.Printf("Failed to rerender avatar headshot %d: %v", userID, err)
			continue
		}
		for _, render := range []struct {
			path string
			data []byte
		}{
			{path: filepath.Join("static", "renders", "avatars", "full", fmt.Sprintf("%d.png", userID)), data: full},
			{path: filepath.Join("static", "renders", "avatars", "headshots", fmt.Sprintf("%d.png", userID)), data: headshot},
		} {
			if err := os.MkdirAll(filepath.Dir(render.path), 0755); err != nil {
				log.Printf("Failed to create avatar render directory for user %d: %v", userID, err)
				continue
			}
			if err := os.WriteFile(render.path, render.data, 0644); err != nil {
				log.Printf("Failed to save avatar render for user %d: %v", userID, err)
			}
		}
	}

	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Views:        engine,
		BodyLimit:    25 * 1024 * 1024,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
		ServerHeader: "",
		ErrorHandler: handlers.ErrorHandler,
	})

	app.Use(recoverer.New())

	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' https: data: blob: wss:; worker-src 'self' blob:; frame-ancestors 'self';")
		return c.Next()
	})

	//We should not allow more than 100 requests X minute from same IP
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.LimiterMax,
		Expiration: cfg.LimiterExpiration,
	}))

	app.Use(favicon.New(favicon.Config{
		File: "./static/branding/favicon.ico",
		URL:  "/favicon.ico",
	}))

	sessionConfig := session.Config{
		CookieHTTPOnly: true,
		CookieSecure:   cfg.SessionSecure,
		CookieSameSite: cfg.SessionSameSite,
		CookiePath:     "/",
	}
	if cfg.SessionIdleTimeout > 0 {
		sessionConfig.IdleTimeout = cfg.SessionIdleTimeout
	}
	if cfg.SessionAbsoluteTimeout > 0 && cfg.SessionAbsoluteTimeout >= cfg.SessionIdleTimeout {
		sessionConfig.AbsoluteTimeout = cfg.SessionAbsoluteTimeout
	}
	app.Use(session.New(sessionConfig))

	routes.Setup(app)

	log.Fatal(app.Listen("127.0.0.1:3000"))
}

func readCachedAvatarIDs() ([]int, error) {
	ids := make(map[int]struct{})
	for _, dir := range []string{
		filepath.Join("static", "renders", "avatars", "full"),
		filepath.Join("static", "renders", "avatars", "headshots"),
	} {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".png" {
				continue
			}
			id, err := strconv.Atoi(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			if err != nil || id < 1 {
				continue
			}
			ids[id] = struct{}{}
		}
	}
	result := make([]int, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	return result, nil
}

func uniqueSortedIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Ints(result)
	return result
}
