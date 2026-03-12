package services

import (
	"context"
	"fmt"
	"log"
	"telegram-communication-bot/internal/config"
	"telegram-communication-bot/internal/database"
	dbmodels "telegram-communication-bot/internal/models"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ForumService struct {
	bot    *tgbot.Bot
	config *config.Config
	db     *database.DB
}

func NewForumService(bot *tgbot.Bot, config *config.Config, db *database.DB) *ForumService {
	return &ForumService{
		bot:    bot,
		config: config,
		db:     db,
	}
}

// CreateOrGetForumTopic creates a new forum topic for a user or returns existing one.
// Returns: threadID, isNewTopic, error
func (fs *ForumService) CreateOrGetForumTopic(ctx context.Context, user *dbmodels.User) (int, bool, error) {
	if !fs.config.HasAdminGroup() {
		return 0, false, fmt.Errorf("admin group not configured")
	}

	if user.MessageThreadID != 0 {
		return user.MessageThreadID, false, nil
	}

	topicName := fmt.Sprintf("%s|%d", fs.getFullName(user), user.UserID)

	topic, err := fs.bot.CreateForumTopic(ctx, &tgbot.CreateForumTopicParams{
		ChatID: fs.config.AdminGroupID,
		Name:   topicName,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to create forum topic: %w", err)
	}

	messageThreadID := topic.MessageThreadID

	user.MessageThreadID = messageThreadID
	if err := fs.db.CreateOrUpdateUser(user); err != nil {
		log.Printf("Error updating user with thread ID: %v", err)
	}

	forumStatus := &dbmodels.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          "opened",
	}
	if err := fs.db.CreateOrUpdateForumStatus(forumStatus); err != nil {
		log.Printf("Error creating forum status: %v", err)
	}

	return messageThreadID, true, nil
}

func (fs *ForumService) CloseForumTopic(ctx context.Context, messageThreadID int) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group not configured")
	}

	_, err := fs.bot.CloseForumTopic(ctx, &tgbot.CloseForumTopicParams{
		ChatID:          fs.config.AdminGroupID,
		MessageThreadID: messageThreadID,
	})
	if err != nil {
		return fmt.Errorf("failed to close forum topic: %w", err)
	}

	forumStatus := &dbmodels.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          "closed",
	}
	if err := fs.db.CreateOrUpdateForumStatus(forumStatus); err != nil {
		log.Printf("Error updating forum status: %v", err)
	}

	return nil
}

func (fs *ForumService) ReopenForumTopic(ctx context.Context, messageThreadID int) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group not configured")
	}

	_, err := fs.bot.ReopenForumTopic(ctx, &tgbot.ReopenForumTopicParams{
		ChatID:          fs.config.AdminGroupID,
		MessageThreadID: messageThreadID,
	})
	if err != nil {
		return fmt.Errorf("failed to reopen forum topic: %w", err)
	}

	forumStatus := &dbmodels.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          "opened",
	}
	if err := fs.db.CreateOrUpdateForumStatus(forumStatus); err != nil {
		log.Printf("Error updating forum status: %v", err)
	}

	return nil
}

func (fs *ForumService) DeleteForumTopic(ctx context.Context, messageThreadID int) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group not configured")
	}

	_, err := fs.bot.DeleteForumTopic(ctx, &tgbot.DeleteForumTopicParams{
		ChatID:          fs.config.AdminGroupID,
		MessageThreadID: messageThreadID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete forum topic: %w", err)
	}

	return fs.db.DB.Where("message_thread_id = ?", messageThreadID).Delete(&dbmodels.ForumStatus{}).Error
}

func (fs *ForumService) GetForumTopicStatus(messageThreadID int) (string, error) {
	status, err := fs.db.GetForumStatus(messageThreadID)
	if err != nil {
		return "unknown", err
	}
	return status.Status, nil
}

func (fs *ForumService) IsForumTopicClosed(messageThreadID int) bool {
	status, err := fs.GetForumTopicStatus(messageThreadID)
	if err != nil {
		return false
	}
	return status == "closed"
}

func (fs *ForumService) HandleForumStatusChange(messageThreadID int, newStatus string) error {
	forumStatus := &dbmodels.ForumStatus{
		MessageThreadID: messageThreadID,
		Status:          newStatus,
	}
	return fs.db.CreateOrUpdateForumStatus(forumStatus)
}

func (fs *ForumService) GetUserByThreadID(messageThreadID int) (*dbmodels.User, error) {
	var user dbmodels.User
	err := fs.db.DB.Where("message_thread_id = ?", messageThreadID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (fs *ForumService) IsForumMessage(message *models.Message) bool {
	return message.MessageThreadID != 0
}

func (fs *ForumService) getFullName(user *dbmodels.User) string {
	fullName := user.FirstName
	if user.LastName != "" {
		fullName += " " + user.LastName
	}
	return fullName
}

func (fs *ForumService) ValidateForumConfiguration(ctx context.Context) error {
	if !fs.config.HasAdminGroup() {
		return fmt.Errorf("admin group ID not configured")
	}

	chat, err := fs.bot.GetChat(ctx, &tgbot.GetChatParams{
		ChatID: fs.config.AdminGroupID,
	})
	if err != nil {
		return fmt.Errorf("cannot access admin group %d: %w", fs.config.AdminGroupID, err)
	}

	if chat.Type != "supergroup" && chat.Type != "channel" {
		return fmt.Errorf("admin group must be a supergroup or channel with topics enabled")
	}

	log.Printf("Forum configuration validated for group: %d (%s)", fs.config.AdminGroupID, chat.Title)
	return nil
}

func (fs *ForumService) GetAllActiveTopics() ([]dbmodels.ForumStatus, error) {
	var topics []dbmodels.ForumStatus
	err := fs.db.DB.Where("status = ?", "opened").Find(&topics).Error
	return topics, err
}

func (fs *ForumService) BulkUpdateTopicStatus(threadIDs []int, status string) error {
	return fs.db.DB.Model(&dbmodels.ForumStatus{}).
		Where("message_thread_id IN ?", threadIDs).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// ResetUserThreadID resets a user's thread ID when their topic is deleted
func (fs *ForumService) ResetUserThreadID(userID int64) error {
	var user dbmodels.User
	if err := fs.db.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("failed to find user %d: %w", userID, err)
	}

	oldThreadID := user.MessageThreadID

	err := fs.db.DB.Model(&dbmodels.User{}).
		Where("user_id = ?", userID).
		Update("message_thread_id", 0).Error
	if err != nil {
		return fmt.Errorf("failed to reset user thread ID: %w", err)
	}

	if oldThreadID != 0 {
		err = fs.db.DB.Where("message_thread_id = ?", oldThreadID).
			Delete(&dbmodels.ForumStatus{}).Error
		if err != nil {
			log.Printf("Warning: failed to delete forum status for user %d (thread %d): %v", userID, oldThreadID, err)
		}
	}

	log.Printf("Reset thread ID for user %d (old thread: %d)", userID, oldThreadID)
	return nil
}
