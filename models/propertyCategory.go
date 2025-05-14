package models

import "gorm.io/gorm"

type PropertyCategory struct {
	gorm.Model
	ID   uint   `gorm:"primarykey"`
	Name string `json:"name"`
}
