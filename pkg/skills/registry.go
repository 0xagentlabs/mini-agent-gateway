package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xagentlabs/mini-agent-gateway/pkg/mcp"
)

// Skill 技能定义
type Skill struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Tools       []ToolDefinition       `json:"tools"`
	MCPConfig   *MCPConfig             `json:"mcp,omitempty"`
	mcpClient   *mcp.Client            // MCP 客户端（如果有）
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     ToolHandler            `json:"-"` // 内置函数（非 MCP）
}

// ToolHandler 工具处理函数
type ToolHandler func(args string) (string, error)

// MCPConfig MCP 服务器配置
type MCPConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// Registry 技能注册表
type Registry struct {
	skills map[string]*Skill
	tools  map[string]*ToolDefinition
}

// NewRegistry 创建技能注册表
func NewRegistry() *Registry {
	r := &Registry{
		skills: make(map[string]*Skill),
		tools:  make(map[string]*ToolDefinition),
	}
	return r
}

// Register 注册技能
func (r *Registry) Register(skill *Skill) error {
	r.skills[skill.Name] = skill
	
	// 注册技能的所有工具
	for i := range skill.Tools {
		tool := &skill.Tools[i]
		fullName := skill.Name + ":" + tool.Name
		r.tools[fullName] = tool
	}
	
	return nil
}

// RegisterBuiltinSkill 注册内置技能
func (r *Registry) RegisterBuiltinSkill(name, description string, tools []ToolDefinition) {
	skill := &Skill{
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		Tools:       tools,
	}
	r.Register(skill)
}

// RegisterMCPSkill 注册 MCP 技能
func (r *Registry) RegisterMCPSkill(name, description string, config MCPConfig) error {
	ctx := context.Background()
	
	// 创建 MCP 客户端
	client, err := mcp.NewClient(config.Command, config.Args...)
	if err != nil {
		return fmt.Errorf("create mcp client: %w", err)
	}
	
	// 初始化
	if _, err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize mcp: %w", err)
	}
	
	// 获取工具列表
	mcpTools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}
	
	// 转换工具定义
	tools := make([]ToolDefinition, len(mcpTools))
	for i, mt := range mcpTools {
		var params map[string]interface{}
		json.Unmarshal(mt.InputSchema, &params)
		
		tools[i] = ToolDefinition{
			Name:        mt.Name,
			Description: mt.Description,
			Parameters:  params,
		}
	}
	
	skill := &Skill{
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		Tools:       tools,
		MCPConfig:   &config,
		mcpClient:   client,
	}
	
	return r.Register(skill)
}

// LoadFromDir 从目录加载技能
func (r *Registry) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read skills dir: %w", err)
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		skillPath := filepath.Join(dir, entry.Name(), "skill.json")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue // 跳过没有 skill.json 的目录
		}
		
		var skill Skill
		if err := json.Unmarshal(data, &skill); err != nil {
			continue
		}
		
		// 如果是 MCP 技能，初始化 MCP 连接
		if skill.MCPConfig != nil {
			if err := r.RegisterMCPSkill(skill.Name, skill.Description, *skill.MCPConfig); err != nil {
				continue
			}
		} else {
			r.Register(&skill)
		}
	}
	
	return nil
}

// GetToolDefinitions 获取所有工具定义（用于 LLM）
func (r *Registry) GetToolDefinitions() []map[string]interface{} {
	defs := make([]map[string]interface{}, 0, len(r.tools))
	
	for fullName, tool := range r.tools {
		def := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        fullName,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		}
		defs = append(defs, def)
	}
	
	return defs
}

// Execute 执行工具
func (r *Registry) Execute(fullName string, args string) (string, error) {
	// 首先检查内置 handler
	if handler, ok := BuiltinHandlers[fullName]; ok {
		return handler(args)
	}
	
	tool, ok := r.tools[fullName]
	if !ok {
		return "", fmt.Errorf("工具不存在: %s", fullName)
	}
	
	// 找到技能
	parts := splitSkillTool(fullName)
	if len(parts) != 2 {
		return "", fmt.Errorf("无效的工具名: %s", fullName)
	}
	
	skillName := parts[0]
	skill, ok := r.skills[skillName]
	if !ok {
		return "", fmt.Errorf("技能不存在: %s", skillName)
	}
	
	// MCP 技能
	if skill.mcpClient != nil {
		var argsMap map[string]interface{}
		if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
			return "", fmt.Errorf("解析参数: %w", err)
		}
		return skill.mcpClient.CallTool(context.Background(), tool.Name, argsMap)
	}
	
	// 内置技能
	if tool.Handler != nil {
		return tool.Handler(args)
	}
	
	return "", fmt.Errorf("工具未实现: %s", fullName)
}

// splitSkillTool 分割技能名和工具名
func splitSkillTool(fullName string) []string {
	// 简单实现，找第一个 :
	for i := 0; i < len(fullName); i++ {
		if fullName[i] == ':' {
			return []string{fullName[:i], fullName[i+1:]}
		}
	}
	return []string{fullName}
}

// Close 关闭所有 MCP 连接
func (r *Registry) Close() {
	for _, skill := range r.skills {
		if skill.mcpClient != nil {
			skill.mcpClient.Close()
		}
	}
}
