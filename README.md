# Telegram Communication Bot

一个完整的 Telegram 客服机器人，支持用户与管理员之间的双向消息转发、论坛话题管理等功能。

**语言选择**: **中文** | [English](README.en.md)

## 功能特性

- 💬 **双向消息转发**：用户消息自动转发到管理群组
- 🎯 **论坛话题管理**：为每个用户创建专属话题，自动显示用户信息
- 🛡️ **防滥用机制**：消息频率限制，防止用户刷屏
- 📡 **管理功能**：广播消息、用户统计、对话管理、话题重置
- 🚫 **用户封禁**：支持永久和临时封禁
- 🔄 **自动恢复**：删除话题后用户重新发消息自动创建新话题
- 👤 **用户信息展示**：新话题自动显示用户名和ID信息

## 快速部署

### 1. 准备工作

#### 创建 Telegram Bot
1. 联系 [@BotFather](https://t.me/botfather)
2. 发送 `/newbot` 创建机器人
3. 获取 Bot Token

#### 获取管理群组 ID
1. 创建一个群组，开启论坛功能
2. 将机器人添加到群组并设为管理员
3. 发送消息到群组，使用 [@userinfobot](https://t.me/userinfobot) 获取群组 ID（负数）

#### 获取管理员用户 ID
联系 [@userinfobot](https://t.me/userinfobot) 获取您的用户 ID

### 2. 使用 Docker Compose 部署

```bash
# 1. 克隆项目（或下载代码）
git clone <your-repo>
cd telegram-communication-bot

# 2. 复制配置文件
cp .env.example .env

# 3. 编辑配置文件
nano .env
```

### 3. 配置环境变量

编辑 `.env` 文件，填入以下必要信息：

```bash
# 必填配置
BOT_TOKEN=你的机器人Token
ADMIN_GROUP_ID=-1001234567890  # 管理群组ID（负数）
ADMIN_USER_IDS=123456789,987654321  # 管理员用户ID（逗号分隔）

# 可选配置
WELCOME_MESSAGE=欢迎使用我们的客服机器人！
MESSAGE_INTERVAL=5     # 用户发送消息间隔（秒）
```

### 4. 启动服务

```bash
# 启动机器人
docker-compose up -d

# 查看日志
docker-compose logs -f telegram-bot
```

## 管理员命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/start` | 检查机器人状态 | `/start` |
| `/clear <用户ID>` | 清理用户对话 | `/clear 123456789` |
| `/reset <用户ID>` | 重置用户话题ID（修复删除话题问题） | `/reset 123456789` |
| `/broadcast` | 广播消息 | 回复消息后发送 `/broadcast` |
| `/stats` | 查看统计信息 | `/stats` |

## 使用说明

### 用户端
1. 搜索并启动您的机器人
2. 发送 `/start` 开始使用
3. 直接发送消息给机器人

### 管理端
1. 在管理群组中查看用户消息（每个用户一个话题）
2. 每个新话题会自动显示用户信息：
   ```
   📋 用户信息

   👤 用户名: @username
   🆔 用户ID: 123456789
   ```
3. 直接在话题中回复用户
4. 使用管理命令管理用户和系统
5. 如果误删话题，用户重新发消息会自动创建新话题

## 配置选项

| 环境变量 | 说明 | 默认值 | 必须 |
|----------|------|--------|------|
| `BOT_TOKEN` | 机器人Token | - | ✅ |
| `ADMIN_GROUP_ID` | 管理群组ID（负数） | - | ✅ |
| `ADMIN_USER_IDS` | 管理员用户ID（逗号分隔） | - | ✅ |
| `APP_NAME` | 应用名称 | TelegramCommunicationBot | ❌ |
| `WELCOME_MESSAGE` | 用户欢迎消息 | 默认中文欢迎词 | ❌ |
| `DELETE_TOPIC_AS_FOREVER_BAN` | 删除话题时永久封禁用户 | false | ❌ |
| `DELETE_USER_MESSAGE_ON_CLEAR_CMD` | 清理命令时删除用户消息 | false | ❌ |
| `MESSAGE_INTERVAL` | 用户消息发送间隔（秒） | 5 | ❌ |
| `DATABASE_PATH` | 数据库文件路径 | ./data/bot.db | ❌ |
| `PORT` | 服务端口（Webhook模式） | 8090 | ❌ |
| `WEBHOOK_URL` | Webhook地址（可选） | - | ❌ |
| `DEBUG` | 调试模式（启用详细日志） | true | ❌ |

## 常见问题

**Q: 机器人无响应？**
A: 检查 Bot Token 是否正确，查看日志 `docker-compose logs telegram-bot`

**Q: 无法创建论坛话题？**
A: 确保：1）群组开启了论坛功能 2）机器人有管理员权限 3）群组ID正确（负数）

**Q: 误删了用户话题怎么办？**
A: 用户重新发消息时会自动创建新话题。或使用 `/reset <用户ID>` 命令手动重置

**Q: 用户发消息后管理员收不到？**
A: 检查话题是否被删除，使用 `/reset <用户ID>` 重置用户状态


**Q: 如何停止机器人？**
```bash
docker-compose down
```

**Q: 如何备份数据？**
```bash
docker cp telegram-communication-bot:/app/data/bot.db ./backup.db
```

## 更新机器人

```bash
# 停止服务
docker-compose down

# 拉取最新代码
git pull

# 重新构建并启动
docker-compose up -d --build
```