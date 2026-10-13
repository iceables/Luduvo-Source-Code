package models

import (
	"database/sql"
	"time"
)

type Game struct {
	ID          int
	Name        string
	Description sql.NullString
	CreatorID   sql.NullInt64
	Genre       sql.NullString
	Visits      int
	Playing     int
	Server      sql.NullString
	Private     bool
	Locked      bool
	CreatedAt   time.Time
	EditedAt    time.Time
	Tags        sql.NullString
}