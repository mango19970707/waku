# Plotline 模块

## 概述

Plotline 模块负责小说剧情线的生成、润色、审核和修改。模块保留原有的 Agent 创建方式和 `compose.Graph` 工作流结构，但会将现有拆分的提取/修订流程统一为一个剧情线工作流。

当前重构目标是围绕 4 个 Agent 展开：

- 生成 Agent：根据小说大纲生成剧情线
- 润色 Agent：优化剧情线表述、节奏和清晰度
- 审核 Agent：审核各条剧情线之间是否冲突，并优先采用与用户建议相关的修改方向
- 修改 Agent：根据审核结论和用户建议修改目标剧情线及必要的关联剧情线

## 统一输出契约

每个剧情线 Agent 都必须输出严格 JSON，顶层只允许这 1 个字段：

```json
{
  "plotlines": [
    {
      "id": 0,
      "title": "剧情线标题",
      "type": "main",
      "description": "剧情线描述",
      "status": "active"
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `plotlines`
- `plotlines` 必须是数组，不能为空
- 每条剧情线至少必须包含 `title`、`type`、`description`
- `id` 在完整生成模式下可以为空或省略，由数据库自增补齐；在快速修改模式下，目标剧情线必须保留原有 `id`
- `type` 当前只允许 `main` 和 `sub`
- `status` 当前建议统一使用 `active`、`paused`、`completed`
- 审核和修改阶段可以在单条剧情线内补充额外字段，例如 `review_suggestions`、`conflict_notes`、`revision_reason`

统一校验入口建议与 `outline` 模块保持一致，单独收敛到剧情线 payload 校验层。

## 工作流

统一工作流保留两种模式：

### 完整生成模式

触发条件：只输入 `NovelID`

执行流程：

```text
START -> init -> generate -> polish -> review -> revise -> saveToDB -> END
```

### 快速修改模式

触发条件：只输入 `PlotlineID + UserSuggestion`

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

说明：

- 输入结构统一包含 `NovelID`、`PlotlineID`、`UserSuggestion`
- 完整生成模式下，`init` 通过 `NovelID` 加载小说和大纲
- 快速修改模式下，`init` 通过 `PlotlineID` 加载目标剧情线，并反查所属 `NovelID`、小说大纲和其他剧情线

## 节点语义

### init

- 初始化工作流状态
- 完整生成模式下从数据库加载 `model.Novel`
- 快速修改模式下从数据库加载目标 `model.Plotline`，并补齐所属小说信息
- 将当前小说大纲、目标剧情线、其他剧情线整理为统一 JSON，供后续 Agent 继续处理

### generate / polish / review / revise

- 每个节点都调用对应 Agent
- 每个节点都会对输出做统一校验和规范化
- 阶段文件仍保存为 `{content, stage, file_type}` 包装结构
- 其中 `content` 保存的是统一 Plotlines JSON 字符串
- 支持断点续传和按节点重试

### review

- 审核 Agent 不再单独输出“冲突列表 JSON”
- 它仍然返回统一 Plotlines JSON
- 审核结果写入单条剧情线的扩展字段，例如 `review_suggestions` 或 `conflict_notes`
- 审核时应优先判断用户建议相关的改动是否会引发冲突，并优先保留用户建议方向

### revise

- 修改 Agent 根据 `review` 阶段结果和用户建议输出最终剧情线集合
- 当只修改单条剧情线时，允许同步调整受影响的其他剧情线
- 若无需调整的剧情线保持原样，也必须以统一 JSON 返回，避免后续保存阶段再做额外拼装

### saveToDB

- 最终只把剧情线核心字段落到 `model.Plotline`
- `Plotline.ID <- id`
- `Plotline.NovelID <- NovelID`
- `Plotline.Title <- title`
- `Plotline.Type <- type`
- `Plotline.Description <- description`
- `Plotline.Status <- status`

因此数据库中只保存核心结构化字段，不直接保存审核扩展字段；扩展字段只作为工作流中间态和阶段文件内容存在。

## 目录结构

```text
internal/plotline/
├── agents/
│   ├── common.go
│   ├── plotline_generator_agent.go
│   ├── plotline_polisher_agent.go
│   ├── plotline_reviewer_agent.go
│   └── plotline_reviser_agent.go
├── compose/
│   ├── agent_executor.go
│   ├── nodes.go
│   ├── plotline_workflow.go
│   └── router.go
├── contracts/
│   └── plotline_payload.go
├── prompts/
│   ├── plotline_generator_prompt.go
│   ├── plotline_polisher_prompt.go
│   ├── plotline_reviewer_prompt.go
│   └── plotline_reviser_prompt.go
├── repository/
│   ├── plotline_repository.go
│   └── repository.go
└── tools/
    └── save_plotline_tool.go
```

## 说明

- 原有 `plotline_extractor` 与独立 `plotline_revision` 工作流最终会统一收敛为一个剧情线工作流
- 原有面向大纲冲突和剧情线互相冲突的零散节点能力，会并入 `review` 和 `revise` 的统一语义
- 旧阶段文件如果仍是历史结构，工作流应尽量兼容读取；新的 Agent 输出必须满足统一 JSON 契约
