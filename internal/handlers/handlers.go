package handlers

import (
	"log"
	"strings"
	"telegram-communication-bot/internal/config"
	"telegram-communication-bot/internal/database"
	"telegram-communication-bot/internal/models"
	"telegram-communication-bot/internal/services"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/robfig/cron/v3"
)

type Handlers struct {
	bot            *api.BotAPI
	config         *config.Config
	db             *database.DB
	messageService *services.MessageService
	forumService   *services.ForumService
	rateLimiter    *services.RateLimiter
	scheduler      *cron.Cron
}

func NewHandlers(
	bot *api.BotAPI,
	config *config.Config,
	db *database.DB,
	messageService *services.MessageService,
	forumService *services.ForumService,
	rateLimiter *services.RateLimiter,
	scheduler *cron.Cron,
) *Handlers {
	return &Handlers{
		bot:            bot,
		config:         config,
		db:             db,
		messageService: messageService,
		forumService:   forumService,
		rateLimiter:    rateLimiter,
		scheduler:      scheduler,
	}
}

// HandleMessage handles incoming messages
func (h *Handlers) HandleMessage(message *api.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in HandleMessage: %v", r)
		}
	}()

	if message.From == nil {
		return
	}

	userID := message.From.ID
	chatID := message.Chat.ID

	// Handle commands
	if message.IsCommand() {
		h.handleCommand(message)
		return
	}

	// Check if this is from admin group
	if h.config.HasAdminGroup() && chatID == h.config.AdminGroupID {
		h.handleAdminGroupMessage(message)
		return
	}

	// Check if user is banned
	if h.db.IsUserBanned(userID) {
		return
	}

	// Handle private chat messages from users
	if message.Chat.Type == "private" {
		h.handleUserMessage(message)
	}
}

// HandleEditedMessage handles edited messages
func (h *Handlers) HandleEditedMessage(message *api.Message) {
	// For now, we don't handle edited messages differently
	// In a full implementation, you might want to update the forwarded messages
	log.Printf("Edited message from user %d", message.From.ID)
}

// HandleCallbackQuery handles callback queries from inline keyboards
func (h *Handlers) HandleCallbackQuery(callbackQuery *api.CallbackQuery) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in HandleCallbackQuery: %v", r)
		}
	}()

	// Answer the callback query
	callback := api.CallbackConfig{
		CallbackQueryID: callbackQuery.ID,
	}
	h.bot.Request(callback)

	// Handle specific callback data
	data := callbackQuery.Data
	switch data {
	default:
		log.Printf("Unknown callback data: %s", data)
	}
}

// handleCommand handles bot commands
func (h *Handlers) handleCommand(message *api.Message) {
	command := message.Command()
	args := message.CommandArguments()
	userID := message.From.ID
	chatID := message.Chat.ID

	switch command {
	case "start":
		h.handleStartCommand(message)
	case "clear":
		if h.config.IsAdminUser(userID) {
			h.handleClearCommand(message, args)
		} else {
			h.sendMessage(chatID, "❌ 您没有权限使用此命令")
		}
	case "broadcast":
		if h.config.IsAdminUser(userID) {
			h.handleBroadcastCommand(message)
		} else {
			h.sendMessage(chatID, "❌ 您没有权限使用此命令")
		}
	case "stats":
		if h.config.IsAdminUser(userID) {
			h.handleStatsCommand(message)
		} else {
			h.sendMessage(chatID, "❌ 您没有权限使用此命令")
		}
	case "reset":
		if h.config.IsAdminUser(userID) {
			h.handleResetCommand(message, args)
		} else {
			h.sendMessage(chatID, "❌ 您没有权限使用此命令")
		}
	default:
		h.sendMessage(chatID, "❓ 未知命令。使用 /start 开始使用机器人。")
	}
}

// handleStartCommand handles the /start command
func (h *Handlers) handleStartCommand(message *api.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	// Check if this is from admin group
	if h.config.HasAdminGroup() && chatID == h.config.AdminGroupID {
		h.sendMessage(chatID, "✅ 机器人在管理群组中正常运行")
		return
	}

	// Private chat start
	if message.Chat.Type == "private" {
		// Update or create user record
		user := &models.User{
			UserID:    userID,
			FirstName: message.From.FirstName,
			LastName:  message.From.LastName,
			Username:  message.From.UserName,
			IsPremium: message.From.IsPremium,
		}

		if err := h.db.CreateOrUpdateUser(user); err != nil {
			log.Printf("Error updating user: %v", err)
		}

		// Send welcome message
		h.sendMessage(chatID, h.config.WelcomeMessage)
	}
}

// handleUserMessage handles messages from users in private chats
func (h *Handlers) handleUserMessage(message *api.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID


	// Check rate limit
	if h.rateLimiter.IsEnabled() {
		canSend, waitTime, err := h.rateLimiter.CheckRateLimit(userID)
		if err != nil {
			log.Printf("Error checking rate limit: %v", err)
		} else if !canSend {
			cooldownMsg := h.rateLimiter.FormatCooldownMessage(waitTime)
			h.sendMessage(chatID, cooldownMsg)
			return
		}
	}

	// Get or create user record
	user, err := h.db.GetUser(userID)
	if err != nil {
		// Create new user
		user = &models.User{
			UserID:    userID,
			FirstName: message.From.FirstName,
			LastName:  message.From.LastName,
			Username:  message.From.UserName,
			IsPremium: message.From.IsPremium,
		}
		if err := h.db.CreateOrUpdateUser(user); err != nil {
			log.Printf("Error creating user: %v", err)
			return
		}
	}

	// Forward message to admin group if configured
	if h.config.HasAdminGroup() {
		h.forwardUserMessageToAdmin(message, user)
	}

	// Record user message for rate limiting
	if err := h.messageService.RecordUserMessage(userID, chatID, message.MessageID); err != nil {
		log.Printf("Error recording user message: %v", err)
	}
}

// forwardUserMessageToAdmin forwards a user message to the admin group
func (h *Handlers) forwardUserMessageToAdmin(message *api.Message, user *models.User) {
	// Get or create forum topic
	threadID, isNewTopic, err := h.forumService.CreateOrGetForumTopic(user)
	if err != nil {
		log.Printf("Error creating forum topic: %v", err)
		return
	}

	// Send user info message for new conversations
	if isNewTopic {
		// Send user info message
		userInfoMsg, err := h.messageService.SendUserInfoMessage(h.bot, user, h.config.AdminGroupID, threadID)
		if err != nil {
			log.Printf("Error sending user info message: %v", err)
		} else {
			// Create message mapping for the user info message
			if err := h.messageService.CreateMessageMap(0, userInfoMsg.MessageID, user.UserID); err != nil {
				log.Printf("Error creating user info message mapping: %v", err)
			}
		}
	}

	// Handle media groups
	if message.MediaGroupID != "" {
		h.messageService.HandleMediaGroup(h.bot, message, h.config.AdminGroupID, threadID)
		return
	}

	// Forward the message
	forwardedMsg, err := h.messageService.ForwardMessageToGroup(h.bot, message, h.config.AdminGroupID, threadID)
	if err != nil {
		// Check if the error is due to thread not found
		if strings.Contains(err.Error(), "message thread not found") {
			log.Printf("Thread %d not found for user %d, resetting thread ID and retrying", threadID, user.UserID)

			// Reset user's thread ID
			if resetErr := h.forumService.ResetUserThreadID(user.UserID); resetErr != nil {
				log.Printf("Error resetting user thread ID: %v", resetErr)
				return
			}

			// Get updated user object from database
			updatedUser, err := h.db.GetUser(user.UserID)
			if err != nil {
				log.Printf("Error getting updated user: %v", err)
				return
			}

			// Retry forwarding message with updated user object
			h.forwardUserMessageToAdmin(message, updatedUser)
			return
		}

		log.Printf("Error forwarding message to admin group: %v", err)
		return
	}

	// Create message mapping
	if err := h.messageService.CreateMessageMap(message.MessageID, forwardedMsg.MessageID, user.UserID); err != nil {
		log.Printf("Error creating message map: %v", err)
	}
}

// handleAdminGroupMessage handles messages from the admin group
func (h *Handlers) handleAdminGroupMessage(message *api.Message) {
	// Check if this is a reply to a forwarded user message
	if message.ReplyToMessage != nil {
		h.handleAdminReply(message)
		return
	}

	// Handle forum topic status changes
	threadID := h.forumService.GetThreadIDFromMessage(message)
	if threadID != 0 {
		// Update forum status based on the message or admin actions
		// This would be implemented based on specific requirements
	}
}

// handleAdminReply handles admin replies to user messages
func (h *Handlers) handleAdminReply(message *api.Message) {
	replyToMessage := message.ReplyToMessage
	var user *models.User

	// First, try to find the original user message mapping
	messageMap, err := h.messageService.GetUserMessageFromGroup(replyToMessage.MessageID)
	if err != nil {
		log.Printf("Error finding message mapping: %v", err)

		// Fallback: try to find user by thread ID
		threadID := h.forumService.GetThreadIDFromMessage(message)
		if threadID != 0 {
			user, err = h.forumService.GetUserByThreadID(threadID)
			if err != nil {
				log.Printf("Error finding user by thread ID: %v", err)
				return
			}
		} else {
			log.Printf("No thread ID found in message")
			return
		}
	} else {
		// Get user info from message mapping
		user, err = h.db.GetUser(messageMap.UserID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			return
		}
	}

	// Forward admin's reply to the user
	forwardedMsg, err := h.messageService.ForwardMessageToUser(h.bot, message, user.UserID)
	if err != nil {
		log.Printf("Error forwarding admin reply: %v", err)
		return
	}

	// Create reverse message mapping
	if err := h.messageService.CreateMessageMap(forwardedMsg.MessageID, message.MessageID, user.UserID); err != nil {
		log.Printf("Error creating reverse message map: %v", err)
	}

	// Reopen forum topic if it was closed
	threadID := h.forumService.GetThreadIDFromMessage(message)
	if threadID != 0 && h.forumService.IsForumTopicClosed(threadID) {
		if err := h.forumService.ReopenForumTopic(threadID); err != nil {
			log.Printf("Error reopening forum topic: %v", err)
		}
	}
}


// sendMessage sends a text message to a chat
func (h *Handlers) sendMessage(chatID int64, text string) {
	msg := api.NewMessage(chatID, text)
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}