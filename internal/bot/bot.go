package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"telegram-communication-bot/internal/config"
	"telegram-communication-bot/internal/database"
	"telegram-communication-bot/internal/handlers"
	"telegram-communication-bot/internal/services"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/robfig/cron/v3"
)

type Bot struct {
	tg             *tgbot.Bot
	Config         *config.Config
	DB             *database.DB
	Scheduler      *cron.Cron
	MessageService *services.MessageService
	ForumService   *services.ForumService
	RateLimiter    *services.RateLimiter
	handlers       *handlers.Handlers
}

func NewBot(cfg *config.Config) (*Bot, error) {
	db, err := database.NewDatabase(cfg.DatabasePath, cfg.Debug)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	scheduler := cron.New(cron.WithSeconds())
	messageService := services.NewMessageService(db)
	rateLimiter := services.NewRateLimiter(cfg.MessageInterval)

	b := &Bot{
		Config:         cfg,
		DB:             db,
		Scheduler:      scheduler,
		MessageService: messageService,
		RateLimiter:    rateLimiter,
	}

	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(b.handleUpdate),
	}
	if cfg.Debug {
		opts = append(opts, tgbot.WithDebug())
	}

	tg, err := tgbot.New(cfg.BotToken, opts...)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	b.tg = tg

	forumService := services.NewForumService(tg, cfg, db)
	b.ForumService = forumService

	h := handlers.NewHandlers(tg, cfg, db, messageService, forumService, rateLimiter)
	b.handlers = h

	b.setupScheduledTasks()

	log.Println("Bot initialized successfully")
	return b, nil
}

// Start starts the bot in either webhook or polling mode based on config.
// It blocks until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) error {
	b.Scheduler.Start()

	if b.Config.WebhookURL != "" {
		return b.startWebhook(ctx)
	}
	b.startPolling(ctx)
	return nil
}

func (b *Bot) startPolling(ctx context.Context) {
	log.Println("Removing any existing webhook...")
	b.tg.DeleteWebhook(ctx, &tgbot.DeleteWebhookParams{})

	log.Println("Bot is running in polling mode. Press Ctrl+C to stop.")
	b.tg.Start(ctx)
}

func (b *Bot) startWebhook(ctx context.Context) error {
	_, err := b.tg.SetWebhook(ctx, &tgbot.SetWebhookParams{
		URL:            b.Config.WebhookURL,
		MaxConnections: 40,
		AllowedUpdates: []string{"message", "edited_message", "callback_query"},
	})
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}
	log.Printf("Webhook set to: %s", b.Config.WebhookURL)

	go b.tg.StartWebhook(ctx)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", b.Config.Port),
		Handler: b.tg.WebhookHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Bot is running in webhook mode on port %d. Press Ctrl+C to stop.", b.Config.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webhook server error: %w", err)
	}
	return nil
}

// Stop releases resources (DB, scheduler). Call after Start returns.
func (b *Bot) Stop() {
	log.Println("Shutting down bot...")
	b.Scheduler.Stop()

	if err := b.DB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
	log.Println("Bot stopped")
}

func (b *Bot) handleUpdate(ctx context.Context, _ *tgbot.Bot, update *models.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleUpdate: %v", r)
		}
	}()

	b.handlers.HandleUpdate(ctx, update)
}

func (b *Bot) setupScheduledTasks() {
	b.Scheduler.AddFunc("@every 1h", func() {
		cutoff := time.Now().Add(-24 * time.Hour)
		if err := b.DB.CleanupOldUserMessages(cutoff); err != nil {
			log.Printf("Error cleaning up old user messages: %v", err)
		}
		b.RateLimiter.CleanupStaleEntries()
	})

	log.Println("Scheduled tasks configured")
}
