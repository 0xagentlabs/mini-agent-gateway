package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Client MCP 客户端
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	mu     sync.Mutex
	
	requestID int
	pending   map[int]chan *JSONRPCResponse
	
	serverInfo *ServerInfo
	tools      []Tool
}

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// ServerInfo MCP 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool MCP 工具定义
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// NewClient 创建 MCP 客户端
func NewClient(command string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}
	
	client := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		requestID: 1,
		pending:   make(map[int]chan *JSONRPCResponse),
	}
	
	// 启动响应读取 goroutine
	go client.readResponses()
	
	return client, nil
}

// Initialize 初始化 MCP 连接
func (c *Client) Initialize(ctx context.Context) (*ServerInfo, error) {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "mini-agent-gateway",
			"version": "0.1.0",
		},
	}
	
	resp, err := c.call(ctx, "initialize", params)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		ProtocolVersion string     `json:"protocolVersion"`
		ServerInfo      ServerInfo `json:"serverInfo"`
	}
	
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}
	
	c.serverInfo = &result.ServerInfo
	
	// 发送 initialized 通知
	c.notify("notifications/initialized", nil)
	
	return c.serverInfo, nil
}

// ListTools 获取工具列表
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Tools []Tool `json:"tools"`
	}
	
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshal tools: %w", err)
	}
	
	c.tools = result.Tools
	return result.Tools, nil
}

// CallTool 调用 MCP 工具
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}
	
	resp, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return "", err
	}
	
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("unmarshal result: %w", err)
	}
	
	// 提取文本内容
	var output string
	for _, c := range result.Content {
		if c.Type == "text" {
			output += c.Text + "\n"
		}
	}
	
	return output, nil
}

// call 发送 JSON-RPC 请求
func (c *Client) call(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	c.mu.Lock()
	id := c.requestID
	c.requestID++
	
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	
	respChan := make(chan *JSONRPCResponse, 1)
	c.pending[id] = respChan
	c.mu.Unlock()
	
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	data = append(data, '\n')
	
	if _, err := c.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}
	
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// notify 发送通知（无响应）
func (c *Client) notify(method string, params interface{}) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	
	data = append(data, '\n')
	_, err = c.stdin.Write(data)
	return err
}

// readResponses 读取服务器响应
func (c *Client) readResponses() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		
		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- &resp
			delete(c.pending, resp.ID)
		}
		c.mu.Unlock()
	}
}

// Close 关闭 MCP 连接
func (c *Client) Close() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// GetTools 获取缓存的工具列表
func (c *Client) GetTools() []Tool {
	return c.tools
}
