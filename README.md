# Mini Agent Gateway

> 用 Go 实现的极简 AI Agent 网关，支持 MCP (Model Context Protocol) 和可插拔 Skills 系统。

## 🎯 项目目标

从零实现一个可运行的 AI Agent 网关，理解以下核心概念：
- **Gateway 模式**: 单一入口，消息路由
- **Channel Adapter**: 多频道协议转换  
- **Agent Loop**: 推理 → 工具调用 → 结果反馈的循环
- **Session 管理**: 对话状态隔离与持久化
- **MCP 协议**: 标准化的 AI 工具集成
- **Skills 系统**: 可插拔的工具集合

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Mini Agent Gateway                      │
│                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐  │
│  │  Telegram   │───▶│   Gateway   │───▶│   Agent Core    │  │
│  │   Adapter   │    │  (Router)   │    │                 │  │
│  └─────────────┘    └─────────────┘    │  ┌───────────┐  │  │
│                                          │  │ LLM Client│  │  │
│  ┌─────────────┐    ┌─────────────┐    │  │(net/http) │  │  │
│  │   Discord   │───▶│   Session   │───▶│  └───────────┘  │  │
│  │   Adapter   │    │   Manager   │    │                 │  │
│  └─────────────┘    └─────────────┘    │  ┌───────────┐  │  │
│                                          │  │  Skills   │  │  │
│                                          │  │ Registry  │  │  │
│                                          │  └───────────┘  │  │
│                                          └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   MCP Client     │◀──── skill.json
                    │   (stdio/SSE)    │
                    └──────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   MCP Servers    │
                    │ (GitHub/SQLite/  │
                    │  Filesystem...)  │
                    └──────────────────┘
```

## 📁 项目结构

```
mini-agent-gateway/
├── cmd/
│   └── main.go                    # 入口
├── pkg/
│   ├── gateway/
│   │   └── gateway.go             # 消息网关
│   ├── channel/
│   │   └── telegram.go            # Telegram 适配器
│   ├── agent/
│   │   └── agent.go               # Agent Loop + LLM Client
│   ├── mcp/
│   │   └── client.go              # MCP 客户端
│   ├── skills/
│   │   ├── registry.go            # 技能注册表
│   │   └── builtin.go             # 内置工具
│   ├── session/
│   │   └── session.go             # 会话管理
│   └── tools/                     # (已合并到 skills)
├── skills/                        # 技能目录
│   ├── filesystem/
│   │   ├── skill.json             # 技能配置
│   │   └── tools.go               # 工具实现
│   └── github-mcp/
│       └── skill.json             # MCP 技能配置
├── docs/
│   └── MCP_SKILLS_DESIGN.md       # 设计文档
├── .env.example
├── go.mod
└── README.md
```

**核心代码统计：约 1500 行 Go 代码**

## 🚀 快速开始

### 1. 安装

```bash
git clone https://github.com/0xagentlabs/mini-agent-gateway.git
cd mini-agent-gateway
go mod tidy
go build -o mini-agent-gateway cmd/main.go
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 填入你的 API Keys
```

```bash
# LLM 配置
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选，支持 OpenRouter/Ollama
export OPENAI_MODEL="gpt-4o-mini"

# Telegram Bot
export TELEGRAM_BOT_TOKEN="..."

# 技能目录
export SKILLS_DIR="./skills"
```

### 3. 运行

```bash
./mini-agent-gateway
```

## 🛠️ Skills 系统

### 内置 Skills

| Skill | 工具 | 说明 |
|-------|------|------|
| `fs` | `fs:read` | 读取文件 |
| `fs` | `fs:write` | 写入文件 |
| `fs` | `fs:exec` | 执行 shell 命令 |
| `fs` | `fs:list` | 列出目录 |

### MCP Skills

支持任何兼容 MCP 协议的服务器：

```json
// skills/github-mcp/skill.json
{
  "name": "github",
  "description": "GitHub MCP 服务器",
  "mcp": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"]
  }
}
```

其他 MCP Servers:
- `@modelcontextprotocol/server-postgres` - PostgreSQL 数据库
- `@modelcontextprotocol/server-sqlite` - SQLite 数据库  
- `@modelcontextprotocol/server-puppeteer` - 浏览器自动化
- [更多 MCP Servers](https://github.com/modelcontextprotocol/servers)

### 自定义 Skill

```bash
mkdir skills/my-skill

# 创建 skill.json
cat > skills/my-skill/skill.json << 'EOF'
{
  "name": "my-skill",
  "description": "我的自定义技能",
  "tools": [
    {
      "name": "hello",
      "description": "打招呼",
      "parameters": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "名字"
          }
        },
        "required": ["name"]
      }
    }
  ]
}
EOF
```

## 💡 使用示例

### 基本对话

在 Telegram 中对 Bot 发送：
```
帮我写一个 hello.py 文件
```

Agent 会：
1. 调用 `fs:write` 创建文件
2. 返回操作结果

### 使用 MCP 工具

```
查询我的 GitHub 仓库列表
```

Agent 会：
1. 通过 MCP 调用 GitHub API
2. 返回仓库列表

## 🔧 配置

### LLM 提供商

**OpenAI (默认)**
```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="gpt-4o-mini"
```

**OpenRouter (多模型)**
```bash
export OPENAI_BASE_URL="https://openrouter.ai/api/v1"
export OPENAI_API_KEY="sk-or-..."
export OPENAI_MODEL="anthropic/claude-sonnet-4"
```

**Ollama (本地)**
```bash
export OPENAI_BASE_URL="http://localhost:11434/v1"
export OPENAI_API_KEY="ollama"
export OPENAI_MODEL="llama3.1"
```

### 多频道支持

目前支持：
- ✅ Telegram
- 🚧 Discord (开发中)
- 🚧 Slack (开发中)

## 📊 对比

| 特性 | Mini Gateway | PicoClaw | OpenClaw |
|------|-------------|----------|----------|
| 语言 | Go | Go | TypeScript |
| 代码量 | ~1500 行 | ~5000 行 | ~70 万行 |
| 内存占用 | ~20MB | <10MB | >1GB |
| MCP 支持 | ✅ | ✅ | ✅ |
| 内置工具 | 4 个 | 10+ | 50+ |
| 多提供商 | ✅ | ✅ | ✅ |
| 部署方式 | 单二进制 | 单二进制 | Node.js + 服务 |

## 🛡️ 安全

- Shell 命令有基础安全检查
- MCP 服务器以独立进程运行
- 建议生产环境使用 Docker 沙箱

## 🔮 路线图

- [ ] Discord 频道支持
- [ ] 向量数据库记忆
- [ ] 定时任务 (Cron)
- [ ] Web UI 控制面板
- [ ] 插件热加载

## 📄 License

MIT

---

*Built with ❤️ for learning AI Agent architecture*
