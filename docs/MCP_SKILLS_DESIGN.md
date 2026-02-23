# MCP & Skills 设计

## MCP (Model Context Protocol)

MCP 是 Anthropic 提出的开放协议，标准化 AI 与外部工具的集成。

### 架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Agent     │────▶│  MCP Client │────▶│ MCP Server  │
│             │     │  (stdio/SSE)│     │ (外部工具)   │
└─────────────┘     └─────────────┘     └─────────────┘
                            │
                            ▼
                    ┌─────────────┐
                    │ Tools/Prompts│
                    │  Resources   │
                    └─────────────┘
```

### 核心概念

| 概念 | 说明 |
|------|------|
| **Tools** | MCP 服务器提供的可调用函数 |
| **Resources** | 可被模型读取的数据源（文件、API 等）|
| **Prompts** | 预定义的提示词模板 |

### 传输方式

1. **stdio**: 本地进程通信（最常用）
2. **SSE**: HTTP Server-Sent Events（远程）

## Skills

Skills 是可插拔的工具集合，每个 Skill 是一个独立模块。

### 结构

```
skills/
├── filesystem/          # 文件操作 Skill
│   ├── skill.yaml      # Skill 元数据
│   └── tools.go        # 工具实现
├── web-search/         # 网络搜索 Skill
│   ├── skill.yaml
│   └── tools.go
└── mcp-bridge/         # MCP 桥接 Skill
    ├── skill.yaml
    └── mcp_client.go
```

## 实现计划

1. **MCP Client**: 连接 MCP Server，发现 Tools
2. **Skill Registry**: 动态加载 Skills
3. **集成**: 将 MCP Tools 和 Skills 统一到工具系统

