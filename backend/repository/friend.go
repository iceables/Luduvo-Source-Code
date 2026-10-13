package repository

import (
	"database/sql"
	"buildoverse/backend/models"
)

type FriendRepository struct {
	db *sql.DB
}

func NewFriendRepository(db *sql.DB) *FriendRepository {
	return &FriendRepository{db: db}
}

func (r *FriendRepository) GetFriendStatus(userID, targetID int) (string, error) {
	if r.db == nil {
		return "none", nil
	}
	var uid, fid int
	var state string
	query := "SELECT uid, fid, state FROM friends WHERE (uid = ? AND fid = ?) OR (uid = ? AND fid = ?)"
	err := r.db.QueryRow(query, userID, targetID, targetID, userID).Scan(&uid, &fid, &state)
	if err != nil {
		if err == sql.ErrNoRows {
			return "none", nil
		}
		return "none", err
	}
	if state == "accepted" {
		return "friends", nil
	}
	if state == "pending" {
		if uid == userID {
			return "pending_sent", nil
		}
		return "pending_received", nil
	}
	return "none", nil
}

func (r *FriendRepository) GetPendingRequestCount(userID int) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	query := "SELECT COUNT(*) FROM friends WHERE fid = ? AND state = 'pending'"
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (r *FriendRepository) GetIncomingRequests(userID int) ([]*models.FriendUserInfo, error) {
	if r.db == nil {
		return nil, nil
	}
	query := `SELECT f.id, f.uid, u.username, u.displayname 
              FROM friends f 
              JOIN users u ON f.uid = u.id 
              WHERE f.fid = ? AND f.state = 'pending' 
              ORDER BY f.id DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.FriendUserInfo
	for rows.Next() {
		var req models.FriendUserInfo
		if err := rows.Scan(&req.FriendshipID, &req.UserID, &req.Username, &req.DisplayName); err != nil {
			return nil, err
		}
		if req.DisplayName == "" {
			req.DisplayName = req.Username
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func (r *FriendRepository) GetOutgoingRequests(userID int) ([]*models.FriendUserInfo, error) {
	if r.db == nil {
		return nil, nil
	}
	query := `SELECT f.id, f.fid, u.username, u.displayname 
              FROM friends f 
              JOIN users u ON f.fid = u.id 
              WHERE f.uid = ? AND f.state = 'pending' 
              ORDER BY f.id DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.FriendUserInfo
	for rows.Next() {
		var req models.FriendUserInfo
		if err := rows.Scan(&req.FriendshipID, &req.UserID, &req.Username, &req.DisplayName); err != nil {
			return nil, err
		}
		if req.DisplayName == "" {
			req.DisplayName = req.Username
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func (r *FriendRepository) GetFriendsList(userID int) ([]*models.FriendUserInfo, error) {
	if r.db == nil {
		return nil, nil
	}
	query := `SELECT f.id, 
                     CASE WHEN f.uid = ? THEN f.fid ELSE f.uid END AS friend_id,
                     u.username, u.displayname
              FROM friends f
              JOIN users u ON u.id = (CASE WHEN f.uid = ? THEN f.fid ELSE f.uid END)
              WHERE (f.uid = ? OR f.fid = ?) AND f.state = 'accepted'
              ORDER BY u.username ASC`
	rows, err := r.db.Query(query, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []*models.FriendUserInfo
	for rows.Next() {
		var f models.FriendUserInfo
		if err := rows.Scan(&f.FriendshipID, &f.UserID, &f.Username, &f.DisplayName); err != nil {
			return nil, err
		}
		if f.DisplayName == "" {
			f.DisplayName = f.Username
		}
		friends = append(friends, &f)
	}
	return friends, nil
}

func (r *FriendRepository) SendRequest(senderID, targetID int) error {
	if r.db == nil {
		return nil
	}
	status, err := r.GetFriendStatus(senderID, targetID)
	if err != nil {
		return err
	}
	if status == "friends" || status == "pending_sent" {
		return nil
	}
	if status == "pending_received" {
		_, err := r.db.Exec("UPDATE friends SET state = 'accepted' WHERE uid = ? AND fid = ? AND state = 'pending'", targetID, senderID)
		return err
	}
	query := "INSERT INTO friends (id, uid, fid, state) VALUES ((SELECT COALESCE(MAX(x.id), 0) + 1 FROM (SELECT id FROM friends) x), ?, ?, 'pending')"
	_, err = r.db.Exec(query, senderID, targetID)
	return err
}

func (r *FriendRepository) AcceptRequest(userID, targetID int) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("UPDATE friends SET state = 'accepted' WHERE uid = ? AND fid = ? AND state = 'pending'", targetID, userID)
	return err
}

func (r *FriendRepository) DeclineRequest(userID, targetID int) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("DELETE FROM friends WHERE uid = ? AND fid = ? AND state = 'pending'", targetID, userID)
	return err
}

func (r *FriendRepository) CancelRequest(userID, targetID int) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("DELETE FROM friends WHERE uid = ? AND fid = ? AND state = 'pending'", userID, targetID)
	return err
}

func (r *FriendRepository) RemoveFriend(userID, targetID int) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.Exec("DELETE FROM friends WHERE ((uid = ? AND fid = ?) OR (uid = ? AND fid = ?)) AND state = 'accepted'", userID, targetID, targetID, userID)
	return err
}