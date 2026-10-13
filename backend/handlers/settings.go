package handlers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

func isAJAXRequest(c fiber.Ctx) bool {
	return c.Get("X-Requested-With") == "XMLHttpRequest" ||
		strings.Contains(c.Get("Accept"), "application/json")
}

func getSettingsUser(c fiber.Ctx) (*models.User, error) {
	username := GetActiveUser(c)
	if username == "" {
		return nil, fiber.ErrUnauthorized
	}
	if service.User == nil {
		return nil, errors.New("Database offline")
	}
	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		return nil, fiber.ErrUnauthorized
	}
	return user, nil
}

func handleSettingsResponse(c fiber.Ctx, err error, successMsg string, extra fiber.Map) error {
	if err != nil {
		if errors.Is(err, fiber.ErrUnauthorized) {
			if isAJAXRequest(c) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
			}
			return c.Redirect().To("/login")
		}
		if isAJAXRequest(c) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Redirect().To("/settings?error=" + err.Error())
	}

	if isAJAXRequest(c) {
		res := fiber.Map{"success": successMsg}
		for k, v := range extra {
			res[k] = v
		}
		return c.JSON(res)
	}
	return c.Redirect().To("/settings?success=" + successMsg)
}

func SettingsGet(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	changesLeft := 2
	if left, err := service.User.GetUsernameChangesLeft(user.ID); err == nil {
		changesLeft = left
	}

	var userMusicID int64
	if user.MusicID.Valid && user.MusicID.Int64 > 0 {
		userMusicID = user.MusicID.Int64
	}

	return Render(c, "pages/settings", fiber.Map{
		"Title":               "Settings - Buildoverse",
		"Error":               c.Query("error"),
		"Success":             c.Query("success"),
		"ActiveTab":           c.Query("tab", "account"),
		"UserBio":             user.Description,
		"UserPronouns":        user.Pronouns,
		"UserSocials":         user.ParsedSocials(),
		"UserMusicID":         userMusicID,
		"CurUsername":         user.Username,
		"CurDisplayName":      user.DisplayName,
		"UsernameChangesLeft": changesLeft,
	}, "layouts/main")
}

func SettingsDisplayNamePost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	newDisplayName := c.FormValue("displayname")
	err = service.User.ChangeDisplayName(user.ID, newDisplayName)
	return handleSettingsResponse(c, err, "Display name changed successfully", fiber.Map{"displayname": newDisplayName})
}

func SettingsUsernamePost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	sess := session.FromContext(c)
	if sess == nil {
		return handleSettingsResponse(c, fiber.ErrUnauthorized, "", nil)
	}

	newUsername := c.FormValue("username")
	if err := service.User.ChangeUsername(user.ID, user.Username, newUsername); err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	sess.Set("username", newUsername)
	SetHashedCookie(c, user.ID)

	updatedUser, _ := service.User.GetUserByID(user.ID)
	newBucks := user.Bucks - 100
	if updatedUser != nil {
		newBucks = updatedUser.Bucks
	}

	changesLeft := 0
	if left, err := service.User.GetUsernameChangesLeft(user.ID); err == nil {
		changesLeft = left
	}

	return handleSettingsResponse(c, nil, "Username changed successfully", fiber.Map{
		"username":     newUsername,
		"bucks":        newBucks,
		"changes_left": changesLeft,
	})
}

func SettingsPasswordPost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	err = service.User.ChangePassword(user.ID, c.FormValue("current_password"), c.FormValue("new_password"), c.FormValue("retype_password"))
	return handleSettingsResponse(c, err, "Password updated successfully", nil)
}

func SettingsBioPost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	bio := c.FormValue("bio")
	err = service.User.UpdateBio(user.ID, bio)
	return handleSettingsResponse(c, err, "Bio updated successfully", fiber.Map{"bio": bio})
}

func SettingsPronounsPost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	pronouns := c.FormValue("pronouns")
	err = service.User.UpdatePronouns(user.ID, pronouns)
	return handleSettingsResponse(c, err, "Pronouns saved successfully", fiber.Map{"pronouns": pronouns})
}

func SettingsSocialsPost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	socialsMap := make(map[string]string)
	if user.Socials != "" {
		_ = json.Unmarshal([]byte(user.Socials), &socialsMap)
	}

	reqPlatform := strings.ToLower(strings.TrimSpace(c.FormValue("platform")))
	reqValue := strings.TrimSpace(c.FormValue("value"))

	if reqPlatform != "" {
		if reqValue == "" {
			delete(socialsMap, reqPlatform)
		} else {
			socialsMap[reqPlatform] = reqValue
		}
	} else {
		for _, p := range service.AllowedSocialPlatforms {
			val := strings.TrimSpace(c.FormValue(p))
			if val != "" {
				socialsMap[p] = val
			} else if c.FormValue("batch") == "true" {
				delete(socialsMap, p)
			}
		}
	}

	err = service.User.UpdateSocials(user.ID, socialsMap)
	return handleSettingsResponse(c, err, "Social links saved successfully", nil)
}

func SettingsMusicPost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	trackIDStr := c.FormValue("track_id")
	if trackIDStr == "" {
		trackIDStr = c.FormValue("music_id")
	}
	trackID, err := strconv.ParseInt(trackIDStr, 10, 64)
	if err != nil || trackID <= 0 {
		return handleSettingsResponse(c, errors.New("Invalid track ID"), "", nil)
	}

	err = service.User.UpdateMusicID(user.ID, trackID)
	return handleSettingsResponse(c, err, "Profile music updated successfully", fiber.Map{"music_id": trackID})
}

func SettingsMusicRemovePost(c fiber.Ctx) error {
	user, err := getSettingsUser(c)
	if err != nil {
		return handleSettingsResponse(c, err, "", nil)
	}

	err = service.User.UpdateMusicID(user.ID, 0)
	return handleSettingsResponse(c, err, "Profile music removed successfully", fiber.Map{"music_id": 0})
}