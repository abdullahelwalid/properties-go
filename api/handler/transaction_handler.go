package handler

import (
	"errors"
	"golang-test/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionHandler struct {
	DB *gorm.DB
}

// CreateTransaction handles buying or renting a property
func (t *TransactionHandler) CreateTransaction(c *gin.Context) {
	var requestBody struct {
		PropertyID uint `json:"propertyId" binding:"required"`
	}
	
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	
	// Get the client ID from JWT
	clientID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	// Start a transaction
	tx := t.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 1. Get the property with its owner
	var property models.Property
	if err := tx.Preload("PropertyType").First(&property, requestBody.PropertyID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch property"})
		}
		return
	}
	
	// 2. Check if property is available
	if property.Status != "available" {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Property is not available"})
		return
	}
	
	// 3. Check that client is not the owner
	if property.OwnerID == clientID.(uint) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot buy/rent your own property"})
		return
	}
	
	// 4. Create transaction record
	transaction := models.Transaction{
		ClientID:   clientID.(uint),
		OwnerID:    property.OwnerID,
		PropertyID: property.ID,
		Type:       property.PropertyType.Name, // "rent" or "buy"
	}
	
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}
	
	// 5. Update property status
	newStatus := "sold"
	if property.PropertyType.Name == "rent" {
		newStatus = "rented"
	}
	
	if err := tx.Model(&property).Update("status", newStatus).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property status"})
		return
	}
	
	// 6. Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete transaction"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Property " + newStatus + " successfully",
		"transaction": transaction,
	})
}

// GetUserTransactions gets all transactions for the current user
func (t *TransactionHandler) GetUserTransactions(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	var transactions []models.Transaction
	if err := t.DB.Preload(clause.Associations).
		Where("client_id = ? OR owner_id = ?", userID, userID).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}
