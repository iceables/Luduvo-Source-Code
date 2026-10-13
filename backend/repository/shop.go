package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"buildoverse/backend/models"
)

type ShopRepository struct {
	db *sql.DB
}

func NewShopRepository(db *sql.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

func (r *ShopRepository) scanShopItem(scanner interface{ Scan(dest ...any) error }) (*models.ShopItem, error) {
	var item models.ShopItem
	err := scanner.Scan(
		&item.ID,
		&item.CID,
		&item.CreatorType,
		&item.CreatorName,
		&item.Name,
		&item.Description,
		&item.Type,
		&item.Bucks,
		&item.Bits,
		&item.Special,
		&item.Stock,
		&item.Deleted,
		&item.UGC,
		&item.ApprovalState,
		&item.OnSale,
		&item.OriginalTitle,
		&item.OriginalDescription,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.OriginalStock,
		&item.RAP,
		&item.OffSale,
	)
	if err != nil {
		return nil, err
	}

	normType, typeName := models.NormalizeShopCategory(item.Type)
	if normType != "all" {
		item.Type = normType
		item.TypeName = typeName
	} else {
		item.TypeName = "Item"
	}

	if item.CreatorName == "" {
		item.CreatorName = "Buildoverse"
	}

	return &item, nil
}

func (r *ShopRepository) buildWhereClause(filter *models.ShopFilter) (string, []any) {
	whereClauses := []string{"s.deleted = 0"}
	var args []any

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		whereClauses = append(whereClauses, "(s.name LIKE ? OR s.description LIKE ?)")
		args = append(args, pattern, pattern)
	}

	cat, _ := models.NormalizeShopCategory(filter.Category)
	switch cat {
	case "hat":
		whereClauses = append(whereClauses, "LOWER(s.type) IN ('hat', 'hats')")
	case "faces":
		whereClauses = append(whereClauses, "LOWER(s.type) IN ('face', 'faces')")
	case "shirts":
		whereClauses = append(whereClauses, "LOWER(s.type) IN ('shirt', 'shirts')")
	case "tshirts":
		whereClauses = append(whereClauses, "LOWER(s.type) IN ('tshirt', 'tshirts')")
	case "pants":
		whereClauses = append(whereClauses, "LOWER(s.type) IN ('pants', 'pant')")
	case "gear":
		whereClauses = append(whereClauses, "LOWER(s.type) IN ('gear', 'tool', 'tools')")
	}

	rawCurr := strings.ToLower(filter.Currency)
	if rawCurr != "" && rawCurr != "all" {
		tokens := strings.Split(rawCurr, ",")
		var currConditions []string
		for _, tok := range tokens {
			tok = strings.TrimSpace(tok)
			switch tok {
			case "free":
				currConditions = append(currConditions, "(s.bucks = 0 AND s.bits = 0)")
			case "bucks", "vertices":
				currConditions = append(currConditions, "s.bucks > 0")
			case "bits", "tickets":
				currConditions = append(currConditions, "s.bits > 0")
			}
		}
		if len(currConditions) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(currConditions, " OR ")+")")
		}
	}

	if filter.MinPrice > 0 {
		whereClauses = append(whereClauses, "(s.bucks >= ? OR s.bits >= ?)")
		args = append(args, filter.MinPrice, filter.MinPrice)
	}

	if filter.MaxPrice > 0 {
		whereClauses = append(whereClauses, "((s.bucks <= ? AND s.bucks > 0) OR (s.bits <= ? AND s.bits > 0) OR (s.bucks = 0 AND s.bits = 0))")
		args = append(args, filter.MaxPrice, filter.MaxPrice)
	}

	if filter.Creator != "" {
		whereClauses = append(whereClauses, "(u.username LIKE ? OR s.cid = ?)")
		creatorID, _ := strconv.Atoi(filter.Creator)
		args = append(args, "%"+filter.Creator+"%", creatorID)
	}

	return "WHERE " + strings.Join(whereClauses, " AND "), args
}

func (r *ShopRepository) buildOrderBy(sort string) string {
	switch strings.ToLower(sort) {
	case "oldest":
		return "ORDER BY s.created_at ASC, s.id ASC"
	case "price_asc":
		return "ORDER BY (s.bucks + s.bits) ASC, s.id ASC"
	case "price_desc":
		return "ORDER BY (s.bucks + s.bits) DESC, s.id DESC"
	case "popular", "bestselling":
		return "ORDER BY s.rap DESC, s.id DESC"
	default:
		return "ORDER BY s.created_at DESC, s.id DESC"
	}
}

func (r *ShopRepository) GetItems(filter *models.ShopFilter) ([]*models.ShopItem, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("database connection is offline")
	}

	whereSQL, args := r.buildWhereClause(filter)
	orderBySQL := r.buildOrderBy(filter.Sort)

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM shop_items s LEFT JOIN users u ON s.cid = u.id AND s.creator_type = 'user' %s`, whereSQL)

	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("ShopRepository query error: %v", err)
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if offset < 0 {
		offset = 0
	}

	selectQuery := fmt.Sprintf(`
		SELECT s.id, s.cid, s.creator_type, COALESCE(u.username, 'Buildoverse') AS creator_name,
		       s.name, COALESCE(s.description, '') AS description, s.type, s.bucks, s.bits, s.special,
		       s.stock, s.deleted, s.ugc, s.approval_state, s.on_sale, s.original_title, s.original_description,
		       s.created_at, s.updated_at, s.original_stock, s.rap, s.off_sale
		FROM shop_items s
		LEFT JOIN users u ON s.cid = u.id AND s.creator_type = 'user'
		%s %s LIMIT ? OFFSET ?`, whereSQL, orderBySQL)

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, filter.Limit, offset)

	rows, err := r.db.Query(selectQuery, queryArgs...)
	if err != nil {
		log.Printf("ShopRepository select query error: %v", err)
		return []*models.ShopItem{}, 0, err
	}
	defer rows.Close()

	items := make([]*models.ShopItem, 0)
	for rows.Next() {
		item, err := r.scanShopItem(rows)
		if err != nil {
			log.Printf("ShopRepository scan error: %v", err)
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ShopRepository) GetItemByID(id int) (*models.ShopItem, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}

	query := `
		SELECT s.id, s.cid, s.creator_type, COALESCE(u.username, 'Buildoverse') AS creator_name,
		       s.name, COALESCE(s.description, '') AS description, s.type, s.bucks, s.bits, s.special,
		       s.stock, s.deleted, s.ugc, s.approval_state, s.on_sale, s.original_title, s.original_description,
		       s.created_at, s.updated_at, s.original_stock, s.rap, s.off_sale
		FROM shop_items s
		LEFT JOIN users u ON s.cid = u.id AND s.creator_type = 'user'
		WHERE s.id = ? AND s.deleted = 0`

	row := r.db.QueryRow(query, id)
	return r.scanShopItem(row)
}
