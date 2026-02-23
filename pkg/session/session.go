package session

import (
	"sync"
	"time"
)

// Session 用户会话
type Session struct {
	UserID   string
	Messages []Message
	LastAt   time.Time
}

// Message 会话中的消息
type Message struct {
	Role      string    // user / assistant / system
	Content   string
	Timestamp time.Time
}

// Manager 会话管理器
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	maxMsgs  int // 保留的最大消息数
}

// NewManager 创建会话管理器
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		maxMsgs:  20, // 保留最近 20 条消息
	}
}

// GetOrCreate 获取或创建会话
func (m *Manager) GetOrCreate(userID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[userID]; ok {
		sess.LastAt = time.Now()
		return sess
	}

	sess := &Session{
		UserID:   userID,
		Messages: make([]Message, 0),
		LastAt:   time.Now(),
	}
	m.sessions[userID] = sess
	return sess
}

// AddMessage 添加消息到会话
func (s *Session) AddMessage(role, content string) {
	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.Messages = append(s.Messages, msg)

	// 限制历史长度
	if len(s.Messages) > 20 {
		s.Messages = s.Messages[len(s.Messages)-20:]
	}
}

// GetMessages 获取所有消息（用于 Agent）
// MessageForAgent 用于 Agent 的消息格式
type MessageForAgent struct {
	Role    string
	Content string
}

// GetMessages 获取所有消息（用于 Agent）
func (s *Session) GetMessages() []MessageForAgent {
	result := make([]MessageForAgent, 0, len(s.Messages))

	for _, m := range s.Messages {
		result = append(result, MessageForAgent{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}

// CleanupOldSessions 清理过期会话（可选）
func (m *Manager) CleanupOldSessions(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, sess := range m.sessions {
		if now.Sub(sess.LastAt) > maxAge {
			delete(m.sessions, id)
		}
	}
}
