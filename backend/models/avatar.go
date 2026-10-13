package models

import "time"

const (
	DefaultBundleItemID = 17
	DefaultFaceItemID   = 19
	DefaultHeadColor    = "FFFFFF"
	DefaultArmColor     = "FFFFFF"
	DefaultTorsoColor   = "8B0000"
	DefaultLegColor     = "FF0000"
)

func DefaultAvatar(id int) *Avatar {
	return &Avatar{
		ID:         id,
		HeadColor:  DefaultHeadColor,
		TorsoColor: DefaultTorsoColor,
		LArmColor:  DefaultArmColor,
		RArmColor:  DefaultArmColor,
		LLegColor:  DefaultLegColor,
		RLegColor:  DefaultLegColor,
		Face:       DefaultFaceItemID,
	}
}

type Avatar struct {
	ID         int    `json:"id"`
	HeadColor  string `json:"head_color"`
	TorsoColor string `json:"torso_color"`
	LArmColor  string `json:"larm_color"`
	RArmColor  string `json:"rarm_color"`
	LLegColor  string `json:"lleg_color"`
	RLegColor  string `json:"rleg_color"`
	Hat1       int    `json:"hat1"`
	Hat2       int    `json:"hat2"`
	Hat3       int    `json:"hat3"`
	Hat4       int    `json:"hat4"`
	Hat5       int    `json:"hat5"`
	Tool       int    `json:"tool"`
	Shirt      int    `json:"shirt"`
	TShirt     int    `json:"tshirt"`
	Pants      int    `json:"pants"`
	Face       int    `json:"face"`
}

type InventoryItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	TypeName    string `json:"type_name"`
	Description string `json:"description"`
	CreatorName string `json:"creator_name"`
	IsEquipped  bool   `json:"is_equipped"`
}

type Outfit struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Name       string    `json:"name"`
	HeadColor  string    `json:"head_color"`
	TorsoColor string    `json:"torso_color"`
	LArmColor  string    `json:"larm_color"`
	RArmColor  string    `json:"rarm_color"`
	LLegColor  string    `json:"lleg_color"`
	RLegColor  string    `json:"rleg_color"`
	Hat1       int       `json:"hat1"`
	Hat2       int       `json:"hat2"`
	Hat3       int       `json:"hat3"`
	Hat4       int       `json:"hat4"`
	Hat5       int       `json:"hat5"`
	Tool       int       `json:"tool"`
	Shirt      int       `json:"shirt"`
	TShirt     int       `json:"tshirt"`
	Pants      int       `json:"pants"`
	Face       int       `json:"face"`
	CreatedAt  time.Time `json:"created_at"`
}
