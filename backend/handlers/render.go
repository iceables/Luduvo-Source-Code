package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"buildoverse/backend/service"
)

func GetActiveUser(c fiber.Ctx) string {
	sess := session.FromContext(c)
	if sess == nil {
		return ""
	}

	usernameVal := sess.Get("username")
	if usernameStr, ok := usernameVal.(string); ok && usernameStr != "" {
		if service.User == nil {
			return usernameStr
		}
		u, err := service.User.GetUserByUsername(usernameStr)
		if err != nil {
			return usernameStr
		}
		if u != nil {
			return usernameStr
		}
	}

	cookieVal := c.Cookies("Buildoverse_remember")
	if cookieVal == "" {
		cookieVal = c.Cookies("buildoverse_remember")
	}
	if cookieVal == "" || service.Auth == nil {
		return ""
	}

	user, err := service.Auth.ValidateRememberToken(cookieVal)
	if err != nil || user == nil {
		return ""
	}

	sess.Set("username", user.Username)
	return user.Username
}

func Render(c fiber.Ctx, view string, data fiber.Map, layouts ...string) error {
	layout := "layouts/main"
	if c.Get("HX-Request") == "true" {
		layout = "layouts/htmx"
	}
	if len(layouts) > 0 {
		layout = layouts[0]
	}

	username := GetActiveUser(c)
	if username != "" {
		data["Username"] = username
		data["IsLoggedIn"] = true

		bucks := 0
		bits := 0
		userID := 0
		pendingFriends := 0
		isAdmin := false
		if service.User != nil {
			u, err := service.User.GetUserByUsername(username)
			if err == nil && u != nil {
				bucks = u.Bucks
				bits = u.Bits
				userID = u.ID
				isAdmin = u.HasAdminAccess()
				if service.Friend != nil {
					pendingFriends, _ = service.Friend.GetPendingRequestCount(u.ID)
				}
			}
		}
		data["Bucks"] = bucks
		data["Bits"] = bits
		data["UserID"] = userID
		data["IsAdmin"] = isAdmin
		data["PendingFriendRequestsCount"] = pendingFriends
	} else {
		data["Username"] = ""
		data["IsLoggedIn"] = false
		data["Bucks"] = 0
		data["Bits"] = 0
		data["UserID"] = 0
		data["IsAdmin"] = false
		data["PendingFriendRequestsCount"] = 0
	}

	return c.Render(view, data, layout)
}