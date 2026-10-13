package models

import (
	"database/sql"
	"time"
)

type FeedPost struct {
	ID           int
	UserID       int
	Username     string
	Content      string
	Removed      string
	Edited       string
	EditDate     sql.NullTime
	CreationDate time.Time
	Reactions    int
	HasReacted   bool
}

type FeedComment struct {
	ID           int            `json:"id"`
	FeedID       int            `json:"feed_id"`
	UserID       int            `json:"user_id"`
	ParentID     sql.NullInt64  `json:"parent_id"`
	FeedType     string         `json:"feed_type"`
	Username     string         `json:"username"`
	Comment      string         `json:"comment"`
	Removed      string         `json:"removed"`
	Edited       string         `json:"edited"`
	CreationDate time.Time      `json:"creation_date"`
	TimeAgo      string         `json:"time_ago"`
	FullDate     string         `json:"full_date"`
	Reactions    int            `json:"reactions"`
	HasReacted   bool           `json:"has_reacted"`
	Replies      []*FeedComment `json:"replies,omitempty"`
}