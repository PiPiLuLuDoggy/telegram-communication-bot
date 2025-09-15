package services

import (
	"fmt"
	"log"
	"telegram-communication-bot/internal/config"
	"telegram-communication-bot/internal/database"
	"telegram-communication-bot/internal/models"

	api "github.com/TGlimmer/telegram-bot-api"
)

type ForumService struct {
	bot    *api.BotAPI
	config *config.Config
	db     *database.DB
}

func NewForumService(bot *api.BotAPI, config *config.Config, db *database.DB) *ForumService {
	return &ForumService{
		bot:    bot,
		config: config,
		db:     db,
	}
}

// CreateOrGetForumTopic creates a new forum topic for a user or returns existing one
func (fs *ForumService) CreateOrGetForumTopic(user *models.User) (int, error) {
	if !fs.config.HasAdminGroup() {
		return 0, fmt.Errorf("admin group not configured")
	}

	// If user already has a thread ID, return it
	if user.MessageThreadID != 0 {
		return user.MessageThreadID, nil
	}

	// Create new forum topic
	topicName := fmt.Sprintf("%s|%d", fs.getFullName(user), user.UserID)

	createTopicConfig := api.NewCreateForumTopicConfig(fs.config.AdminGroupID, topicName)

	topic, err := fs.bot.CreateForumTopic(createTopicConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to create forum topic: %w", err)
	}

	messageThreadID := topic.MessageThreadID

	// Update user with the new thread ID
	user.MessageThreadID = messageThreadID
	if err := fs.db.CreateOrUpdateUser(user); err != nil {
		log.Printf("Error updating user with thread ID: %v", err)
	}

	// Create forum status record
	forumStatus := &models.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          "opened",
	}
	if err := fs.db.CreateOrUpdateForumStatus(forumStatus); err != nil {
		log.Printf("Error creating forum status: %v", err)
	}

	return messageThreadID, nil
}

// CloseForumTopic closes a forum topic
func (fs *ForumService) CloseForumTopic(messageThreadID int) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group not configured")
	}

	// Create close forum topic config
	closeConfig := api.CloseForumTopicConfig{
		BaseForum: api.BaseForum{
			ChatConfig: api.ChatConfig{
				ChatID: fs.config.AdminGroupID,
			},
			MessageThreadID: messageThreadID,
		},
	}

	_, err := fs.bot.Request(closeConfig)
	if err != nil {
		return fmt.Errorf("failed to close forum topic: %w", err)
	}

	// Update forum status in database
	forumStatus := &models.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          "closed",
	}
	if err := fs.db.CreateOrUpdateForumStatus(forumStatus); err != nil {
		log.Printf("Error updating forum status: %v", err)
	}

	return nil
}

// ReopenForumTopic reopens a forum topic
func (fs *ForumService) ReopenForumTopic(messageThreadID int) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group not configured")
	}

	// Create reopen forum topic config
	reopenConfig := api.ReopenForumTopicConfig{
		BaseForum: api.BaseForum{
			ChatConfig: api.ChatConfig{
				ChatID: fs.config.AdminGroupID,
			},
			MessageThreadID: messageThreadID,
		},
	}

	_, err := fs.bot.Request(reopenConfig)
	if err != nil {
		return fmt.Errorf("failed to reopen forum topic: %w", err)
	}

	// Update forum status in database
	forumStatus := &models.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          "opened",
	}
	if err := fs.db.CreateOrUpdateForumStatus(forumStatus); err != nil {
		log.Printf("Error updating forum status: %v", err)
	}

	return nil
}

// DeleteForumTopic deletes a forum topic
func (fs *ForumService) DeleteForumTopic(messageThreadID int) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group not configured")
	}

	// Create delete forum topic config
	deleteConfig := api.DeleteForumTopicConfig{
		BaseForum: api.BaseForum{
			ChatConfig: api.ChatConfig{
				ChatID: fs.config.AdminGroupID,
			},
			MessageThreadID: messageThreadID,
		},
	}

	_, err := fs.bot.Request(deleteConfig)
	if err != nil {
		return fmt.Errorf("failed to delete forum topic: %w", err)
	}

	// Remove from database after successful deletion
	return fs.db.DB.Where("message_thread_id = ?", messageThreadID).Delete(&models.ForumStatus{}).Error
}

// GetForumTopicStatus gets the status of a forum topic
func (fs *ForumService) GetForumTopicStatus(messageThreadID int) (string, error) {
	status, err := fs.db.GetForumStatus(messageThreadID)
	if err != nil {
		return "unknown", err
	}
	return status.Status, nil
}

// IsForumTopicClosed checks if a forum topic is closed
func (fs *ForumService) IsForumTopicClosed(messageThreadID int) bool {
	status, err := fs.GetForumTopicStatus(messageThreadID)
	if err != nil {
		return false
	}
	return status == "closed"
}

// HandleForumStatusChange handles forum topic status changes from admin actions
func (fs *ForumService) HandleForumStatusChange(messageThreadID int, newStatus string) error {
	forumStatus := &models.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          newStatus,
	}
	return fs.db.CreateOrUpdateForumStatus(forumStatus)
}

// GetUserByThreadID finds a user by their forum thread ID
func (fs *ForumService) GetUserByThreadID(messageThreadID int) (*models.User, error) {
	// This requires a database query to find user by message_thread_id
	var user models.User
	err := fs.db.DB.Where("message_thread_id = ?", messageThreadID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetThreadIDFromMessage extracts thread ID from a message if it's in a forum topic
func (fs *ForumService) GetThreadIDFromMessage(message *api.Message) int {
	return message.MessageThreadID
}

// IsForumMessage checks if a message is from a forum topic
func (fs *ForumService) IsForumMessage(message *api.Message) bool {
	return fs.GetThreadIDFromMessage(message) != 0
}

// getFullName returns the full name of a user
func (fs *ForumService) getFullName(user *models.User) string {
	fullName := user.FirstName
	if user.LastName != "" {
		fullName += " " + user.LastName
	}
	return fullName
}

// ValidateForumConfiguration validates forum-related configuration
func (fs *ForumService) ValidateForumConfiguration() error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group ID not configured")
	}

	// Test if the group exists and is accessible
	chatConfig := api.ChatInfoConfig{
		ChatConfig: api.ChatConfig{
			ChatID: fs.config.AdminGroupID,
		},
	}

	chat, err := fs.bot.GetChat(chatConfig)
	if err != nil {
		return fmt.Errorf("cannot access admin group %d: %w", fs.config.AdminGroupID, err)
	}

	if chat.Type != "supergroup" && chat.Type != "channel" {
		return fmt.Errorf("admin group must be a supergroup or channel with topics enabled")
	}

	log.Printf("Forum configuration validated for group: %d (%s)", fs.config.AdminGroupID, chat.Title)
	return nil
}

// GetAllActiveTopics returns all active forum topics (for maintenance)
func (fs *ForumService) GetAllActiveTopics() ([]models.ForumStatus, error) {
	var topics []models.ForumStatus
	err := fs.db.DB.Where("status = ?", "opened").Find(&topics).Error
	return topics, err
}

// BulkUpdateTopicStatus updates multiple topic statuses (for maintenance)
func (fs *ForumService) BulkUpdateTopicStatus(threadIDs []int, status string) error {
	return fs.db.DB.Model(&models.ForumStatus{}).
		Where("message_thread_id IN ?", threadIDs).
		Update("status", status).
		Update("updated_at", "NOW()").Error
}