package utils

import (
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash for the given password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// VerifyPassword verifies if the given password matches the stored hash.
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func verifyToken(tokenString string) (uint, uint, error) {
	var secretKey = []byte("secret-key")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return 0, 0, err
	}

	if !token.Valid {
		return 0, 0, fmt.Errorf("invalid token")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Extract user ID
		userIdFloat, ok := claims["userId"].(float64)
		if !ok {
			return 0, 0, fmt.Errorf("invalid userId claim")
		}
		userId := uint(userIdFloat)
		
		// Extract role ID if available
		var roleId uint = 0
		if roleIdValue, exists := claims["roleId"]; exists {
			if roleIdFloat, ok := roleIdValue.(float64); ok {
				roleId = uint(roleIdFloat)
			}
		}
		
		return userId, roleId, nil
	} else {
		return 0, 0, fmt.Errorf("An error has occurred")
	}
}

func AuthMiddleware(roles []uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		// Decode token
		tokenString := authHeader[len("Bearer "):]
		userId, roleId, err := verifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Token"})
			c.Abort()
			return
		}

		authorized := false
		for _, role := range(roles){
			if role == roleId || role == 0 {
				authorized = true
			}
		}
		if !authorized {
			fmt.Println(roles, roleId)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Insufficient permissions"})
			c.Abort()
			return

		}
		c.Set("userId", userId)
		c.Set("roleId", roleId)
		c.Next()
	}
}
