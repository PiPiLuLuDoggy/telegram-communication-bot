package services

import (
	"fmt"
	"log"
	"telegram-communication-bot/internal/database"
	"time"
)

type RateLimiter struct {
	interval int // seconds between messages
	db       *database.DB
}

func NewRateLimiter(interval int, db *database.DB) *RateLimiter {
	return &RateLimiter{
		interval: interval,
		db:       db,
	}
}

// CheckRateLimit checks if a user can send a message now
func (rl *RateLimiter) CheckRateLimit(userID int64) (bool, time.Duration, error) {
	if rl.interval <= 0 {
		// Rate limiting disabled
		return true, 0, nil
	}

	// Get recent messages from this user
	since := time.Now().Add(-time.Duration(rl.interval) * time.Second)
	recentMessages, err := rl.db.GetRecentUserMessages(userID, since)
	if err != nil {
		log.Printf("Error checking rate limit for user %d: %v", userID, err)
		// Allow message on error to avoid blocking users
		return true, 0, nil
	}

	if len(recentMessages) == 0 {
		// No recent messages, allow
		return true, 0, nil
	}

	// Find the most recent message
	var mostRecent time.Time
	for _, msg := range recentMessages {
		if msg.SentAt.After(mostRecent) {
			mostRecent = msg.SentAt
		}
	}

	// Calculate remaining wait time
	nextAllowedTime := mostRecent.Add(time.Duration(rl.interval) * time.Second)
	now := time.Now()

	if now.Before(nextAllowedTime) {
		// Still need to wait
		waitTime := nextAllowedTime.Sub(now)
		return false, waitTime, nil
	}

	// Can send now
	return true, 0, nil
}

// RecordMessage records that a user sent a message
func (rl *RateLimiter) RecordMessage(userID int64, chatID int64, messageID int) error {
	// This is handled by the MessageService, but we provide this method for consistency
	// In practice, the message recording is done when the message is processed
	return nil
}

// GetRemainingCooldown returns the remaining cooldown time for a user
func (rl *RateLimiter) GetRemainingCooldown(userID int64) time.Duration {
	canSend, waitTime, err := rl.CheckRateLimit(userID)
	if err != nil || canSend {
		return 0
	}
	return waitTime
}

// IsRateLimited returns true if the user is currently rate limited
func (rl *RateLimiter) IsRateLimited(userID int64) bool {
	canSend, _, _ := rl.CheckRateLimit(userID)
	return !canSend
}

// SetInterval updates the rate limit interval
func (rl *RateLimiter) SetInterval(interval int) {
	rl.interval = interval
}

// GetInterval returns the current rate limit interval
func (rl *RateLimiter) GetInterval() int {
	return rl.interval
}

// FormatCooldownMessage returns a formatted message about the cooldown
func (rl *RateLimiter) FormatCooldownMessage(waitTime time.Duration) string {
	seconds := int(waitTime.Seconds())
	if seconds <= 0 {
		return "您可以立即发送消息。"
	}

	if seconds < 60 {
		return fmt.Sprintf("⏰ 请等待 %d 秒后再发送消息", seconds)
	}

	minutes := seconds / 60
	remainingSeconds := seconds % 60

	if remainingSeconds == 0 {
		return fmt.Sprintf("⏰ 请等待 %d 分钟后再发送消息", minutes)
	}

	return fmt.Sprintf("⏰ 请等待 %d 分 %d 秒后再发送消息", minutes, remainingSeconds)
}

// IsEnabled returns true if rate limiting is enabled
func (rl *RateLimiter) IsEnabled() bool {
	return rl.interval > 0
}

// Disable disables rate limiting by setting interval to 0
func (rl *RateLimiter) Disable() {
	rl.interval = 0
}

// Enable enables rate limiting with the specified interval
func (rl *RateLimiter) Enable(interval int) {
	rl.interval = interval
}

// CleanupOldMessages removes old message records (called by scheduled tasks)
func (rl *RateLimiter) CleanupOldMessages() error {
	// Keep messages for up to 1 hour to handle edge cases
	cutoff := time.Now().Add(-1 * time.Hour)
	return rl.db.CleanupOldUserMessages(cutoff)
}