package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/0xagentlabs/mini-agent-gateway/pkg/skill"
	"github.com/0xagentlabs/mini-agent-gateway/pkg/tools"
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
	client    *LLMClient
	skillReg  *skill.Registry
	toolReg   *tools.Registry
	workspace string
}

// New 创建 Agent 实例
func New(apiKey string) *Agent {
	// 从环境变量读取配置，或使用默认值
	baseURL := getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")
	model := getEnv("OPENAI_MODEL", "gpt-4o-mini")
	workspace := getEnv("WORKSPACE", ".")

	// 创建技能注册表
	skillReg := skill.NewRegistry(filepath.Join(workspace, "skills"))
	if err := skillReg.LoadAll(); err != nil {
		fmt.Printf("加载技能失败: %v\n", err)
	}
	
	// 创建工具注册表
	toolReg := tools.NewRegistry()

	return &Agent{
		client:    NewLLMClient(baseURL, apiKey, model),
		skillReg:  skillReg,
		toolReg:   toolReg,
		workspace: workspace,
	}
}

// Run 执行 Agent Loop
func (a *Agent) Run(ctx context.Context, history []Message) (string, error) {
	// 构建系统消息
	systemMsg := a.buildSystemPrompt()
	
	messages := []Message{
		{Role: "system", Content: systemMsg},
	}
	messages = append(messages, history...)

	// 获取工具定义
	toolDefs := a.toolReg.GetToolDefinitions()

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

// buildSystemPrompt 构建系统提示词
func (a *Agent) buildSystemPrompt() string {
	prompt := `你是一个有用的 AI 助手。你可以使用以下工具来帮助用户：

1. fs:read - 读取文件内容
2. fs:write - 写入文件内容  
3. fs:exec - 执行 shell 命令

当你收到用户请求时：
- 分析需要使用哪些工具
- 按顺序执行工具
- 根据结果给出最终回复
`
	
	// 添加技能说明
	skillsPrompt := a.skillReg.BuildSystemPrompt()
	if skillsPrompt != "" {
		prompt += "\n\n" + skillsPrompt
	}
	
	return prompt
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
		result, err := a.toolReg.Execute(tc.Function.Name, tc.Function.Arguments)
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

// getEnv 获取环境变量，如果不存在返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
