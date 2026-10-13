package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/service"
)

func isSafeRedirect(ref string) bool {
	if ref == "" || strings.HasPrefix(ref, "//") || strings.HasPrefix(ref, "/\\") {
		return false
	}
	return strings.HasPrefix(ref, "/")
}

func handleFriendAction(c fiber.Ctx, action func(userID, targetID int) error, fallbackPath string) error {
	username := GetActiveUser(c)
	if username == "" {
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", "/login")
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Redirect().To("/login")
	}

	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID <= 0 {
		return c.Redirect().To("/")
	}

	if service.User == nil || service.Friend == nil {
		if fallbackPath != "" {
			return c.Redirect().To(fallbackPath)
		}
		return c.Redirect().To("/user/" + strconv.Itoa(targetID))
	}

	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", "/login")
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Redirect().To("/login")
	}

	_ = action(user.ID, targetID)

	if c.Get("HX-Request") == "true" {
		ref := c.Get("Referer")
		if strings.Contains(ref, "/user/") {
			targetUser, _ := service.User.GetUserByID(targetID)
			if targetUser != nil {
				friendStatus, _ := service.Friend.GetFriendStatus(user.ID, targetID)
				return c.Render("components/profile/actions", fiber.Map{
					"ProfileUser":  targetUser,
					"FriendStatus": friendStatus,
					"IsOwnProfile": false,
				})
			}
		}
		return c.SendString("")
	}

	ref := c.Get("Referer")
	if isSafeRedirect(ref) {
		return c.Redirect().To(ref)
	}
	if fallbackPath != "" {
		return c.Redirect().To(fallbackPath)
	}
	return c.Redirect().To("/user/" + strconv.Itoa(targetID))
}

func FriendsGet(c fiber.Ctx) error {
	username := GetActiveUser(c)
	if username == "" {
		return c.Redirect().To("/login")
	}
	if service.User == nil || service.Friend == nil {
		return c.Redirect().To("/")
	}
	user, err := service.User.GetUserByUsername(username)
	if err != nil || user == nil {
		return c.Redirect().To("/login")
	}

	idParam := c.Params("id")
	if idParam != "" {
		targetID, err := strconv.Atoi(idParam)
		if err == nil && targetID > 0 && targetID != user.ID {
			targetUser, err := service.User.GetUserByID(targetID)
			if err != nil || targetUser == nil {
				return c.Status(fiber.StatusNotFound).SendString("User not found")
			}
			friends, err := service.Friend.GetFriendsList(targetID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Error loading friends list")
			}
			return Render(c, "pages/friends", fiber.Map{
				"Title":            targetUser.DisplayName + " (@" + targetUser.Username + ")'s Friends - Buildoverse",
				"ActiveTab":        "friends",
				"FriendsList":      friends,
				"IsOwnFriendsPage": false,
				"TargetUser":       targetUser,
			}, "layouts/main")
		}
	}

	incoming, outgoing, friends, err := service.Friend.GetFriendRequestsPageData(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading friend requests")
	}

	return Render(c, "pages/friends", fiber.Map{
		"Title":            "Friends & Requests - Buildoverse",
		"ActiveTab":        "requests",
		"IncomingRequests": incoming,
		"OutgoingRequests": outgoing,
		"FriendsList":      friends,
		"IsOwnFriendsPage": true,
	}, "layouts/main")
}

func FriendSendPost(c fiber.Ctx) error {
	return handleFriendAction(c, service.Friend.SendFriendRequest, "")
}

func FriendAcceptPost(c fiber.Ctx) error {
	return handleFriendAction(c, service.Friend.AcceptFriendRequest, "")
}

func FriendDeclinePost(c fiber.Ctx) error {
	return handleFriendAction(c, service.Friend.DeclineFriendRequest, "/friends")
}

func FriendCancelPost(c fiber.Ctx) error {
	return handleFriendAction(c, service.Friend.CancelFriendRequest, "")
}

func FriendRemovePost(c fiber.Ctx) error {
	return handleFriendAction(c, service.Friend.RemoveFriend, "")
}