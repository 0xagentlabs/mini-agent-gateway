package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill AgentSkills 定义
type Skill struct {
	Name               string                 `yaml:"name"`
	Description        string                 `yaml:"description"`
	Metadata           map[string]interface{} `yaml:"metadata,omitempty"`
	UserInvocable      bool                   `yaml:"user-invocable,omitempty"`
	DisableModelInvoke bool                   `yaml:"disable-model-invocation,omitempty"`
	CommandDispatch    string                 `yaml:"command-dispatch,omitempty"`
	CommandTool        string                 `yaml:"command-tool,omitempty"`
	
	Content   string // SKILL.md 内容（frontmatter 之后）
	Path      string // 技能目录路径
	Source    string // 来源：bundled | managed | workspace
}

// IsEligible 检查技能是否可用（环境检查）
func (s *Skill) IsEligible() bool {
	// 解析 metadata.openclaw.requires
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

// ToPrompt 转换为 prompt 文本
func (s *Skill) ToPrompt() string {
	var b strings.Builder
	
	b.WriteString(fmt.Sprintf("## Skill: %s\n", s.Name))
	b.WriteString(fmt.Sprintf("Description: %s\n\n", s.Description))
	b.WriteString(s.Content)
	
	return b.String()
}

// Registry 技能注册表
type Registry struct {
	skills map[string]*Skill
	
	// 加载路径（按优先级排序）
	workspaceDir string
	managedDir   string
	bundledDir   string
}

// NewRegistry 创建技能注册表
func NewRegistry(workspaceDir string) *Registry {
	home, _ := os.UserHomeDir()
	
	return &Registry{
		skills:       make(map[string]*Skill),
		workspaceDir: workspaceDir,
		managedDir:   filepath.Join(home, ".mini-agent", "skills"),
		bundledDir:   "./skills",
	}
}

// LoadAll 加载所有技能
func (r *Registry) LoadAll() error {
	// 按优先级加载：bundled → managed → workspace
	paths := []struct {
		path   string
		source string
	}{
		{r.bundledDir, "bundled"},
		{r.managedDir, "managed"},
		{r.workspaceDir, "workspace"},
	}
	
	for _, p := range paths {
		if err := r.loadFromDir(p.path, p.source); err != nil {
			// 目录不存在不报错
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
			fmt.Printf("加载技能 %s 失败: %v\n", entry.Name(), err)
			continue
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
	
	return &skill, nil
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

// Get 获取特定技能
func (r *Registry) Get(name string) *Skill {
	if s, ok := r.skills[name]; ok && s.IsEligible() {
		return s
	}
	return nil
}

// BuildPrompt 构建技能部分的 prompt
func (r *Registry) BuildPrompt() string {
	skills := r.GetAll()
	if len(skills) == 0 {
		return ""
	}
	
	var b strings.Builder
	b.WriteString("# Available Skills\n\n")
	b.WriteString("You have access to the following skills. Use them when appropriate:\n\n")
	
	for _, s := range skills {
		b.WriteString(s.ToPrompt())
		b.WriteString("\n---\n\n")
	}
	
	return b.String()
}
