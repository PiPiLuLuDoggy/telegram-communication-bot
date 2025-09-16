package database

import (
	"fmt"
	"os"
	"path/filepath"
	"telegram-communication-bot/internal/models"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

type DB struct {
	*gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(databasePath string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM with custom settings
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// Open SQLite database with pure-Go driver
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        databasePath,
	}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings similar to Python version
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate all models
	if err := models.AutoMigrateAll(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// User operations
func (db *DB) CreateOrUpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	return db.DB.Save(user).Error
}

func (db *DB) GetUser(userID int64) (*models.User, error) {
	var user models.User
	err := db.DB.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *DB) GetAllUsers() ([]models.User, error) {
	var users []models.User
	err := db.DB.Find(&users).Error
	return users, err
}

// MessageMap operations
func (db *DB) CreateMessageMap(messageMap *models.MessageMap) error {
	messageMap.CreatedAt = time.Now()
	return db.DB.Create(messageMap).Error
}

func (db *DB) GetMessageMapByUserMessage(userChatMessageID int, userID int64) (*models.MessageMap, error) {
	var messageMap models.MessageMap
	err := db.DB.Where("user_chat_message_id = ? AND user_id = ?", userChatMessageID, userID).First(&messageMap).Error
	if err != nil {
		return nil, err
	}
	return &messageMap, nil
}

func (db *DB) GetMessageMapByGroupMessage(groupChatMessageID int) (*models.MessageMap, error) {
	var messageMap models.MessageMap
	err := db.DB.Where("group_chat_message_id = ?", groupChatMessageID).First(&messageMap).Error
	if err != nil {
		return nil, err
	}
	return &messageMap, nil
}

// MediaGroupMessage operations
func (db *DB) CreateMediaGroupMessage(msg *models.MediaGroupMessage) error {
	msg.CreatedAt = time.Now()
	return db.DB.Create(msg).Error
}

func (db *DB) GetMediaGroupMessages(mediaGroupID string) ([]models.MediaGroupMessage, error) {
	var messages []models.MediaGroupMessage
	err := db.DB.Where("media_group_id = ?", mediaGroupID).Find(&messages).Error
	return messages, err
}

func (db *DB) DeleteMediaGroupMessages(mediaGroupID string) error {
	return db.DB.Where("media_group_id = ?", mediaGroupID).Delete(&models.MediaGroupMessage{}).Error
}

// ForumStatus operations
func (db *DB) CreateOrUpdateForumStatus(status *models.ForumStatus) error {
	status.UpdatedAt = time.Now()
	return db.DB.Save(status).Error
}

func (db *DB) GetForumStatus(messageThreadID int) (*models.ForumStatus, error) {
	var status models.ForumStatus
	err := db.DB.Where("message_thread_id = ?", messageThreadID).First(&status).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}


// UserMessage operations for rate limiting
func (db *DB) CreateUserMessage(msg *models.UserMessage) error {
	return db.DB.Create(msg).Error
}

func (db *DB) GetRecentUserMessages(userID int64, since time.Time) ([]models.UserMessage, error) {
	var messages []models.UserMessage
	err := db.DB.Where("user_id = ? AND sent_at > ?", userID, since).Find(&messages).Error
	return messages, err
}

func (db *DB) CleanupOldUserMessages(before time.Time) error {
	return db.DB.Where("sent_at < ?", before).Delete(&models.UserMessage{}).Error
}

// BanStatus operations
func (db *DB) CreateOrUpdateBanStatus(banStatus *models.BanStatus) error {
	banStatus.UpdatedAt = time.Now()
	if banStatus.IsBanned {
		banStatus.BannedAt = time.Now()
	}
	return db.DB.Save(banStatus).Error
}

func (db *DB) GetBanStatus(userID int64) (*models.BanStatus, error) {
	var banStatus models.BanStatus
	err := db.DB.First(&banStatus, userID).Error
	if err != nil {
		return nil, err
	}
	return &banStatus, nil
}

func (db *DB) IsUserBanned(userID int64) bool {
	var banStatus models.BanStatus
	err := db.DB.First(&banStatus, userID).Error
	if err != nil {
		return false
	}
	return banStatus.IsBanned
}