package handler

import (
	"errors"
	"fmt"
	"golang-test/models"
	"golang-test/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserHandler struct {
	DB *gorm.DB
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	type reqBodyFields struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		RoleID   uint   `json:"roleId"`
	}
	var reqBody reqBodyFields
	if err := c.BindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	var user models.User
	//check if user exist
	h.DB.Model(models.User{}).Where("email = ?", reqBody.Email).First(&user)
	if user.ID != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exist"})
		return
	}
	// Check if role exist
	var role models.Role
	h.DB.Model(models.Role{}).First(&role)
	if role.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role doesn't exist"})
		return
	}
	hashedPassword, err := utils.HashPassword(reqBody.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error has occurred while creating your account"})
		return
	}
	user.Password = hashedPassword
	user.Email = reqBody.Email
	user.Name = reqBody.Name
	user.RoleID = reqBody.RoleID
	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "data": user.Serialize()})
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	var users []models.User
	if err := h.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	var concurrency int
	if len(users) > 5 && len(users) < 100 {
		concurrency = 10
	} else if len(users) > 100 {
		concurrency = 20
	} else {
		concurrency = 2
	}
	fmt.Println(concurrency)
	serializedUsersInit := utils.Pool{Concurrency: concurrency}
	serializedUsers := serializedUsersInit.Run(users)
	c.JSON(http.StatusOK, gin.H{"users": serializedUsers})
}

func (p *UserHandler) GetUserById(c *gin.Context) {
	id := c.Param("id")
	
	var user models.User
	if err := p.DB.Preload(clause.Associations).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"user": user.Serialize()})
}
