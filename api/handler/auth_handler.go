package handler

import (
	"encoding/json"
	"golang-test/models"
	"golang-test/utils"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB *gorm.DB
}

var secretKey = []byte("secret-key")

func (h *AuthHandler) Login(c *gin.Context) {
	type reqBodyFields struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var reqBody reqBodyFields
	err := json.NewDecoder(c.Request.Body).Decode(&reqBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request, invalid request body"})
		return
	}

	var user models.User
	// Check if user exist
	h.DB.Model(models.User{}).Where("email = ?", reqBody.Email).First(&user)
	if user.ID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email or password is invalid"})
		return
	}
	validPassword := utils.VerifyPassword(reqBody.Password, user.Password)
	if !validPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email or password is invalid"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"userId": user.ID,
			"roleId": user.RoleID,
			"exp":    time.Now().Add(time.Hour * 1).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error has occurred while generating auth creds"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}
