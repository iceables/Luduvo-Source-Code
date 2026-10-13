package service

import (
	"errors"
	"strings"

	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type AssetService struct {
	assetRepo *repository.AssetRepository
}

func NewAssetService(assetRepo *repository.AssetRepository) *AssetService {
	return &AssetService{assetRepo: assetRepo}
}

func (s *AssetService) CreateAsset(uid int, name, description, assetType, filePath string) (int, error) {
	if s.assetRepo == nil {
		return 0, errors.New("asset service unavailable")
	}
	if uid <= 0 {
		return 0, errors.New("invalid user")
	}

	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return 0, errors.New("asset name is required")
	}
	if len(name) > 60 {
		return 0, errors.New("asset name is too long")
	}
	if len(description) > 500 {
		return 0, errors.New("asset description is too long")
	}

	normalizedType, _ := models.NormalizeAssetType(assetType)
	return s.assetRepo.Create(uid, name, description, normalizedType, filePath)
}

func (s *AssetService) GetUserAssets(uid int) ([]*models.Asset, error) {
	if s.assetRepo == nil {
		return nil, nil
	}
	return s.assetRepo.GetByUserID(uid)
}

func (s *AssetService) GetByID(id int) (*models.Asset, error) {
	if s.assetRepo == nil {
		return nil, errors.New("asset service unavailable")
	}
	if id <= 0 {
		return nil, errors.New("invalid asset ID")
	}
	return s.assetRepo.GetByID(id)
}

func (s *AssetService) GetQueue(assetType string, page, limit int) ([]*models.Asset, int, error) {
	if s.assetRepo == nil {
		return nil, 0, errors.New("asset service unavailable")
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 12
	}

	filter := ""
	if t := strings.ToLower(strings.TrimSpace(assetType)); t != "" {
		if _, ok := models.AssetTypeNames[t]; ok {
			filter = t
		}
	}

	return s.assetRepo.GetQueue(filter, limit, (page-1)*limit)
}

func (s *AssetService) Review(id int, state string, reviewerID int, note string) error {
	if s.assetRepo == nil {
		return errors.New("asset service unavailable")
	}
	if id <= 0 {
		return errors.New("invalid asset ID")
	}

	state = models.NormalizeAssetApproval(state)
	if state == models.AssetApprovalPending {
		return errors.New("invalid review state")
	}
	if reviewerID <= 0 {
		return errors.New("invalid reviewer")
	}

	note = strings.TrimSpace(note)
	if len(note) > 500 {
		return errors.New("review note is too long")
	}

	asset, err := s.assetRepo.GetByID(id)
	if err != nil {
		return err
	}
	if asset == nil {
		return errors.New("asset not found")
	}
	if !asset.IsPending() {
		return errors.New("asset has already been reviewed")
	}

	return s.assetRepo.UpdateApproval(id, state, reviewerID, note)
}