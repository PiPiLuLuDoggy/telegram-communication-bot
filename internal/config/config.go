package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Telegram Bot Configuration
	BotToken    string
	AppName     string
	WelcomeMessage string

	// Admin Configuration
	AdminGroupID  int64
	AdminUserIDs  []int64

	// Bot Behavior Settings
	DeleteTopicAsForeverBan      bool
	DeleteUserMessageOnClearCmd  bool
	DisableCaptcha               bool
	MessageInterval              int

	// Database Settings
	DatabasePath string

	// Server Settings
	Port       int
	WebhookURL string

	// Debug mode
	Debug bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{}

	// Load required configuration
	config.BotToken = os.Getenv("BOT_TOKEN")
	if config.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	// Load app settings
	config.AppName = getEnvWithDefault("APP_NAME", "TelegramCommunicationBot")
	config.WelcomeMessage = getEnvWithDefault("WELCOME_MESSAGE", "欢迎使用我们的客服机器人！请发送您的问题，我们的客服人员将尽快回复您。")

	// Load admin configuration
	adminGroupIDStr := os.Getenv("ADMIN_GROUP_ID")
	if adminGroupIDStr != "" {
		adminGroupID, err := strconv.ParseInt(adminGroupIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ADMIN_GROUP_ID: %w", err)
		}
		config.AdminGroupID = adminGroupID
	}

	// Load admin user IDs
	adminUserIDsStr := os.Getenv("ADMIN_USER_IDS")
	if adminUserIDsStr != "" {
		adminUserIDStrSlice := strings.Split(adminUserIDsStr, ",")
		config.AdminUserIDs = make([]int64, len(adminUserIDStrSlice))
		for i, idStr := range adminUserIDStrSlice {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid admin user ID '%s': %w", idStr, err)
			}
			config.AdminUserIDs[i] = id
		}
	}

	// Load bot behavior settings
	config.DeleteTopicAsForeverBan = getBoolEnv("DELETE_TOPIC_AS_FOREVER_BAN", false)
	config.DeleteUserMessageOnClearCmd = getBoolEnv("DELETE_USER_MESSAGE_ON_CLEAR_CMD", false)
	config.DisableCaptcha = getBoolEnv("DISABLE_CAPTCHA", false)
	config.MessageInterval = getIntEnv("MESSAGE_INTERVAL", 5)

	// Load database settings
	config.DatabasePath = getEnvWithDefault("DATABASE_PATH", "./data/bot.db")

	// Load server settings
	config.Port = getIntEnv("PORT", 8080)
	config.WebhookURL = os.Getenv("WEBHOOK_URL")

	// Debug mode
	config.Debug = getBoolEnv("DEBUG", false)

	return config, nil
}

// IsAdminUser checks if a user ID is in the admin list
func (c *Config) IsAdminUser(userID int64) bool {
	for _, adminID := range c.AdminUserIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// HasAdminGroup checks if admin group is configured
func (c *Config) HasAdminGroup() bool {
	return c.AdminGroupID != 0
}

// ValidateConfig validates the configuration
func (c *Config) ValidateConfig() error {
	if c.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	if c.MessageInterval < 0 {
		return fmt.Errorf("MESSAGE_INTERVAL must be non-negative")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}

	return nil
}

// Helper functions
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		result, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Warning: Invalid boolean value for %s: %s, using default: %v", key, value, defaultValue)
			return defaultValue
		}
		return result
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		result, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
			return defaultValue
		}
		return result
	}
	return defaultValue
}