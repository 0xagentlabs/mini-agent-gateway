package channel

import (
	"log"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/0xcevin/mini-agent-gateway/pkg/gateway"
)

// TelegramAdapter Telegram 频道适配器
type TelegramAdapter struct {
	bot     *tgbotapi.BotAPI
	gateway *gateway.Gateway
	updates tgbotapi.UpdatesChannel
}

// NewTelegramAdapter 创建 Telegram 适配器
func NewTelegramAdapter(token string, gw *gateway.Gateway) *TelegramAdapter {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("创建 Telegram Bot 失败: %v", err)
	}

	bot.Debug = false
	log.Printf("已授权 Telegram Bot: %s", bot.Self.UserName)

	return &TelegramAdapter{
		bot:     bot,
		gateway: gw,
	}
}

// Start 开始接收消息
func (t *TelegramAdapter) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	t.updates = t.bot.GetUpdatesChan(u)

	for update := range t.updates {
		if update.Message == nil {
			continue
		}

		msg := gateway.Message{
			ID:        string(rune(update.Message.MessageID)),
			UserID:    string(rune(update.Message.From.ID)),
			ChatID:    string(rune(update.Message.Chat.ID)),
			Text:      update.Message.Text,
			Channel:   "telegram",
			Timestamp: time.Now(),
		}

		// 发送到网关处理
		go t.gateway.HandleMessage(msg)
		
		// 立即回复处理中（可选）
		if update.Message.Text != "" {
			log.Printf("[Telegram] 收到消息 from @%s: %s", 
				update.Message.From.UserName, update.Message.Text)
		}
	}

	return nil
}

// SendMessage 发送消息到 Telegram
func (t *TelegramAdapter) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := t.bot.Send(msg)
	return err
}

// Stop 停止接收
func (t *TelegramAdapter) Stop() {
	t.bot.StopReceivingUpdates()
}
