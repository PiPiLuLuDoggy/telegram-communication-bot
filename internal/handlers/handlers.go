package handlers

import (
	"context"
	"log"
	"strings"
	"telegram-communication-bot/internal/config"
	"telegram-communication-bot/internal/database"
	dbmodels "telegram-communication-bot/internal/models"
	"telegram-communication-bot/internal/services"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handlers struct {
	bot            *tgbot.Bot
	config         *config.Config
	db             *database.DB
	messageService *services.MessageService
	forumService   *services.ForumService
	rateLimiter    *services.RateLimiter
}

func NewHandlers(
	bot *tgbot.Bot,
	config *config.Config,
	db *database.DB,
	messageService *services.MessageService,
	forumService *services.ForumService,
	rateLimiter *services.RateLimiter,
) *Handlers {
	return &Handlers{
		bot:            bot,
		config:         config,
		db:             db,
		messageService: messageService,
		forumService:   forumService,
		rateLimiter:    rateLimiter,
	}
}

// HandleUpdate dispatches an incoming update to the appropriate handler.
func (h *Handlers) HandleUpdate(ctx context.Context, update *models.Update) {
	switch {
	case update.Message != nil:
		h.handleMessage(ctx, update.Message)
	case update.EditedMessage != nil:
		h.handleEditedMessage(ctx, update.EditedMessage)
	case update.CallbackQuery != nil:
		h.handleCallbackQuery(ctx, update.CallbackQuery)
	}
}

func (h *Handlers) handleMessage(ctx context.Context, message *models.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleMessage: %v", r)
		}
	}()

	if message.From == nil {
		return
	}

	userID := message.From.ID
	chatID := message.Chat.ID

	if isCommand(message) {
		h.handleCommand(ctx, message)
		return
	}

	if h.config.HasAdminGroup() && chatID == h.config.AdminGroupID {
		h.handleAdminGroupMessage(ctx, message)
		return
	}

	if h.db.IsUserBanned(userID) {
		return
	}

	if message.Chat.Type == "private" {
		h.handleUserMessage(ctx, message)
	}
}

func (h *Handlers) handleEditedMessage(ctx context.Context, message *models.Message) {
	if message.From != nil {
		log.Printf("Edited message from user %d", message.From.ID)
	}
}

func (h *Handlers) handleCallbackQuery(ctx context.Context, callbackQuery *models.CallbackQuery) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleCallbackQuery: %v", r)
		}
	}()

	h.bot.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQuery.ID,
	})

	data := callbackQuery.Data
	switch data {
	default:
		log.Printf("Unknown callback data: %s", data)
	}
}

func (h *Handlers) handleCommand(ctx context.Context, message *models.Message) {
	command := extractCommand(message)
	args := extractCommandArgs(message)
	userID := message.From.ID
	chatID := message.Chat.ID

	switch command {
	case "start":
		h.handleStartCommand(ctx, message)
	case "clear":
		if h.config.IsAdminUser(userID) {
			h.handleClearCommand(ctx, message, args)
		} else {
			h.sendMessage(ctx, chatID, "❌ 您没有权限使用此命令")
		}
	case "broadcast":
		if h.config.IsAdminUser(userID) {
			h.handleBroadcastCommand(ctx, message)
		} else {
			h.sendMessage(ctx, chatID, "❌ 您没有权限使用此命令")
		}
	case "stats":
		if h.config.IsAdminUser(userID) {
			h.handleStatsCommand(ctx, message)
		} else {
			h.sendMessage(ctx, chatID, "❌ 您没有权限使用此命令")
		}
	case "reset":
		if h.config.IsAdminUser(userID) {
			h.handleResetCommand(ctx, message, args)
		} else {
			h.sendMessage(ctx, chatID, "❌ 您没有权限使用此命令")
		}
	default:
		h.sendMessage(ctx, chatID, "❓ 未知命令。使用 /start 开始使用机器人。")
	}
}

func (h *Handlers) handleStartCommand(ctx context.Context, message *models.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	if h.config.HasAdminGroup() && chatID == h.config.AdminGroupID {
		h.sendMessage(ctx, chatID, "✅ 机器人在管理群组中正常运行")
		return
	}

	if message.Chat.Type == "private" {
		user := &dbmodels.User{
			UserID:    userID,
			FirstName: message.From.FirstName,
			LastName:  message.From.LastName,
			Username:  message.From.Username,
			IsPremium: message.From.IsPremium,
		}
		if err := h.db.CreateOrUpdateUser(user); err != nil {
			log.Printf("Error updating user: %v", err)
		}
		h.sendMessage(ctx, chatID, h.config.WelcomeMessage)
	}
}

func (h *Handlers) handleUserMessage(ctx context.Context, message *models.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	if h.rateLimiter.IsEnabled() {
		canSend, waitTime := h.rateLimiter.CheckAndRecord(userID)
		if !canSend {
			h.sendMessage(ctx, chatID, h.rateLimiter.FormatCooldownMessage(waitTime))
			return
		}
	}

	user, err := h.db.GetUser(userID)
	if err != nil {
		user = &dbmodels.User{
			UserID:    userID,
			FirstName: message.From.FirstName,
			LastName:  message.From.LastName,
			Username:  message.From.Username,
			IsPremium: message.From.IsPremium,
		}
		if err := h.db.CreateOrUpdateUser(user); err != nil {
			log.Printf("Error creating user: %v", err)
			return
		}
	}

	if h.config.HasAdminGroup() {
		h.forwardUserMessageToAdmin(ctx, message, user)
	}

	go func() {
		if err := h.messageService.RecordUserMessage(userID, chatID, message.ID); err != nil {
			log.Printf("Error recording user message: %v", err)
		}
	}()
}

// forwardUserMessageToAdmin forwards a user message to the admin group.
// Uses a retry loop (max 1 retry) to handle deleted topics.
func (h *Handlers) forwardUserMessageToAdmin(ctx context.Context, message *models.Message, user *dbmodels.User) {
	const maxAttempts = 2

	for attempt := 0; attempt < maxAttempts; attempt++ {
		threadID, isNewTopic, err := h.forumService.CreateOrGetForumTopic(ctx, user)
		if err != nil {
			log.Printf("Error creating forum topic: %v", err)
			return
		}

		if isNewTopic {
			userInfoMsg, err := h.messageService.SendUserInfoMessage(ctx, h.bot, user, h.config.AdminGroupID, threadID)
			if err != nil {
				log.Printf("Error sending user info message: %v", err)
			} else {
				if err := h.messageService.CreateMessageMap(0, userInfoMsg.ID, user.UserID); err != nil {
					log.Printf("Error creating user info message mapping: %v", err)
				}
			}
		}

		if message.MediaGroupID != "" {
			h.messageService.HandleMediaGroup(ctx, h.bot, message, h.config.AdminGroupID, threadID)
			return
		}

		forwardedMsg, err := h.messageService.ForwardMessageToGroup(ctx, h.bot, message, h.config.AdminGroupID, threadID)
		if err != nil {
			if strings.Contains(err.Error(), "message thread not found") && attempt < maxAttempts-1 {
				log.Printf("Thread %d not found for user %d, resetting and retrying", threadID, user.UserID)

				if resetErr := h.forumService.ResetUserThreadID(user.UserID); resetErr != nil {
					log.Printf("Error resetting user thread ID: %v", resetErr)
					return
				}

				updatedUser, getErr := h.db.GetUser(user.UserID)
				if getErr != nil {
					log.Printf("Error getting updated user: %v", getErr)
					return
				}
				user = updatedUser
				continue
			}

			log.Printf("Error forwarding message to admin group: %v", err)
			return
		}

		if err := h.messageService.CreateMessageMap(message.ID, forwardedMsg.ID, user.UserID); err != nil {
			log.Printf("Error creating message map: %v", err)
		}
		return
	}
}

func (h *Handlers) handleAdminGroupMessage(ctx context.Context, message *models.Message) {
	if message.ReplyToMessage != nil {
		h.handleAdminReply(ctx, message)
		return
	}
}

func (h *Handlers) handleAdminReply(ctx context.Context, message *models.Message) {
	replyToMessage := message.ReplyToMessage
	var user *dbmodels.User

	messageMap, err := h.messageService.GetUserMessageFromGroup(replyToMessage.ID)
	if err != nil {
		log.Printf("Error finding message mapping: %v", err)

		threadID := message.MessageThreadID
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
		user, err = h.db.GetUser(messageMap.UserID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			return
		}
	}

	forwardedMsg, err := h.messageService.ForwardMessageToUser(ctx, h.bot, message, user.UserID)
	if err != nil {
		log.Printf("Error forwarding admin reply: %v", err)
		return
	}

	if err := h.messageService.CreateMessageMap(forwardedMsg.ID, message.ID, user.UserID); err != nil {
		log.Printf("Error creating reverse message map: %v", err)
	}

	threadID := message.MessageThreadID
	if threadID != 0 && h.forumService.IsForumTopicClosed(threadID) {
		if err := h.forumService.ReopenForumTopic(ctx, threadID); err != nil {
			log.Printf("Error reopening forum topic: %v", err)
		}
	}
}

func (h *Handlers) sendMessage(ctx context.Context, chatID int64, text string) {
	_, err := h.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// --- Command parsing helpers ---

func isCommand(msg *models.Message) bool {
	if msg == nil || msg.Text == "" || len(msg.Entities) == 0 {
		return false
	}
	return msg.Entities[0].Type == models.MessageEntityTypeBotCommand && msg.Entities[0].Offset == 0
}

func extractCommand(msg *models.Message) string {
	if !isCommand(msg) {
		return ""
	}
	cmd := msg.Text[1:msg.Entities[0].Length]
	if i := strings.Index(cmd, "@"); i != -1 {
		cmd = cmd[:i]
	}
	return cmd
}

func extractCommandArgs(msg *models.Message) string {
	if !isCommand(msg) {
		return ""
	}
	rest := msg.Text[msg.Entities[0].Length:]
	return strings.TrimSpace(rest)
}
