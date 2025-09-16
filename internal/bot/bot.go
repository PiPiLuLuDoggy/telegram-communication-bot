package bot

import (
	"fmt"
	"log"
	"time"
	"telegram-communication-bot/internal/config"
	"telegram-communication-bot/internal/database"
	"telegram-communication-bot/internal/handlers"
	"telegram-communication-bot/internal/services"

	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/robfig/cron/v3"
)

type Bot struct {
	API             *api.BotAPI
	Config          *config.Config
	DB              *database.DB
	Scheduler       *cron.Cron
	MessageService  *services.MessageService
	ForumService    *services.ForumService
	RateLimiter     *services.RateLimiter
	handlers        *handlers.Handlers
	stopChan        chan struct{}
}

// NewBot creates a new bot instance
func NewBot(cfg *config.Config) (*Bot, error) {
	// Initialize Telegram Bot API
	botAPI, err := api.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	botAPI.Debug = cfg.Debug
	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	// Initialize database
	db, err := database.NewDatabase(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize scheduler
	scheduler := cron.New(cron.WithSeconds())

	// Initialize services
	messageService := services.NewMessageService(db)
	forumService := services.NewForumService(botAPI, cfg, db)
	rateLimiter := services.NewRateLimiter(cfg.MessageInterval, db)

	// Initialize handlers
	handlers := handlers.NewHandlers(botAPI, cfg, db, messageService, forumService, rateLimiter, scheduler)

	bot := &Bot{
		API:            botAPI,
		Config:         cfg,
		DB:             db,
		Scheduler:      scheduler,
		MessageService: messageService,
		ForumService:   forumService,
		RateLimiter:    rateLimiter,
		handlers:       handlers,
		stopChan:       make(chan struct{}),
	}

	// Setup scheduled tasks
	bot.setupScheduledTasks()

	return bot, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	log.Println("Starting bot...")

	// Try to remove any existing webhook first
	log.Println("Removing any existing webhook...")
	if err := b.RemoveWebhook(); err != nil {
		log.Printf("Warning: Failed to remove webhook: %v", err)
	}

	// Start the scheduler
	b.Scheduler.Start()

	// Configure update settings
	updateConfig := api.NewUpdate(0)
	updateConfig.Timeout = 60
	updateConfig.AllowedUpdates = []string{
		api.UpdateTypeMessage,
		api.UpdateTypeCallbackQuery,
		api.UpdateTypeEditedMessage,
	}

	// Get update channel
	updates := b.API.GetUpdatesChan(updateConfig)

	// Start processing updates
	log.Println("Bot is running. Press Ctrl+C to stop.")
	for {
		select {
		case update := <-updates:
			// Process update in a goroutine
			go b.processUpdate(update)
		case <-b.stopChan:
			log.Println("Stopping bot...")
			return nil
		}
	}
}

// Stop stops the bot
func (b *Bot) Stop() {
	log.Println("Shutting down bot...")

	// Stop the scheduler
	b.Scheduler.Stop()

	// Close database connection
	if err := b.DB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	// Stop update processing
	close(b.stopChan)

	log.Println("Bot stopped")
}

// processUpdate processes an incoming update
func (b *Bot) processUpdate(update api.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in processUpdate: %v", r)
		}
	}()

	switch {
	case update.Message != nil:
		b.handlers.HandleMessage(update.Message)
	case update.EditedMessage != nil:
		b.handlers.HandleEditedMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		b.handlers.HandleCallbackQuery(update.CallbackQuery)
	}
}

// setupScheduledTasks sets up recurring scheduled tasks
func (b *Bot) setupScheduledTasks() {

	// Cleanup old user messages every hour
	b.Scheduler.AddFunc("@every 1h", func() {
		cutoff := time.Now().Add(-24 * time.Hour)
		if err := b.DB.CleanupOldUserMessages(cutoff); err != nil {
			log.Printf("Error cleaning up old user messages: %v", err)
		}
	})


	log.Println("Scheduled tasks configured")
}

// SetWebhook configures webhook mode
func (b *Bot) SetWebhook(webhookURL string) error {
	webhook, err := api.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to create webhook config: %w", err)
	}

	webhook.MaxConnections = 40
	webhook.AllowedUpdates = []string{
		api.UpdateTypeMessage,
		api.UpdateTypeCallbackQuery,
		api.UpdateTypeEditedMessage,
	}

	_, err = b.API.Request(webhook)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	log.Printf("Webhook set to: %s", webhookURL)
	return nil
}

// RemoveWebhook removes the webhook
func (b *Bot) RemoveWebhook() error {
	_, err := b.API.Request(api.DeleteWebhookConfig{})
	if err != nil {
		return fmt.Errorf("failed to remove webhook: %w", err)
	}

	log.Println("Webhook removed")
	return nil
}