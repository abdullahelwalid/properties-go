package models

import "gorm.io/gorm"

type Transaction struct {
	gorm.Model
	ID         uint     `gorm:"primarykey"`
	ClientID   uint     `json:"clientId"`
	OwnerID    uint     `json:"ownerId"`
	PropertyID uint     `json:"propertyId"`
	Type       string   `json:"type"` // "rent" or "buy"
	Client     User     `json:"client" gorm:"foreignKey:ClientID;references:ID"`
	Owner      User     `json:"owner" gorm:"foreignKey:OwnerID;references:ID"`
	Property   Property `json:"property"`
}
