package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
	cooldown *CooldownService
}

func NewUserService(userRepo *repository.UserRepository, cooldown *CooldownService) *UserService {
	return &UserService{
		userRepo: userRepo,
		cooldown: cooldown,
	}
}

func (s *UserService) GetUserByID(id int) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) TouchOnline(userID int) error {
	if s == nil || s.userRepo == nil {
		return nil
	}
	return s.userRepo.UpdateLastOnline(userID)
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	return s.userRepo.GetByUsername(username)
}

func (s *UserService) SetEmailVerification(userID int, token string, expiry int64) error {
	if s == nil || s.userRepo == nil {
		return errors.New("user service offline")
	}
	return s.userRepo.SetEmailVerification(userID, token, expiry)
}

func (s *UserService) MarkEmailVerified(userID int, email string) error {
	if s == nil || s.userRepo == nil {
		return errors.New("user service offline")
	}
	return s.userRepo.MarkEmailVerified(userID, email)
}

func (s *UserService) GetUserByEmailToken(token string) (*models.User, error) {
	if s == nil || s.userRepo == nil {
		return nil, errors.New("user service offline")
	}
	return s.userRepo.GetByEmailToken(token)
}

func (s *UserService) GetUserByUnikey(unikey string) (*models.User, error) {
	if s.userRepo == nil {
		return nil, errors.New("user repository uninitialized")
	}
	return s.userRepo.GetByUnikey(unikey)
}

func (s *UserService) SearchAdminUsers(search string, powerFilter int, limit, offset int) ([]*models.User, int, error) {
	return s.userRepo.SearchAdminUsers(search, powerFilter, limit, offset)
}

func (s *UserService) SearchUsers(query string, limit int) ([]*models.User, error) {
	if s.userRepo == nil {
		return []*models.User{}, nil
	}
	return s.userRepo.SearchUsers(query, limit)
}

func (s *UserService) GetUserCount() (int, error) {
	return s.userRepo.GetUserCount()
}

func (s *UserService) GetRecentUsers(excludeUsername string, limit int) ([]*models.User, error) {
	return s.userRepo.GetRecentUsers(excludeUsername, limit)
}

func (s *UserService) UserExists(userID int) (bool, error) {
	return s.userRepo.UserExists(userID)
}

func (s *UserService) GetUsernameChangesLeft(userID int) (int, error) {
	return s.userRepo.GetUsernameChangesLeft(userID)
}

func (s *UserService) GetOldUsernames(userID, limit, offset int) ([]models.UsernameHistory, int, error) {
	if s == nil || s.userRepo == nil {
		return nil, 0, errors.New("user service offline")
	}
	return s.userRepo.GetOldUsernames(userID, limit, offset)
}

func (s *UserService) ChangeDisplayName(userID int, newDisplayName string) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:displayname:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}

	newDisplayName = html.EscapeString(strings.TrimSpace(newDisplayName))
	if newDisplayName == "" {
		return errors.New("Display name cannot be empty")
	}

	if utf8.RuneCountInString(newDisplayName) < 3 || utf8.RuneCountInString(newDisplayName) > 30 {
		return errors.New("Display name must be between 3 and 30 characters")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.New("User not found")
	}

	if newDisplayName == user.DisplayName {
		return errors.New("New display name must be different from current display name")
	}

	return s.userRepo.ChangeDisplayName(userID, newDisplayName, user.DisplayName)
}

func (s *UserService) ChangeUsername(userID int, currentUsername, newUsername string) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:username:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}

	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return errors.New("Username cannot be empty")
	}

	if strings.Contains(newUsername, " ") {
		return errors.New("Username cannot contain spaces")
	}

	if newUsername == currentUsername {
		return errors.New("New username must be different from current username")
	}

	if len(newUsername) < 3 || len(newUsername) > 25 {
		return errors.New("Username must be between 3 and 25 characters")
	}

	matched, _ := regexp.MatchString("^[a-zA-Z0-9]+$", newUsername)
	if !matched {
		return errors.New("Username can only contain alphanumeric characters")
	}

	return s.userRepo.ChangeUsername(userID, currentUsername, newUsername, 100)
}

func (s *UserService) ChangePassword(userID int, currentPassword, newPassword, retypePassword string) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:password:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}

	if currentPassword == "" || newPassword == "" || retypePassword == "" {
		return errors.New("All password fields are required")
	}

	if newPassword != retypePassword {
		return errors.New("New passwords do not match")
	}

	if len(newPassword) < 8 {
		return errors.New("New password must be at least 8 characters long")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.New("User not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword))
	if err != nil {
		return errors.New("Current password is incorrect")
	}

	newHashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("Hashing error")
	}

	return s.userRepo.UpdatePassword(userID, string(newHashed))
}

func (s *UserService) UpdateBio(userID int, bio string) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:bio:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}

	escapedBio := html.EscapeString(bio)
	if utf8.RuneCountInString(escapedBio) > 1024 {
		return errors.New("Bio cannot be longer than 1024 characters")
	}

	return s.userRepo.UpdateBio(userID, escapedBio)
}

func (s *UserService) UpdatePronouns(userID int, pronouns string) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:pronouns:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}

	escapedPronouns := html.EscapeString(strings.TrimSpace(pronouns))
	if utf8.RuneCountInString(escapedPronouns) > 16 {
		return errors.New("Pronouns cannot be longer than 16 characters")
	}

	return s.userRepo.UpdatePronouns(userID, escapedPronouns)
}

var AllowedSocialPlatforms = []string{"discord", "twitter", "youtube", "twitch", "github", "instagram", "tiktok", "steam"}

func (s *UserService) UpdateSocials(userID int, socialsMap map[string]string) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:socials:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}

	cleanMap := make(map[string]string)
	for platform, val := range socialsMap {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if !slices.Contains(AllowedSocialPlatforms, platform) {
			continue
		}
		val = html.EscapeString(strings.TrimSpace(val))
		if len(val) > 120 {
			val = val[:120]
		}
		if val != "" {
			cleanMap[platform] = val
		}
	}

	bytesData, err := json.Marshal(cleanMap)
	if err != nil {
		return errors.New("Failed to encode socials")
	}

	return s.userRepo.UpdateSocials(userID, string(bytesData))
}

func (s *UserService) UpdateMusicID(userID int, trackID int64) error {
	if s.cooldown != nil {
		key := fmt.Sprintf("action:music:%d", userID)
		if allowed, remaining := s.cooldown.Allow(key); !allowed {
			return fmt.Errorf("Please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}
	return s.userRepo.UpdateMusicID(userID, trackID)
}

func (s *UserService) GetProfileData(userID int, activeUsername string) (*models.User, bool, int, int, int, error) {
	targetUser, err := s.userRepo.GetByID(userID)
	if err != nil || targetUser == nil {
		return nil, false, 0, 0, 0, errors.New("User not found")
	}

	if targetUser.DisplayName == "" {
		targetUser.DisplayName = targetUser.Username
	}

	var isOwnProfile bool
	if activeUsername != "" {
		activeUser, _ := s.userRepo.GetByUsername(activeUsername)
		if activeUser != nil && activeUser.ID == targetUser.ID {
			isOwnProfile = true
		}
	}

	friends, followers, following := s.userRepo.GetUserSocialCounts(userID)
	return targetUser, isOwnProfile, friends, followers, following, nil
}
