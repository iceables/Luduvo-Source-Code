package repository

import (
	"database/sql"
	"errors"
	"time"

	"buildoverse/backend/models"
)

type AssetRepository struct {
	db *sql.DB
}

func NewAssetRepository(db *sql.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

func (r *AssetRepository) scanAsset(scanner interface{ Scan(dest ...any) error }) (*models.Asset, error) {
	var asset models.Asset
	var reviewerID sql.NullInt64
	var reviewNote sql.NullString
	var reviewedAt sql.NullTime
	err := scanner.Scan(
		&asset.ID,
		&asset.UID,
		&asset.OwnerName,
		&asset.Name,
		&asset.Description,
		&asset.Type,
		&asset.FilePath,
		&asset.ApprovalState,
		&reviewerID,
		&reviewNote,
		&asset.CreatedAt,
		&reviewedAt,
	)
	if err != nil {
		return nil, err
	}
	if reviewerID.Valid {
		value := int(reviewerID.Int64)
		asset.ReviewerID = &value
	}
	if reviewNote.Valid {
		value := reviewNote.String
		asset.ReviewNote = &value
	}
	if reviewedAt.Valid {
		value := reviewedAt.Time
		asset.ReviewedAt = &value
	}
	return &asset, nil
}

const assetSelectColumns = `a.id, a.uid, COALESCE(u.username, 'Buildoverse'), a.name, COALESCE(a.description, ''), a.type, a.file_path, a.approval_state, a.reviewer_id, a.review_note, a.created_at, a.reviewed_at`

func (r *AssetRepository) Create(uid int, name, description, assetType, filePath string) (int, error) {
	if r.db == nil {
		return 0, errors.New("database connection is offline")
	}

	res, err := r.db.Exec(
		`INSERT INTO assets (uid, name, description, type, file_path, approval_state, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uid, name, description, assetType, filePath, models.AssetApprovalPending, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *AssetRepository) GetByID(id int) (*models.Asset, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}

	query := `SELECT ` + assetSelectColumns + ` FROM assets a LEFT JOIN users u ON a.uid = u.id WHERE a.id = ?`
	asset, err := r.scanAsset(r.db.QueryRow(query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return asset, nil
}

func (r *AssetRepository) GetByUserID(uid int) ([]*models.Asset, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `SELECT ` + assetSelectColumns + ` FROM assets a LEFT JOIN users u ON a.uid = u.id WHERE a.uid = ? ORDER BY a.created_at DESC, a.id DESC`
	rows, err := r.db.Query(query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Asset
	for rows.Next() {
		asset, err := r.scanAsset(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, asset)
	}
	return list, nil
}

func (r *AssetRepository) GetQueue(assetType string, limit, offset int) ([]*models.Asset, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("database connection is offline")
	}

	where := `WHERE a.approval_state = 'pending'`
	var args []any
	if assetType != "" {
		where += ` AND a.type = ?`
		args = append(args, assetType)
	}

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM assets a `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + assetSelectColumns + ` FROM assets a LEFT JOIN users u ON a.uid = u.id ` + where + ` ORDER BY a.created_at ASC, a.id ASC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), limit, offset)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*models.Asset
	for rows.Next() {
		asset, err := r.scanAsset(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, asset)
	}
	return list, total, nil
}

func (r *AssetRepository) UpdateApproval(id int, state string, reviewerID int, note string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}

	var noteArg any
	if note != "" {
		noteArg = note
	}

	_, err := r.db.Exec(
		`UPDATE assets SET approval_state = ?, reviewer_id = ?, review_note = ?, reviewed_at = ? WHERE id = ?`,
		state, reviewerID, noteArg, time.Now(), id,
	)
	return err
}