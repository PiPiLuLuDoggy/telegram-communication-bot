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
func NewDatabase(databasePath string, debug bool) (*DB, error) {
	dir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        databasePath,
	}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode and busy timeout for better concurrent performance
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// SQLite only supports one writer at a time; keep pool small to avoid lock contention
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(time.Hour)

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

// CountUsers returns the total number of users
func (db *DB) CountUsers() (int64, error) {
	var count int64
	err := db.DB.Model(&models.User{}).Count(&count).Error
	return count, err
}

// CountPremiumUsers returns the number of premium users
func (db *DB) CountPremiumUsers() (int64, error) {
	var count int64
	err := db.DB.Model(&models.User{}).Where("is_premium = ?", true).Count(&count).Error
	return count, err
}

// CountBannedUsers returns the number of banned users
func (db *DB) CountBannedUsers() (int64, error) {
	var count int64
	err := db.DB.Model(&models.BanStatus{}).Where("is_banned = ?", true).Count(&count).Error
	return count, err
}

// IsUserVerified checks if a user has passed CAPTCHA verification
func (db *DB) IsUserVerified(userID int64) bool {
	var user models.User
	if err := db.DB.Select("verified").First(&user, userID).Error; err != nil {
		return false
	}
	return user.Verified
}

// SetUserVerified updates the verified status for a user
func (db *DB) SetUserVerified(userID int64, verified bool) error {
	return db.DB.Model(&models.User{}).Where("user_id = ?", userID).Update("verified", verified).Error
}