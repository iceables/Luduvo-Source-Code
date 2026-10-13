package service

import (
	"errors"
	"fmt"
	"strings"

	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type ModHistoryService struct {
	modHistRepo *repository.ModHistoryRepository
	userRepo    *repository.UserRepository
}

func NewModHistoryService(modHistRepo *repository.ModHistoryRepository, userRepo *repository.UserRepository) *ModHistoryService {
	return &ModHistoryService{
		modHistRepo: modHistRepo,
		userRepo:    userRepo,
	}
}

func (s *ModHistoryService) GetByUserID(userID int) ([]*models.ModHistory, error) {
	if s.modHistRepo == nil {
		return nil, nil
	}
	return s.modHistRepo.GetByUserID(userID)
}

func (s *ModHistoryService) GetAllLogs(limit, offset int) ([]*models.ModHistory, int, error) {
	if s.modHistRepo == nil {
		return nil, 0, nil
	}
	return s.modHistRepo.GetAll(limit, offset)
}

func (s *ModHistoryService) normalizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "No reason provided"
	}
	if len(reason) > 500 {
		return reason[:500]
	}
	return reason
}

func (s *ModHistoryService) ensureCanModerate(admin *models.User, targetID int) (*models.User, error) {
	if admin == nil {
		return nil, errors.New("Unauthorized")
	}
	if !admin.HasPower(models.PowerModerator) {
		return nil, errors.New("Insufficient permission")
	}
	if s.userRepo == nil || s.modHistRepo == nil {
		return nil, errors.New("moderation service unavailable")
	}
	if targetID == admin.ID {
		return nil, errors.New("You cannot moderate your own account")
	}

	target, err := s.userRepo.GetByID(targetID)
	if err != nil || target == nil {
		return nil, errors.New("User not found")
	}
	if admin.Power <= target.Power {
		return nil, errors.New("You cannot moderate a user of equal or higher rank")
	}
	return target, nil
}

func (s *ModHistoryService) applyScrub(admin *models.User, targetID int, actionType, reason, alreadyMsg string, getCurrent func(*models.User) string, scrubbed func() string, apply func(string) error) error {
	target, err := s.ensureCanModerate(admin, targetID)
	if err != nil {
		return err
	}

	current := getCurrent(target)
	if current == scrubbed() {
		return errors.New(alreadyMsg)
	}

	if err := apply(scrubbed()); err != nil {
		return err
	}

	_, err = s.modHistRepo.Create(targetID, admin.ID, actionType, s.normalizeReason(reason), current, models.StatusActive)
	return err
}

func (s *ModHistoryService) ScrubDescription(admin *models.User, targetID int, reason string) error {
	return s.applyScrub(admin, targetID, models.ActionScrubDescription, reason,
		"Description is already scrubbed",
		func(u *models.User) string { return u.Description },
		func() string { return models.ScrubbedDescription },
		func(v string) error { return s.userRepo.UpdateBio(targetID, v) })
}

func (s *ModHistoryService) ScrubDisplayName(admin *models.User, targetID int, reason string) error {
	return s.applyScrub(admin, targetID, models.ActionScrubDisplayName, reason,
		"Display name is already scrubbed",
		func(u *models.User) string { return u.DisplayName },
		func() string { return models.ScrubbedDisplayName },
		func(v string) error { return s.userRepo.AdminSetDisplayName(targetID, v) })
}

func (s *ModHistoryService) ScrubPronouns(admin *models.User, targetID int, reason string) error {
	return s.applyScrub(admin, targetID, models.ActionScrubPronouns, reason,
		"Pronouns are already empty",
		func(u *models.User) string { return u.Pronouns },
		func() string { return "" },
		func(v string) error { return s.userRepo.UpdatePronouns(targetID, v) })
}

func (s *ModHistoryService) ScrubUsername(admin *models.User, targetID int, reason string) error {
	return s.applyScrub(admin, targetID, models.ActionScrubUsername, reason,
		"Username is already scrubbed",
		func(u *models.User) string { return u.Username },
		func() string { return models.ScrubbedUsername(targetID) },
		func(v string) error { return s.userRepo.AdminSetUsername(targetID, v) })
}

func (s *ModHistoryService) Record(admin *models.User, targetID int, actionType, reason, note string) error {
	_, err := s.ensureCanModerate(admin, targetID)
	if err != nil {
		return err
	}
	_, err = s.modHistRepo.Create(targetID, admin.ID, actionType, s.normalizeReason(reason), note, models.StatusActive)
	return err
}

func (s *ModHistoryService) RecordAssetReview(admin *models.User, targetUID, assetID int, assetName, actionType, note string) error {
	if admin == nil {
		return errors.New("Unauthorized")
	}
	if !admin.HasAdminAccess() {
		return errors.New("Insufficient permission")
	}
	if s.modHistRepo == nil {
		return errors.New("moderation service unavailable")
	}
	reason := strings.TrimSpace(note)
	if reason == "" {
		if actionType == models.ActionApproveAsset {
			reason = fmt.Sprintf("Asset #%d (%s) approved", assetID, assetName)
		} else {
			reason = fmt.Sprintf("Asset #%d (%s) rejected", assetID, assetName)
		}
	}
	_, err := s.modHistRepo.Create(targetUID, admin.ID, actionType, s.normalizeReason(reason), fmt.Sprintf("Asset #%d (%s)", assetID, assetName), models.StatusActive)
	return err
}

func (s *ModHistoryService) restoreField(actionType string, uid int, note string) error {
	switch actionType {
	case models.ActionScrubDescription:
		return s.userRepo.UpdateBio(uid, note)
	case models.ActionScrubDisplayName:
		return s.userRepo.AdminSetDisplayName(uid, note)
	case models.ActionScrubPronouns:
		return s.userRepo.UpdatePronouns(uid, note)
	case models.ActionScrubUsername:
		if note == "" {
			return errors.New("The previous username was not recorded for this action")
		}
		if taken, _ := s.userRepo.IsOldUsernameExceptUser(note, uid); taken {
			return fmt.Errorf("Cannot restore username %q: it is no longer available", note)
		}
		if existing, _ := s.userRepo.GetByUsername(note); existing != nil && existing.ID != uid {
			return fmt.Errorf("Cannot restore username %q: it is no longer available", note)
		}
		return s.userRepo.AdminSetUsername(uid, note)
	default:
		return errors.New("This action type cannot be retracted")
	}
}

func (s *ModHistoryService) Retract(admin *models.User, modHistID int) error {
	if admin == nil {
		return errors.New("Unauthorized")
	}
	if !admin.HasPower(models.PowerModerator) {
		return errors.New("Insufficient permission")
	}
	if s.userRepo == nil || s.modHistRepo == nil {
		return errors.New("moderation service unavailable")
	}

	entry, err := s.modHistRepo.GetByID(modHistID)
	if err != nil {
		return err
	}
	if entry == nil {
		return errors.New("Moderation record not found")
	}
	if entry.Status != models.StatusActive {
		return errors.New("Only active moderation records can be retracted")
	}
	if admin.Power < entry.AdminPower && admin.ID != entry.AdminID {
		return errors.New("You must outrank the issuing moderator to retract this action")
	}

	target, err := s.userRepo.GetByID(entry.UID)
	if err != nil || target == nil {
		return errors.New("Target user not found")
	}
	if admin.Power <= target.Power {
		return errors.New("You cannot retract actions on a user of equal or higher rank")
	}

	prev := ""
	if entry.Note.Valid {
		prev = entry.Note.String
	}
	if err := s.restoreField(entry.ActionType, entry.UID, prev); err != nil {
		return err
	}

	return s.modHistRepo.UpdateStatus(modHistID, models.StatusRetracted)
}