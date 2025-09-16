package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"telegram-communication-bot/internal/bot"
	"telegram-communication-bot/internal/config"
)

func main() {
	log.Println("Starting Telegram Communication Bot...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.ValidateConfig(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("App: %s", cfg.AppName)
	log.Printf("Admin Group: %d", cfg.AdminGroupID)
	log.Printf("Admin Users: %v", cfg.AdminUserIDs)
	log.Printf("Message Interval: %d seconds", cfg.MessageInterval)

	// Create bot instance
	botInstance, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		cancel()
		botInstance.Stop()
	}()

	// Set webhook if URL is provided
	if cfg.WebhookURL != "" {
		log.Printf("Setting webhook to: %s", cfg.WebhookURL)
		if err := botInstance.SetWebhook(cfg.WebhookURL); err != nil {
			log.Fatalf("Failed to set webhook: %v", err)
		}
		log.Println("Webhook mode enabled")
	} else {
		log.Println("Using polling mode")
	}

	// Start the bot
	log.Println("Bot is starting...")
	if err := botInstance.Start(); err != nil {
		log.Printf("Bot stopped with error: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Bot shutdown completed")
}

