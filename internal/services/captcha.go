package services

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-telegram/bot/models"
)

type CaptchaChallenge struct {
	Answer    int
	MessageID int
	ChatID    int64
	ExpiresAt time.Time
}

type CaptchaService struct {
	mu               sync.RWMutex
	challenges       map[int64]*CaptchaChallenge
	cooldowns        map[int64]time.Time
	expiration       time.Duration
	cooldownDuration time.Duration
}

func NewCaptchaService() *CaptchaService {
	return &CaptchaService{
		challenges:       make(map[int64]*CaptchaChallenge),
		cooldowns:        make(map[int64]time.Time),
		expiration:       5 * time.Minute,
		cooldownDuration: 3 * time.Minute,
	}
}

// GenerateChallenge creates a math CAPTCHA and returns the question text with an inline keyboard.
// If a non-expired challenge already exists for the user, it is replaced.
func (s *CaptchaService) GenerateChallenge(userID int64) (string, models.InlineKeyboardMarkup) {
	a := rand.Intn(20) + 1
	b := rand.Intn(20) + 1
	answer := a + b
	question := fmt.Sprintf("🔒 请完成人机验证\n\n❓ %d + %d = ?", a, b)

	options := generateOptions(answer)

	var buttons []models.InlineKeyboardButton
	for _, opt := range options {
		buttons = append(buttons, models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%d", opt),
			CallbackData: fmt.Sprintf("captcha_%d", opt),
		})
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{buttons},
	}

	s.mu.Lock()
	s.challenges[userID] = &CaptchaChallenge{
		Answer:    answer,
		ExpiresAt: time.Now().Add(s.expiration),
	}
	s.mu.Unlock()

	return question, keyboard
}

// SetMessageInfo records the bot's CAPTCHA message so it can be deleted later.
func (s *CaptchaService) SetMessageInfo(userID int64, messageID int, chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.challenges[userID]; ok {
		ch.MessageID = messageID
		ch.ChatID = chatID
	}
}

// GetMessageInfo returns the CAPTCHA message ID and chat ID for deletion.
func (s *CaptchaService) GetMessageInfo(userID int64) (messageID int, chatID int64, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, exists := s.challenges[userID]
	if !exists {
		return 0, 0, false
	}
	return ch.MessageID, ch.ChatID, ch.MessageID != 0
}

// Verify checks the user's answer. Returns true if correct.
// On failure the challenge is removed and a cooldown is set.
func (s *CaptchaService) Verify(userID int64, answer int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.challenges[userID]
	if !ok || time.Now().After(ch.ExpiresAt) {
		delete(s.challenges, userID)
		s.cooldowns[userID] = time.Now().Add(s.cooldownDuration)
		return false
	}

	if ch.Answer == answer {
		delete(s.challenges, userID)
		delete(s.cooldowns, userID)
		return true
	}

	delete(s.challenges, userID)
	s.cooldowns[userID] = time.Now().Add(s.cooldownDuration)
	return false
}

// IsInCooldown reports whether the user is in a post-failure cooldown period.
func (s *CaptchaService) IsInCooldown(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	until, ok := s.cooldowns[userID]
	return ok && time.Now().Before(until)
}

// GetCooldownRemaining returns how long the user must wait before retrying.
func (s *CaptchaService) GetCooldownRemaining(userID int64) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	until, ok := s.cooldowns[userID]
	if !ok {
		return 0
	}
	remaining := time.Until(until)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// HasActiveChallenge reports whether a non-expired challenge exists for the user.
func (s *CaptchaService) HasActiveChallenge(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.challenges[userID]
	if !ok {
		return false
	}
	return time.Now().Before(ch.ExpiresAt)
}

// RemoveChallenge deletes any pending challenge for the user.
func (s *CaptchaService) RemoveChallenge(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.challenges, userID)
}

// CleanupExpired removes all expired challenges and cooldowns to prevent memory leaks.
func (s *CaptchaService) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for userID, ch := range s.challenges {
		if now.After(ch.ExpiresAt) {
			delete(s.challenges, userID)
		}
	}
	for userID, until := range s.cooldowns {
		if now.After(until) {
			delete(s.cooldowns, userID)
		}
	}
}

// generateOptions returns 4 unique positive integers including the correct answer, shuffled.
func generateOptions(answer int) []int {
	options := []int{answer}
	seen := map[int]bool{answer: true}

	for len(options) < 4 {
		offset := rand.Intn(10) + 1
		if rand.Intn(2) == 0 {
			offset = -offset
		}
		wrong := answer + offset
		if wrong > 0 && !seen[wrong] {
			seen[wrong] = true
			options = append(options, wrong)
		}
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	return options
}
