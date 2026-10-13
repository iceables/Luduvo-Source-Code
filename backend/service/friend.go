package service

import (
	"errors"
	"fmt"
	"time"

	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type FriendService struct {
	friendRepo *repository.FriendRepository
	userRepo   *repository.UserRepository
	cooldown   *CooldownService
}

func NewFriendService(friendRepo *repository.FriendRepository, userRepo *repository.UserRepository, cooldown *CooldownService) *FriendService {
	return &FriendService{
		friendRepo: friendRepo,
		userRepo:   userRepo,
		cooldown:   cooldown,
	}
}

func (s *FriendService) checkFriendAction(action string, userID, targetID int) error {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return errors.New("invalid user")
	}
	if s.cooldown != nil {
		key := fmt.Sprintf("action:friend_%s:%d", action, userID)
		if allowed, remaining := s.cooldown.Allow(key, 2*time.Second); !allowed {
			return fmt.Errorf("please wait %s before performing this action again", s.cooldown.FormatRemaining(remaining))
		}
	}
	return nil
}

func (s *FriendService) GetFriendStatus(userID, targetID int) (string, error) {
	if userID <= 0 || targetID <= 0 || userID == targetID {
		return "none", nil
	}
	return s.friendRepo.GetFriendStatus(userID, targetID)
}

func (s *FriendService) GetPendingRequestCount(userID int) (int, error) {
	if userID <= 0 {
		return 0, nil
	}
	return s.friendRepo.GetPendingRequestCount(userID)
}

func (s *FriendService) GetFriendsList(userID int) ([]*models.FriendUserInfo, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}
	return s.friendRepo.GetFriendsList(userID)
}

func (s *FriendService) GetFriendRequestsPageData(userID int) ([]*models.FriendUserInfo, []*models.FriendUserInfo, []*models.FriendUserInfo, error) {
	if userID <= 0 {
		return nil, nil, nil, errors.New("invalid user")
	}
	incoming, err := s.friendRepo.GetIncomingRequests(userID)
	if err != nil {
		return nil, nil, nil, err
	}
	outgoing, err := s.friendRepo.GetOutgoingRequests(userID)
	if err != nil {
		return nil, nil, nil, err
	}
	friends, err := s.friendRepo.GetFriendsList(userID)
	if err != nil {
		return nil, nil, nil, err
	}
	return incoming, outgoing, friends, nil
}

func (s *FriendService) SendFriendRequest(senderID, targetID int) error {
	if err := s.checkFriendAction("send", senderID, targetID); err != nil {
		return err
	}
	exists, err := s.userRepo.UserExists(targetID)
	if err != nil || !exists {
		return errors.New("user not found")
	}
	return s.friendRepo.SendRequest(senderID, targetID)
}

func (s *FriendService) AcceptFriendRequest(userID, targetID int) error {
	if err := s.checkFriendAction("accept", userID, targetID); err != nil {
		return err
	}
	return s.friendRepo.AcceptRequest(userID, targetID)
}

func (s *FriendService) DeclineFriendRequest(userID, targetID int) error {
	if err := s.checkFriendAction("decline", userID, targetID); err != nil {
		return err
	}
	return s.friendRepo.DeclineRequest(userID, targetID)
}

func (s *FriendService) CancelFriendRequest(userID, targetID int) error {
	if err := s.checkFriendAction("cancel", userID, targetID); err != nil {
		return err
	}
	return s.friendRepo.CancelRequest(userID, targetID)
}

func (s *FriendService) RemoveFriend(userID, targetID int) error {
	if err := s.checkFriendAction("remove", userID, targetID); err != nil {
		return err
	}
	return s.friendRepo.RemoveFriend(userID, targetID)
}