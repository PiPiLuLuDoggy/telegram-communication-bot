package services

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"telegram-communication-bot/internal/database"
	"telegram-communication-bot/internal/models"
	"time"

	api "github.com/TGlimmer/telegram-bot-api"
)

type CaptchaService struct {
	db          *database.DB
	captchaPath string
}

func NewCaptchaService(db *database.DB) *CaptchaService {
	return &CaptchaService{
		db:          db,
		captchaPath: "./assets/captcha",
	}
}

// SendCaptcha sends a captcha challenge to the user
func (cs *CaptchaService) SendCaptcha(bot *api.BotAPI, chatID int64, userID int64) (*api.Message, error) {
	// Get a random captcha image
	captchaFile, code, err := cs.getRandomCaptchaImage()
	if err != nil {
		return nil, fmt.Errorf("failed to get captcha image: %w", err)
	}

	// Send the captcha image
	photo := api.NewPhoto(chatID, api.FilePath(captchaFile))
	photo.Caption = "🔐 请输入图片中的验证码以完成人机验证\n⏰ 您有2分钟时间完成验证"

	message, err := bot.Send(photo)
	if err != nil {
		return nil, fmt.Errorf("failed to send captcha: %w", err)
	}

	// Save captcha session
	session := &models.CaptchaSession{
		UserID:    userID,
		Code:      code,
		MessageID: message.MessageID,
		ChatID:    chatID,
		ExpiresAt: time.Now().Add(2 * time.Minute),
	}

	if err := cs.db.CreateCaptchaSession(session); err != nil {
		return nil, fmt.Errorf("failed to save captcha session: %w", err)
	}

	// Schedule message deletion after 60 seconds
	go func() {
		time.Sleep(60 * time.Second)
		deleteConfig := api.NewDeleteMessage(chatID, message.MessageID)
		bot.Request(deleteConfig)
	}()

	return &message, nil
}

// VerifyCaptcha verifies the user's captcha input
func (cs *CaptchaService) VerifyCaptcha(userID int64, input string) (bool, error) {
	session, err := cs.db.GetCaptchaSession(userID)
	if err != nil {
		return false, err
	}

	// Check if session is expired
	if session.IsExpired() {
		cs.db.DeleteCaptchaSession(userID)
		return false, fmt.Errorf("captcha session expired")
	}

	// Verify the code (case insensitive)
	if strings.ToLower(strings.TrimSpace(input)) == strings.ToLower(session.Code) {
		// Delete the session after successful verification
		cs.db.DeleteCaptchaSession(userID)
		return true, nil
	}

	return false, nil
}

// GetCaptchaSession gets the user's captcha session
func (cs *CaptchaService) GetCaptchaSession(userID int64) (*models.CaptchaSession, error) {
	return cs.db.GetCaptchaSession(userID)
}

// DeleteCaptchaSession deletes the user's captcha session
func (cs *CaptchaService) DeleteCaptchaSession(userID int64) error {
	return cs.db.DeleteCaptchaSession(userID)
}

// getRandomCaptchaImage returns a random captcha image file and its code
func (cs *CaptchaService) getRandomCaptchaImage() (string, string, error) {
	// Read all PNG files from captcha directory
	files, err := filepath.Glob(filepath.Join(cs.captchaPath, "*.png"))
	if err != nil {
		return "", "", fmt.Errorf("failed to read captcha directory: %w", err)
	}

	if len(files) == 0 {
		return "", "", fmt.Errorf("no captcha images found in %s", cs.captchaPath)
	}

	// Select a random file
	rand.Seed(time.Now().UnixNano())
	selectedFile := files[rand.Intn(len(files))]

	// Extract the code from the filename
	filename := filepath.Base(selectedFile)
	code := strings.TrimSuffix(filename, filepath.Ext(filename))

	return selectedFile, code, nil
}

// CleanupOldCaptchaImages cleans up old captcha sessions (for maintenance)
func (cs *CaptchaService) CleanupOldCaptchaImages(cutoff time.Time) {
	// This method is called by scheduled tasks to clean up
	// In the Go version, we rely on the database cleanup
	// The actual captcha images are static assets and don't need cleanup
}

// HasCaptchaSession checks if user has an active captcha session
func (cs *CaptchaService) HasCaptchaSession(userID int64) bool {
	session, err := cs.db.GetCaptchaSession(userID)
	if err != nil {
		return false
	}
	return !session.IsExpired()
}

// GenerateRandomCaptchaCode generates a random 5-character code
func (cs *CaptchaService) GenerateRandomCaptchaCode() string {
	const charset = "ABCDEFGHIJKLMNPQRSTUVWXYZ123456789" // Excluded O and 0 to avoid confusion
	code := make([]byte, 5)
	rand.Seed(time.Now().UnixNano())

	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}

// CreateCaptchaFromCode creates a captcha session with a specific code (for testing)
func (cs *CaptchaService) CreateCaptchaFromCode(userID int64, chatID int64, code string, messageID int) error {
	session := &models.CaptchaSession{
		UserID:    userID,
		Code:      code,
		MessageID: messageID,
		ChatID:    chatID,
		ExpiresAt: time.Now().Add(2 * time.Minute),
	}

	return cs.db.CreateCaptchaSession(session)
}

// GetCaptchaImageCount returns the number of available captcha images
func (cs *CaptchaService) GetCaptchaImageCount() (int, error) {
	files, err := filepath.Glob(filepath.Join(cs.captchaPath, "*.png"))
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// ValidateCaptchaDirectory ensures the captcha directory exists and has images
func (cs *CaptchaService) ValidateCaptchaDirectory() error {
	if _, err := os.Stat(cs.captchaPath); os.IsNotExist(err) {
		return fmt.Errorf("captcha directory does not exist: %s", cs.captchaPath)
	}

	files, err := filepath.Glob(filepath.Join(cs.captchaPath, "*.png"))
	if err != nil {
		return fmt.Errorf("failed to read captcha directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no captcha images found in %s", cs.captchaPath)
	}

	return nil
}

// ExtractCodeFromFilename extracts the captcha code from filename
func (cs *CaptchaService) ExtractCodeFromFilename(filename string) string {
	base := filepath.Base(filename)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// IsValidCaptchaCode validates if a captcha code format is correct
func (cs *CaptchaService) IsValidCaptchaCode(code string) bool {
	// Check if code is exactly 5 characters and contains only alphanumeric
	if len(code) != 5 {
		return false
	}

	for _, char := range code {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	return true
}