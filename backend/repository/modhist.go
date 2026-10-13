package repository

import (
	"database/sql"
	"errors"
	"time"

	"buildoverse/backend/models"
)

type ModHistoryRepository struct {
	db *sql.DB
}

func NewModHistoryRepository(db *sql.DB) *ModHistoryRepository {
	return &ModHistoryRepository{db: db}
}

func (r *ModHistoryRepository) GetByUserID(userID int) ([]*models.ModHistory, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `SELECT m.id, m.uid, m.admin_id, COALESCE(u.username, 'System'), COALESCE(u.power, 0), m.action_type, m.reason, m.note, m.status, m.creation_date, m.expires_at, COALESCE(tu.username, '')
              FROM modhist m
              LEFT JOIN users u ON m.admin_id = u.id
              LEFT JOIN users tu ON m.uid = tu.id
              WHERE m.uid = ?
              ORDER BY m.creation_date DESC, m.id DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.ModHistory
	for rows.Next() {
		var mh models.ModHistory
		if err := rows.Scan(
			&mh.ID, &mh.UID, &mh.AdminID, &mh.AdminName, &mh.AdminPower,
			&mh.ActionType, &mh.Reason, &mh.Note, &mh.Status,
			&mh.CreationDate, &mh.ExpiresAt, &mh.TargetName,
		); err != nil {
			return nil, err
		}
		list = append(list, &mh)
	}

	return list, nil
}

func (r *ModHistoryRepository) GetByID(id int) (*models.ModHistory, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}

	query := `SELECT m.id, m.uid, m.admin_id, COALESCE(u.username, 'System'), COALESCE(u.power, 0), m.action_type, m.reason, m.note, m.status, m.creation_date, m.expires_at, COALESCE(tu.username, '')
              FROM modhist m
              LEFT JOIN users u ON m.admin_id = u.id
              LEFT JOIN users tu ON m.uid = tu.id
              WHERE m.id = ?`

	var mh models.ModHistory
	err := r.db.QueryRow(query, id).Scan(
		&mh.ID, &mh.UID, &mh.AdminID, &mh.AdminName, &mh.AdminPower,
		&mh.ActionType, &mh.Reason, &mh.Note, &mh.Status,
		&mh.CreationDate, &mh.ExpiresAt, &mh.TargetName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &mh, nil
}

func (r *ModHistoryRepository) GetAll(limit, offset int) ([]*models.ModHistory, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("database connection is offline")
	}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM modhist").Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT m.id, m.uid, m.admin_id, COALESCE(u.username, 'System'), COALESCE(u.power, 0), m.action_type, m.reason, m.note, m.status, m.creation_date, m.expires_at, COALESCE(tu.username, '')
              FROM modhist m
              LEFT JOIN users u ON m.admin_id = u.id
              LEFT JOIN users tu ON m.uid = tu.id
              ORDER BY m.creation_date DESC, m.id DESC
              LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*models.ModHistory
	for rows.Next() {
		var mh models.ModHistory
		if err := rows.Scan(
			&mh.ID, &mh.UID, &mh.AdminID, &mh.AdminName, &mh.AdminPower,
			&mh.ActionType, &mh.Reason, &mh.Note, &mh.Status,
			&mh.CreationDate, &mh.ExpiresAt, &mh.TargetName,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, &mh)
	}

	return list, total, nil
}

func (r *ModHistoryRepository) Create(uid, adminID int, actionType, reason, note, status string) (int, error) {
	if r.db == nil {
		return 0, errors.New("database connection is offline")
	}
	if status == "" {
		status = models.StatusActive
	}

	var noteArg any
	if note != "" {
		noteArg = note
	}

	res, err := r.db.Exec(
		`INSERT INTO modhist (uid, admin_id, action_type, reason, note, status, creation_date) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uid, adminID, actionType, reason, noteArg, status, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, nil
	}
	return int(id), nil
}

func (r *ModHistoryRepository) UpdateStatus(id int, status string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	_, err := r.db.Exec(`UPDATE modhist SET status = ? WHERE id = ?`, status, id)
	return err
}
