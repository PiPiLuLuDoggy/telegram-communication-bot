package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"telegram-communication-bot/internal/bot"
	"telegram-communication-bot/internal/config"
)

func main() {
	log.Println("Starting Telegram Communication Bot...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.ValidateConfig(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("App: %s", cfg.AppName)
	log.Printf("Admin Group: %d", cfg.AdminGroupID)
	log.Printf("Admin Users: %v", cfg.AdminUserIDs)
	log.Printf("Message Interval: %d seconds", cfg.MessageInterval)
	if cfg.WebhookURL != "" {
		log.Printf("Mode: Webhook (%s)", cfg.WebhookURL)
	} else {
		log.Printf("Mode: Polling")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	botInstance, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	defer botInstance.Stop()

	log.Println("Bot is starting...")
	if err := botInstance.Start(ctx); err != nil {
		log.Printf("Bot stopped with error: %v", err)
	}

	log.Println("Bot shutdown completed")
}
