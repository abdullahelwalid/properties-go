package models

import "gorm.io/gorm"

type Property struct {
	gorm.Model
	ID                 uint             `gorm:"primarykey"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	Status             string           `json:"status" gorm:"default:available"`
	Price              float32          `json:"price"`
	Location           string           `json:"location"`
	OwnerID            uint             `json:"ownerId"`
	Owner              User             `json:"owner" gorm:"foreignKey:OwnerID;references:ID"`
	ImagePrefix           string        `json:"imagePrefix"`
	PropertyTypeID     uint             `json:"propertyTypeId"`
	PropertyType       PropertyType     `json:"propertyType"`
	PropertyCategoryID uint             `json:"propertyCategoryId"`
	PropertyCategory   PropertyCategory `json:"propertyCategory"`
}
