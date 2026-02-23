package gateway

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/openclaw/mini-agent-gateway/pkg/agent"
	"github.com/openclaw/mini-agent-gateway/pkg/session"
)

// Message 标准化消息格式
type Message struct {
	ID        string
	UserID    string
	ChatID    string
	Text      string
	Channel   string // telegram / discord / slack
	Timestamp time.Time
}

// Gateway 是核心消息路由
type Gateway struct {
	agent   *agent.Agent
	session *session.Manager
	msgChan chan Message
}

// New 创建网关实例
func New() *Gateway {
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量")
	}

	return &Gateway{
		agent:   agent.New(openaiKey),
		session: session.NewManager(),
		msgChan: make(chan Message, 100),
	}
}

// HandleMessage 接收来自各频道的消息
func (g *Gateway) HandleMessage(msg Message) {
	g.msgChan <- msg
}

// Start 开始处理消息循环
func (g *Gateway) Start() {
	for msg := range g.msgChan {
		go g.processMessage(msg)
	}
}

// processMessage 处理单条消息
func (g *Gateway) processMessage(msg Message) {
	ctx := context.Background()
	
	// 获取或创建会话
	sess := g.session.GetOrCreate(msg.UserID)
	
	// 记录用户消息
	sess.AddMessage("user", msg.Text)
	
	log.Printf("[%s] %s: %s", msg.Channel, msg.UserID, msg.Text)

	// 调用 Agent 处理
	reply, err := g.agent.Run(ctx, sess.GetMessages())
	if err != nil {
		log.Printf("Agent 错误: %v", err)
		reply = "抱歉，处理消息时出错了"
	}

	// 记录助手回复
	sess.AddMessage("assistant", reply)

	// 发送回复到对应频道
	g.sendReply(msg, reply)
}

// sendReply 发送回复到原频道
func (g *Gateway) sendReply(msg Message, reply string) {
	// 通过回调或直接调用 channel 的方法发送
	// 这里简化处理，实际可以通过 channel 的接口发送
	fmt.Printf("[回复 %s]: %s\n", msg.ChatID, reply)
}
