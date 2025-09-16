package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"telegram-communication-bot/internal/models"

	api "github.com/OvyFlash/telegram-bot-api"
)

// handleClearCommand handles the /clear command for admins
func (h *Handlers) handleClearCommand(message *api.Message, args string) {
	chatID := message.Chat.ID

	if args == "" {
		h.sendMessage(chatID, "❌ 请提供用户ID\n用法: /clear <user_id>")
		return
	}

	userID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		h.sendMessage(chatID, "❌ 无效的用户ID")
		return
	}

	// Get user info
	user, err := h.db.GetUser(userID)
	if err != nil {
		h.sendMessage(chatID, "❌ 用户不存在")
		return
	}

	// Delete forum topic if configured
	if h.config.DeleteTopicAsForeverBan && user.MessageThreadID != 0 {
		if err := h.forumService.DeleteForumTopic(user.MessageThreadID); err != nil {
			log.Printf("Error deleting forum topic: %v", err)
		}

		// Ban user permanently if configured
		banStatus := &models.BanStatus{
			UserID:   userID,
			IsBanned: true,
			Reason:   "Forum topic deleted by admin",
		}
		if err := h.db.CreateOrUpdateBanStatus(banStatus); err != nil {
			log.Printf("Error banning user: %v", err)
		}
	} else {
		// Just close the forum topic
		if user.MessageThreadID != 0 {
			if err := h.forumService.CloseForumTopic(user.MessageThreadID); err != nil {
				log.Printf("Error closing forum topic: %v", err)
			}
		}
	}

	// Delete user messages if configured
	if h.config.DeleteUserMessageOnClearCmd {
		// This would require implementing message deletion functionality
		// For now, we'll just log it
		log.Printf("Would delete messages for user %d", userID)
	}

	action := "已关闭"
	if h.config.DeleteTopicAsForeverBan {
		action = "已删除并永久禁止"
	}

	h.sendMessage(chatID, fmt.Sprintf("✅ 用户 %d (%s) 的对话%s", userID, user.FirstName, action))
}

// handleBroadcastCommand handles the /broadcast command
func (h *Handlers) handleBroadcastCommand(message *api.Message) {
	chatID := message.Chat.ID

	// Check if this is a reply to a message
	if message.ReplyToMessage == nil {
		h.sendMessage(chatID, "❌ 请回复一条消息以进行广播")
		return
	}

	// Get all users
	users, err := h.db.GetAllUsers()
	if err != nil {
		h.sendMessage(chatID, "❌ 获取用户列表失败")
		log.Printf("Error getting users for broadcast: %v", err)
		return
	}

	if len(users) == 0 {
		h.sendMessage(chatID, "❌ 没有用户可以广播")
		return
	}

	// Start broadcasting
	h.sendMessage(chatID, fmt.Sprintf("📡 开始广播消息给 %d 个用户...", len(users)))

	go h.performBroadcast(message.ReplyToMessage, users, chatID)
}

// performBroadcast performs the actual broadcast operation
func (h *Handlers) performBroadcast(broadcastMsg *api.Message, users []models.User, adminChatID int64) {
	successCount := 0
	failCount := 0

	for _, user := range users {
		// Skip banned users
		if h.db.IsUserBanned(user.UserID) {
			failCount++
			continue
		}

		// Forward the message to each user
		_, err := h.messageService.ForwardMessageToUser(h.bot, broadcastMsg, user.UserID)
		if err != nil {
			log.Printf("Error broadcasting to user %d: %v", user.UserID, err)
			failCount++
		} else {
			successCount++
		}

		// Small delay to avoid rate limiting
		// time.Sleep(50 * time.Millisecond) // Uncomment if needed
	}

	// Send summary to admin
	summary := fmt.Sprintf("📡 广播完成!\n✅ 成功: %d\n❌ 失败: %d", successCount, failCount)
	h.sendMessage(adminChatID, summary)
}

// handleStatsCommand handles the /stats command
func (h *Handlers) handleStatsCommand(message *api.Message) {
	chatID := message.Chat.ID

	// Get user statistics
	users, err := h.db.GetAllUsers()
	if err != nil {
		h.sendMessage(chatID, "❌ 获取统计信息失败")
		log.Printf("Error getting stats: %v", err)
		return
	}

	totalUsers := len(users)
	activeUsers := 0
	premiumUsers := 0
	bannedUsers := 0

	for _, user := range users {
		if h.db.IsUserBanned(user.UserID) {
			bannedUsers++
		} else {
			activeUsers++
		}

		if user.IsPremium {
			premiumUsers++
		}
	}

	// Get active topics count
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

	msg := api.NewMessage(chatID, statsText)
	msg.ParseMode = api.ModeHTML
	h.bot.Send(msg)
}


// getBoolString returns a Chinese string representation of a boolean
func (h *Handlers) getBoolString(value bool) string {
	if value {
		return "启用"
	}
	return "禁用"
}

// Additional admin helper methods

// banUser bans a user
func (h *Handlers) banUser(userID int64, reason string) error {
	banStatus := &models.BanStatus{
		UserID:   userID,
		IsBanned: true,
		Reason:   reason,
	}
	return h.db.CreateOrUpdateBanStatus(banStatus)
}

// unbanUser unbans a user
func (h *Handlers) unbanUser(userID int64) error {
	banStatus := &models.BanStatus{
		UserID:   userID,
		IsBanned: false,
		Reason:   "",
	}
	return h.db.CreateOrUpdateBanStatus(banStatus)
}

// getUserInfo formats user information for display
func (h *Handlers) getUserInfo(user *models.User) string {
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

	// Check ban status
	if h.db.IsUserBanned(user.UserID) {
		info.WriteString("🚫 <b>状态:</b> 已禁止\n")
	} else {
		info.WriteString("✅ <b>状态:</b> 正常\n")
	}

	return info.String()
}

// handleResetCommand handles the /reset command to reset user's thread ID
func (h *Handlers) handleResetCommand(message *api.Message, args string) {
	chatID := message.Chat.ID

	if args == "" {
		h.sendMessage(chatID, "❌ 请提供用户ID\n用法: /reset <user_id>")
		return
	}

	userID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		h.sendMessage(chatID, "❌ 无效的用户ID")
		return
	}

	// Get user info
	user, err := h.db.GetUser(userID)
	if err != nil {
		h.sendMessage(chatID, "❌ 用户不存在")
		return
	}

	// Reset user's thread ID
	if err := h.forumService.ResetUserThreadID(userID); err != nil {
		h.sendMessage(chatID, fmt.Sprintf("❌ 重置用户 %d 的对话ID失败: %v", userID, err))
		log.Printf("Error resetting thread ID for user %d: %v", userID, err)
		return
	}

	h.sendMessage(chatID, fmt.Sprintf("✅ 已重置用户 %d (%s) 的对话ID\n用户下次发消息时将创建新的对话", userID, user.FirstName))
}