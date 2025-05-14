package models

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	ID   uint   `gorm:"primarykey"`
	Name string `json:"name" gorm:"unique"`
}
