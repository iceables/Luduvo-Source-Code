package service

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"buildoverse/backend/models"
	"buildoverse/backend/repository"
)

type FeedService struct {
	feedRepo *repository.FeedRepository
	userRepo *repository.UserRepository
	cooldown *CooldownService
}

func NewFeedService(feedRepo *repository.FeedRepository, userRepo *repository.UserRepository, cooldown *CooldownService) *FeedService {
	return &FeedService{
		feedRepo: feedRepo,
		userRepo: userRepo,
		cooldown: cooldown,
	}
}

func FormatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return "Just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins <= 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if d < 24*time.Hour {
		hrs := int(d.Hours())
		if hrs <= 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hrs)
	}
	if d < 30*24*time.Hour {
		days := int(d.Hours() / 24)
		if days <= 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	return t.Format("Jan 02, 2006")
}

func (s *FeedService) GetRecentFeed(limit, currentUserID int, feedType string) ([]*models.FeedPost, error) {
	if feedType == "friends" {
		return s.feedRepo.GetRecentFriendsFeedPosts(limit, currentUserID)
	}
	return s.feedRepo.GetRecentFeedPosts(limit, currentUserID)
}

func (s *FeedService) GetRecentFeedPaginated(limit, offset, currentUserID int, feedType string) ([]*models.FeedPost, error) {
	if feedType == "friends" {
		return s.feedRepo.GetRecentFriendsFeedPostsPaginated(limit, offset, currentUserID)
	}
	return s.feedRepo.GetRecentFeedPostsPaginated(limit, offset, currentUserID)
}

func (s *FeedService) PostToFeed(username, content, feedType string) (int, int, error) {
	if username == "" {
		return 0, 0, errors.New("Unauthorized")
	}

	content = html.EscapeString(strings.TrimSpace(content))
	if content == "" {
		return 0, 0, errors.New("Content cannot be empty")
	}

	if len(content) > 150 {
		content = content[:150]
	}

	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return 0, 0, errors.New("User not found")
	}

	if s.cooldown != nil {
		key := fmt.Sprintf("feed_post:%d", user.ID)
		if feedType == "friends" {
			key = fmt.Sprintf("friends_feed_post:%d", user.ID)
		}
		if allowed, remaining := s.cooldown.Allow(key, 15*time.Second); !allowed {
			return 0, user.ID, fmt.Errorf("The feed has a %s cooldown per message.", s.cooldown.FormatRemaining(remaining))
		}
	}

	var postID int
	if feedType == "friends" {
		postID, err = s.feedRepo.CreateFriendsPost(user.ID, content)
	} else {
		postID, err = s.feedRepo.CreatePost(user.ID, content)
	}

	if err != nil {
		return 0, user.ID, err
	}

	return postID, user.ID, nil
}

func (s *FeedService) ToggleReaction(username string, feedID int, feedType string) (int, error) {
	if username == "" || feedID <= 0 {
		return 0, errors.New("invalid request")
	}

	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return 0, errors.New("user not found")
	}

	if s.cooldown != nil {
		key := fmt.Sprintf("reaction:%d:%d:%s", user.ID, feedID, feedType)
		if allowed, remaining := s.cooldown.Allow(key, 2*time.Second); !allowed {
			return 0, fmt.Errorf("Please wait %s before reacting again.", s.cooldown.FormatRemaining(remaining))
		}
	}

	if feedType == "friends" {
		hasReacted, err := s.feedRepo.HasUserReactedFriends(user.ID, feedID)
		if err != nil {
			return 0, err
		}

		if hasReacted {
			if err := s.feedRepo.RemoveReactionFriends(user.ID, feedID); err != nil {
				return 0, err
			}
		} else {
			if err := s.feedRepo.AddReactionFriends(user.ID, feedID); err != nil {
				return 0, err
			}
		}

		return s.feedRepo.GetReactionCountFriends(feedID)
	}

	hasReacted, err := s.feedRepo.HasUserReacted(user.ID, feedID)
	if err != nil {
		return 0, err
	}

	if hasReacted {
		if err := s.feedRepo.RemoveReaction(user.ID, feedID); err != nil {
			return 0, err
		}
	} else {
		if err := s.feedRepo.AddReaction(user.ID, feedID); err != nil {
			return 0, err
		}
	}

	return s.feedRepo.GetReactionCount(feedID)
}

func (s *FeedService) HasUserReacted(username string, feedID int, feedType string) bool {
	if username == "" || feedID <= 0 {
		return false
	}
	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return false
	}
	if feedType == "friends" {
		hasReacted, _ := s.feedRepo.HasUserReactedFriends(user.ID, feedID)
		return hasReacted
	}
	hasReacted, _ := s.feedRepo.HasUserReacted(user.ID, feedID)
	return hasReacted
}

func (s *FeedService) PostComment(username string, feedID int, parentID *int, feedType, comment string) (*models.FeedComment, error) {
	if username == "" {
		return nil, errors.New("Unauthorized")
	}

	comment = html.EscapeString(strings.TrimSpace(comment))
	if comment == "" {
		return nil, errors.New("Comment cannot be empty")
	}

	if len(comment) > 5000 {
		comment = comment[:5000]
	}

	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return nil, errors.New("User not found")
	}

	if s.cooldown != nil {
		key := fmt.Sprintf("feed_comment:%d", user.ID)
		if allowed, remaining := s.cooldown.Allow(key, 5*time.Second); !allowed {
			return nil, fmt.Errorf("Please wait %s before replying again.", s.cooldown.FormatRemaining(remaining))
		}
	}

	commentID, err := s.feedRepo.CreateComment(feedID, user.ID, parentID, feedType, comment)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	res := &models.FeedComment{
		ID:           commentID,
		FeedID:       feedID,
		UserID:       user.ID,
		FeedType:     feedType,
		Username:     user.Username,
		Comment:      comment,
		Removed:      "false",
		Edited:       "false",
		CreationDate: now,
		TimeAgo:      "Just now",
		FullDate:     now.Format("January 02, 2006 at 03:04 PM"),
		Reactions:    0,
		HasReacted:   false,
	}
	if parentID != nil && *parentID > 0 {
		res.ParentID.Int64 = int64(*parentID)
		res.ParentID.Valid = true
	}

	return res, nil
}

func (s *FeedService) GetFeedComments(feedID int, feedType, currentUsername string) ([]*models.FeedComment, error) {
	var currentUserID int
	if currentUsername != "" {
		u, _ := s.userRepo.GetByUsername(currentUsername)
		if u != nil {
			currentUserID = u.ID
		}
	}

	flatList, err := s.feedRepo.GetCommentsByFeedID(feedID, feedType, currentUserID)
	if err != nil {
		return nil, err
	}

	commentMap := make(map[int]*models.FeedComment)
	var rootComments []*models.FeedComment

	for _, c := range flatList {
		c.TimeAgo = FormatTimeAgo(c.CreationDate)
		c.FullDate = c.CreationDate.Format("January 02, 2006 at 03:04 PM")
		commentMap[c.ID] = c
	}

	for _, c := range flatList {
		if c.ParentID.Valid && c.ParentID.Int64 > 0 {
			parentID := int(c.ParentID.Int64)
			if parent, exists := commentMap[parentID]; exists {
				parent.Replies = append(parent.Replies, c)
			} else {
				rootComments = append(rootComments, c)
			}
		} else {
			rootComments = append(rootComments, c)
		}
	}

	return rootComments, nil
}

func (s *FeedService) ToggleCommentReaction(username string, commentID int) (int, error) {
	if username == "" || commentID <= 0 {
		return 0, errors.New("invalid request")
	}

	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return 0, errors.New("user not found")
	}

	if s.cooldown != nil {
		key := fmt.Sprintf("creaction:%d:%d", user.ID, commentID)
		if allowed, remaining := s.cooldown.Allow(key, 2*time.Second); !allowed {
			return 0, fmt.Errorf("Please wait %s before reacting again.", s.cooldown.FormatRemaining(remaining))
		}
	}

	hasReacted, err := s.feedRepo.HasUserReactedComment(user.ID, commentID)
	if err != nil {
		return 0, err
	}

	if hasReacted {
		if err := s.feedRepo.RemoveCommentReaction(user.ID, commentID); err != nil {
			return 0, err
		}
	} else {
		if err := s.feedRepo.AddCommentReaction(user.ID, commentID); err != nil {
			return 0, err
		}
	}

	return s.feedRepo.GetCommentReactionCount(commentID)
}

func (s *FeedService) HasUserReactedComment(username string, commentID int) bool {
	if username == "" || commentID <= 0 {
		return false
	}
	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return false
	}
	hasReacted, _ := s.feedRepo.HasUserReactedComment(user.ID, commentID)
	return hasReacted
}