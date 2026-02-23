package skill

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill AgentSkills / Claude Code Skills 定义
type Skill struct {
	Name               string                 `yaml:"name"`
	Description        string                 `yaml:"description"`
	Metadata           map[string]interface{} `yaml:"metadata,omitempty"`
	UserInvocable      bool                   `yaml:"user-invocable,omitempty"`      // 默认可用户调用
	DisableModelInvoke bool                   `yaml:"disable-model-invocation,omitempty"` // 禁止自动调用
	CommandDispatch    string                 `yaml:"command-dispatch,omitempty"`     // tool | null
	CommandTool        string                 `yaml:"command-tool,omitempty"`         // 指定工具名
	
	Content   string // SKILL.md 内容（frontmatter 之后）
	Path      string // 技能目录路径
	Source    string // 来源：project | personal | bundled
}

// GetSlashCommand 获取 slash command 名称
func (s *Skill) GetSlashCommand() string {
	return "/" + s.Name
}

// CanAutoInvoke 是否可以自动调用（根据描述匹配）
func (s *Skill) CanAutoInvoke() bool {
	return !s.DisableModelInvoke
}

// CanUserInvoke 是否可以用户直接调用
func (s *Skill) CanUserInvoke() bool {
	if s.UserInvocable == false {
		return false
	}
	return true // 默认 true
}

// IsEligible 检查技能是否可用（环境检查）
func (s *Skill) IsEligible() bool {
	requires := s.getRequires()
	if requires == nil {
		return true
	}
	
	// 检查 bins
	if bins, ok := requires["bins"].([]interface{}); ok {
		for _, bin := range bins {
			if _, err := exec.LookPath(bin.(string)); err != nil {
				return false
			}
		}
	}
	
	// 检查 env
	if envs, ok := requires["env"].([]interface{}); ok {
		for _, env := range envs {
			if os.Getenv(env.(string)) == "" {
				return false
			}
		}
	}
	
	return true
}

// getRequires 获取 requires 配置
func (s *Skill) getRequires() map[string]interface{} {
	if metadata, ok := s.Metadata["openclaw"].(map[string]interface{}); ok {
		if requires, ok := metadata["requires"].(map[string]interface{}); ok {
			return requires
		}
	}
	return nil
}

// BuildPromptForLLM 构建给 LLM 的 prompt
func (s *Skill) BuildPromptForLLM() string {
	var b strings.Builder
	
	b.WriteString(fmt.Sprintf("## Skill: %s\n", s.Name))
	b.WriteString(fmt.Sprintf("Description: %s\n", s.Description))
	
	if s.CanUserInvoke() {
		b.WriteString(fmt.Sprintf("Slash Command: %s\n", s.GetSlashCommand()))
	}
	
	if s.CanAutoInvoke() {
		b.WriteString("Auto-invoke: When the user's request matches the description above.\n")
	}
	
	b.WriteString("\n")
	b.WriteString(s.Content)
	
	return b.String()
}

// Registry 技能注册表
type Registry struct {
	skills map[string]*Skill
	
	// 加载路径（按优先级排序）
	projectDir   string // .claude/skills/ 或 ./skills/
	personalDir  string // ~/.claude/skills/
}

// NewRegistry 创建技能注册表
func NewRegistry(projectDir string) *Registry {
	home, _ := os.UserHomeDir()
	
	return &Registry{
		skills:      make(map[string]*Skill),
		projectDir:  projectDir,
		personalDir: filepath.Join(home, ".claude", "skills"),
	}
}

// LoadAll 加载所有技能
func (r *Registry) LoadAll() error {
	// 按优先级加载：personal → project（project 优先级更高）
	paths := []struct {
		path   string
		source string
	}{
		{r.personalDir, "personal"},
		{r.projectDir, "project"},
	}
	
	for _, p := range paths {
		if err := r.loadFromDir(p.path, p.source); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("加载技能目录 %s 失败: %v\n", p.path, err)
			}
		}
	}
	
	return nil
}

// loadFromDir 从目录加载技能
func (r *Registry) loadFromDir(dir, source string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		skillPath := filepath.Join(dir, entry.Name())
		skill, err := loadSkill(skillPath, source)
		if err != nil {
			continue // 静默跳过无效技能
		}
		
		// 高优先级覆盖低优先级
		r.skills[skill.Name] = skill
	}
	
	return nil
}

// loadSkill 加载单个技能
func loadSkill(dir, source string) (*Skill, error) {
	skillFile := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}
	
	// 解析 frontmatter
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("missing frontmatter")
	}
	
	// 提取 frontmatter
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid frontmatter")
	}
	
	var skill Skill
	if err := yaml.Unmarshal([]byte(parts[0]), &skill); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	
	skill.Content = strings.TrimSpace(parts[1])
	skill.Path = dir
	skill.Source = source
	
	// 默认值
	if skill.UserInvocable == false && !strings.Contains(parts[0], "user-invocable") {
		skill.UserInvocable = true // 默认可用户调用
	}
	
	return &skill, nil
}

// Get 获取技能
func (r *Registry) Get(name string) *Skill {
	if s, ok := r.skills[name]; ok && s.IsEligible() {
		return s
	}
	return nil
}

// GetBySlashCommand 通过 slash command 获取技能
func (r *Registry) GetBySlashCommand(cmd string) *Skill {
	// 去掉 / 前缀
	name := strings.TrimPrefix(cmd, "/")
	return r.Get(name)
}

// GetAll 获取所有可用技能
func (r *Registry) GetAll() []*Skill {
	var result []*Skill
	for _, s := range r.skills {
		if s.IsEligible() {
			result = append(result, s)
		}
	}
	return result
}

// GetAutoInvokable 获取可自动调用的技能
func (r *Registry) GetAutoInvokable() []*Skill {
	var result []*Skill
	for _, s := range r.skills {
		if s.IsEligible() && s.CanAutoInvoke() {
			result = append(result, s)
		}
	}
	return result
}

// GetUserInvokable 获取可用户调用的技能
func (r *Registry) GetUserInvokable() []*Skill {
	var result []*Skill
	for _, s := range r.skills {
		if s.IsEligible() && s.CanUserInvoke() {
			result = append(result, s)
		}
	}
	return result
}

// BuildSystemPrompt 构建系统 prompt 中的技能部分
func (r *Registry) BuildSystemPrompt() string {
	skills := r.GetAutoInvokable()
	if len(skills) == 0 {
		return ""
	}
	
	var b strings.Builder
	b.WriteString("# Available Skills\n\n")
	b.WriteString("You have access to the following skills. " +
		"Use them automatically when the user's request matches the description, " +
		"or when the user explicitly invokes them with /command.\n\n")
	
	for _, s := range skills {
		b.WriteString(s.BuildPromptForLLM())
		b.WriteString("\n---\n\n")
	}
	
	return b.String()
}

// BuildSlashCommandsHelp 构建 slash commands 帮助
func (r *Registry) BuildSlashCommandsHelp() string {
	skills := r.GetUserInvokable()
	if len(skills) == 0 {
		return "No slash commands available."
	}
	
	var b strings.Builder
	b.WriteString("# Slash Commands\n\n")
	
	for _, s := range skills {
		b.WriteString(fmt.Sprintf("**%s** - %s\n", s.GetSlashCommand(), s.Description))
	}
	
	return b.String()
}

// TryInvokeByCommand 尝试通过 command 调用技能
func (r *Registry) TryInvokeByCommand(ctx context.Context, cmd string, args string) (string, bool) {
	skill := r.GetBySlashCommand(cmd)
	if skill == nil {
		return "", false
	}
	
	// 如果设置了 command-dispatch: tool，直接调用指定工具
	if skill.CommandDispatch == "tool" && skill.CommandTool != "" {
		// 返回工具调用信息，由上层处理
		return fmt.Sprintf("Tool call: %s with args: %s", skill.CommandTool, args), true
	}
	
	// 否则返回技能内容作为 prompt
	return skill.Content, true
}
