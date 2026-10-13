package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/database"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

var (
	groupsSchemaMu    sync.Mutex
	groupsSchemaReady bool
)

func ensureGroupsSchema() error {
	if database.DB == nil {
		return errors.New("database unavailable")
	}

	groupsSchemaMu.Lock()
	defer groupsSchemaMu.Unlock()
	if groupsSchemaReady {
		return nil
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS bv_groups (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			description TEXT NOT NULL,
			owner_id INT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX bv_groups_created_at (created_at),
			INDEX bv_groups_owner_id (owner_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS bv_group_members (
			group_id BIGINT NOT NULL,
			user_id INT NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'member',
			joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (group_id, user_id),
			INDEX bv_group_members_user_id (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, statement := range statements {
		if _, err := database.DB.Exec(statement); err != nil {
			return err
		}
	}
	groupsSchemaReady = true
	return nil
}

func GroupsCreate(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid group"})
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)
	if utf8.RuneCountInString(request.Name) < 3 || utf8.RuneCountInString(request.Name) > 50 ||
		utf8.RuneCountInString(request.Description) > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Group name or description is invalid"})
	}
	if err := ensureGroupsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}

	tx, err := database.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(context.Background(),
		"INSERT INTO bv_groups (name, description, owner_id) VALUES (?, ?, ?)",
		request.Name, request.Description, user.ID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create group"})
	}
	groupID, err := result.LastInsertId()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create group"})
	}
	if _, err := tx.ExecContext(context.Background(),
		"INSERT INTO bv_group_members (group_id, user_id, role) VALUES (?, ?, 'owner')",
		groupID, user.ID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create group"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create group"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":             groupID,
		"name":           request.Name,
		"description":    request.Description,
		"owner_id":       user.ID,
		"owner_username": user.Username,
		"member_count":   1,
		"icon_url":       "",
		"created_at":     time.Now().Unix(),
		"is_member":      true,
		"can_edit":       true,
	})
}

func GroupsAPI(c fiber.Ctx) error {
	if err := ensureGroupsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}
	limit := parsePageSize(c.Query("limit"), 20)
	offset := parsePageOffset(c.Query("offset"))
	query := strings.TrimSpace(c.Query("q"))
	if utf8.RuneCountInString(query) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Search query is too long"})
	}
	sort := strings.ToLower(strings.TrimSpace(c.Query("sort", "trending")))
	orderBy := "member_count DESC, g.created_at DESC"
	switch sort {
	case "trending", "members":
	case "created", "latest":
		orderBy = "g.created_at DESC"
	case "name":
		orderBy = "g.name ASC"
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid group sort order"})
	}

	search := "%" + query + "%"
	var total int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM bv_groups WHERE name LIKE ? OR description LIKE ?",
		search, search,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load groups"})
	}

	user := getCurrentUser(c)
	userID := 0
	if user != nil {
		userID = user.ID
	}
	rows, err := database.DB.Query(fmt.Sprintf(
		`SELECT g.id, g.name, g.description, g.owner_id, COALESCE(u.username, ''),
		        COUNT(m.user_id) AS member_count, MAX(CASE WHEN m.user_id = ? THEN 1 ELSE 0 END), g.created_at
		 FROM bv_groups g
		 LEFT JOIN users u ON u.id = g.owner_id
		 LEFT JOIN bv_group_members m ON m.group_id = g.id
		 WHERE g.name LIKE ? OR g.description LIKE ?
		 GROUP BY g.id, g.name, g.description, g.owner_id, u.username, g.created_at
		 ORDER BY %s
		 LIMIT ? OFFSET ?`, orderBy,
	), userID, search, search, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load groups"})
	}
	defer rows.Close()

	groups := make([]fiber.Map, 0)
	for rows.Next() {
		var (
			id, ownerID, members             int64
			isMember                         int
			name, description, ownerUsername string
			createdAt                        time.Time
		)
		if err := rows.Scan(&id, &name, &description, &ownerID, &ownerUsername, &members, &isMember, &createdAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load groups"})
		}
		groups = append(groups, groupResponse(id, name, description, ownerID, ownerUsername, members, createdAt, user, isMember != 0))
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load groups"})
	}
	return c.JSON(fiber.Map{"groups": groups, "total": total, "limit": limit, "offset": offset})
}

func GroupDetail(c fiber.Ctx) error {
	groupID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || groupID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid group ID"})
	}
	if err := ensureGroupsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}

	user := getCurrentUser(c)
	userID := 0
	if user != nil {
		userID = user.ID
	}
	var (
		id, ownerID, members             int64
		isMember                         int
		name, description, ownerUsername string
		createdAt                        time.Time
	)
	err = database.DB.QueryRow(
		`SELECT g.id, g.name, g.description, g.owner_id, COALESCE(u.username, ''),
		        COUNT(m.user_id), MAX(CASE WHEN m.user_id = ? THEN 1 ELSE 0 END), g.created_at
		 FROM bv_groups g
		 LEFT JOIN users u ON u.id = g.owner_id
		 LEFT JOIN bv_group_members m ON m.group_id = g.id
		 WHERE g.id = ?
		 GROUP BY g.id, g.name, g.description, g.owner_id, u.username, g.created_at`,
		userID, groupID,
	).Scan(&id, &name, &description, &ownerID, &ownerUsername, &members, &isMember, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Group not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load group"})
	}
	return c.JSON(groupResponse(id, name, description, ownerID, ownerUsername, members, createdAt, user, isMember != 0))
}

func groupResponse(id int64, name, description string, ownerID int64, ownerUsername string, members int64, createdAt time.Time, user *models.User, isMember bool) fiber.Map {
	canEdit := false
	if user != nil {
		userID := int64(user.ID)
		canEdit = userID == ownerID
		if canEdit {
			isMember = true
		}
	}
	return fiber.Map{
		"id":             id,
		"name":           name,
		"description":    description,
		"owner_id":       ownerID,
		"owner_username": ownerUsername,
		"member_count":   members,
		"icon_url":       "",
		"created_at":     createdAt.Unix(),
		"is_member":      isMember,
		"can_edit":       canEdit,
	}
}

func GroupJoin(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	groupID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || groupID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid group ID"})
	}
	if err := ensureGroupsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}

	var exists int
	if err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM bv_groups WHERE id = ?)", groupID).Scan(&exists); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not join group"})
	}
	if exists == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Group not found"})
	}
	_, err = database.DB.Exec(
		`INSERT INTO bv_group_members (group_id, user_id, role) VALUES (?, ?, 'member')
		 ON DUPLICATE KEY UPDATE user_id = VALUES(user_id)`,
		groupID, user.ID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not join group"})
	}
	return c.JSON(fiber.Map{"ok": true, "group_id": groupID})
}

func GroupLeave(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	groupID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || groupID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid group ID"})
	}
	if err := ensureGroupsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}

	var ownerID int
	if err := database.DB.QueryRow("SELECT owner_id FROM bv_groups WHERE id = ?", groupID).Scan(&ownerID); errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Group not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not leave group"})
	}
	if ownerID == user.ID {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "The group owner cannot leave the group"})
	}
	result, err := database.DB.Exec("DELETE FROM bv_group_members WHERE group_id = ? AND user_id = ?", groupID, user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not leave group"})
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not leave group"})
	}
	if affected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "You are not a member of this group"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func MeGroups(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	return listGroupsForMember(c, user.ID)
}

func UserGroups(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil || userID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if service.User == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "User service unavailable"})
	}
	target, err := service.User.GetUserByID(userID)
	if errors.Is(err, sql.ErrNoRows) || target == nil && err == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user groups"})
	}
	return listGroupsForMember(c, userID)
}

func listGroupsForMember(c fiber.Ctx, memberID int) error {
	if err := ensureGroupsSchema(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Groups service unavailable"})
	}
	limit := parsePageSize(c.Query("limit"), 20)
	offset := parsePageOffset(c.Query("offset"))
	var total int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM bv_group_members WHERE user_id = ?",
		memberID,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user groups"})
	}

	user := getCurrentUser(c)
	viewerID := 0
	if user != nil {
		viewerID = user.ID
	}
	rows, err := database.DB.Query(
		`SELECT g.id, g.name, g.description, g.owner_id, COALESCE(u.username, ''),
		        COUNT(m.user_id), MAX(CASE WHEN m.user_id = ? THEN 1 ELSE 0 END), g.created_at
		 FROM bv_group_members member
		 JOIN bv_groups g ON g.id = member.group_id
		 LEFT JOIN users u ON u.id = g.owner_id
		 LEFT JOIN bv_group_members m ON m.group_id = g.id
		 WHERE member.user_id = ?
		 GROUP BY g.id, g.name, g.description, g.owner_id, u.username, g.created_at
		 ORDER BY g.created_at DESC, g.id DESC
		 LIMIT ? OFFSET ?`,
		viewerID, memberID, limit, offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user groups"})
	}
	defer rows.Close()
	groups := make([]fiber.Map, 0)
	for rows.Next() {
		var (
			id, ownerID, members             int64
			isMember                         int
			name, description, ownerUsername string
			createdAt                        time.Time
		)
		if err := rows.Scan(&id, &name, &description, &ownerID, &ownerUsername, &members, &isMember, &createdAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user groups"})
		}
		groups = append(groups, groupResponse(id, name, description, ownerID, ownerUsername, members, createdAt, user, isMember != 0))
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load user groups"})
	}
	return c.JSON(fiber.Map{"groups": groups, "total": total, "limit": limit, "offset": offset})
}
