package handlers

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

func FriendsGetAPI(c fiber.Ctx) error {
	return MeFriends(c)
}

func MeFriends(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Friend == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Friends service unavailable"})
	}
	friends, err := service.Friend.GetFriendsList(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load friends"})
	}
	items := friendResponses(friends)
	limit := parsePageSize(c.Query("limit"), 200)
	offset := parsePageOffset(c.Query("offset"))
	return c.JSON(fiber.Map{
		"friends": itemsOnPage(items, offset, limit),
		"total":   len(items),
		"limit":   limit,
		"offset":  offset,
	})
}

func MeFriendsPending(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Friend == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Friends service unavailable"})
	}
	incoming, outgoing, _, err := service.Friend.GetFriendRequestsPageData(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load friend requests"})
	}
	limit := parsePageSize(c.Query("limit"), 200)
	offset := parsePageOffset(c.Query("offset"))
	incomingItems := friendResponses(incoming)
	outgoingItems := friendResponses(outgoing)
	pageIncoming := itemsOnPage(incomingItems, offset, limit)
	pageOutgoing := itemsOnPage(outgoingItems, offset, limit)
	return c.JSON(fiber.Map{
		"friend_requests": pageIncoming,
		"incoming":        pageIncoming,
		"outgoing":        pageOutgoing,
		"total":           len(incomingItems),
		"incoming_total":  len(incomingItems),
		"outgoing_total":  len(outgoingItems),
		"limit":           limit,
		"offset":          offset,
	})
}

func UserFriends(c fiber.Ctx) error {
	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if service.User == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User service unavailable"})
	}
	target, err := service.User.GetUserByID(targetID)
	if errors.Is(err, sql.ErrNoRows) || target == nil && err == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user friends"})
	}
	allowed, err := profileListAllowed(c, targetID, "friends_visibility")
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Could not verify profile privacy"})
	}
	if !allowed {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "This user's friends are private"})
	}
	if service.Friend == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Friends service unavailable"})
	}
	friends, err := service.Friend.GetFriendsList(targetID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user friends"})
	}
	items := friendResponses(friends)
	limit := parsePageSize(c.Query("limit"), 200)
	offset := parsePageOffset(c.Query("offset"))
	return c.JSON(fiber.Map{
		"users":  itemsOnPage(items, offset, limit),
		"total":  len(items),
		"limit":  limit,
		"offset": offset,
	})
}

func profileListAllowed(c fiber.Ctx, targetID int, settingKey string) (bool, error) {
	viewer := getCurrentUser(c)
	if viewer != nil && viewer.ID == targetID {
		return true, nil
	}
	settings, err := readUserSettings(targetID)
	if err != nil {
		return false, err
	}
	visibility, _ := settings[settingKey].(string)
	switch visibility {
	case "everyone":
		return true, nil
	case "friends":
		if viewer == nil || service.Friend == nil {
			return false, nil
		}
		status, err := service.Friend.GetFriendStatus(viewer.ID, targetID)
		return status == "friends", err
	default:
		return false, nil
	}
}

func FriendByUsernameRequest(c fiber.Ctx) error {
	if getCurrentUser(c) == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.User == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User service unavailable"})
	}
	target, err := service.User.GetUserByUsername(strings.TrimSpace(c.Params("username")))
	if errors.Is(err, sql.ErrNoRows) || target == nil && err == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not find user"})
	}
	return updateFriendRelation(c, target.ID, "send")
}

func FriendSendPostAPI(c fiber.Ctx) error {
	return updateFriendRelationFromPath(c, "send")
}

func FriendAcceptPostAPI(c fiber.Ctx) error {
	return updateFriendRelationFromPath(c, "accept")
}

func FriendDeclinePostAPI(c fiber.Ctx) error {
	return updateFriendRelationFromPath(c, "decline")
}

func FriendCancelPostAPI(c fiber.Ctx) error {
	return updateFriendRelationFromPath(c, "cancel")
}

func FriendRemovePostAPI(c fiber.Ctx) error {
	return updateFriendRelationFromPath(c, "remove")
}

func FriendBlock(c fiber.Ctx) error     { return AddBlockedUser(c) }
func FriendUnblock(c fiber.Ctx) error   { return RemoveBlockedUser(c) }
func FriendBlockPost(c fiber.Ctx) error { return AddBlockedUser(c) }
func FriendUnblockPost(c fiber.Ctx) error {
	return RemoveBlockedUser(c)
}

func updateFriendRelationFromPath(c fiber.Ctx, action string) error {
	targetID, err := strconv.Atoi(c.Params("id"))
	if err != nil || targetID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	return updateFriendRelation(c, targetID, action)
}

func updateFriendRelation(c fiber.Ctx, targetID int, action string) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.User == nil || service.Friend == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Friends service unavailable"})
	}
	target, err := service.User.GetUserByID(targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not find user"})
	}
	if target == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	status, err := service.Friend.GetFriendStatus(user.ID, targetID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not check friend status"})
	}
	var mutation func(int, int) error
	switch action {
	case "send":
		if status == "friends" || status == "pending_sent" {
			return c.JSON(fiber.Map{"ok": true, "status": status})
		}
		mutation = service.Friend.SendFriendRequest
	case "accept":
		if status != "pending_received" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "No incoming friend request to accept"})
		}
		mutation = service.Friend.AcceptFriendRequest
	case "decline":
		if status != "pending_received" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "No incoming friend request to decline"})
		}
		mutation = service.Friend.DeclineFriendRequest
	case "cancel":
		if status != "pending_sent" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "No outgoing friend request to cancel"})
		}
		mutation = service.Friend.CancelFriendRequest
	case "remove":
		if status != "friends" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Users are not friends"})
		}
		mutation = service.Friend.RemoveFriend
	default:
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "Friend action is not implemented"})
	}
	if err := mutation(user.ID, targetID); err != nil {
		if strings.Contains(err.Error(), "please wait") {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Please wait before trying again"})
		}
		if strings.Contains(err.Error(), "invalid user") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update friend relationship"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func friendResponses(friends []*models.FriendUserInfo) []fiber.Map {
	items := make([]fiber.Map, 0, len(friends))
	for _, friend := range friends {
		items = append(items, fiber.Map{
			"id":            friend.UserID,
			"user_id":       friend.UserID,
			"friendship_id": friend.FriendshipID,
			"username":      friend.Username,
			"display_name":  friend.DisplayName,
		})
	}
	return items
}

func itemsOnPage[T any](items []T, offset, limit int) []T {
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
