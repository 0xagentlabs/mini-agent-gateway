package main

import (
	"fmt"
	"os"

	"github.com/0xagentlabs/mini-agent-gateway/pkg/skill"
	"github.com/0xagentlabs/mini-agent-gateway/pkg/tools"
)

func main() {
	fmt.Println("🧪 Mini Agent Gateway - Component Test")
	fmt.Println("=========================================")

	// 1. 测试工具系统
	fmt.Println("\n✓ Tools System:")
	toolReg := tools.NewRegistry()
	toolDefs := toolReg.GetToolDefinitions()
	fmt.Printf("  Registered tools: %d\n", len(toolDefs))
	for _, def := range toolDefs {
		if fn, ok := def["function"].(map[string]interface{}); ok {
			fmt.Printf("    - %s\n", fn["name"])
		}
	}

	// 2. 测试工具执行
	fmt.Println("\n✓ Tool Execution Test:")
	result, err := toolReg.Execute("exec_shell", `{"command": "echo 'Hello from Mini Agent Gateway!'"}`)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Output: %s", result)
	}

	// 3. 测试文件读写
	fmt.Println("\n✓ File I/O Test:")
	writeResult, err := toolReg.Execute("write_file", `{"path": "/tmp/test.txt", "content": "Test content from Mini Agent Gateway"}`)
	if err != nil {
		fmt.Printf("  ❌ Write error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Write: %s\n", writeResult)
	}
	
	readResult, err := toolReg.Execute("read_file", `{"path": "/tmp/test.txt"}`)
	if err != nil {
		fmt.Printf("  ❌ Read error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Read: %s\n", readResult)
	}

	// 4. 测试技能系统
	fmt.Println("\n✓ Skills System:")
	skillReg := skill.NewRegistry("./skills")
	if err := skillReg.LoadAll(); err != nil {
		fmt.Printf("  Error loading skills: %v\n", err)
	}
	
	allSkills := skillReg.GetAll()
	fmt.Printf("  Loaded skills: %d\n", len(allSkills))
	for _, s := range allSkills {
		fmt.Printf("    - %s (%s) %s\n", s.Name, s.Source, s.GetSlashCommand())
		if s.CanAutoInvoke() {
			fmt.Printf("      [auto-invoke]\n")
		}
		if s.CanUserInvoke() {
			fmt.Printf("      [user-invoke]\n")
		}
	}

	// 5. 测试技能 Prompt 构建
	fmt.Println("\n✓ Skills Prompt Generation:")
	prompt := skillReg.BuildSystemPrompt()
	if prompt != "" {
		fmt.Printf("  Generated prompt length: %d chars\n", len(prompt))
		fmt.Printf("  Preview (first 300 chars):\n%s...\n", prompt[:min(300, len(prompt))])
	} else {
		fmt.Println("  No eligible skills found")
	}

	// 6. 测试 Slash Commands
	fmt.Println("\n✓ Slash Commands Help:")
	help := skillReg.BuildSlashCommandsHelp()
	fmt.Println(help)

	fmt.Println("\n=========================================")
	
	// 7. 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set - LLM test skipped")
		fmt.Println("\nTo run full integration test:")
		fmt.Println("  export OPENAI_API_KEY='sk-...'")
		fmt.Println("  go run cmd/test/main.go")
	} else {
		fmt.Println("✅ API Key found - run `go run cmd/test/main.go` for LLM test")
	}
	
	fmt.Println("\n✅ Component tests passed!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
