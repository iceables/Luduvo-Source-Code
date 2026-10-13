package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/config"
)

func calculateLiveTimerData() (fiber.Map, bool) {
	if config.Global == nil || !config.Global.IsLiveTimerActive() {
		return fiber.Map{
			"active":               false,
			"remaining_seconds":    0,
			"formatted":            "00:00",
			"target_end_ms":        0,
			"total_duration_ms":    0,
			"should_preload_video": false,
		}, false
	}

	remaining := config.Global.LiveTimerRemaining()
	totalSec := int(remaining.Seconds())
	if totalSec < 0 {
		totalSec = 0
	}
	if totalSec > 3600 {
		totalSec = 3600
	}

	mins := totalSec / 60
	secs := totalSec % 60
	formatted := fmt.Sprintf("%02d:%02d", mins, secs)

	targetEndMs := config.Global.LiveTimerEnd.UnixMilli()
	remainingMs := remaining.Milliseconds()
	totalDurationMs := config.Global.LiveTimerDuration.Milliseconds()

	shouldPreloadVideo := (totalDurationMs > 0 && remainingMs <= totalDurationMs/2) || (totalDurationMs <= 0 && remainingMs <= 1800000)

	return fiber.Map{
		"active":               true,
		"remaining_seconds":    totalSec,
		"formatted":            formatted,
		"target_end_ms":        targetEndMs,
		"total_duration_ms":    totalDurationMs,
		"should_preload_video": shouldPreloadVideo,
		"FormattedRemaining":   formatted,
		"TargetEndMs":          targetEndMs,
		"TotalDurationMs":      totalDurationMs,
		"RemainingSeconds":     totalSec,
		"ShouldPreloadVideo":   shouldPreloadVideo,
	}, true
}

func LiveTimerGet(c fiber.Ctx) error {
	data, active := calculateLiveTimerData()
	if !active {
		return c.Redirect().To("/")
	}
	data["Title"] = "Buildoverse - LiveTimer"
	return c.Render("pages/livetimer", data, "")
}

func LiveTimerAPIGet(c fiber.Ctx) error {
	data, _ := calculateLiveTimerData()
	return c.JSON(data)
}