# Outline 模块约束

大纲工作流当前采用统一 Novel JSON 输出契约。

## 必须满足

- 每个大纲 Agent 只输出 `title`、`synopsis`、`outline`
- `outline` 必须是 JSON 字符串
- `outline` 内层必须至少包含：
  - `outline`
  - `worldview`
  - `cultivation_system`
  - `core_conflicts`
  - `major_characters`
- 当 `genre` 为 `玄幻` 或 `修真` 时，上述字段都必须完整可用

## 模式约定

工作流保留两种模式：

- 完整生成模式
- 快速预修改模式

快速预修改模式的含义是：

- 输入 `NovelID + Genre + UserSuggestion`
- 从 `polish` 节点进入后续流程
- 这里的 `polish` 节点语义是“按用户建议预修改 + 润色整理”
- `review` 节点继续检查预修改结果是否与用户建议冲突
- 若冲突，审核建议必须优先站在用户建议一侧并要求消除冲突
- `revise` 节点负责落实审核建议

## 数据落盘

- 阶段文件中的 `content` 保存统一 Novel JSON
- 数据库 `model.Novel.Outline` 保存内层 `outline` JSON 字符串

详细说明见 [README.md](/G:/code/waku/internal/outline/README.md)。
