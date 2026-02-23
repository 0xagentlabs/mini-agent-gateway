package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadFile 读取文件
func ReadFile(args string) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	return string(content), nil
}

// WriteFile 写入文件
func WriteFile(args string) (string, error) {
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
}

// ListDir 列出目录
func ListDir(args string) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
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
}
