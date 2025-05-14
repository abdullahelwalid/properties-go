package models

import "gorm.io/gorm"

// User defines the structure of the user model
type User struct {
	gorm.Model
	ID       uint   `gorm:"primarykey"`
	Name     string `json:"name"`
	Email    string `json:"email" gorm:"unique"`
	Password string `json:"password"`
	RoleID   uint   `json:"roleId" gorm:"default:1"`
	Role     Role
}

func (u *User) Serialize() *map[string]interface{} {
	return &map[string]interface{}{
		"name":   u.Name,
		"email":  u.Email,
		"roleId": u.RoleID,
		"id":     u.ID,
	}
}
