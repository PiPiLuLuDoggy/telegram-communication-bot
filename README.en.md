# Telegram Communication Bot

A complete Telegram customer service bot that supports bidirectional message forwarding between users and administrators, CAPTCHA verification, forum topic management, and more.

**Language**: [中文](README.md) | **English**

## Features

- 🔐 **Image CAPTCHA**: Prevents bot abuse
- 💬 **Bidirectional Message Forwarding**: User messages automatically forwarded to admin group
- 🎯 **Forum Topic Management**: Create dedicated topics for each user
- 🛡️ **Anti-Abuse Mechanism**: Message rate limiting
- 📡 **Admin Functions**: Broadcast messages, user statistics, conversation management
- 🚫 **User Ban System**: Supports permanent and temporary bans

## Quick Deployment

### 1. Prerequisites

#### Create Telegram Bot
1. Contact [@BotFather](https://t.me/botfather)
2. Send `/newbot` to create a bot
3. Get the Bot Token

#### Get Admin Group ID
1. Create a group and enable forum functionality
2. Add the bot to the group and make it an admin
3. Send a message to the group, use [@userinfobot](https://t.me/userinfobot) to get the group ID (negative number)

#### Get Admin User ID
Contact [@userinfobot](https://t.me/userinfobot) to get your user ID

### 2. Deploy with Docker Compose

```bash
# 1. Clone the project (or download the code)
git clone https://github.com/PiPiLuLuDoggy/telegram-communication-bot.git
cd telegram-communication-bot

# 2. Copy configuration file
cp .env.example .env

# 3. Edit configuration file
nano .env
```

### 3. Configure Environment Variables

Edit the `.env` file with the following required information:

```bash
# Required Configuration
BOT_TOKEN=your_bot_token_here
ADMIN_GROUP_ID=-1001234567890  # Admin group ID (negative number)
ADMIN_USER_IDS=123456789,987654321  # Admin user IDs (comma separated)

# Optional Configuration
WELCOME_MESSAGE=Welcome to our customer service bot!
DISABLE_CAPTCHA=false  # Whether to disable CAPTCHA
MESSAGE_INTERVAL=5     # User message interval (seconds)
```

### 4. Start Service

```bash
# Start the bot
docker-compose up -d

# View logs
docker-compose logs -f telegram-bot
```

## Admin Commands

| Command | Description | Example |
|---------|-------------|---------|
| `/start` | Check bot status | `/start` |
| `/clear <user_id>` | Clear user conversation | `/clear 123456789` |
| `/broadcast` | Broadcast message | Reply to a message then send `/broadcast` |
| `/stats` | View statistics | `/stats` |

## Usage Instructions

### User Side
1. Search and start your bot
2. Send `/start` to begin
3. Complete CAPTCHA verification (if enabled)
4. Send messages directly to the bot

### Admin Side
1. View user messages in the admin group (one topic per user)
2. Reply directly in the topic to users
3. Use admin commands to manage users and system

## Configuration Options

| Environment Variable | Description | Default Value | Required |
|---------------------|-------------|---------------|----------|
| `BOT_TOKEN` | Bot Token | - | ✅ |
| `ADMIN_GROUP_ID` | Admin Group ID (negative) | - | ✅ |
| `ADMIN_USER_IDS` | Admin User IDs (comma separated) | - | ✅ |
| `APP_NAME` | Application Name | TelegramCommunicationBot | ❌ |
| `WELCOME_MESSAGE` | User Welcome Message | Default Chinese Welcome | ❌ |
| `DELETE_TOPIC_AS_FOREVER_BAN` | Permanently ban user when deleting topic | false | ❌ |
| `DELETE_USER_MESSAGE_ON_CLEAR_CMD` | Delete user messages on clear command | false | ❌ |
| `DISABLE_CAPTCHA` | Disable CAPTCHA verification | false | ❌ |
| `MESSAGE_INTERVAL` | User message sending interval (seconds) | 5 | ❌ |
| `DATABASE_PATH` | Database file path | ./data/bot.db | ❌ |
| `PORT` | Service port (Webhook mode) | 8080 | ❌ |
| `WEBHOOK_URL` | Webhook URL (optional) | - | ❌ |
| `DEBUG` | Debug mode (enable detailed logging) | true | ❌ |

## FAQ

**Q: Bot not responding?**
A: Check if the Bot Token is correct, view logs: `docker-compose logs telegram-bot`

**Q: Cannot create forum topics?**
A: Ensure: 1) Group has forum functionality enabled 2) Bot has admin permissions 3) Group ID is correct (negative number)

**Q: CAPTCHA not displaying?**
A: CAPTCHA images are included in the image. If issues persist, set `DISABLE_CAPTCHA=true`

**Q: How to stop the bot?**
```bash
docker-compose down
```

**Q: How to backup data?**
```bash
docker cp telegram-communication-bot:/app/data/bot.db ./backup.db
```

## Update Bot

```bash
# Stop service
docker-compose down

# Pull latest code
git pull

# Rebuild and start
docker-compose up -d --build
```