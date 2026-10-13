package handlers

import (
	"html"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/service"
)

func ProfileRedirect(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Redirect().To("/login")
	}
	if service.User == nil {
		return c.Redirect().To("/login")
	}
	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		return c.Redirect().To("/login")
	}
	return c.Redirect().To("/user/" + strconv.Itoa(user.ID))
}

func ProfileGet(c fiber.Ctx) error {
	idParam := c.Params("id")
	userID, err := strconv.Atoi(idParam)
	if err != nil || userID <= 0 {
		return c.Status(fiber.StatusNotFound).SendString("User not found")
	}

	if service.User == nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Database error")
	}

	activeUsername := GetActiveUser(c)
	targetUser, isOwnProfile, friends, followers, following, err := service.User.GetProfileData(userID, activeUsername)
	if err != nil || targetUser == nil {
		return c.Status(fiber.StatusNotFound).SendString("User not found")
	}

	friendStatus := "none"
	if activeUsername != "" && !isOwnProfile && service.Friend != nil {
		activeUser, _ := service.User.GetUserByUsername(activeUsername)
		if activeUser != nil {
			friendStatus, _ = service.Friend.GetFriendStatus(activeUser.ID, targetUser.ID)
		}
	}

	var profileMusicID int64
	if targetUser.MusicID.Valid && targetUser.MusicID.Int64 > 0 {
		profileMusicID = targetUser.MusicID.Int64
	}

	return Render(c, "pages/profile", fiber.Map{
		"Title":          html.EscapeString(targetUser.DisplayName) + " (@" + html.EscapeString(targetUser.Username) + ") - Buildoverse",
		"ProfileUser":    targetUser,
		"IsOwnProfile":   isOwnProfile,
		"FriendsCount":   friends,
		"FollowersCount": followers,
		"FollowingCount": following,
		"FriendStatus":   friendStatus,
		"ProfileMusicID": profileMusicID,
	}, "layouts/main")
}