package api

import (
	"context"
	"fmt"
	"log"

	"golang-test/api/handler"
	"golang-test/api/route"
	"golang-test/config"
	"golang-test/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartServer() {
	// Load configuration
	cfg := config.AppConfig

	dbURL := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Run database migrations
	// Migrate function will apply the migration
	err = func() error {
		return db.AutoMigrate(&models.User{}, &models.Role{}, &models.Property{}, &models.PropertyCategory{}, &models.PropertyType{}, &models.Transaction{})
	}()
	if err != nil {
		log.Fatal("Failed to run migrations:", err)
	}
	// Redis configuration setup function (to be called during application startup)
	redisClient := func () *redis.Client {
		// Read configuration from environment or config file
		redisAddr := config.AppConfig.RedisAddr
		redisPassword := config.AppConfig.RedisPass
		redisDB := config.AppConfig.RedisDB

		// Set defaults if not provided
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		}
		if redisDB == 0 {
			redisDB = 0 // Default Redis DB
		}

		// Create Redis client
		client := redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		})

		// Ping Redis to check connection
		_, err := client.Ping(context.Background()).Result()
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v", err)
			// Return nil to indicate Redis is not available
			return nil
		}

		log.Println("Connected to Redis successfully")
		return client
	}()

	// Initialize handlers
	userHandler := &handler.UserHandler{DB: db}
	authHandler := &handler.AuthHandler{DB: db}
	roleHandler := &handler.RoleHandler{DB: db}
	propertiesHandler := &handler.PropertiesHandler{
		DB: db,
		Redis: redisClient,
	}
	categoryHandler := &handler.PropertyCategoryHandler{DB: db}
	propertyTypesHandler := &handler.PropertyTypeHandler{DB: db}
	transactionHandler := &handler.TransactionHandler{DB: db}

	// Set up routes
	r := route.SetupRouter(userHandler, authHandler, roleHandler, propertiesHandler, categoryHandler, propertyTypesHandler, transactionHandler)

	// Start the server
	log.Println("Starting server on port 8085...")
	if err := r.Run(":8085"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
