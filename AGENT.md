# Waku Agent 指南

## 项目概览

Waku 是基于 Cloudwego Eino 的 AI 小说创作系统，通过 Agent 与 Graph Workflow 自动生成、扩写、润色、审核、修改并保存小说大纲。

## 核心结构

```text
waku/
├── main.go                  # 程序入口：启动、迁移、重置
├── config/                  # YAML 配置
├── data/                    # SQLite 与内容文件
└── internal/
    ├── model/               # GORM 模型
    ├── config/              # 配置加载
    ├── database/            # 数据库连接与迁移
    ├── utils/               # ChatModel、Agent、文件、回调工具
    └── novel/
        ├── agents/          # 生成、扩写、润色、审核、修改、提取 Agent
        ├── compose/         # 大纲工作流、节点、路由、执行器
        ├── prompts/         # Agent 提示词
        ├── repository/      # 小说数据访问
        ├── tools/           # 大纲保存工具
        └── utils/           # 小说模块工具
```

## 关键模型

`internal/model.Novel` 保存小说元信息：标题、作者、类型、简介、大纲、世界观、修行体系、状态与时间戳。`Outline` 字段存储 JSON 字符串，便于结构化管理章节、卷和阶段产物。

## 大纲工作流

完整生成模式：

```text
START → init → generate → expand → polish → review → revise → extract → saveToDB → END
```

快速修改模式：

```text
START → init → polish → review → revise → extract → saveToDB → END
```

- `init`：根据输入决定完整生成或快速修改；快速修改会按 `NovelID` 从数据库加载内容。
- `generate` / `expand`：仅完整生成模式执行，生成并扩写初始大纲。
- `polish` / `review` / `revise`：润色、审核不合理性，并按审核或用户建议修改。
- `extract`：提取标题、简介、世界观、修行体系等结构化信息。
- `saveToDB`：保存到数据库，支持用户审批式中断恢复。
- 各处理节点支持断点续传与失败重试，阶段文件保存到 `data/content/{novelID-novelName}/abstract/`。

## Agent 列表

| Agent | 作用 |
| --- | --- |
| `OutlineGeneratorAgent` | 根据构思和类型生成初始大纲 |
| `OutlineExpanderAgent` | 扩写大纲细节 |
| `OutlinePolisherAgent` | 润色并优化表达与结构 |
| `OutlineReviewerAgent` | 审核逻辑漏洞、设定冲突、节奏问题 |
| `OutlineReviserAgent` | 根据审核或用户建议修改大纲 |
| `OutlineExtractorAgent` | 提取结构化小说信息并输出 JSON |

## 文件格式

阶段文件统一使用 JSON：

```json
{
  "content": "阶段内容",
  "stage": "生成/扩写/润色/审核/修改",
  "file_type": "outline"
}
```

`utils.LoadFromFile()` 需兼容旧纯文本文件；`utils.SaveToFile()` 写入结构化 JSON。

## 开发规范

- 代码保持小步、聚焦，优先修根因，避免无关重构。
- 导出包、类型、函数使用中文注释，复杂业务逻辑补充必要说明。
- 错误使用 `%w` 包装，信息包含上下文；需要判断时使用 `errors.Is` / `errors.As`。
- 避免重复逻辑，公共流程提取函数；硬编码值提取为常量。
- 字符串拼接优先 `strings.Builder`，共享状态必须考虑并发安全。
- 测试优先表驱动；核心逻辑尽量覆盖成功、空输入、外部依赖失败等场景。

## 常用命令

```bash
go run main.go -config config/config.yaml
go run main.go -migrate
go run main.go -reset
gofmt -w .
go test ./... -v -cover
go test ./... -race
golangci-lint run
```

## 修改入口

- 新增 Agent：添加 `internal/novel/agents/`、`internal/novel/prompts/`，再接入工作流节点与路由。
- 修改工作流：优先看 `internal/novel/compose/nodes.go`、`router.go`、`outline_workflow.go`。
- 新增模型提供商：修改 `internal/utils/chat_model.go` 与配置结构。
- 调试工作流：查看 `data/content/` 中间文件，必要时设置 `DB_LOG_LEVEL=debug`。

## Commit 规范

使用 `<type>: <subject>`，常见类型：`feat`、`fix`、`refactor`、`docs`、`test`、`chore`、`perf`、`ci`。
