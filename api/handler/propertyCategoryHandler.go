package handler

import (
	"golang-test/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PropertyCategoryHandler struct {
	DB *gorm.DB
}

func (p *PropertyCategoryHandler) CreateCategory(c *gin.Context) {
	var category models.PropertyCategory
	if err := c.BindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	category.Name = strings.ToLower(category.Name)
	// check if type exist
	p.DB.Model(models.PropertyCategory{}).Where("name = ?", category.Name).First(&category)
	if category.ID == 0{
		c.JSON(http.StatusConflict, gin.H{"error": "type already exist"})
		return

	}

	p.DB.Create(&category)
	c.JSON(http.StatusCreated, gin.H{"message": "property category created successfully"})
}

func (p *PropertyCategoryHandler) GetCategories(c *gin.Context) {
	var categories []models.PropertyCategory
	p.DB.Model(models.PropertyCategory{}).Find(&categories)
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
