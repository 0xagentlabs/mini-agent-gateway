package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BuiltinHandlers 内置工具处理函数映射
var BuiltinHandlers = map[string]ToolHandler{
	// fs:read - 读取文件
	"fs:read": func(args string) (string, error) {
		var params struct{ Path string `json:"path"` }
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", err
		}
		content, err := os.ReadFile(params.Path)
		if err != nil {
			return "", fmt.Errorf("读取文件失败: %w", err)
		}
		return string(content), nil
	},
	
	// fs:write - 写入文件
	"fs:write": func(args string) (string, error) {
		var params struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", err
		}
		if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
			return "", fmt.Errorf("写入文件失败: %w", err)
		}
		return fmt.Sprintf("文件已写入: %s", params.Path), nil
	},
	
	// fs:exec - 执行命令
	"fs:exec": func(args string) (string, error) {
		var params struct{ Command string `json:"command"` }
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", err
		}
		
		// 安全限制
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
	
	// fs:list - 列出目录
	"fs:list": func(args string) (string, error) {
		var params struct{ Path string `json:"path"` }
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", err
		}
		
		entries, err := os.ReadDir(params.Path)
		if err != nil {
			return "", fmt.Errorf("读取目录失败: %w", err)
		}
		
		var result string
		for _, entry := range entries {
			if entry.IsDir() {
				result += "[DIR]  " + entry.Name() + "\n"
			} else {
				result += "[FILE] " + entry.Name() + "\n"
			}
		}
		return result, nil
	},
}

// isSafeCommand 检查命令安全性
func isSafeCommand(cmd string) bool {
	// 禁止的危险命令
	dangerous := []string{"rm -rf /", "> /dev/sda", "mkfs", "dd if=/dev/zero", ":(){ :|:& };:"}
	for _, d := range dangerous {
		if strings.Contains(cmd, d) {
			return false
		}
	}
	return true
}
