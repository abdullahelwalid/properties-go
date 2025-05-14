package handler

import (
	"golang-test/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RoleHandler struct {
	DB *gorm.DB
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	type reqBodyFields struct {
		Name string `json:"name"`
	}
	var reqBody reqBodyFields
	if err := c.BindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	var role models.Role
	reqBody.Name = strings.ToLower(reqBody.Name)
	//check if user exist
	h.DB.Model(models.Role{}).Where("name = ?", reqBody.Name).First(&role)
	if role.ID != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "role already exist"})
		return
	}
	role.Name = reqBody.Name
	if err := h.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role created successfully", "data": role})
}
