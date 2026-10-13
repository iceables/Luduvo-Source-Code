package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"buildoverse/backend/models"
)

type AvatarRepository struct {
	db *sql.DB
}

var (
	ErrInventoryItemNotOwned = errors.New("item is not in the user's inventory")
	ErrInvalidEquippedItem   = errors.New("item type cannot be equipped on an avatar")
)

func NewAvatarRepository(db *sql.DB) *AvatarRepository {
	return &AvatarRepository{db: db}
}

func (r *AvatarRepository) GetAvatar(userID int) (*models.Avatar, error) {
	if r.db == nil {
		return models.DefaultAvatar(userID), nil
	}

	query := `SELECT head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color,
	                 hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face
              FROM avatar WHERE id = ?`

	var a models.Avatar
	a.ID = userID
	err := r.db.QueryRow(query, userID).Scan(
		&a.HeadColor, &a.LArmColor, &a.RArmColor, &a.TorsoColor, &a.LLegColor, &a.RLegColor,
		&a.Hat1, &a.Hat2, &a.Hat3, &a.Hat4, &a.Hat5, &a.Tool, &a.Shirt, &a.TShirt, &a.Pants, &a.Face,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.DefaultAvatar(userID), nil
		}
		return nil, err
	}

	formatColor := func(hex, fallback string) string {
		hex = strings.TrimPrefix(hex, "#")
		if hex == "" {
			return fallback
		}
		return hex
	}

	a.HeadColor = formatColor(a.HeadColor, models.DefaultHeadColor)
	a.TorsoColor = formatColor(a.TorsoColor, models.DefaultTorsoColor)
	a.LArmColor = formatColor(a.LArmColor, models.DefaultArmColor)
	a.RArmColor = formatColor(a.RArmColor, models.DefaultArmColor)
	a.LLegColor = formatColor(a.LLegColor, models.DefaultLegColor)
	a.RLegColor = formatColor(a.RLegColor, models.DefaultLegColor)

	return &a, nil
}

func (r *AvatarRepository) UpdateBodyColor(userID int, bodyPart string, color string) error {
	if r.db == nil {
		return nil
	}
	color = strings.TrimPrefix(color, "#")
	if len(color) != 6 {
		return errors.New("invalid color format")
	}

	bodyPart = strings.ToLower(strings.TrimSpace(bodyPart))

	switch bodyPart {
	case "head":
		_, err := r.db.Exec("UPDATE avatar SET head_color = ? WHERE id = ?", color, userID)
		return err
	case "torso":
		_, err := r.db.Exec("UPDATE avatar SET torso_color = ? WHERE id = ?", color, userID)
		return err
	case "larm", "leftarm", "left_arm":
		_, err := r.db.Exec("UPDATE avatar SET larm_color = ? WHERE id = ?", color, userID)
		return err
	case "rarm", "rightarm", "right_arm":
		_, err := r.db.Exec("UPDATE avatar SET rarm_color = ? WHERE id = ?", color, userID)
		return err
	case "lleg", "leftleg", "left_leg":
		_, err := r.db.Exec("UPDATE avatar SET lleg_color = ? WHERE id = ?", color, userID)
		return err
	case "rleg", "rightleg", "right_leg":
		_, err := r.db.Exec("UPDATE avatar SET rleg_color = ? WHERE id = ?", color, userID)
		return err
	case "all":
		_, err := r.db.Exec("UPDATE avatar SET head_color = ?, torso_color = ?, larm_color = ?, rarm_color = ?, lleg_color = ?, rleg_color = ? WHERE id = ?", color, color, color, color, color, color, userID)
		return err
	default:
		return errors.New("invalid body part")
	}
}

func (r *AvatarRepository) EquipItem(userID int, itemType string, itemID int) error {
	if r.db == nil || itemID <= 0 {
		return nil
	}
	av, err := r.GetAvatar(userID)
	if err != nil {
		return err
	}

	itemType = strings.ToLower(strings.TrimSpace(itemType))

	switch itemType {
	case "hat", "hats":
		if av.Hat1 == itemID || av.Hat2 == itemID || av.Hat3 == itemID || av.Hat4 == itemID || av.Hat5 == itemID {
			return nil
		}
		if av.Hat1 == 0 {
			_, err = r.db.Exec("UPDATE avatar SET hat1 = ? WHERE id = ?", itemID, userID)
		} else if av.Hat2 == 0 {
			_, err = r.db.Exec("UPDATE avatar SET hat2 = ? WHERE id = ?", itemID, userID)
		} else if av.Hat3 == 0 {
			_, err = r.db.Exec("UPDATE avatar SET hat3 = ? WHERE id = ?", itemID, userID)
		} else if av.Hat4 == 0 {
			_, err = r.db.Exec("UPDATE avatar SET hat4 = ? WHERE id = ?", itemID, userID)
		} else if av.Hat5 == 0 {
			_, err = r.db.Exec("UPDATE avatar SET hat5 = ? WHERE id = ?", itemID, userID)
		} else {
			_, err = r.db.Exec("UPDATE avatar SET hat1 = ? WHERE id = ?", itemID, userID)
		}
	case "shirt", "shirts":
		_, err = r.db.Exec("UPDATE avatar SET shirt = ? WHERE id = ?", itemID, userID)
	case "pants":
		_, err = r.db.Exec("UPDATE avatar SET pants = ? WHERE id = ?", itemID, userID)
	case "tshirt", "tshirts":
		_, err = r.db.Exec("UPDATE avatar SET tshirt = ? WHERE id = ?", itemID, userID)
	case "face", "faces":
		_, err = r.db.Exec("UPDATE avatar SET face = ? WHERE id = ?", itemID, userID)
	case "gear", "tool", "tools":
		_, err = r.db.Exec("UPDATE avatar SET tool = ? WHERE id = ?", itemID, userID)
	default:
		return errors.New("invalid item type")
	}

	return err
}

func (r *AvatarRepository) UnequipItem(userID int, itemType string, itemID int) error {
	if r.db == nil || itemID <= 0 {
		return nil
	}
	itemType = strings.ToLower(strings.TrimSpace(itemType))

	switch itemType {
	case "hat", "hats":
		query := "UPDATE avatar SET hat1 = IF(hat1 = ?, 0, hat1), hat2 = IF(hat2 = ?, 0, hat2), hat3 = IF(hat3 = ?, 0, hat3), hat4 = IF(hat4 = ?, 0, hat4), hat5 = IF(hat5 = ?, 0, hat5) WHERE id = ?"
		_, err := r.db.Exec(query, itemID, itemID, itemID, itemID, itemID, userID)
		return err
	case "shirt", "shirts":
		_, err := r.db.Exec("UPDATE avatar SET shirt = 0 WHERE id = ? AND shirt = ?", userID, itemID)
		return err
	case "pants":
		_, err := r.db.Exec("UPDATE avatar SET pants = 0 WHERE id = ? AND pants = ?", userID, itemID)
		return err
	case "tshirt", "tshirts":
		_, err := r.db.Exec("UPDATE avatar SET tshirt = 0 WHERE id = ? AND tshirt = ?", userID, itemID)
		return err
	case "face", "faces":
		_, err := r.db.Exec("UPDATE avatar SET face = 0 WHERE id = ? AND face = ?", userID, itemID)
		return err
	case "gear", "tool", "tools":
		_, err := r.db.Exec("UPDATE avatar SET tool = 0 WHERE id = ? AND tool = ?", userID, itemID)
		return err
	default:
		return errors.New("invalid item type")
	}
}

func (r *AvatarRepository) SetEquippedItems(userID int, itemIDs []int) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var hats []int
	clothing := map[string]int{
		"shirts":  0,
		"pants":   0,
		"tshirts": 0,
		"faces":   0,
		"gear":    0,
	}
	seen := make(map[int]struct{}, len(itemIDs))
	for _, id := range itemIDs {
		if id < 1 {
			return ErrInvalidEquippedItem
		}
		if _, duplicate := seen[id]; duplicate {
			return ErrInvalidEquippedItem
		}
		seen[id] = struct{}{}

		var itemType string
		err := tx.QueryRow(`SELECT s.type FROM user_inventory i
			JOIN shop_items s ON s.id = i.item_id
			WHERE i.user_id = ? AND i.item_id = ?`, userID, id).Scan(&itemType)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInventoryItemNotOwned
		}
		if err != nil {
			return err
		}

		item := &models.InventoryItem{Type: itemType}
		normalizeItemType(item)
		switch item.Type {
		case "hat":
			if len(hats) == 5 {
				return ErrInvalidEquippedItem
			}
			hats = append(hats, id)
		case "shirts", "pants", "tshirts", "faces", "gear":
			if clothing[item.Type] != 0 {
				return ErrInvalidEquippedItem
			}
			clothing[item.Type] = id
		default:
			return ErrInvalidEquippedItem
		}
	}

	_, err = tx.Exec(`INSERT INTO avatar
		(id, head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color,
		 hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, 0, 0, 0, 0, 0, ?)
		ON DUPLICATE KEY UPDATE id = id`,
		userID, models.DefaultHeadColor, models.DefaultArmColor, models.DefaultArmColor,
		models.DefaultTorsoColor, models.DefaultLegColor, models.DefaultLegColor, models.DefaultFaceItemID)
	if err != nil {
		return err
	}

	for len(hats) < 5 {
		hats = append(hats, 0)
	}
	equippedFace := clothing["faces"]
	if equippedFace == 0 {
		equippedFace = models.DefaultFaceItemID
	}
	_, err = tx.Exec(`UPDATE avatar SET hat1 = ?, hat2 = ?, hat3 = ?, hat4 = ?, hat5 = ?,
		shirt = ?, pants = ?, tshirt = ?, face = ?, tool = ? WHERE id = ?`,
		hats[0], hats[1], hats[2], hats[3], hats[4],
		clothing["shirts"], clothing["pants"], clothing["tshirts"], equippedFace, clothing["gear"], userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeItemType(item *models.InventoryItem) {
	switch strings.ToLower(item.Type) {
	case "hat", "hats":
		item.TypeName = "Hat"
		item.Type = "hat"
	case "shirt", "shirts":
		item.TypeName = "Shirt"
		item.Type = "shirts"
	case "pants":
		item.TypeName = "Pants"
		item.Type = "pants"
	case "tshirt", "tshirts":
		item.TypeName = "T-Shirt"
		item.Type = "tshirts"
	case "face", "faces":
		item.TypeName = "Face"
		item.Type = "faces"
	case "gear", "tool", "tools":
		item.TypeName = "Gear"
		item.Type = "gear"
	}
}

func (r *AvatarRepository) lookupShopItems(ids []int) map[int]*models.InventoryItem {
	found := make(map[int]*models.InventoryItem)
	if r.db == nil || len(ids) == 0 {
		return found
	}

	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := `SELECT s.id, s.name, s.type, COALESCE(s.description, ''), COALESCE(u.username, s.creator_name, 'Polyoria')
              FROM shop_items s
              LEFT JOIN users u ON s.cid = u.id AND s.creator_type = 'user'
              WHERE s.id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return found
	}
	defer rows.Close()

	for rows.Next() {
		var item models.InventoryItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Description, &item.CreatorName); err == nil {
			normalizeItemType(&item)
			found[item.ID] = &item
		}
	}
	return found
}

func (r *AvatarRepository) GetInventory(userID int, category, search string) ([]*models.InventoryItem, error) {
	avatar, err := r.GetAvatar(userID)
	if err != nil {
		return nil, err
	}

	equippedMap := make(map[string]bool)
	if avatar.Hat1 > 0 {
		equippedMap[fmt.Sprintf("hat:%d", avatar.Hat1)] = true
	}
	if avatar.Hat2 > 0 {
		equippedMap[fmt.Sprintf("hat:%d", avatar.Hat2)] = true
	}
	if avatar.Hat3 > 0 {
		equippedMap[fmt.Sprintf("hat:%d", avatar.Hat3)] = true
	}
	if avatar.Hat4 > 0 {
		equippedMap[fmt.Sprintf("hat:%d", avatar.Hat4)] = true
	}
	if avatar.Hat5 > 0 {
		equippedMap[fmt.Sprintf("hat:%d", avatar.Hat5)] = true
	}
	if avatar.Shirt > 0 {
		equippedMap[fmt.Sprintf("shirts:%d", avatar.Shirt)] = true
	}
	if avatar.Pants > 0 {
		equippedMap[fmt.Sprintf("pants:%d", avatar.Pants)] = true
	}
	if avatar.TShirt > 0 {
		equippedMap[fmt.Sprintf("tshirts:%d", avatar.TShirt)] = true
	}
	if avatar.Face > 0 {
		equippedMap[fmt.Sprintf("faces:%d", avatar.Face)] = true
	}
	if avatar.Tool > 0 {
		equippedMap[fmt.Sprintf("gear:%d", avatar.Tool)] = true
	}

	var items []*models.InventoryItem

	if r.db != nil {
		query := `SELECT s.id, s.name, s.type, COALESCE(s.description, ''), COALESCE(u.username, s.creator_name, 'Polyoria')
	              FROM user_inventory i
	              JOIN shop_items s ON i.item_id = s.id
	              LEFT JOIN users u ON s.cid = u.id AND s.creator_type = 'user'
	              WHERE i.user_id = ?`
		rows, err := r.db.Query(query, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.InventoryItem
			if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Description, &item.CreatorName); err != nil {
				return nil, err
			}
			items = append(items, &item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("database connection is offline")
	}

	category = strings.ToLower(strings.TrimSpace(category))
	search = strings.ToLower(strings.TrimSpace(search))

	var filtered []*models.InventoryItem
	for _, item := range items {
		normalizeItemType(item)

		item.IsEquipped = equippedMap[fmt.Sprintf("%s:%d", item.Type, item.ID)]

		if category != "" && category != "all" {
			matchCat := false
			switch category {
			case "hat", "hats":
				matchCat = (item.Type == "hat")
			case "shirt", "shirts":
				matchCat = (item.Type == "shirts")
			case "pants":
				matchCat = (item.Type == "pants")
			case "tshirt", "tshirts":
				matchCat = (item.Type == "tshirts")
			case "face", "faces":
				matchCat = (item.Type == "faces")
			case "gear", "tool", "tools":
				matchCat = (item.Type == "gear")
			}
			if !matchCat {
				continue
			}
		}

		if search != "" {
			if !strings.Contains(strings.ToLower(item.Name), search) &&
				!strings.Contains(strings.ToLower(item.CreatorName), search) {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	return filtered, nil
}

func (r *AvatarRepository) GetEquippedItems(userID int) ([]*models.InventoryItem, error) {
	avatar, err := r.GetAvatar(userID)
	if err != nil {
		return nil, err
	}

	var ids []int
	for _, id := range []int{avatar.Hat1, avatar.Hat2, avatar.Hat3, avatar.Hat4, avatar.Hat5, avatar.Shirt, avatar.Pants, avatar.TShirt, avatar.Face, avatar.Tool} {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	shopMap := r.lookupShopItems(ids)

	var equipped []*models.InventoryItem

	addEquipped := func(itemType string, id int) {
		if id <= 0 {
			return
		}
		if item, ok := shopMap[id]; ok {
			cp := *item
			cp.IsEquipped = true
			equipped = append(equipped, &cp)
		} else {
			typeName := "Item"
			switch itemType {
			case "hat":
				typeName = "Hat"
			case "shirts":
				typeName = "Shirt"
			case "pants":
				typeName = "Pants"
			case "tshirts":
				typeName = "T-Shirt"
			case "faces":
				typeName = "Face"
			case "gear":
				typeName = "Gear"
			}
			equipped = append(equipped, &models.InventoryItem{
				ID:          id,
				Name:        fmt.Sprintf("%s #%d", typeName, id),
				Type:        itemType,
				TypeName:    typeName,
				CreatorName: "Polyoria",
				IsEquipped:  true,
			})
		}
	}

	addEquipped("hat", avatar.Hat1)
	addEquipped("hat", avatar.Hat2)
	addEquipped("hat", avatar.Hat3)
	addEquipped("hat", avatar.Hat4)
	addEquipped("hat", avatar.Hat5)
	addEquipped("shirts", avatar.Shirt)
	addEquipped("pants", avatar.Pants)
	addEquipped("tshirts", avatar.TShirt)
	addEquipped("faces", avatar.Face)
	addEquipped("gear", avatar.Tool)

	return equipped, nil
}

func (r *AvatarRepository) GetOutfits(userID int) ([]*models.Outfit, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}

	query := `SELECT id, user_id, name, head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color,
	                 hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face, created_at
              FROM outfits WHERE user_id = ? ORDER BY id DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outfits []*models.Outfit
	for rows.Next() {
		var o models.Outfit
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Name, &o.HeadColor, &o.LArmColor, &o.RArmColor, &o.TorsoColor, &o.LLegColor, &o.RLegColor,
			&o.Hat1, &o.Hat2, &o.Hat3, &o.Hat4, &o.Hat5, &o.Tool, &o.Shirt, &o.TShirt, &o.Pants, &o.Face, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		outfits = append(outfits, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return outfits, nil
}

func (r *AvatarRepository) GetOutfitByID(outfitID int) (*models.Outfit, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}

	query := `SELECT id, user_id, name, head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color,
	                 hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face, created_at
              FROM outfits WHERE id = ?`

	var o models.Outfit
	err := r.db.QueryRow(query, outfitID).Scan(
		&o.ID, &o.UserID, &o.Name, &o.HeadColor, &o.LArmColor, &o.RArmColor, &o.TorsoColor, &o.LLegColor, &o.RLegColor,
		&o.Hat1, &o.Hat2, &o.Hat3, &o.Hat4, &o.Hat5, &o.Tool, &o.Shirt, &o.TShirt, &o.Pants, &o.Face, &o.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *AvatarRepository) CreateOutfit(userID int, name string) (*models.Outfit, error) {
	if r.db == nil {
		return nil, errors.New("database connection is offline")
	}

	av, err := r.GetAvatar(userID)
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO outfits (user_id, name, head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color, hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.Exec(query,
		userID, name, av.HeadColor, av.LArmColor, av.RArmColor, av.TorsoColor, av.LLegColor, av.RLegColor,
		av.Hat1, av.Hat2, av.Hat3, av.Hat4, av.Hat5, av.Tool, av.Shirt, av.TShirt, av.Pants, av.Face,
	)
	if err != nil {
		return nil, err
	}

	outfitID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &models.Outfit{
		ID:         int(outfitID),
		UserID:     userID,
		Name:       name,
		HeadColor:  av.HeadColor,
		LArmColor:  av.LArmColor,
		RArmColor:  av.RArmColor,
		TorsoColor: av.TorsoColor,
		LLegColor:  av.LLegColor,
		RLegColor:  av.RLegColor,
		Hat1:       av.Hat1,
		Hat2:       av.Hat2,
		Hat3:       av.Hat3,
		Hat4:       av.Hat4,
		Hat5:       av.Hat5,
		Tool:       av.Tool,
		Shirt:      av.Shirt,
		TShirt:     av.TShirt,
		Pants:      av.Pants,
		Face:       av.Face,
		CreatedAt:  time.Now(),
	}, nil
}

func (r *AvatarRepository) WearOutfit(userID, outfitID int) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}

	outfit, err := r.GetOutfitByID(outfitID)
	if err != nil {
		return err
	}
	if outfit == nil {
		return errors.New("outfit not found")
	}

	if outfit.UserID != userID {
		return errors.New("unauthorized")
	}

	query := `UPDATE avatar SET 
	          head_color = ?, larm_color = ?, rarm_color = ?, torso_color = ?, lleg_color = ?, rleg_color = ?,
	          hat1 = ?, hat2 = ?, hat3 = ?, hat4 = ?, hat5 = ?, tool = ?, shirt = ?, tshirt = ?, pants = ?, face = ?
	          WHERE id = ?`
	_, err = r.db.Exec(query,
		outfit.HeadColor, outfit.LArmColor, outfit.RArmColor, outfit.TorsoColor, outfit.LLegColor, outfit.RLegColor,
		outfit.Hat1, outfit.Hat2, outfit.Hat3, outfit.Hat4, outfit.Hat5, outfit.Tool, outfit.Shirt, outfit.TShirt, outfit.Pants, outfit.Face,
		userID,
	)
	return err
}

func (r *AvatarRepository) DeleteOutfit(userID, outfitID int) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	result, err := r.db.Exec("DELETE FROM outfits WHERE id = ? AND user_id = ?", outfitID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("outfit not found")
	}
	return nil
}

func (r *AvatarRepository) RenameOutfit(userID, outfitID int, name string) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	result, err := r.db.Exec("UPDATE outfits SET name = ? WHERE id = ? AND user_id = ?", name, outfitID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows > 0 {
		return nil
	}

	outfit, err := r.GetOutfitByID(outfitID)
	if err != nil {
		return err
	}
	if outfit == nil || outfit.UserID != userID {
		return errors.New("outfit not found")
	}
	return nil
}

func (r *AvatarRepository) ResetAvatar(userID int) error {
	if r.db == nil {
		return errors.New("database connection is offline")
	}
	query := `UPDATE avatar SET
	          head_color = ?, larm_color = ?, rarm_color = ?, torso_color = ?, lleg_color = ?, rleg_color = ?,
	          hat1 = 0, hat2 = 0, hat3 = 0, hat4 = 0, hat5 = 0, tool = 0, shirt = 0, tshirt = 0, pants = 0, face = ?
	          WHERE id = ?`
	_, err := r.db.Exec(query, models.DefaultHeadColor, models.DefaultArmColor, models.DefaultArmColor,
		models.DefaultTorsoColor, models.DefaultLegColor, models.DefaultLegColor, models.DefaultFaceItemID, userID)
	return err
}

func (r *AvatarRepository) MigrateDefaultAvatarPalette() ([]int, error) {
	if r.db == nil {
		return nil, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const normalized = `LOWER(REPLACE(COALESCE(%s, ''), '#', ''))`
	legacyCondition := fmt.Sprintf(
		normalized+` = 'f3b700' AND `+normalized+` = 'f3b700' AND `+normalized+` = 'f3b700' AND `+
			normalized+` IN ('c60000', '3292d3', '85ad00', 'e58700') AND `+
			normalized+` IN ('650013', '1c4399', '1d6a19', '76603f') AND `+
			normalized+` IN ('650013', '1c4399', '1d6a19', '76603f')`,
		"head_color", "larm_color", "rarm_color", "torso_color", "lleg_color", "rleg_color",
	)
	rows, err := tx.Query(`SELECT u.id FROM users u
		LEFT JOIN avatar a ON a.id = u.id
		WHERE a.id IS NULL OR a.face IN (0, 1) OR (` + legacyCondition + `)
		ORDER BY u.id`)
	if err != nil {
		return nil, err
	}
	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if len(userIDs) == 0 {
		return nil, tx.Commit()
	}

	if _, err := tx.Exec(`INSERT INTO avatar
		(id, head_color, larm_color, rarm_color, torso_color, lleg_color, rleg_color,
		 hat1, hat2, hat3, hat4, hat5, tool, shirt, tshirt, pants, face)
		SELECT u.id, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, 0, 0, 0, 0, 0, ?
		FROM users u LEFT JOIN avatar a ON a.id = u.id WHERE a.id IS NULL`,
		models.DefaultHeadColor, models.DefaultArmColor, models.DefaultArmColor,
		models.DefaultTorsoColor, models.DefaultLegColor, models.DefaultLegColor, models.DefaultFaceItemID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE avatar SET
		head_color = ?, larm_color = ?, rarm_color = ?, torso_color = ?, lleg_color = ?, rleg_color = ?
		WHERE `+legacyCondition,
		models.DefaultHeadColor, models.DefaultArmColor, models.DefaultArmColor,
		models.DefaultTorsoColor, models.DefaultLegColor, models.DefaultLegColor); err != nil {
		return nil, err
	}
	if _, err := tx.Exec("UPDATE avatar SET face = ? WHERE face IN (0, 1)", models.DefaultFaceItemID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return userIDs, nil
}
