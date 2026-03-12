package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"telegram-communication-bot/internal/database"
	dbmodels "telegram-communication-bot/internal/models"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type MessageService struct {
	db                  *database.DB
	mediaGroupScheduled sync.Map
}

func NewMessageService(db *database.DB) *MessageService {
	return &MessageService{
		db: db,
	}
}

func (ms *MessageService) CreateMessageMap(userChatMessageID int, groupChatMessageID int, userID int64) error {
	messageMap := &dbmodels.MessageMap{
		UserChatMessageID:  userChatMessageID,
		GroupChatMessageID: groupChatMessageID,
		UserID:             userID,
	}
	return ms.db.CreateMessageMap(messageMap)
}

func (ms *MessageService) GetUserMessageFromGroup(groupChatMessageID int) (*dbmodels.MessageMap, error) {
	return ms.db.GetMessageMapByGroupMessage(groupChatMessageID)
}

func (ms *MessageService) GetGroupMessageFromUser(userChatMessageID int, userID int64) (*dbmodels.MessageMap, error) {
	return ms.db.GetMessageMapByUserMessage(userChatMessageID, userID)
}

// copyMessage copies a message to a target chat, optionally into a forum thread.
func (ms *MessageService) copyMessage(ctx context.Context, b *tgbot.Bot, fromMessage *models.Message, toChatID int64, threadID int) (*models.Message, error) {
	switch {
	case fromMessage.Text != "":
		return b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Text:            fromMessage.Text,
			Entities:        fromMessage.Entities,
		})

	case len(fromMessage.Photo) > 0:
		largest := fromMessage.Photo[len(fromMessage.Photo)-1]
		return b.SendPhoto(ctx, &tgbot.SendPhotoParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Photo:           &models.InputFileString{Data: largest.FileID},
			Caption:         fromMessage.Caption,
			CaptionEntities: fromMessage.CaptionEntities,
		})

	case fromMessage.Document != nil:
		return b.SendDocument(ctx, &tgbot.SendDocumentParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Document:        &models.InputFileString{Data: fromMessage.Document.FileID},
			Caption:         fromMessage.Caption,
			CaptionEntities: fromMessage.CaptionEntities,
		})

	case fromMessage.Video != nil:
		return b.SendVideo(ctx, &tgbot.SendVideoParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Video:           &models.InputFileString{Data: fromMessage.Video.FileID},
			Caption:         fromMessage.Caption,
			CaptionEntities: fromMessage.CaptionEntities,
		})

	case fromMessage.Audio != nil:
		return b.SendAudio(ctx, &tgbot.SendAudioParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Audio:           &models.InputFileString{Data: fromMessage.Audio.FileID},
			Caption:         fromMessage.Caption,
			CaptionEntities: fromMessage.CaptionEntities,
		})

	case fromMessage.Voice != nil:
		return b.SendVoice(ctx, &tgbot.SendVoiceParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Voice:           &models.InputFileString{Data: fromMessage.Voice.FileID},
			Caption:         fromMessage.Caption,
			CaptionEntities: fromMessage.CaptionEntities,
		})

	case fromMessage.VideoNote != nil:
		return b.SendVideoNote(ctx, &tgbot.SendVideoNoteParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			VideoNote:       &models.InputFileString{Data: fromMessage.VideoNote.FileID},
			Length:          fromMessage.VideoNote.Length,
		})

	case fromMessage.Sticker != nil:
		return b.SendSticker(ctx, &tgbot.SendStickerParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Sticker:         &models.InputFileString{Data: fromMessage.Sticker.FileID},
		})

	case fromMessage.Animation != nil:
		return b.SendAnimation(ctx, &tgbot.SendAnimationParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Animation:       &models.InputFileString{Data: fromMessage.Animation.FileID},
			Caption:         fromMessage.Caption,
			CaptionEntities: fromMessage.CaptionEntities,
		})

	case fromMessage.Location != nil:
		return b.SendLocation(ctx, &tgbot.SendLocationParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			Latitude:        fromMessage.Location.Latitude,
			Longitude:       fromMessage.Location.Longitude,
		})

	case fromMessage.Contact != nil:
		return b.SendContact(ctx, &tgbot.SendContactParams{
			ChatID:          toChatID,
			MessageThreadID: threadID,
			PhoneNumber:     fromMessage.Contact.PhoneNumber,
			FirstName:       fromMessage.Contact.FirstName,
			LastName:        fromMessage.Contact.LastName,
		})

	default:
		return nil, fmt.Errorf("unsupported message type")
	}
}

func (ms *MessageService) ForwardMessageToGroup(ctx context.Context, b *tgbot.Bot, fromMessage *models.Message, groupChatID int64, messageThreadID int) (*models.Message, error) {
	return ms.copyMessage(ctx, b, fromMessage, groupChatID, messageThreadID)
}

func (ms *MessageService) ForwardMessageToUser(ctx context.Context, b *tgbot.Bot, fromMessage *models.Message, userChatID int64) (*models.Message, error) {
	return ms.copyMessage(ctx, b, fromMessage, userChatID, 0)
}

// HandleMediaGroup processes media group messages with deduplication.
func (ms *MessageService) HandleMediaGroup(ctx context.Context, b *tgbot.Bot, message *models.Message, groupChatID int64, messageThreadID int) {
	if message.MediaGroupID == "" {
		return
	}

	mediaGroupMsg := &dbmodels.MediaGroupMessage{
		MediaGroupID: message.MediaGroupID,
		ChatID:       message.Chat.ID,
		MessageID:    message.ID,
		CaptionHTML:  message.Caption,
	}

	if err := ms.db.CreateMediaGroupMessage(mediaGroupMsg); err != nil {
		log.Printf("Error storing media group message: %v", err)
		return
	}

	if _, alreadyScheduled := ms.mediaGroupScheduled.LoadOrStore(message.MediaGroupID, true); !alreadyScheduled {
		go func() {
			time.Sleep(3 * time.Second)
			defer ms.mediaGroupScheduled.Delete(message.MediaGroupID)
			ms.processMediaGroup(context.Background(), b, message.MediaGroupID, groupChatID, messageThreadID)
		}()
	}
}

func (ms *MessageService) processMediaGroup(ctx context.Context, b *tgbot.Bot, mediaGroupID string, groupChatID int64, messageThreadID int) {
	messages, err := ms.db.GetMediaGroupMessages(mediaGroupID)
	if err != nil {
		log.Printf("Error getting media group messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	for _, msg := range messages {
		_, err := b.ForwardMessage(ctx, &tgbot.ForwardMessageParams{
			ChatID:          groupChatID,
			FromChatID:      msg.ChatID,
			MessageID:       msg.MessageID,
			MessageThreadID: messageThreadID,
		})
		if err != nil {
			log.Printf("Error forwarding media group message: %v", err)
			continue
		}
	}

	if err := ms.db.DeleteMediaGroupMessages(mediaGroupID); err != nil {
		log.Printf("Error cleaning up media group messages: %v", err)
	}
}

func (ms *MessageService) RecordUserMessage(userID int64, chatID int64, messageID int) error {
	userMessage := &dbmodels.UserMessage{
		UserID:    userID,
		ChatID:    chatID,
		MessageID: messageID,
		SentAt:    time.Now(),
	}
	return ms.db.CreateUserMessage(userMessage)
}

func (ms *MessageService) SendUserInfoMessage(ctx context.Context, b *tgbot.Bot, user *dbmodels.User, groupChatID int64, messageThreadID int) (*models.Message, error) {
	var infoText strings.Builder
	infoText.WriteString("📋 <b>用户信息</b>\n\n")

	if user.Username != "" {
		infoText.WriteString(fmt.Sprintf("👤 用户名: @%s\n", user.Username))
	} else {
		infoText.WriteString("👤 用户名: <i>无</i>\n")
	}
	infoText.WriteString(fmt.Sprintf("🆔 用户ID: <code>%d</code>", user.UserID))

	return b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:          groupChatID,
		MessageThreadID: messageThreadID,
		Text:            infoText.String(),
		ParseMode:       models.ParseModeHTML,
	})
}

func (ms *MessageService) SendContactCard(ctx context.Context, b *tgbot.Bot, user *dbmodels.User, groupChatID int64, messageThreadID int) (*models.Message, error) {
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

	return b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:          groupChatID,
		MessageThreadID: messageThreadID,
		Text:            cardText.String(),
		ParseMode:       models.ParseModeHTML,
	})
}

func (ms *MessageService) PinMessage(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) error {
	_, err := b.PinChatMessage(ctx, &tgbot.PinChatMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	return err
}
