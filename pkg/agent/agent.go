package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/openclaw/mini-agent-gateway/pkg/tools"
)

// Agent 是核心智能体
type Agent struct {
	client *openai.Client
	tools  *tools.Registry
	model  string
}

// New 创建 Agent 实例
func New(apiKey string) *Agent {
	client := openai.NewClient(apiKey)
	
	return &Agent{
		client: client,
		tools:  tools.NewRegistry(),
		model:  openai.GPT4oMini, // 使用 GPT-4o-mini 降低成本
	}
}

// Message 对话消息
type Message struct {
	Role    string
	Content string
}

// Run 执行 Agent Loop
func (a *Agent) Run(ctx context.Context, history []Message) (string, error) {
	// 转换历史消息格式
	messages := make([]openai.ChatCompletionMessage, 0, len(history)+1)
	
	// 系统提示词
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "system",
		Content: a.systemPrompt(),
	})

	for _, h := range history {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	// 第一次调用，获取响应
	resp, err := a.callLLM(ctx, messages)
	if err != nil {
		return "", err
	}

	// 处理工具调用
	if len(resp.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, messages, resp.ToolCalls)
	}

	return resp.Content, nil
}

// callLLM 调用大模型
func (a *Agent) callLLM(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionMessage, error) {
	req := openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: messages,
		Tools:    a.tools.GetDefinitions(),
	}

	resp, err := a.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空响应")
	}

	return &resp.Choices[0].Message, nil
}

// handleToolCalls 处理工具调用
func (a *Agent) handleToolCalls(ctx context.Context, messages []openai.ChatCompletionMessage, toolCalls []openai.ToolCall) (string, error) {
	// 添加 assistant 的 tool_calls 消息
	messages = append(messages, openai.ChatCompletionMessage{
		Role:      "assistant",
		ToolCalls: toolCalls,
	})

	// 执行每个工具调用
	for _, tc := range toolCalls {
		result, err := a.tools.Execute(tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("错误: %v", err)
		}

		// 添加 tool 结果到消息
		messages = append(messages, openai.ChatCompletionMessage{
			Role:       "tool",
			Content:    result,
			ToolCallID: tc.ID,
		})
	}

	// 再次调用 LLM 获取最终回复
	finalResp, err := a.callLLM(ctx, messages)
	if err != nil {
		return "", err
	}

	return finalResp.Content, nil
}

// systemPrompt 系统提示词
func (a *Agent) systemPrompt() string {
	return `你是一个有用的 AI 助手。你可以使用以下工具来帮助用户：

1. read_file - 读取文件内容
2. write_file - 写入文件内容  
3. exec_shell - 执行 shell 命令
4. web_search - 搜索网络信息

请根据用户需求选择合适的工具。如果不确定，可以直接回答。`
}
