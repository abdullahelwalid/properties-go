package config

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/viper"

)

type Config struct {
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	Port       string `mapstructure:"PORT"`
	S3Bucket string `mapstructure:"AWS_S3_BUCKET"`
	RedisAddr string `mapstructure:"REDIS_ADDR"`
	RedisPass string `mapstructure:"REDIS_PASS"`
	RedisDB int `mapstructure:"REDIS_DB"`
}

var AppConfig Config

func LoadConfig() {

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Error reading config file: %s", err)
		log.Println("Using environment variables only")
	}

	// Handle PORT separately
	portValue := viper.GetString("PORT")
	if strings.Contains(portValue, ":") {
		parts := strings.Split(portValue, ":")
		if len(parts) == 2 {
			viper.Set("HOST", parts[0])
			portValue = parts[1]
		}
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		log.Printf("Invalid PORT value: %s. Using default 8080", portValue)
		port = 8080
	}
	viper.Set("PORT", port)

	// Unmarshal config
	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		panic(fmt.Errorf("fatal error when unmarshaling config: %s", err))
	}

	// Override config with environment variables
	t := reflect.TypeOf(AppConfig)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envKey := field.Tag.Get("mapstructure")
		if envKey != "" && envKey != "PORT" { // Skip PORT as we've already handled it
			envVal := viper.GetString(envKey)
			if envVal != "" {
				viper.Set(envKey, envVal)
			}
		}
	}

	// Re-unmarshal to ensure all values are updated
	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		panic(fmt.Errorf("fatal error when re-unmarshaling config: %s", err))
	}
}



