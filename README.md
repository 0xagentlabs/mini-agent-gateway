# Mini Agent Gateway

> 用 Go 实现的极简 AI Agent 网关，仅 500-800 行代码，演示核心 Agent Loop 原理。

## 🎯 项目目标

从零实现一个可运行的 AI Agent 网关，理解以下核心概念：
- **Gateway 模式**: 单一入口，消息路由
- **Channel Adapter**: 多频道协议转换
- **Agent Loop**: 推理 → 工具调用 → 结果反馈的循环
- **Session 管理**: 对话状态隔离与持久化

## 📁 项目结构

```
mini-agent-gateway/
├── cmd/
│   └── main.go              # 入口
├── pkg/
│   ├── gateway/
│   │   └── gateway.go       # WebSocket 网关核心 (~100行)
│   ├── channel/
│   │   └── telegram.go      # Telegram 适配器 (~100行)
│   ├── agent/
│   │   └── agent.go         # Agent Loop 核心 (~150行)
│   ├── tools/
│   │   └── tools.go         # 工具系统 (~200行)
│   └── session/
│       └── session.go       # 会话管理 (~100行)
├── .env.example
├── go.mod
└── README.md
```

**总计约 650 行代码**

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/xingyue/mini-agent-gateway.git
cd mini-agent-gateway
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 填入你的 API Keys
```

### 3. 运行

```bash
go mod tidy
go run cmd/main.go
```

## ⚙️ 环境变量

| 变量 | 说明 | 必需 |
|------|------|------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token | ✅ |
| `OPENAI_API_KEY` | OpenAI API Key | ✅ |

## 🔧 支持的工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `read_file` | 读取文件 | 读取配置文件 |
| `write_file` | 写入文件 | 生成代码文件 |
| `exec_shell` | 执行命令 | 运行 git/status |
| `web_search` | 网络搜索 | 查资料 |

## 💡 使用示例

在 Telegram 中对你的 Bot 发送：

```
帮我查一下 Go 1.22 的新特性
```

Agent 会：
1. 调用 `web_search` 搜索
2. 获取结果
3. 总结回复

```
帮我写个 hello.go 文件
```

Agent 会：
1. 调用 `write_file` 创建文件
2. 返回操作结果

## 📚 核心代码解读

### Agent Loop 流程

```go
// 1. 接收用户消息
msg := gateway.ReceiveMessage()

// 2. 加载会话历史
history := session.GetMessages(userID)

// 3. 调用 LLM (支持 Function Calling)
resp := llm.Chat(history, tools)

// 4. 如果需要工具调用
if resp.HasToolCalls() {
    results := tools.Execute(resp.ToolCalls)
    // 递归：工具结果再发给 LLM
    return agent.Run(ctx, append(history, results))
}

// 5. 返回最终回复
return resp.Content
```

### 消息流转

```
Telegram Bot ──▶ Gateway ──▶ Agent ──▶ LLM API
                                 │
                                 ▼
                            工具执行 (Shell/File/Search)
                                 │
                                 ▼
Telegram Bot ◀── Gateway ◀── Agent
```

## 🛡️ 安全说明

- Shell 命令有基础安全检查（禁止 `rm -rf /` 等）
- 生产环境建议：
  - 使用 Docker 沙箱执行命令
  - 添加用户白名单
  - 限制文件访问路径

## 🔮 扩展方向

| 功能 | 实现思路 |
|------|----------|
| 多频道 | 添加 Discord/Slack Adapter |
| 长期记忆 | 接入 sqlite-vec 向量搜索 |
| 并发控制 | Session 级消息队列 |
| 插件系统 | 动态加载 .so 插件 |
| Web UI | 加 WebSocket 浏览器客户端 |

## 📖 对比 OpenClaw

| 特性 | Mini Gateway | OpenClaw |
|------|-------------|----------|
| 代码量 | ~650 行 | ~70 万行 |
| 频道 | 1 (Telegram) | 10+ |
| 工具 | 4 个基础 | 50+ |
| 部署 | `go run` | systemd + 多服务 |
| 学习成本 | 低 | 高 |

**这个项目是为了理解原理，OpenClaw 是为了生产使用。**

## 📄 License

MIT

---

*Built with ❤️ for learning AI Agent architecture*
