package handler

import (
	"golang-test/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PropertyTypeHandler struct {
	DB *gorm.DB
}

func (p *PropertyTypeHandler) CreateType(c *gin.Context) {
	var propertyType models.PropertyType
	if err := c.BindJSON(&propertyType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	propertyType.Name = strings.ToLower(propertyType.Name)
	// check if type exist
	p.DB.Model(models.Property{}).Where("name = ?", propertyType.Name).First(&propertyType)
	if propertyType.ID == 0{
		c.JSON(http.StatusConflict, gin.H{"error": "type already exist"})
		return

	}
	p.DB.Create(&propertyType)
	c.JSON(http.StatusCreated, gin.H{"message": "property type created successfully"})
}

func (p *PropertyTypeHandler) GetTypes(c *gin.Context) {
	var propertyTypes []models.PropertyType
	p.DB.Model(models.PropertyType{}).Find(&propertyTypes)
	c.JSON(http.StatusOK, gin.H{"categories": propertyTypes})
}
