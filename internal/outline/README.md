# Outline 模块

## 概述

Outline 模块负责小说大纲的生成、扩写、润色、审核和修改。模块保留原有的 Agent 创建方式和 `compose.Graph` 工作流结构，但统一了所有大纲 Agent 的输出契约。

当前仅要求兼容 `玄幻` 和 `修真` 两类小说。

## 统一输出契约

每个大纲 Agent 都必须输出严格 JSON，顶层只允许这 3 个字段：

```json
{
  "title": "小说标题",
  "synopsis": "小说简介",
  "outline": "{\"outline\":\"大纲正文\",\"worldview\":\"世界观\",\"cultivation_system\":\"修行体系\"}"
}
```

约束如下：

- 顶层字段只能是 `title`、`synopsis`、`outline`
- `outline` 必须是 JSON 字符串，不能直接输出对象
- `outline` 内层 JSON 至少必须包含 `outline`
- 当 `genre` 包含 `玄幻` 或 `修真` 时，`outline` 内层 JSON 还必须包含 `worldview` 和 `cultivation_system`
- 内层 JSON 可以继续补充其他字段，例如 `story_arcs`、`major_characters`、`core_conflicts`

统一校验入口位于 [novel_payload.go](/G:/code/waku/internal/outline/contracts/novel_payload.go)。

## 工作流

统一工作流仍然保留两种模式：

### 完整生成模式

触发条件：输入 `Genre + OriginalOutline`

执行流程：

```text
START -> init -> generate -> expand -> polish -> review -> revise -> saveToDB -> END
```

### 快速修改模式

触发条件：输入 `NovelID + Genre + UserSuggestion`

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

## 节点语义

### init

- 初始化工作流状态
- 快速修改模式下从数据库加载 `model.Novel`
- 将数据库中的 `Title`、`Synopsis`、`Outline` 重新组装成统一 Novel JSON，供后续 Agent 继续处理

### generate / expand / polish / review / revise

- 每个节点都调用对应 Agent
- 每个节点都会使用 `NormalizeNovelPayload(...)` 对输出做统一校验和规范化
- 阶段文件仍保存为 `{content, stage, file_type}` 包装结构
- 其中 `content` 保存的是统一 Novel JSON 字符串
- 支持断点续传和按节点重试

### review

- 审核 Agent 不再单独返回“审核建议列表 JSON”
- 它仍然返回统一 Novel JSON
- 审核建议写入 `outline` 内层 JSON 的 `review_suggestions` 字段

### saveToDB

- 最终只把 3 个核心字段落到 `model.Novel`
- `Novel.Title <- title`
- `Novel.Synopsis <- synopsis`
- `Novel.Outline <- outline` 内层 JSON 字符串

因此数据库中的 `Novel.Outline` 不是顶层 Novel JSON，而是其中的内层 `outline` JSON 字符串。

## 目录结构

```text
internal/outline/
├── agents/
│   ├── common.go
│   ├── outline_generator_agent.go
│   ├── outline_expander_agent.go
│   ├── outline_polisher_agent.go
│   ├── outline_reviewer_agent.go
│   └── outline_reviser_agent.go
├── compose/
│   ├── agent_executor.go
│   ├── nodes.go
│   ├── outline_generate_workflow.go
│   └── router.go
├── contracts/
│   └── novel_payload.go
├── prompts/
│   ├── outline_generator_prompt.go
│   ├── outline_expander_prompt.go
│   ├── outline_polisher_prompt.go
│   ├── outline_reviewer_prompt.go
│   └── outline_reviser_prompt.go
├── repository/
│   └── repository.go
└── tools/
    └── save_outline_tool.go
```

## 说明

- 原有 `outline_extractor_agent` 与提取节点逻辑已经移除
- 原有独立修订工作流已移除，统一收敛到一个大纲工作流
- 旧阶段文件如果仍是纯文本，工作流会尽量做兼容转换；新的 Agent 输出必须满足统一 JSON 契约
