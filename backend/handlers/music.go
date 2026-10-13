package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/service"
)

func MusicSearchGet(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return c.JSON(fiber.Map{
			"data": []any{},
		})
	}

	if service.Music == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Music service uninitialized",
		})
	}

	tracks, err := service.Music.SearchTracks(q)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": tracks,
	})
}

func MusicTrackGet(c fiber.Ctx) error {
	idParam := c.Params("id")
	trackID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || trackID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid track ID",
		})
	}

	if service.Music == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Music service uninitialized",
		})
	}

	track, err := service.Music.GetTrackByID(trackID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Track not found",
		})
	}

	return c.JSON(track)
}