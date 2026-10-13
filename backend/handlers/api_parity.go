package handlers

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/models"
	"buildoverse/backend/repository"
	"buildoverse/backend/service"
)

func MeOutfits(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	out := []fiber.Map{}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Outfit service unavailable"})
	}
	rows, err := service.Avatar.GetOutfits(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load outfits"})
	}
	for _, o := range rows {
		createdAt := o.CreatedAt.Unix()
		if createdAt <= 0 {
			createdAt = time.Now().Unix()
		}
		out = append(out, fiber.Map{
			"id":                    o.ID,
			"name":                  o.Name,
			"created_at":            createdAt,
			"updated_at":            createdAt,
			"skin_tone":             nil,
			"headshot_key":          nil,
			"headshot_has_variants": false,
		})
	}
	return c.JSON(fiber.Map{"outfits": out})
}

func MeOutfitsCreate(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Outfit service unavailable"})
	}
	var request struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid outfit"})
	}
	outfit, err := service.Avatar.CreateOutfit(user.ID, request.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(outfitResponse(outfit))
}

func MeOutfitRename(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Outfit service unavailable"})
	}
	outfitID, err := strconv.Atoi(c.Params("id"))
	if err != nil || outfitID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid outfit ID"})
	}
	var request struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid outfit"})
	}
	if err := service.Avatar.RenameOutfit(user.ID, outfitID, request.Name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Outfit not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

func MeOutfitDelete(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Outfit service unavailable"})
	}
	outfitID, err := strconv.Atoi(c.Params("id"))
	if err != nil || outfitID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid outfit ID"})
	}
	if err := service.Avatar.DeleteOutfit(user.ID, outfitID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Outfit not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete outfit"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func MeOutfitWear(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Outfit service unavailable"})
	}
	outfitID, err := strconv.Atoi(c.Params("id"))
	if err != nil || outfitID < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid outfit ID"})
	}
	if err := service.Avatar.WearOutfit(user.ID, outfitID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Outfit not found"})
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Outfit belongs to another user"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not wear outfit"})
	}
	return c.JSON(fiber.Map{"status": "worn"})
}

func outfitResponse(outfit *models.Outfit) fiber.Map {
	createdAt := outfit.CreatedAt.Unix()
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	return fiber.Map{
		"id":                    outfit.ID,
		"name":                  outfit.Name,
		"created_at":            createdAt,
		"updated_at":            createdAt,
		"skin_tone":             nil,
		"headshot_key":          nil,
		"headshot_has_variants": false,
	}
}

func MeInventory(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Inventory service unavailable"})
	}

	limit := parsePageSize(c.Query("limit"), 100)
	offset := parsePageOffset(c.Query("offset"))
	category := inventoryCategory(c)
	items, err := service.Avatar.GetInventory(user.ID, category, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load inventory"})
	}
	total := len(items)
	page := itemsOnPage(items, offset, limit)
	result := make([]fiber.Map, 0, len(page))
	for _, item := range page {
		result = append(result, inventoryItemResponse(item))
	}
	return c.JSON(fiber.Map{"items": result, "total": total, "limit": limit, "offset": offset})
}

func inventoryCategory(c fiber.Ctx) string {
	category := c.Query("asset_type")
	switch category {
	case "accessory":
		return c.Query("slot", "hat")
	case "flat_shirt":
		return "shirts"
	case "flat_pant":
		return "pants"
	case "tshirt":
		return "tshirts"
	case "face":
		return "faces"
	case "mesh":
		return "gear"
	default:
		return category
	}
}

func inventoryItemResponse(item *models.InventoryItem) fiber.Map {
	assetType := item.Type
	slot := item.Type
	switch item.Type {
	case "hat":
		assetType, slot = "accessory", "hat"
	case "shirts":
		assetType = "flat_shirt"
	case "tshirts":
		assetType = "tshirt"
	case "pants":
		assetType = "flat_pant"
	case "faces":
		assetType = "face"
	case "gear":
		assetType = "mesh"
	}
	return fiber.Map{
		"id":               item.ID,
		"item_id":          item.ID,
		"name":             item.Name,
		"item_name":        item.Name,
		"type":             item.Type,
		"type_name":        item.TypeName,
		"asset_type":       assetType,
		"slot":             slot,
		"description":      item.Description,
		"creator_username": item.CreatorName,
		"variant_index":    0,
		"is_equipped":      item.IsEquipped,
		"thumbnail_url":    "/static/renders/shop/" + item.Type + "/" + strconv.Itoa(item.ID),
	}
}

func MeEquipped(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Avatar service unavailable"})
	}
	items := []any{}
	equipped, err := service.Avatar.GetEquippedItems(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load equipped items"})
	}
	for _, item := range equipped {
		items = append(items, fiber.Map{
			"item_id":       item.ID,
			"variant_index": 0,
			"name":          item.Name,
			"type":          item.Type,
			"slot":          item.Type,
		})
	}
	return c.JSON(fiber.Map{"equipped_items": items, "items": items})
}

func MeEquippedPut(c fiber.Ctx) error {
	user := getCurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	if service.Avatar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Avatar service unavailable"})
	}
	var request struct {
		Items []struct {
			ItemID       int `json:"item_id"`
			VariantIndex int `json:"variant_index"`
		} `json:"items"`
	}
	if err := json.Unmarshal(c.Body(), &request); err != nil || request.Items == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid equipped items"})
	}
	itemIDs := make([]int, 0, len(request.Items))
	for _, item := range request.Items {
		if item.VariantIndex != 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Item variants are not supported"})
		}
		itemIDs = append(itemIDs, item.ItemID)
	}
	if err := service.Avatar.SetEquippedItems(user.ID, itemIDs); err != nil {
		switch {
		case errors.Is(err, repository.ErrInventoryItemNotOwned):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Item is not in your inventory"})
		case errors.Is(err, repository.ErrInvalidEquippedItem):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid equipped items"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update equipped items"})
		}
	}
	return c.JSON(fiber.Map{"ok": true})
}

func MeBundle(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"bundle": nil})
}

func MeSubscription(c fiber.Ctx) error {
	user := getCurrentUser(c)
	tier := "free"
	status := ""
	if user != nil {
		if b, err := os.ReadFile("data/subscriptions.json"); err == nil {
			var all map[string]subRecord
			if json.Unmarshal(b, &all) == nil {
				if rec, ok := all[strings.ToLower(user.Username)]; ok && rec.Tier != "" {
					tier = rec.Tier
					status = "active"
				}
			}
		}
	}
	return c.JSON(fiber.Map{
		"tier":                 tier,
		"status":               status,
		"interval":             "monthly",
		"cancel_at_period_end": false,
	})
}

func MeSubscriptionCancel(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": true, "status": "canceled"})
}

func MePendingPayments(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"pending": []any{}})
}

func MeCheckout(c fiber.Ctx) error {
	return UpgradeCheckout(c)
}

func NotificationsList(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"notifications": []any{}, "unread": 0})
}

func GroupsList(c fiber.Ctx) error {
	return GroupsAPI(c)
}

func AuthLoginMFA(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"mfa_required": false})
}

func AuthLogoutAll(c fiber.Ctx) error {
	return LogoutAPI(c)
}
