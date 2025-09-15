module telegram-communication-bot

go 1.23

toolchain go1.23.4

require (
	github.com/TGlimmer/telegram-bot-api v0.0.0-20240101000000-000000000000
	github.com/joho/godotenv v1.5.1
	github.com/robfig/cron/v3 v3.0.1
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.5
)

replace github.com/TGlimmer/telegram-bot-api => /root/code/telegram-bot-api

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
)
