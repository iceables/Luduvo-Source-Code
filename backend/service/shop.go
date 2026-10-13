package service

import (
	"errors"
	"strings"

	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type ShopService struct {
	shopRepo *repository.ShopRepository
}

func NewShopService(shopRepo *repository.ShopRepository) *ShopService {
	return &ShopService{shopRepo: shopRepo}
}

func (s *ShopService) GetShopItems(filter *models.ShopFilter) ([]*models.ShopItem, int, error) {
	if s.shopRepo == nil {
		return nil, 0, errors.New("shop repository uninitialized")
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 12
	}

	filter.Search = strings.TrimSpace(filter.Search)
	filter.Creator = strings.TrimSpace(filter.Creator)

	return s.shopRepo.GetItems(filter)
}

func (s *ShopService) GetShopItemByID(id int) (*models.ShopItem, error) {
	if s.shopRepo == nil {
		return nil, errors.New("shop repository uninitialized")
	}
	if id <= 0 {
		return nil, errors.New("invalid item ID")
	}
	return s.shopRepo.GetItemByID(id)
}
