package services

import (
	"fmt"
	"sync"
	"time"
)

type RateLimiter struct {
	interval int // seconds between messages
	mu       sync.Mutex
	lastMsg  map[int64]time.Time
}

func NewRateLimiter(interval int) *RateLimiter {
	return &RateLimiter{
		interval: interval,
		lastMsg:  make(map[int64]time.Time),
	}
}

// CheckAndRecord atomically checks if a user can send and records the timestamp.
// Returns (canSend, waitTime).
func (rl *RateLimiter) CheckAndRecord(userID int64) (bool, time.Duration) {
	if rl.interval <= 0 {
		return true, 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if lastTime, ok := rl.lastMsg[userID]; ok {
		nextAllowed := lastTime.Add(time.Duration(rl.interval) * time.Second)
		if now.Before(nextAllowed) {
			return false, nextAllowed.Sub(now)
		}
	}

	rl.lastMsg[userID] = now
	return true, 0
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

// SetInterval updates the rate limit interval
func (rl *RateLimiter) SetInterval(interval int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.interval = interval
}

// GetInterval returns the current rate limit interval
func (rl *RateLimiter) GetInterval() int {
	return rl.interval
}

// CleanupStaleEntries removes entries older than 2x the interval to prevent memory leaks
func (rl *RateLimiter) CleanupStaleEntries() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-2 * time.Duration(rl.interval) * time.Second)
	for userID, lastTime := range rl.lastMsg {
		if lastTime.Before(cutoff) {
			delete(rl.lastMsg, userID)
		}
	}
}
