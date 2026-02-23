# AgentSkills 规范实现

## 什么是 AgentSkills?

AgentSkills 是一种**声明式技能定义**，通过 SKILL.md 文件教 AI Agent 如何使用工具，而不是直接提供工具实现。

## 核心概念

| 层级 | 作用 |
|------|------|
| **SKILL.md** | Markdown 文件，包含元数据 + 使用说明 |
| **ClawHub** | 公共技能仓库 (类似 npm registry) |
| **技能加载** | workspace > ~/.openclaw/skills > bundled |

## SKILL.md 格式

```markdown
---
name: web-search
description: Search the web using DuckDuckGo
metadata:
  {"openclaw": {"requires": {"env": ["DDG_API_KEY"]}}}
---

## 使用 web-search 技能

当用户需要搜索信息时，使用 fs:exec 工具执行 curl：

```bash
curl -s "https://duckduckgo.com/html/?q={query}"
```

或者使用内置的 web_search 工具（如果可用）。

## 最佳实践

1. 优先搜索最新信息
2. 总结前 3 个结果
3. 提供来源链接
```

## 技能加载优先级

```
<workspace>/skills/     ← 最高优先级 (项目级)
~/.openclaw/skills/     ← 用户级
<bundled>/skills/       ← 内置 (最低优先级)
```

## 元数据字段

| 字段 | 说明 |
|------|------|
| `name` | 技能标识 |
| `description` | 简短描述 |
| `metadata.openclaw.requires.bins` | 需要的可执行文件 |
| `metadata.openclaw.requires.env` | 需要的环境变量 |
| `metadata.openclaw.requires.config` | 需要的配置项 |
| `user-invocable` | 是否可作为 /command 调用 |
| `disable-model-invocation` | 是否仅从 prompt 中隐藏 |

## 与 MCP 的区别

| | AgentSkills | MCP |
|---|---|---|
| **本质** | 教学文档 | 协议接口 |
| **实现** | SKILL.md (markdown) | Server (stdio/SSE) |
| **工具** | 复用现有工具 | 提供新工具 |
| **用途** | 教会 Agent 使用工具 | 标准化工具集成 |

## 使用流程

1. 从 ClawHub 安装技能：`clawhub install web-search`
2. SKILL.md 被加载到 prompt
3. Agent 根据说明使用已有工具
4. 完成任务

