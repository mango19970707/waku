# Content 模块

## 概述

Content 模块负责小说章节正文的生成、润色、审核和修改。正文写作以 Chapter 模块产出的章节规划为基础，默认按单章执行，避免一次性加载整卷正文导致 token 消耗过高。模块围绕 4 个 Agent 展开：

- 正文生成 Agent：基于小说设定、分卷信息、章节大纲和必要上下文生成章节正文
- 正文润色 Agent：优化正文表达、节奏、对话、画面感和风格一致性，并可根据用户建议修改内容
- 正文审核 Agent：检查正文是否符合章节大纲、设定约束、上下文承接和内容质量要求
- 正文修改 Agent：根据审核建议对正文进行定向修改

## 输入契约

统一工作流输入结构：

```json
{
  "novel_id": 1,
  "volume_id": 2,
  "chapter_id": 15,
  "chapter_ids": [15, 16, 17],
  "user_suggestion": "强化主角突破时的压迫感，增加师姐在旁护法的细节"
}
```

约束如下：

- `novel_id` 为必填，类型为 `uint`
- `volume_id` 为必填，类型为 `uint`，正文必须归属到某个分卷
- `chapter_id` 为可选，类型为 `uint`，用于指定单章正文生成或修改
- `chapter_ids` 为可选，类型为 `[]uint`，用于批量生成多个指定章节
- `user_suggestion` 为可选，类型为 `string`，用于补充写作方向或修改要求
- 当输入具备 `novel_id`、`volume_id`、`chapter_id`，且目标章节无正文时，走单章完整正文工作流
- 当输入具备 `novel_id`、`volume_id`、`chapter_id`、`user_suggestion`，且目标章节已有正文时，走单章快速修改工作流
- 当输入具备 `novel_id`、`volume_id`、`chapter_ids` 时，按章节顺序逐章执行单章工作流，不建议一次性并发生成整卷正文
- 当输入只有 `novel_id` 和 `volume_id` 时，默认生成当前卷中正文为空的章节，并按章节号顺序串行执行
- `volume_id` 指向的卷必须属于当前 `novel_id`
- `chapter_id` 和 `chapter_ids` 指向的章节必须属于当前 `volume_id`
- `chapter_id` 与 `chapter_ids` 不应同时传入；如同时传入，优先返回参数错误

对应代码结构规划：

```go
type ContentWorkflowInput struct {
    NovelID        uint
    VolumeID       uint
    ChapterID      uint
    ChapterIDs     []uint
    UserSuggestion string
}
```

## 输出契约

### 正文生成/润色/修改 Agent 输出格式

正文生成、润色、修改 Agent 的输出都必须收敛为统一 JSON，顶层只允许 `chapter`。其中润色 Agent 不只是语言优化，也可以在 `user_suggestion` 存在时直接按用户建议修改正文内容：

```json
{
  "chapter": {
    "id": 15,
    "novel_id": 1,
    "volume_id": 2,
    "chapter_number": 3,
    "title": "夜入丹阁",
    "content": "夜色沉入青云宗后山时，丹阁外的铜铃忽然无风自响……",
    "word_count": 3200,
    "status": "writing"
  }
}
```

约束如下：

- 顶层字段只能是 `chapter`
- `chapter` 必须是对象，且最终能映射到 `model.Chapter`
- `id`、`novel_id`、`volume_id`、`chapter_number` 必须保留原章节数据
- `title` 默认沿用章节规划标题，除非用户明确要求修改标题
- `content` 为最终正文，不能为空
- `word_count` 必须根据 `content` 实际统计，保存前可由服务层重新计算覆盖
- `status` 建议使用 `writing` 或 `completed`，首次生成正文后默认进入 `writing`
- `synopsis`、`outline`、`word_count_estimate`、`key_events`、`key_characters`、`plotline_progress` 默认不由正文 Agent 修改
- `created_at`、`updated_at` 不要求 Agent 输出，保存层自动维护
- `deleted_at` 不出现在 Agent 输出中

### 正文审核 Agent 输出格式

正文审核 Agent 输出格式为 `suggestions` 数组，每条建议包含修改建议、修改目标类型和目标 ID：

```json
{
  "suggestions": [
    {
      "suggestion": "正文第4段突破过程过快，缺少灵力失控到稳定的过渡，建议补充两段修炼细节。",
      "target": "content",
      "id": 15
    },
    {
      "suggestion": "师姐苏婉的台词语气与角色设定偏差较大，建议改得更克制冷静。",
      "target": "character",
      "id": 3
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `suggestions`
- `suggestions` 必须是数组，可以为空，表示正文审核通过
- 每条建议必须包含 `suggestion`、`target`、`id`
- `target` 必须是 `content`、`chapter`、`character`、`plotline`、`volume`、`outline` 之一
- 当 `target` 为 `content` 或 `chapter` 时，`id` 为章节 ID
- 建议必须具体可执行，避免只给出“加强描写”“优化节奏”等泛化描述
- 审核建议只用于驱动正文修改，不直接写入数据库

## 工作流设计

### 完整单章正文工作流

```text
init -> generate -> polish -> review -> route -> revise -> saveToDB
```

节点职责如下：

- `init`：校验输入，加载小说、分卷、目标章节、角色、剧情线和相邻章节上下文
- `generate`：根据章节规划生成初始正文
- `polish`：统一文风，优化表达、节奏、对话和段落衔接；当存在用户建议时，同时按建议修改正文内容
- `review`：检查正文质量、设定一致性、剧情承接和章节规划覆盖度
- `route`：若审核无建议则进入保存；若存在建议则进入修改
- `revise`：按审核建议定向修改正文，并保持章节 ID 与结构信息不变
- `saveToDB`：只更新目标章节的 `Content`、`WordCount`、`Status` 和 `UpdatedAt`

### 快速修改工作流

```text
init -> polish -> review -> route -> revise -> saveToDB
```

适用场景如下：

- 用户对已有正文提出修改建议
- 章节正文存在，但需要按最新角色、剧情线或分卷设定修正
- 审核发现局部问题，需要低成本重写目标片段

快速修改约束如下：

- 必须加载目标章节原正文
- 默认只修改目标章节正文，不修改章节规划字段
- 用户建议优先级高于审核建议，优先在 `polish` 节点处理；如审核后仍有问题，再由 `revise` 节点按审核建议修正
- 修改时尽量保留原正文中未受影响的段落，减少重复生成成本

### 批量正文工作流

```text
initBatch -> foreachChapter -> singleChapterWorkflow -> collectResult
```

设计原则如下：

- 批量模式只是单章工作流的顺序编排，不新增一次性整卷生成 Agent
- 每次只把当前章节、上一章摘要、下一章大纲和必要设定传给 Agent
- 当前章节保存成功后，再进入下一章，确保后续章节能读取最新上下文
- 单章失败时停止批量流程，并返回失败章节 ID 和错误信息
- 不建议并发生成相邻章节，避免上下文竞争和剧情承接不一致

## 上下文加载策略

为减少 token 消耗，正文工作流只加载与目标章节强相关的信息：

- 小说上下文：小说名称、类型、核心大纲、世界观、修行体系等高层设定
- 分卷上下文：当前卷标题、主题、简介、分卷大纲
- 章节上下文：目标章节标题、简介、大纲、关键事件、关键角色、剧情线进度、预估字数
- 相邻章节：上一章正文摘要、上一章结尾、下一章标题和大纲
- 角色上下文：只加载目标章节 `key_characters` 对应的角色设定和关系
- 剧情线上下文：只加载目标章节 `plotline_progress` 涉及的剧情线
- 历史正文：默认不加载整卷正文；必要时只加载前 1 章正文摘要和结尾片段

上下文压缩规则如下：

- 已完成正文优先转为摘要后传入 Agent
- 章节大纲优先于完整正文，完整正文只用于快速修改目标章节
- 角色信息只保留名称、定位、性格、当前关系和本章相关设定
- 剧情线信息只保留目标章节前后的进度，不传入无关支线
- 用户建议原文必须完整传入，不做摘要压缩

## 阶段文件

每次工作流执行都应参考分卷工作流的实现，在生成、润色、审核、修改等节点内部保存 Agent 阶段输出，便于调试和回溯；阶段文件保存不需要设计为独立工作流节点。

单章正文阶段文件路径：

```text
volume_{volumeNumber}/content/chapter_{chapterNumber}/
├── 生成.json
├── 润色.json
├── 审核.json
└── 修改.json
```

批量正文阶段文件路径：

```text
volume_{volumeNumber}/content/
├── chapter_1/
│   ├── 生成.json
│   ├── 润色.json
│   ├── 审核.json
│   └── 修改.json
└── chapter_2/
    ├── 生成.json
    ├── 润色.json
    ├── 审核.json
    └── 修改.json
```

文件结构统一为：

```json
{
  "content": "{...payload json...}",
  "stage": "正文生成",
  "file_type": "content"
}
```

## 目录结构

目标目录结构如下：

```text
internal/content/
├── README.md
├── agents/
│   ├── common.go
│   ├── content_generator_agent.go
│   ├── content_polisher_agent.go
│   ├── content_reviewer_agent.go
│   └── content_reviser_agent.go
├── compose/
│   ├── agent_executor.go
│   ├── logger.go
│   ├── nodes.go
│   ├── router.go
│   └── content_workflow.go
├── contracts/
│   └── content_payload.go
├── prompts/
│   ├── content_generator_prompt.go
│   ├── content_polisher_prompt.go
│   ├── content_reviewer_prompt.go
│   └── content_reviser_prompt.go
└── repository/
    └── content_repository.go
```

## 仓储接口

正文仓储接口至少应包含：

- `GetChapterByID(id uint) (*model.Chapter, error)`
- `GetChaptersByVolumeID(volumeID uint) ([]*model.Chapter, error)`
- `GetEmptyContentChapters(volumeID uint) ([]*model.Chapter, error)`
- `UpdateContent(chapterID uint, content string, wordCount int, status string) error`
- `UpdateContentBatch(chapters []*model.Chapter) error`
- `GetPreviousChapter(chapterID uint) (*model.Chapter, error)`
- `GetNextChapter(chapterID uint) (*model.Chapter, error)`

## 测试

统一工作流测试至少应覆盖以下场景：

- 单章正文完整生成工作流
- 已有正文快速修改工作流
- 批量生成空正文章节工作流
- 上下文压缩与相邻章节提取测试
- JSON 契约校验测试
- 正文保存只更新允许字段测试
- 审核建议为空时跳过修改节点测试
- 批量流程中单章失败后停止测试

至少应覆盖三个核心场景：

- 提供 `NovelID`、`VolumeID`、`ChapterID` 时，基于章节规划生成目标章节正文
- 提供 `ChapterID` 和 `UserSuggestion` 时，仅修改目标章节正文并保留章节规划字段
- 仅提供 `NovelID` 和 `VolumeID` 时，按章节号顺序生成当前卷中正文为空的章节

## 正文写作原则

### 1. 大纲覆盖

- 正文必须覆盖章节大纲中的核心事件
- 不得擅自跳过关键剧情节点
- 可以补充细节、动作、心理和环境描写，但不能改变章节目标
- 如用户建议与章节大纲冲突，应优先返回可理解的中文错误或审核建议

### 2. 上下文承接

- 开头应承接上一章结尾的时间、地点、人物和情绪状态
- 结尾应自然导向下一章大纲或留下合理悬念
- 不得让角色、道具、境界、伤势等状态突然变化
- 不得重复上一章已经完成的关键事件

### 3. 人物一致性

- 角色行为必须符合角色设定、关系和当前剧情阶段
- 对话语气应体现角色身份、性格和情绪
- 主角能力成长必须有过程，不应无铺垫突破
- 配角出场应服务本章剧情，不应无意义堆砌

### 4. 剧情节奏

- 常规章节应保持铺垫、推进、转折、收束的基本结构
- 高潮章节应增加冲突密度和情绪张力
- 过渡章节应控制信息量，避免纯说明文
- 单章内不应同时推进过多无关剧情线

### 5. 文风控制

- 同一小说应保持叙事视角、语言风格和节奏稳定
- 玄幻、修仙、都市、悬疑等题材应匹配对应表达习惯
- 避免模板化句式和重复形容词
- 避免现代网络口吻误入古风或严肃题材

### 6. 字数控制

- 正文应尽量贴近章节 `word_count_estimate`
- 常规允许偏差建议控制在 ±20%
- 若用户指定字数，以用户指定为准
- 字数不足时优先补充冲突过程、心理变化和场景动作
- 字数过多时优先压缩重复描写和无关对白

## 与其他模块的关系

### 依赖关系

- **依赖 Outline 模块**：需要小说大纲、世界观、修行体系和整体风格
- **依赖 Volume 模块**：需要当前分卷主题、简介和分卷大纲
- **依赖 Chapter 模块**：需要章节标题、简介、大纲、关键事件和预估字数
- **依赖 Plotline 模块**：需要目标章节涉及的剧情线和阶段进度
- **依赖 Character 模块**：需要目标章节涉及角色的设定和关系

### 被依赖关系

- **被 Review 模块依赖**：内容审核需要读取章节正文
- **被后续章节依赖**：后续正文生成需要上一章摘要和结尾片段
- **被导出模块依赖**：小说导出需要读取章节正文并按卷章顺序拼接

## 注意事项

1. **默认单章生成**：正文生成默认以章节为最小单位，避免一次性生成整卷正文造成 token 浪费和上下文漂移
2. **只更新正文相关字段**：正文保存默认只更新 `Content`、`WordCount`、`Status` 和 `UpdatedAt`，不改章节规划字段
3. **上下文必须压缩**：除快速修改目标章节外，不应传入大段历史正文
4. **章节顺序必须稳定**：批量生成必须按 `chapter_number` 升序执行，确保上下文承接正确
5. **正文不能为空**：Agent 输出的 `content` 为空时不得保存，应返回中文错误信息
6. **字数必须复核**：保存前应由服务层重新统计 `word_count`，不要完全信任 Agent 返回值
7. **用户建议优先**：用户建议应完整传入 `polish` 节点优先处理，但不得覆盖基础设定和章节归属约束
8. **审核建议可为空**：审核通过时应直接保存润色结果，不应强制进入修改节点
9. **阶段文件可回溯**：每个阶段文件必须包含原始 Agent 输出，并参考分卷工作流在对应节点内保存，便于定位 JSON 格式和内容质量问题

