package service

import (
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type AvatarService struct {
	avatarRepo *repository.AvatarRepository
	userRepo   *repository.UserRepository
}

func NewAvatarService(avatarRepo *repository.AvatarRepository, userRepo *repository.UserRepository) *AvatarService {
	return &AvatarService{
		avatarRepo: avatarRepo,
		userRepo:   userRepo,
	}
}

func (s *AvatarService) InvalidateRenderCache(userID int) {
	idStr := fmt.Sprintf("%d.png", userID)
	_ = os.Remove(filepath.Join("static", "renders", "avatars", "full", idStr))
	_ = os.Remove(filepath.Join("static", "renders", "avatars", "headshots", idStr))
}

func (s *AvatarService) MigrateDefaultAvatarPalette() ([]int, error) {
	if s.avatarRepo == nil {
		return nil, errors.New("avatar repository uninitialized")
	}
	userIDs, err := s.avatarRepo.MigrateDefaultAvatarPalette()
	if err != nil {
		return nil, err
	}
	for _, userID := range userIDs {
		s.InvalidateRenderCache(userID)
	}
	return userIDs, nil
}

func (s *AvatarService) GetAvatar(userID int) (*models.Avatar, error) {
	if s.avatarRepo == nil {
		return nil, errors.New("avatar repository uninitialized")
	}
	return s.avatarRepo.GetAvatar(userID)
}

func (s *AvatarService) UpdateBodyColor(userID int, bodyPart, color string) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	err := s.avatarRepo.UpdateBodyColor(userID, bodyPart, color)
	if err == nil {
		s.InvalidateRenderCache(userID)
	}
	return err
}

func (s *AvatarService) EquipItem(userID int, itemType string, itemID int) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	err := s.avatarRepo.EquipItem(userID, itemType, itemID)
	if err == nil {
		s.InvalidateRenderCache(userID)
	}
	return err
}

func (s *AvatarService) UnequipItem(userID int, itemType string, itemID int) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	err := s.avatarRepo.UnequipItem(userID, itemType, itemID)
	if err == nil {
		s.InvalidateRenderCache(userID)
	}
	return err
}

func (s *AvatarService) SetEquippedItems(userID int, itemIDs []int) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	if err := s.avatarRepo.SetEquippedItems(userID, itemIDs); err != nil {
		return err
	}
	s.InvalidateRenderCache(userID)
	return nil
}

func (s *AvatarService) GetInventory(userID int, category, search string) ([]*models.InventoryItem, error) {
	if s.avatarRepo == nil {
		return nil, errors.New("avatar repository uninitialized")
	}
	return s.avatarRepo.GetInventory(userID, category, search)
}

func (s *AvatarService) GetEquippedItems(userID int) ([]*models.InventoryItem, error) {
	if s.avatarRepo == nil {
		return nil, errors.New("avatar repository uninitialized")
	}
	return s.avatarRepo.GetEquippedItems(userID)
}

func (s *AvatarService) GetOutfits(userID int) ([]*models.Outfit, error) {
	if s.avatarRepo == nil {
		return nil, errors.New("avatar repository uninitialized")
	}
	return s.avatarRepo.GetOutfits(userID)
}

func (s *AvatarService) CreateOutfit(userID int, name string) (*models.Outfit, error) {
	if s.avatarRepo == nil {
		return nil, errors.New("avatar repository uninitialized")
	}
	name = html.EscapeString(strings.TrimSpace(name))
	if name == "" {
		return nil, errors.New("outfit name cannot be empty")
	}
	if len(name) > 30 {
		name = name[:30]
	}
	outfits, err := s.avatarRepo.GetOutfits(userID)
	if err != nil {
		return nil, err
	}
	if len(outfits) >= 20 {
		return nil, errors.New("maximum limit of 20 saved outfits reached")
	}
	return s.avatarRepo.CreateOutfit(userID, name)
}

func (s *AvatarService) RenameOutfit(userID, outfitID int, name string) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	name = html.EscapeString(strings.TrimSpace(name))
	if name == "" {
		return errors.New("outfit name cannot be empty")
	}
	if len(name) > 30 {
		name = name[:30]
	}
	return s.avatarRepo.RenameOutfit(userID, outfitID, name)
}

func (s *AvatarService) WearOutfit(userID, outfitID int) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	err := s.avatarRepo.WearOutfit(userID, outfitID)
	if err == nil {
		s.InvalidateRenderCache(userID)
	}
	return err
}

func (s *AvatarService) DeleteOutfit(userID, outfitID int) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	err := s.avatarRepo.DeleteOutfit(userID, outfitID)
	if err == nil {
		_ = os.Remove(filepath.Join("static", "renders", "outfits", fmt.Sprintf("%d.png", outfitID)))
	}
	return err
}

func (s *AvatarService) ResetAvatar(userID int) error {
	if s.avatarRepo == nil {
		return errors.New("avatar repository uninitialized")
	}
	err := s.avatarRepo.ResetAvatar(userID)
	if err == nil {
		s.InvalidateRenderCache(userID)
	}
	return err
}
