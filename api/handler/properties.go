package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang-test/config"
	"golang-test/models"
	"golang-test/utils"
	"log"
	"net/http"
	"time"

	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PropertiesHandler struct {
	DB *gorm.DB
	Redis *redis.Client
}

// Function to cache properties results
func (p *PropertiesHandler) cacheProperties(key string, properties []models.Property) error {
	// Serialize properties to JSON
	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	
	// Cache with expiration (e.g., 10 minutes)
	return p.Redis.Set(context.Background(), key, propertiesJSON, 120*time.Minute).Err()
}

// Function to get properties from cache
func (p *PropertiesHandler) getPropertiesFromCache(key string) ([]models.Property, error) {
	// Try to get data from cache
	val, err := p.Redis.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist
			return nil, nil
		}
		return nil, err
	}
	
	// Deserialize JSON to properties
	var properties []models.Property
	if err := json.Unmarshal([]byte(val), &properties); err != nil {
		return nil, err
	}
	
	return properties, nil
}

// Function to invalidate property cache
func (p *PropertiesHandler) invalidatePropertyCache() error {
	// Use KEYS pattern to find all property-related cache keys
	// Note: In production, consider using SCAN for large datasets
	keys, err := p.Redis.Keys(context.Background(), "properties:*").Result()
	if err != nil {
		return err
	}
	
	// If there are keys to delete
	if len(keys) > 0 {
		return p.Redis.Del(context.Background(), keys...).Err()
	}
	
	return nil
}


func (p *PropertiesHandler) GetProperties(c *gin.Context) {
	// Get filter parameters
	categoryID := c.Query("categoryId")
	propertyTypeID := c.Query("typeId")
	description := c.Query("description")
	minPrice := c.Query("minPrice")
	maxPrice := c.Query("maxPrice")
	
	// Create params map for cache key generation
	params := map[string]string{
		"categoryId": categoryID,
		"typeId":     propertyTypeID,
		"description": description,
		"minPrice":   minPrice,
		"maxPrice":   maxPrice,
	}
	
	// Generate cache key
	cacheKey := utils.GeneratePropertiesCacheKey(params)
	
	// Try to get from cache first
	cachedProperties, err := p.getPropertiesFromCache(cacheKey)
	if err != nil {
		// Log the cache error but continue with database query
		log.Printf("Cache error: %v", err)
	} else if cachedProperties != nil {
		// Return cached data if available
		c.JSON(http.StatusOK, gin.H{"properties": cachedProperties, "source": "cache"})
		return
	}
	
	// Build query
	query := p.DB.Model(models.Property{}).Preload(clause.Associations)
	
	// Apply filters if provided
	if categoryID != "" {
		if id, err := strconv.Atoi(categoryID); err == nil {
			query = query.Where("property_category_id = ?", id)
		}
	}
	
	if propertyTypeID != "" {
		if id, err := strconv.Atoi(propertyTypeID); err == nil {
			query = query.Where("property_type_id = ?", id)
		}
	}
	
	// Apply description search if provided
	if description != "" {
		query = query.Where("description LIKE ?", "%"+description+"%")
	}
	
	// Apply price range filters if provided
	if minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			query = query.Where("price >= ?", price)
		}
	}
	
	if maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			query = query.Where("price <= ?", price)
		}
	}
	
	// Execute query
	var properties []models.Property
	if err := query.Find(&properties).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch properties"})
		return
	}
	
	// Cache the results
	if err := p.cacheProperties(cacheKey, properties); err != nil {
		log.Printf("Failed to cache properties: %v", err)
	}
	
	c.JSON(http.StatusOK, gin.H{"properties": properties, "source": "database"})
}


// GetPropertyByID retrieves a single property by ID
func (p *PropertiesHandler) GetPropertyByID(c *gin.Context) {
	id := c.Param("id")
	
	var property models.Property
	if err := p.DB.Preload(clause.Associations).First(&property, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch property"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"property": property})
}


// CreateProperty creates a new property
func (p *PropertiesHandler) CreateProperty(c *gin.Context) {
	// Parse multipart form with a reasonable max memory
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max memory
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}
	
	// Validate required form fields
	name := c.PostForm("name")
	description := c.PostForm("description")
	location := c.PostForm("location")
	priceStr := c.PostForm("price")
	propertyTypeIDStr := c.PostForm("propertyTypeId")
	propertyCategoryIDStr := c.PostForm("propertyCategoryId")
	
	// Validate all required fields
	if name == "" || description == "" || location == "" || priceStr == "" || 
	   propertyTypeIDStr == "" || propertyCategoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "All fields are required"})
		return
	}
	
	// Parse numeric values
	price, err := strconv.ParseFloat(priceStr, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price format"})
		return
	}
	
	propertyTypeID, err := strconv.ParseUint(propertyTypeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property type ID"})
		return
	}
	
	propertyCategoryID, err := strconv.ParseUint(propertyCategoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property category ID"})
		return
	}
	// Validate that propertyTypeID exists in the database
	var propertyType models.PropertyType
	if err := p.DB.First(&propertyType, propertyTypeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Property type does not exist"})
		return
	}
	// Validate that propertyCategoryID exists in the database
	var propertyCategory models.PropertyCategory
	if err := p.DB.First(&propertyCategory, propertyCategoryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Property category does not exist"})
		return
	}
	
	// Set the OwnerID from the JWT token claim
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	// Create a new property object
	property := models.Property{
		Name:               name,
		Description:        description,
		Status:             "available", // Set default status as available
		Price:              float32(price),
		Location:           location,
		OwnerID:            userId.(uint),
		PropertyTypeID:     uint(propertyTypeID),
		PropertyCategoryID: uint(propertyCategoryID),
	}
	
	// Create property record first to get the ID
	if err := p.DB.Create(&property).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create property"})
		return
	}
	
	// Process file uploads
	form, _ := c.MultipartForm()
	files := form.File["images"]
	
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one image is required"})
		// Clean up the created property
		p.DB.Delete(&property)
		return
	}
	
	// Create a unique image prefix using property ID
	imagePrefix := fmt.Sprintf("property/property_%d", property.ID)
	property.ImagePrefix = imagePrefix
	
	cfg := config.AppConfig
	bucketName := cfg.S3Bucket
	
	// Upload files using a worker pool
	success := utils.UploadFilesWithWorkerPool(c, bucketName, imagePrefix, files)
	if !success {
		// If file uploads failed, clean up the property
		p.DB.Delete(&property)
		return
	}
	
	// Update the property with the image prefix
	if err := p.DB.Save(&property).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property with image information"})
		// Clean up all uploaded files
		cleanupS3Files(bucketName, imagePrefix, len(files))
		return
	}
	
	// Invalidate property cache after successful creation
	if err := p.invalidatePropertyCache(); err != nil {
		// Log the error but don't fail the request
		log.Printf("Failed to invalidate property cache: %v", err)
	}
	
	c.JSON(http.StatusCreated, gin.H{"message": "Property created successfully", "data": property})
}


// UpdateProperty updates an existing property
func (p *PropertiesHandler) UpdateProperty(c *gin.Context) {
	id := c.Param("id")
	
	// First, get the existing property
	var property models.Property
	if err := p.DB.First(&property, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch property"})
		return
	}
	
	// Check authorization (only owner or admin can update)
	if !p.canModifyProperty(c, property.OwnerID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this property"})
		return
	}
	
	// Bind new data
	var updatedProperty models.Property
	if err := c.BindJSON(&updatedProperty); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	
	// Preserve data that shouldn't be changed
	updatedProperty.ID = property.ID
	updatedProperty.OwnerID = property.OwnerID
	updatedProperty.Status = property.Status // Status should be changed via transactions, not direct updates
	
	// Update the property
	if err := p.DB.Model(&property).Updates(updatedProperty).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Property updated successfully", "data": property})
}

// DeleteProperty deletes a property
func (p *PropertiesHandler) DeleteProperty(c *gin.Context) {
	id := c.Param("id")
	
	// First, get the existing property
	var property models.Property
	if err := p.DB.First(&property, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch property"})
		return
	}
	
	// Check authorization (only owner or admin can delete)
	if !p.canModifyProperty(c, property.OwnerID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this property"})
		return
	}
	
	// Check if property is involved in any transactions
	var transactionCount int64
	p.DB.Model(&models.Transaction{}).Where("property_id = ?", property.ID).Count(&transactionCount)
	if transactionCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete property with existing transactions"})
		return
	}
	
	// Delete the property
	if err := p.DB.Delete(&property).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete property"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Property deleted successfully"})
}

// Helper to check if user is authorized to modify a property
func (p *PropertiesHandler) canModifyProperty(c *gin.Context, ownerID uint) bool {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userId")
	if !exists {
		return false
	}
	
	// Check if user is the owner
	if userID.(uint) == ownerID {
		return true
	}
	
	// Check if user is admin
	roleID, exists := c.Get("roleId")
	if !exists {
		return false
	}
	
	// Get the role name
	var role models.Role
	if err := p.DB.First(&role, roleID).Error; err != nil {
		return false
	}
	
	// Return true if admin
	return role.Name == "admin"
}

func cleanupS3Files(bucketName, imagePrefix string, count int) {
	for i := 0; i < count; i++ {
		fileKey := fmt.Sprintf("%s/image_%d", imagePrefix, i)
		utils.DeleteFileFromS3(bucketName, fileKey)
	}
}
