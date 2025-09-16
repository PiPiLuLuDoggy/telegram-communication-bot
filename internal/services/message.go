package services

import (
	"fmt"
	"log"
	"strings"
	"telegram-communication-bot/internal/database"
	"telegram-communication-bot/internal/models"
	"time"

	api "github.com/OvyFlash/telegram-bot-api"
)

type MessageService struct {
	db *database.DB
}

func NewMessageService(db *database.DB) *MessageService {
	return &MessageService{
		db: db,
	}
}

// CreateMessageMap creates a mapping between user and admin group messages
func (ms *MessageService) CreateMessageMap(userChatMessageID int, groupChatMessageID int, userID int64) error {
	messageMap := &models.MessageMap{
		UserChatMessageID:  userChatMessageID,
		GroupChatMessageID: groupChatMessageID,
		UserID:             userID,
	}
	return ms.db.CreateMessageMap(messageMap)
}

// GetUserMessageFromGroup finds the user message ID from a group message ID
func (ms *MessageService) GetUserMessageFromGroup(groupChatMessageID int) (*models.MessageMap, error) {
	return ms.db.GetMessageMapByGroupMessage(groupChatMessageID)
}

// GetGroupMessageFromUser finds the group message ID from a user message ID
func (ms *MessageService) GetGroupMessageFromUser(userChatMessageID int, userID int64) (*models.MessageMap, error) {
	return ms.db.GetMessageMapByUserMessage(userChatMessageID, userID)
}

// ForwardMessageToGroup forwards a message from user to admin group
func (ms *MessageService) ForwardMessageToGroup(bot *api.BotAPI, fromMessage *api.Message, groupChatID int64, messageThreadID int) (*api.Message, error) {
	var sentMessage api.Message
	var err error

	switch {
	case fromMessage.Text != "":
		// Text message
		msg := api.NewMessage(groupChatID, fromMessage.Text)
		msg.MessageThreadID = messageThreadID
		if fromMessage.Entities != nil {
			msg.Entities = fromMessage.Entities
		}
		sentMessage, err = bot.Send(msg)

	case fromMessage.Photo != nil:
		// Photo message
		photo := api.NewPhoto(groupChatID, api.FileID(fromMessage.Photo[len(fromMessage.Photo)-1].FileID))
		photo.MessageThreadID = messageThreadID
		if fromMessage.Caption != "" {
			photo.Caption = fromMessage.Caption
			photo.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(photo)

	case fromMessage.Document != nil:
		// Document message
		doc := api.NewDocument(groupChatID, api.FileID(fromMessage.Document.FileID))
		doc.MessageThreadID = messageThreadID
		if fromMessage.Caption != "" {
			doc.Caption = fromMessage.Caption
			doc.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(doc)

	case fromMessage.Video != nil:
		// Video message
		video := api.NewVideo(groupChatID, api.FileID(fromMessage.Video.FileID))
		video.MessageThreadID = messageThreadID
		if fromMessage.Caption != "" {
			video.Caption = fromMessage.Caption
			video.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(video)

	case fromMessage.Audio != nil:
		// Audio message
		audio := api.NewAudio(groupChatID, api.FileID(fromMessage.Audio.FileID))
		audio.MessageThreadID = messageThreadID
		if fromMessage.Caption != "" {
			audio.Caption = fromMessage.Caption
			audio.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(audio)

	case fromMessage.Voice != nil:
		// Voice message
		voice := api.NewVoice(groupChatID, api.FileID(fromMessage.Voice.FileID))
		voice.MessageThreadID = messageThreadID
		if fromMessage.Caption != "" {
			voice.Caption = fromMessage.Caption
			voice.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(voice)

	case fromMessage.VideoNote != nil:
		// Video note
		videoNote := api.NewVideoNote(groupChatID, fromMessage.VideoNote.Length, api.FileID(fromMessage.VideoNote.FileID))
		videoNote.MessageThreadID = messageThreadID
		sentMessage, err = bot.Send(videoNote)

	case fromMessage.Sticker != nil:
		// Sticker
		sticker := api.NewSticker(groupChatID, api.FileID(fromMessage.Sticker.FileID))
		sticker.MessageThreadID = messageThreadID
		sentMessage, err = bot.Send(sticker)

	case fromMessage.Animation != nil:
		// Animation (GIF)
		animation := api.NewAnimation(groupChatID, api.FileID(fromMessage.Animation.FileID))
		animation.MessageThreadID = messageThreadID
		if fromMessage.Caption != "" {
			animation.Caption = fromMessage.Caption
			animation.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(animation)

	case fromMessage.Location != nil:
		// Location
		location := api.NewLocation(groupChatID, fromMessage.Location.Latitude, fromMessage.Location.Longitude)
		location.MessageThreadID = messageThreadID
		sentMessage, err = bot.Send(location)

	case fromMessage.Contact != nil:
		// Contact
		contact := api.NewContact(groupChatID, fromMessage.Contact.PhoneNumber, fromMessage.Contact.FirstName)
		contact.MessageThreadID = messageThreadID
		contact.LastName = fromMessage.Contact.LastName
		sentMessage, err = bot.Send(contact)

	default:
		return nil, fmt.Errorf("unsupported message type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to forward message: %w", err)
	}

	return &sentMessage, nil
}

// ForwardMessageToUser forwards a message from admin group to user
func (ms *MessageService) ForwardMessageToUser(bot *api.BotAPI, fromMessage *api.Message, userChatID int64) (*api.Message, error) {
	var sentMessage api.Message
	var err error

	switch {
	case fromMessage.Text != "":
		// Text message
		msg := api.NewMessage(userChatID, fromMessage.Text)
		if fromMessage.Entities != nil {
			msg.Entities = fromMessage.Entities
		}
		sentMessage, err = bot.Send(msg)

	case fromMessage.Photo != nil:
		// Photo message
		photo := api.NewPhoto(userChatID, api.FileID(fromMessage.Photo[len(fromMessage.Photo)-1].FileID))
		if fromMessage.Caption != "" {
			photo.Caption = fromMessage.Caption
			photo.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(photo)

	case fromMessage.Document != nil:
		// Document message
		doc := api.NewDocument(userChatID, api.FileID(fromMessage.Document.FileID))
		if fromMessage.Caption != "" {
			doc.Caption = fromMessage.Caption
			doc.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(doc)

	case fromMessage.Video != nil:
		// Video message
		video := api.NewVideo(userChatID, api.FileID(fromMessage.Video.FileID))
		if fromMessage.Caption != "" {
			video.Caption = fromMessage.Caption
			video.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(video)

	case fromMessage.Audio != nil:
		// Audio message
		audio := api.NewAudio(userChatID, api.FileID(fromMessage.Audio.FileID))
		if fromMessage.Caption != "" {
			audio.Caption = fromMessage.Caption
			audio.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(audio)

	case fromMessage.Voice != nil:
		// Voice message
		voice := api.NewVoice(userChatID, api.FileID(fromMessage.Voice.FileID))
		if fromMessage.Caption != "" {
			voice.Caption = fromMessage.Caption
			voice.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(voice)

	case fromMessage.VideoNote != nil:
		// Video note
		videoNote := api.NewVideoNote(userChatID, fromMessage.VideoNote.Length, api.FileID(fromMessage.VideoNote.FileID))
		sentMessage, err = bot.Send(videoNote)

	case fromMessage.Sticker != nil:
		// Sticker
		sticker := api.NewSticker(userChatID, api.FileID(fromMessage.Sticker.FileID))
		sentMessage, err = bot.Send(sticker)

	case fromMessage.Animation != nil:
		// Animation (GIF)
		animation := api.NewAnimation(userChatID, api.FileID(fromMessage.Animation.FileID))
		if fromMessage.Caption != "" {
			animation.Caption = fromMessage.Caption
			animation.CaptionEntities = fromMessage.CaptionEntities
		}
		sentMessage, err = bot.Send(animation)

	case fromMessage.Location != nil:
		// Location
		location := api.NewLocation(userChatID, fromMessage.Location.Latitude, fromMessage.Location.Longitude)
		sentMessage, err = bot.Send(location)

	case fromMessage.Contact != nil:
		// Contact
		contact := api.NewContact(userChatID, fromMessage.Contact.PhoneNumber, fromMessage.Contact.FirstName)
		contact.LastName = fromMessage.Contact.LastName
		sentMessage, err = bot.Send(contact)

	default:
		return nil, fmt.Errorf("unsupported message type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to forward message: %w", err)
	}

	return &sentMessage, nil
}

// HandleMediaGroup processes media group messages
func (ms *MessageService) HandleMediaGroup(bot *api.BotAPI, message *api.Message, groupChatID int64, messageThreadID int) {
	if message.MediaGroupID == "" {
		return
	}

	// Store the media group message
	mediaGroupMsg := &models.MediaGroupMessage{
		MediaGroupID: message.MediaGroupID,
		ChatID:       message.Chat.ID,
		MessageID:    message.MessageID,
		CaptionHTML:  message.Caption,
	}

	if err := ms.db.CreateMediaGroupMessage(mediaGroupMsg); err != nil {
		log.Printf("Error storing media group message: %v", err)
		return
	}

	// Schedule delayed sending after 5 seconds
	go func() {
		time.Sleep(5 * time.Second)
		ms.processMediaGroup(bot, message.MediaGroupID, groupChatID, messageThreadID)
	}()
}

// processMediaGroup processes and sends all messages in a media group
func (ms *MessageService) processMediaGroup(bot *api.BotAPI, mediaGroupID string, groupChatID int64, messageThreadID int) {
	messages, err := ms.db.GetMediaGroupMessages(mediaGroupID)
	if err != nil {
		log.Printf("Error getting media group messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	// Forward each message individually
	for _, msg := range messages {
		// Since we can't easily extract media from stored messages,
		// we'll forward them individually with thread ID
		forwardConfig := api.NewForward(groupChatID, msg.ChatID, msg.MessageID)

		_, err := bot.Request(forwardConfig)
		if err != nil {
			log.Printf("Error forwarding media group message: %v", err)
			continue
		}
	}

	// Clean up the stored media group messages
	if err := ms.db.DeleteMediaGroupMessages(mediaGroupID); err != nil {
		log.Printf("Error cleaning up media group messages: %v", err)
	}
}

// RecordUserMessage records a user message for rate limiting
func (ms *MessageService) RecordUserMessage(userID int64, chatID int64, messageID int) error {
	userMessage := &models.UserMessage{
		UserID:    userID,
		ChatID:    chatID,
		MessageID: messageID,
		SentAt:    time.Now(),
	}
	return ms.db.CreateUserMessage(userMessage)
}

// SendContactCard sends a user's contact information to the admin group
func (ms *MessageService) SendContactCard(bot *api.BotAPI, user *models.User, groupChatID int64, messageThreadID int) (*api.Message, error) {
	// Create contact card text
	var cardText strings.Builder
	cardText.WriteString("👤 <b>用户信息</b>\n\n")
	cardText.WriteString(fmt.Sprintf("🆔 <b>用户ID:</b> <code>%d</code>\n", user.UserID))
	cardText.WriteString(fmt.Sprintf("👤 <b>姓名:</b> %s", user.FirstName))

	if user.LastName != "" {
		cardText.WriteString(" " + user.LastName)
	}
	cardText.WriteString("\n")

	if user.Username != "" {
		cardText.WriteString(fmt.Sprintf("📱 <b>用户名:</b> @%s\n", user.Username))
	}

	if user.IsPremium {
		cardText.WriteString("⭐ <b>Telegram Premium</b>\n")
	}

	cardText.WriteString(fmt.Sprintf("📅 <b>首次联系:</b> %s\n", user.CreatedAt.Format("2006-01-02 15:04:05")))
	cardText.WriteString(fmt.Sprintf("🔄 <b>最后活跃:</b> %s", user.UpdatedAt.Format("2006-01-02 15:04:05")))

	// Send the contact card
	msg := api.NewMessage(groupChatID, cardText.String())
	msg.MessageThreadID = messageThreadID
	msg.ParseMode = api.ModeHTML

	sentMessage, err := bot.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send contact card: %w", err)
	}

	return &sentMessage, nil
}