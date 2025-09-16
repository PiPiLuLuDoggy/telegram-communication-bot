package models

import (
	"time"
	"gorm.io/gorm"
)

// MediaGroupMessage represents a message in a media group
type MediaGroupMessage struct {
	ID            uint   `gorm:"primarykey" json:"id"`
	MediaGroupID  string `gorm:"index;not null" json:"media_group_id"`
	ChatID        int64  `gorm:"not null" json:"chat_id"`
	MessageID     int    `gorm:"not null" json:"message_id"`
	CaptionHTML   string `json:"caption_html"`
	CreatedAt     time.Time `json:"created_at"`
}

// ForumStatus represents the status of a forum topic
type ForumStatus struct {
	ID              uint   `gorm:"primarykey" json:"id"`
	MessageThreadID int    `gorm:"uniqueIndex;not null" json:"message_thread_id"`
	Status          string `gorm:"not null;default:'opened'" json:"status"` // "opened" or "closed"
	UpdatedAt       time.Time `json:"updated_at"`
}

// MessageMap maps messages between user chat and admin group
type MessageMap struct {
	ID                  uint  `gorm:"primarykey" json:"id"`
	UserChatMessageID   int   `gorm:"not null" json:"user_chat_message_id"`
	GroupChatMessageID  int   `gorm:"not null" json:"group_chat_message_id"`
	UserID              int64 `gorm:"not null;index" json:"user_id"`
	CreatedAt           time.Time `json:"created_at"`
}

// User represents a telegram user
type User struct {
	UserID          int64     `gorm:"primarykey" json:"user_id"`
	FirstName       string    `gorm:"not null" json:"first_name"`
	LastName        string    `json:"last_name"`
	Username        string    `json:"username"`
	IsPremium       bool      `gorm:"default:false" json:"is_premium"`
	MessageThreadID int       `json:"message_thread_id"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedAt       time.Time `json:"created_at"`
}


// UserMessage tracks user messages for rate limiting
type UserMessage struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	ChatID    int64     `gorm:"not null" json:"chat_id"`
	MessageID int       `gorm:"not null" json:"message_id"`
	SentAt    time.Time `gorm:"not null" json:"sent_at"`
}

// BanStatus represents a user's ban status
type BanStatus struct {
	UserID    int64     `gorm:"primarykey" json:"user_id"`
	IsBanned  bool      `gorm:"default:false" json:"is_banned"`
	BannedAt  time.Time `json:"banned_at"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AutoMigrateAll performs database migration for all models
func AutoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&MediaGroupMessage{},
		&ForumStatus{},
		&MessageMap{},
		&User{},
		&UserMessage{},
		&BanStatus{},
	)
}