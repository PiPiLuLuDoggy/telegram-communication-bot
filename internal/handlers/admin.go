package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	dbmodels "telegram-communication-bot/internal/models"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handlers) handleClearCommand(ctx context.Context, message *models.Message, args string) {
	chatID := message.Chat.ID

	if args == "" {
		h.sendMessage(ctx, chatID, "❌ 请提供用户ID\n用法: /clear <user_id>")
		return
	}

	userID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		h.sendMessage(ctx, chatID, "❌ 无效的用户ID")
		return
	}

	user, err := h.db.GetUser(userID)
	if err != nil {
		h.sendMessage(ctx, chatID, "❌ 用户不存在")
		return
	}

	if h.config.DeleteTopicAsForeverBan && user.MessageThreadID != 0 {
		if err := h.forumService.DeleteForumTopic(ctx, user.MessageThreadID); err != nil {
			log.Printf("Error deleting forum topic: %v", err)
		}
		banStatus := &dbmodels.BanStatus{
			UserID:   userID,
			IsBanned: true,
			Reason:   "Forum topic deleted by admin",
		}
		if err := h.db.CreateOrUpdateBanStatus(banStatus); err != nil {
			log.Printf("Error banning user: %v", err)
		}
	} else {
		if user.MessageThreadID != 0 {
			if err := h.forumService.CloseForumTopic(ctx, user.MessageThreadID); err != nil {
				log.Printf("Error closing forum topic: %v", err)
			}
		}
	}

	if h.config.DeleteUserMessageOnClearCmd {
		log.Printf("Would delete messages for user %d", userID)
	}

	action := "已关闭"
	if h.config.DeleteTopicAsForeverBan {
		action = "已删除并永久禁止"
	}

	h.sendMessage(ctx, chatID, fmt.Sprintf("✅ 用户 %d (%s) 的对话%s", userID, user.FirstName, action))
}

func (h *Handlers) handleBroadcastCommand(ctx context.Context, message *models.Message) {
	chatID := message.Chat.ID

	if message.ReplyToMessage == nil {
		h.sendMessage(ctx, chatID, "❌ 请回复一条消息以进行广播")
		return
	}

	users, err := h.db.GetAllUsers()
	if err != nil {
		h.sendMessage(ctx, chatID, "❌ 获取用户列表失败")
		log.Printf("Error getting users for broadcast: %v", err)
		return
	}

	if len(users) == 0 {
		h.sendMessage(ctx, chatID, "❌ 没有用户可以广播")
		return
	}

	h.sendMessage(ctx, chatID, fmt.Sprintf("📡 开始广播消息给 %d 个用户...", len(users)))

	go h.performBroadcast(context.Background(), message.ReplyToMessage, users, chatID)
}

func (h *Handlers) performBroadcast(ctx context.Context, broadcastMsg *models.Message, users []dbmodels.User, adminChatID int64) {
	successCount := 0
	failCount := 0

	for _, user := range users {
		if h.db.IsUserBanned(user.UserID) {
			failCount++
			continue
		}

		_, err := h.messageService.ForwardMessageToUser(ctx, h.bot, broadcastMsg, user.UserID)
		if err != nil {
			log.Printf("Error broadcasting to user %d: %v", user.UserID, err)
			failCount++
		} else {
			successCount++
		}

		time.Sleep(50 * time.Millisecond)
	}

	summary := fmt.Sprintf("📡 广播完成!\n✅ 成功: %d\n❌ 失败: %d", successCount, failCount)
	h.sendMessage(ctx, adminChatID, summary)
}

func (h *Handlers) handleStatsCommand(ctx context.Context, message *models.Message) {
	chatID := message.Chat.ID

	totalUsers, err := h.db.CountUsers()
	if err != nil {
		h.sendMessage(ctx, chatID, "❌ 获取统计信息失败")
		log.Printf("Error counting users: %v", err)
		return
	}

	bannedUsers, err := h.db.CountBannedUsers()
	if err != nil {
		log.Printf("Error counting banned users: %v", err)
	}

	premiumUsers, err := h.db.CountPremiumUsers()
	if err != nil {
		log.Printf("Error counting premium users: %v", err)
	}

	activeUsers := totalUsers - bannedUsers

	activeTopics, err := h.forumService.GetAllActiveTopics()
	if err != nil {
		log.Printf("Error getting active topics: %v", err)
	}

	statsText := fmt.Sprintf(`📊 <b>机器人统计</b>

👥 <b>用户统计:</b>
• 总用户数: %d
• 活跃用户: %d
• 被禁用户: %d
• Premium用户: %d

💬 <b>对话统计:</b>
• 活跃对话: %d

🔧 <b>系统设置:</b>
• 消息间隔: %d秒
• 删除对话永久禁止: %s
• 清除时删除消息: %s`,
		totalUsers,
		activeUsers,
		bannedUsers,
		premiumUsers,
		len(activeTopics),
		h.config.MessageInterval,
		h.getBoolString(h.config.DeleteTopicAsForeverBan),
		h.getBoolString(h.config.DeleteUserMessageOnClearCmd))

	h.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      statsText,
		ParseMode: models.ParseModeHTML,
	})
}

func (h *Handlers) getBoolString(value bool) string {
	if value {
		return "启用"
	}
	return "禁用"
}

func (h *Handlers) banUser(userID int64, reason string) error {
	banStatus := &dbmodels.BanStatus{
		UserID:   userID,
		IsBanned: true,
		Reason:   reason,
	}
	return h.db.CreateOrUpdateBanStatus(banStatus)
}

func (h *Handlers) unbanUser(userID int64) error {
	banStatus := &dbmodels.BanStatus{
		UserID:   userID,
		IsBanned: false,
		Reason:   "",
	}
	return h.db.CreateOrUpdateBanStatus(banStatus)
}

func (h *Handlers) getUserInfo(user *dbmodels.User) string {
	var info strings.Builder

	info.WriteString(fmt.Sprintf("👤 <b>用户信息</b>\n\n"))
	info.WriteString(fmt.Sprintf("🆔 <b>ID:</b> <code>%d</code>\n", user.UserID))
	info.WriteString(fmt.Sprintf("📝 <b>姓名:</b> %s", user.FirstName))

	if user.LastName != "" {
		info.WriteString(" " + user.LastName)
	}
	info.WriteString("\n")

	if user.Username != "" {
		info.WriteString(fmt.Sprintf("👤 <b>用户名:</b> @%s\n", user.Username))
	}

	if user.IsPremium {
		info.WriteString("⭐ <b>Premium用户</b>\n")
	}

	info.WriteString(fmt.Sprintf("📅 <b>创建时间:</b> %s\n", user.CreatedAt.Format("2006-01-02 15:04:05")))
	info.WriteString(fmt.Sprintf("🔄 <b>更新时间:</b> %s\n", user.UpdatedAt.Format("2006-01-02 15:04:05")))

	if user.MessageThreadID != 0 {
		info.WriteString(fmt.Sprintf("💬 <b>对话ID:</b> %d\n", user.MessageThreadID))
	}

	if h.db.IsUserBanned(user.UserID) {
		info.WriteString("🚫 <b>状态:</b> 已禁止\n")
	} else {
		info.WriteString("✅ <b>状态:</b> 正常\n")
	}

	return info.String()
}

func (h *Handlers) handleResetCommand(ctx context.Context, message *models.Message, args string) {
	chatID := message.Chat.ID

	if args == "" {
		h.sendMessage(ctx, chatID, "❌ 请提供用户ID\n用法: /reset <user_id>")
		return
	}

	userID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		h.sendMessage(ctx, chatID, "❌ 无效的用户ID")
		return
	}

	user, err := h.db.GetUser(userID)
	if err != nil {
		h.sendMessage(ctx, chatID, "❌ 用户不存在")
		return
	}

	if err := h.forumService.ResetUserThreadID(userID); err != nil {
		h.sendMessage(ctx, chatID, fmt.Sprintf("❌ 重置用户 %d 的对话ID失败: %v", userID, err))
		log.Printf("Error resetting thread ID for user %d: %v", userID, err)
		return
	}

	h.sendMessage(ctx, chatID, fmt.Sprintf("✅ 已重置用户 %d (%s) 的对话ID\n用户下次发消息时将创建新的对话", userID, user.FirstName))
}
