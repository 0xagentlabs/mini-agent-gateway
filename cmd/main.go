package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/xingyue/mini-agent-gateway/pkg/channel"
	"github.com/xingyue/mini-agent-gateway/pkg/gateway"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，使用环境变量")
	}

	// 创建网关
	gw := gateway.New()

	// 创建 Telegram 频道适配器
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("请设置 TELEGRAM_BOT_TOKEN 环境变量")
	}

	telegramAdapter := channel.NewTelegramAdapter(telegramToken, gw)
	
	// 启动 Telegram 接收消息
	go func() {
		if err := telegramAdapter.Start(); err != nil {
			log.Fatalf("Telegram 启动失败: %v", err)
		}
	}()

	// 启动网关处理消息
	go gw.Start()

	log.Println("🚀 Mini Agent Gateway 已启动")
	log.Println("按 Ctrl+C 停止服务")

	// 等待中断信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("正在关闭服务...")
	telegramAdapter.Stop()
}
