package handlers

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"buildoverse/backend/models"
	"buildoverse/backend/service"
)

type SearchResultItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	TypeName    string `json:"type_name"`
	CreatorName string `json:"creator_name"`
	Bucks       int    `json:"bucks"`
	Bits        int    `json:"bits"`
}

type SearchResultUser struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type SearchResponse struct {
	Query     string             `json:"query"`
	ShopItems []SearchResultItem `json:"shop_items"`
	Users     []SearchResultUser `json:"users"`
}

func SearchAPI(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		q = strings.TrimSpace(c.Query("search"))
	}

	limit, _ := strconv.Atoi(c.Query("limit", "5"))
	if limit <= 0 || limit > 20 {
		limit = 5
	}

	res := SearchResponse{
		Query:     q,
		ShopItems: []SearchResultItem{},
		Users:     []SearchResultUser{},
	}

	if q == "" {
		return c.JSON(res)
	}

	if service.Shop != nil {
		filter := &models.ShopFilter{
			Search: q,
			Page:   1,
			Limit:  limit,
		}
		items, _, err := service.Shop.GetShopItems(filter)
		if err == nil && items != nil {
			for _, item := range items {
				res.ShopItems = append(res.ShopItems, SearchResultItem{
					ID:          item.ID,
					Name:        item.Name,
					Type:        item.Type,
					TypeName:    item.TypeName,
					CreatorName: item.CreatorName,
					Bucks:       item.Bucks,
					Bits:        item.Bits,
				})
			}
		}
	}

	if service.User != nil {
		users, err := service.User.SearchUsers(q, limit)
		if err == nil && users != nil {
			for _, u := range users {
				dispName := u.DisplayName
				if dispName == "" {
					dispName = u.Username
				}
				res.Users = append(res.Users, SearchResultUser{
					ID:          u.ID,
					Username:    u.Username,
					DisplayName: dispName,
				})
			}
		}
	}

	return c.JSON(res)
}

func SearchRedirect(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		q = strings.TrimSpace(c.Query("search"))
	}
	return c.Redirect().To("/shop?q=" + url.QueryEscape(q))
}