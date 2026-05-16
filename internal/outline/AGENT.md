# Outline 模块约束

大纲工作流当前采用统一 Novel JSON 输出契约。

## 必须满足

- 每个大纲 Agent 只输出 `title`、`synopsis`、`outline`
- `outline` 必须是 JSON 字符串
- `outline` 内层必须至少包含 `outline`
- `genre` 为 `玄幻` 或 `修真` 时，内层还必须包含 `worldview` 和 `cultivation_system`

## 数据落盘

- 阶段文件中的 `content` 保存统一 Novel JSON
- 数据库 `model.Novel.Outline` 保存内层 `outline` JSON 字符串

详细说明见 [README.md](/G:/code/waku/internal/outline/README.md)。
