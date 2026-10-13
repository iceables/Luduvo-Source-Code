package models

type FriendRelation struct {
	ID    int
	UID   int
	FID   int
	State string
}

type FriendUserInfo struct {
	FriendshipID int
	UserID       int
	Username     string
	DisplayName  string
}