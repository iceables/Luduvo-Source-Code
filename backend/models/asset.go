package models

import (
	"strings"
	"time"
)

const (
	AssetTypeImage = "image"
	AssetTypeMesh  = "mesh"
	AssetTypeSound = "sound"

	AssetApprovalPending  = "pending"
	AssetApprovalApproved = "approved"
	AssetApprovalRejected = "rejected"
)

var AssetTypeNames = map[string]string{
	AssetTypeImage: "Image",
	AssetTypeMesh:  "Mesh",
	AssetTypeSound: "Sound",
}

func NormalizeAssetType(value string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AssetTypeImage:
		return AssetTypeImage, AssetTypeNames[AssetTypeImage]
	case AssetTypeMesh:
		return AssetTypeMesh, AssetTypeNames[AssetTypeMesh]
	case AssetTypeSound:
		return AssetTypeSound, AssetTypeNames[AssetTypeSound]
	default:
		return AssetTypeImage, AssetTypeNames[AssetTypeImage]
	}
}

func NormalizeAssetApproval(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AssetApprovalApproved, AssetApprovalRejected, AssetApprovalPending:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return AssetApprovalPending
	}
}

type Asset struct {
	ID            int        `json:"id"`
	UID           int        `json:"uid"`
	OwnerName     string     `json:"owner_name"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Type          string     `json:"type"`
	FilePath      string     `json:"file_path"`
	ApprovalState string     `json:"approval_state"`
	ReviewerID    *int       `json:"reviewer_id"`
	ReviewNote    *string    `json:"review_note"`
	CreatedAt     time.Time  `json:"created_at"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
}

func (a *Asset) TypeName() string {
	if name, ok := AssetTypeNames[a.Type]; ok {
		return name
	}
	return a.Type
}

func (a *Asset) ApprovalLabel() string {
	switch a.ApprovalState {
	case AssetApprovalApproved:
		return "Approved"
	case AssetApprovalRejected:
		return "Rejected"
	default:
		return "Pending"
	}
}

func (a *Asset) IsPending() bool {
	return a.ApprovalState == AssetApprovalPending
}