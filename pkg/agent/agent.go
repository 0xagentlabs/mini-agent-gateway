package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/0xagentlabs/mini-agent-gateway/pkg/skills"
)

// LLMClient 轻量级 OpenAI 兼容客户端
type LLMClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewLLMClient 创建 LLM 客户端
func NewLLMClient(baseURL, apiKey, model string) *LLMClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &LLMClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ChatCompletionRequest OpenAI 聊天完成请求
type ChatCompletionRequest struct {
	Model    string                   `json:"model"`
	Messages []Message                `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

// ChatCompletionResponse OpenAI 聊天完成响应
type ChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Message 对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat 发送聊天请求
func (c *LLMClient) Chat(ctx context.Context, messages []Message, tools []map[string]interface{}) (*ChatCompletionResponse, error) {
	reqBody := ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// Agent 核心智能体
type Agent struct {
	client *LLMClient
	skills *skills.Registry
}

// New 创建 Agent 实例
func New(apiKey string) *Agent {
	// 从环境变量读取配置，或使用默认值
	baseURL := getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")
	model := getEnv("OPENAI_MODEL", "gpt-4o-mini")

	// 创建技能注册表
	skillsReg := skills.NewRegistry()
	
	// 加载内置技能
	loadBuiltinSkills(skillsReg)
	
	// 从目录加载技能
	skillsDir := getEnv("SKILLS_DIR", "./skills")
	if err := skillsReg.LoadFromDir(skillsDir); err != nil {
		fmt.Printf("加载技能目录失败: %v\n", err)
	}

	return &Agent{
		client: NewLLMClient(baseURL, apiKey, model),
		skills: skillsReg,
	}
}

// loadBuiltinSkills 加载内置技能
func loadBuiltinSkills(r *skills.Registry) {
	// 文件系统技能
	fsTools := []skills.ToolDefinition{
		{
			Name:        "read",
			Description: "读取文件内容",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{
						"type":        "string",
						"description": "文件路径",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write",
			Description: "写入文件内容",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{
						"type":        "string",
						"description": "文件路径",
					},
					"content": map[string]string{
						"type":        "string",
						"description": "文件内容",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "exec",
			Description: "执行 shell 命令",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]string{
						"type":        "string",
						"description": "要执行的命令",
					},
				},
				"required": []string{"command"},
			},
		},
	}
	r.RegisterBuiltinSkill("fs", "文件系统操作", fsTools)
}

// Run 执行 Agent Loop
func (a *Agent) Run(ctx context.Context, history []Message) (string, error) {
	// 构建系统消息
	systemMsg := a.systemPrompt()
	
	// 如果有技能，添加到系统提示词
	skillsDesc := a.getSkillsDescription()
	if skillsDesc != "" {
		systemMsg += "\n\n" + skillsDesc
	}
	
	messages := []Message{
		{Role: "system", Content: systemMsg},
	}
	messages = append(messages, history...)

	// 获取所有工具定义
	toolDefs := a.skills.GetToolDefinitions()

	// 调用 LLM
	resp, err := a.client.Chat(ctx, messages, toolDefs)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空响应")
	}

	choice := resp.Choices[0].Message

	// 处理工具调用
	if len(choice.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, messages, choice.ToolCalls)
	}

	return choice.Content, nil
}

// handleToolCalls 处理工具调用
func (a *Agent) handleToolCalls(ctx context.Context, messages []Message, toolCalls []ToolCall) (string, error) {
	// 添加 assistant 的 tool_calls 消息
	assistantMsg := Message{
		Role:    "assistant",
		Content: "",
	}
	messages = append(messages, assistantMsg)

	// 执行每个工具调用
	for _, tc := range toolCalls {
		result, err := a.skills.Execute(tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("错误: %v", err)
		}

		// 添加 tool 结果到消息
		toolMsg := Message{
			Role:    "tool",
			Content: result,
		}
		messages = append(messages, toolMsg)
	}

	// 再次调用 LLM 获取最终回复
	finalResp, err := a.client.Chat(ctx, messages, nil)
	if err != nil {
		return "", err
	}

	if len(finalResp.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空响应")
	}

	return finalResp.Choices[0].Message.Content, nil
}

// systemPrompt 系统提示词
func (a *Agent) systemPrompt() string {
	return `你是一个有用的 AI 助手。你可以使用技能系统提供的工具来帮助用户。`
}

// getSkillsDescription 获取技能描述
func (a *Agent) getSkillsDescription() string {
	tools := a.skills.GetToolDefinitions()
	if len(tools) == 0 {
		return ""
	}
	
	desc := "可用工具:\n"
	for _, tool := range tools {
		if fn, ok := tool["function"].(map[string]interface{}); ok {
			name := fn["name"].(string)
			description := fn["description"].(string)
			desc += fmt.Sprintf("- %s: %s\n", name, description)
		}
	}
	return desc
}

// Close 关闭 Agent
func (a *Agent) Close() {
	if a.skills != nil {
		a.skills.Close()
	}
}

// getEnv 获取环境变量，如果不存在返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
