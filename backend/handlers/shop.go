package handlers

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

func extractShopFilter(c fiber.Ctx) *models.ShopFilter {
	search := strings.TrimSpace(c.Query("q"))
	if search == "" {
		search = strings.TrimSpace(c.Query("search"))
	}

	category := strings.TrimSpace(c.Query("category"))
	if category == "" {
		category = strings.TrimSpace(c.Query("cat"))
	}
	if category == "" {
		category = "all"
	}

	currency := strings.TrimSpace(c.Query("currency"))
	minPrice, _ := strconv.Atoi(c.Query("min_price", "0"))
	maxPrice, _ := strconv.Atoi(c.Query("max_price", "0"))
	creator := strings.TrimSpace(c.Query("creator"))
	sort := strings.TrimSpace(c.Query("sort", "newest"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return &models.ShopFilter{
		Search:   search,
		Category: category,
		Currency: currency,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		Creator:  creator,
		Sort:     sort,
		Page:     page,
		Limit:    limit,
	}
}

func ShopGet(c fiber.Ctx) error {
	filter := extractShopFilter(c)

	var items []*models.ShopItem
	totalItems := 0

	if service.Shop != nil {
		var err error
		items, totalItems, err = service.Shop.GetShopItems(filter)
		if err != nil {
			log.Printf("ShopGet error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Could not load store items")
		}
	} else {
		return c.Status(fiber.StatusServiceUnavailable).SendString("Shop service unavailable")
	}

	totalPages := 1
	if totalItems > 0 && filter.Limit > 0 {
		totalPages = (totalItems + filter.Limit - 1) / filter.Limit
	}

	normCat, normCatName := models.NormalizeShopCategory(filter.Category)

	return Render(c, "pages/shop", fiber.Map{
		"Title":         "Shop - Buildoverse",
		"Items":         items,
		"TotalItems":    totalItems,
		"TotalPages":    totalPages,
		"CurrentPage":   filter.Page,
		"Filter":        filter,
		"ActiveCat":     normCat,
		"ActiveCatName": normCatName,
	}, "layouts/main")
}

func ShopItemsAPI(c fiber.Ctx) error {
	filter := extractShopFilter(c)

	if service.Shop == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Shop service unavailable"})
	}

	items, total, err := service.Shop.GetShopItems(filter)
	if err != nil {
		log.Printf("ShopItemsAPI error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load store items"})
	}

	totalPages := 1
	if total > 0 && filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	return c.JSON(fiber.Map{
		"items":       items,
		"total":       total,
		"page":        filter.Page,
		"limit":       filter.Limit,
		"total_pages": totalPages,
	})
}

func ShopItemDetailAPI(c fiber.Ctx) error {
	itemID, err := strconv.Atoi(c.Params("id"))
	if err != nil || itemID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid item ID"})
	}

	if service.Shop == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Shop service uninitialized"})
	}

	item, err := service.Shop.GetShopItemByID(itemID)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Item not found"})
	}
	if err != nil {
		log.Printf("ShopItemDetailAPI error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not load store item"})
	}
	if item == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Item not found"})
	}

	return c.JSON(item)
}
