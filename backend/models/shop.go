package models

import (
	"database/sql"
	"strings"
	"time"
)

type ShopItem struct {
	ID                  int            `json:"id"`
	CID                 int            `json:"cid"`
	CreatorType         string         `json:"creator_type"`
	CreatorName         string         `json:"creator_name"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	Type                string         `json:"type"`
	TypeName            string         `json:"type_name"`
	Bucks               int            `json:"bucks"`
	Bits                int            `json:"bits"`
	Special             string         `json:"special"`
	Stock               int            `json:"stock"`
	Deleted             int            `json:"deleted"`
	UGC                 string         `json:"ugc"`
	ApprovalState       string         `json:"approval_state"`
	OnSale              string         `json:"onsale"`
	OriginalTitle       sql.NullString `json:"original_title"`
	OriginalDescription sql.NullString `json:"original_description"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	OriginalStock       int            `json:"original_stock"`
	RAP                 int            `json:"rap"`
	OffSale             string         `json:"offsale"`
}

type ShopFilter struct {
	Search   string `json:"search"`
	Category string `json:"category"`
	Currency string `json:"currency"`
	MinPrice int    `json:"min_price"`
	MaxPrice int    `json:"max_price"`
	Creator  string `json:"creator"`
	Sort     string `json:"sort"`
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
}

func NormalizeShopCategory(cat string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "hat", "hats":
		return "hat", "Hat"
	case "face", "faces":
		return "faces", "Face"
	case "shirt", "shirts":
		return "shirts", "Shirt"
	case "tshirt", "tshirts":
		return "tshirts", "T-Shirt"
	case "pants":
		return "pants", "Pants"
	case "gear", "tool", "tools":
		return "gear", "Gear"
	default:
		return "all", "All Items"
	}
}