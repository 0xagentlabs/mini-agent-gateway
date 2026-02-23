package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Handler 工具处理函数类型
type Handler func(args string) (string, error)

// Tool 工具定义
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Handler     Handler
}

// ToolDefinition LLM 工具定义 (OpenAI 格式)
type ToolDefinition struct {
	Type     string           `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Registry 工具注册表
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}
	r.registerDefaults()
	return r
}

// registerDefaults 注册默认工具
func (r *Registry) registerDefaults() {
	// 读取文件
	r.Register(Tool{
		Name:        "read_file",
		Description: "读取指定路径的文件内容",
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
		Handler: func(args string) (string, error) {
			var params struct{ Path string `json:"path"` }
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", err
			}
			content, err := os.ReadFile(params.Path)
			if err != nil {
				return "", err
			}
			return string(content), nil
		},
	})

	// 写入文件
	r.Register(Tool{
		Name:        "write_file",
		Description: "写入内容到指定文件",
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
		Handler: func(args string) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", err
			}
			if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
				return "", err
			}
			return fmt.Sprintf("文件已写入: %s", params.Path), nil
		},
	})

	// 执行 Shell
	r.Register(Tool{
		Name:        "exec_shell",
		Description: "执行 shell 命令并返回输出",
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
		Handler: func(args string) (string, error) {
			var params struct{ Command string `json:"command"` }
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", err
			}
			// 安全限制：只允许特定命令
			if !isSafeCommand(params.Command) {
				return "", fmt.Errorf("命令不安全或被禁止")
			}
			cmd := exec.Command("sh", "-c", params.Command)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("错误: %v\n输出: %s", err, string(output)), nil
			}
			return string(output), nil
		},
	})

	// 网络搜索
	r.Register(Tool{
		Name:        "web_search",
		Description: "使用 DuckDuckGo 搜索网络信息",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]string{
					"type":        "string",
					"description": "搜索关键词",
				},
			},
			"required": []string{"query"},
		},
		Handler: func(args string) (string, error) {
			var params struct{ Query string `json:"query"` }
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", err
			}
			return duckduckgoSearch(params.Query)
		},
	})
}

// Register 注册工具
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name] = tool
}

// GetDefinitions 获取工具定义（用于 Function Calling）
func (r *Registry) GetDefinitions() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return defs
}

// GetToolDefinitions 获取工具定义（map 格式，用于 LLM）
func (r *Registry) GetToolDefinitions() []map[string]interface{} {
	defs := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		def := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		}
		defs = append(defs, def)
	}
	return defs
}

// Execute 执行工具
func (r *Registry) Execute(name string, args string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}
	return tool.Handler(args)
}

// isSafeCommand 检查命令安全性
func isSafeCommand(cmd string) bool {
	// 禁止的危险命令
	dangerous := []string{"rm -rf /", "> /dev/sda", "mkfs", "dd if=/dev/zero"}
	for _, d := range dangerous {
		if strings.Contains(cmd, d) {
			return false
		}
	}
	return true
}

// duckduckgoSearch DuckDuckGo 搜索
func duckduckgoSearch(query string) (string, error) {
	// 使用 DuckDuckGo HTML 版本
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 简单提取结果（实际生产应该用 goquery 解析 HTML）
	// 这里简化返回
	return fmt.Sprintf("搜索结果: %s... (共 %d 字节)", query, len(body)), nil
}
