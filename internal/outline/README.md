# Outline 模块

## 概览

Outline 模块负责小说大纲的生成、扩写、预修改润色、审核和修改。模块保留原有的 Agent 创建方式和 `compose.Graph` 工作流结构，同时统一了所有大纲 Agent 的输入输出契约。

当前优先支持 `玄幻` 和 `修真` 两类小说。

## 统一输出契约

每个大纲 Agent 都必须输出严格 JSON，顶层只允许这 3 个字段：

```json
{
  "title": "小说标题",
  "synopsis": "小说简介",
  "outline": "{\"outline\":\"大纲正文\",\"worldview\":\"世界观\",\"cultivation_system\":\"修行体系\",\"core_conflicts\":\"核心冲突\",\"major_characters\":\"主要人物\"}"
}
```

约束如下：

- 顶层字段只能是 `title`、`synopsis`、`outline`
- `outline` 必须是 JSON 字符串，不能直接输出对象
- `outline` 内层 JSON 必须稳定包含：
  - `outline`
  - `worldview`
  - `cultivation_system`
  - `core_conflicts`
  - `major_characters`
- 当 `genre` 包含 `玄幻` 或 `修真` 时，上述 5 个字段都不能为空
- 内层 JSON 可以继续补充其他字段，例如 `story_highlights`、`themes`、`foreshadowing`

统一校验入口位于 [novel_payload.go](/G:/code/waku/internal/outline/contracts/novel_payload.go)。

## 工作流

统一工作流保留两种模式：

### 完整生成模式

触发条件：输入 `Genre + OriginalOutline`

执行流程：

```text
START -> init -> generate -> expand -> polish -> review -> revise -> saveToDB -> END
```

### 快速预修改模式

触发条件：输入 `NovelID + Genre + UserSuggestion`

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

说明：

- 这里的 `polish` 节点在该模式下不是单纯文风润色。
- 它承担“按用户建议预修改 + 润色整理”的职责。
- `review` 节点会继续检查预修改结果是否与用户建议冲突；如果冲突，审核建议会明确要求优先采用用户建议并消除冲突。
- `revise` 节点主要负责落实审核建议，而不是再次直接消费原始用户建议。

## 节点语义

### init

- 初始化工作流状态
- 在快速预修改模式下从数据库加载 `model.Novel`
- 将数据库中的 `Title`、`Synopsis`、`Outline` 重新组装成统一 Novel JSON，供后续 Agent 继续处理

### generate / expand

- 调用对应 Agent
- 使用 `NormalizeNovelPayload(...)` 统一校验和规范化输出
- 阶段文件保存为 `{content, stage, file_type}` 包装结构
- 其中 `content` 保存的是统一 Novel JSON 字符串
- 支持断点续传和按节点重试

### polish

- 在完整生成模式下：负责常规润色与结构整理
- 在快速预修改模式下：负责“按用户建议预修改 + 润色整理”
- 当存在 `user_suggestion` 时，优先吸收用户建议，并联动修正受影响的设定、人物、冲突和阶段推进

### review

- 审核 Agent 只返回修改建议，不再返回修改后的大纲
- 审核时会检查当前结果是否与 `user_suggestion` 冲突
- 如果冲突，审核建议会明确要求优先采用用户建议并消除冲突
- 审核结果写入工作流状态中的 `ReviewSuggestions`

### revise

- 基于最新大纲和审核建议执行定向修改
- 重点是落实审核建议，并消除预修改后仍然残留的不一致问题

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
- 原有独立修订工作流已经移除，统一收敛到一个大纲工作流
- 旧阶段文件如果仍是纯文本，工作流会尽量做兼容转换；新的 Agent 输出必须满足统一 JSON 契约
