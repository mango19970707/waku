# Chapter 模块

## 概述

Chapter 模块负责小说章节的生成、润色、审核和修改。围绕 4 个 Agent 展开：

- 章节生成 Agent：基于分卷大纲、剧情线和角色信息生成初始章节列表
- 章节润色 Agent：优化章节标题、简介、大纲和节奏表达，同时根据用户建议修改内容
- 章节审核 Agent：检查章节之间的逻辑连续性、节奏分布、剧情承接和设定冲突
- 章节修改 Agent：根据审核建议修改章节内容

## 输入契约

统一工作流输入结构：

```json
{
  "novel_id": 1,
  "volume_id": 2,
  "chapter_id": 15,
  "user_suggestion": "增加主角与师姐的互动，强化感情线铺垫"
}
```

约束如下：

- `novel_id` 为必填，类型为 `uint`
- `volume_id` 为必填，类型为 `uint`（章节必须属于某个分卷）
- `chapter_id` 为可选，类型为 `uint`
- `user_suggestion` 为可选，类型为 `string`
- 当输入只有 `novel_id` 和 `volume_id` 时，走完整工作流（为该卷重新进行章节划分）
- 当输入同时具备 `novel_id`、`volume_id`、`user_suggestion`（无 `chapter_id`）时，从 `init` 直接跳到 `polish`（快速新增章节）
- 当输入同时具备 `novel_id`、`volume_id`、`chapter_id`、`user_suggestion` 时，从 `init` 直接跳到 `polish`（快速修改章节）
- `volume_id` 指向的卷必须属于当前 `novel_id`
- `chapter_id` 指向的章节必须属于当前 `volume_id`

对应代码结构：

```go
type ChapterWorkflowInput struct {
    NovelID        uint
    VolumeID       uint
    ChapterID      uint
    UserSuggestion string
}
```

## 输出契约

### 章节生成/润色/修改 Agent 输出格式

所有章节 Agent（生成、润色、修改）的输出都必须收敛为统一 JSON，顶层只允许 `chapters`：

```json
{
  "chapters": [
    {
      "id": 1,
      "novel_id": 1,
      "volume_id": 2,
      "chapter_number": 1,
      "title": "初入宗门",
      "synopsis": "主角林凡通过考核，正式成为青云宗外门弟子，结识师兄李明和师姐苏婉。",
      "outline": "1. 宗门大门前，主角观察宗门建筑和弟子\n2. 参加入门考核，展现天赋\n3. 获得外门弟子身份\n4. 师兄李明带领参观宗门\n5. 偶遇师姐苏婉，埋下感情线伏笔\n6. 分配到外门第三峰居住",
      "word_count_estimate": 3000,
      "key_events": ["入门考核", "结识李明", "初遇苏婉"],
      "key_characters": ["林凡", "李明", "苏婉"],
      "plotline_progress": {
        "1": "主线剧情：主角正式进入修仙世界",
        "3": "感情线：与苏婉初次相遇"
      },
      "status": "draft"
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `chapters`
- `chapters` 必须是数组，且不能为空
- 每个章节对象最终都必须能映射到 `model.Chapter`
- 至少应包含 `volume_id`、`chapter_number`、`title`、`outline`
- 已有章节在修改流程中必须保留原有 `id`
- 新增章节可以省略 `id`，保存时由数据库自增生成
- `novel_id` 和 `volume_id` 在保存阶段统一写入当前工作流的 `NovelID` 和 `VolumeID`
- `chapter_number` 必须在当前卷内从 1 开始连续且唯一；不同分卷之间的章节号互不影响
- `word_count_estimate` 为预估字数，建议范围 2000-5000
- `key_events` 为本章关键事件列表，用于剧情追踪
- `key_characters` 为本章出场的关键角色列表
- `plotline_progress` 为本章推进的剧情线进度，key 为剧情线 ID，value 为进度描述
- `status` 建议使用 `draft`、`writing`、`completed`
- `created_at`、`updated_at` 不要求 Agent 输出，保存层自动维护
- `deleted_at` 不出现在 Agent 输出中

### 章节审核 Agent 输出格式

章节审核 Agent 输出格式为 `suggestions` 数组，每条建议包含修改建议、修改目标类型和目标ID：

```json
{
  "suggestions": [
    {
      "suggestion": "第3章与第4章之间缺少过渡，主角从外门突然进入内门缺乏铺垫，建议在第3章结尾增加考核预告",
      "target": "chapter",
      "id": 3
    },
    {
      "suggestion": "第5章的节奏过快，主角突破境界缺乏过程描写，建议扩充修炼细节",
      "target": "chapter",
      "id": 5
    },
    {
      "suggestion": "第7章中主角使用的功法与角色设定中的修炼体系不符，建议修改功法描述",
      "target": "character",
      "id": 1
    },
    {
      "suggestion": "第10章的宗门大比情节与剧情线2的发展节奏不匹配，建议调整大比时间点",
      "target": "plotline",
      "id": 2
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `suggestions`
- `suggestions` 必须是数组，可以为空（表示所有章节都合理）
- 每条建议必须包含 `suggestion`（修改建议）、`target`（修改目标类型）、`id`（目标ID）
- `target` 必须是 `outline`、`plotline`、`character`、`volume`、`chapter` 之一
- `id` 必须是对应目标的真实ID
- 如果所有章节都合理，输出空数组：`{"suggestions": []}`

## 数据模型

当前章节模型核心字段如下：

```go
type Chapter struct {
    ID                 uint
    NovelID            uint
    VolumeID           uint
    ChapterNumber      int
    Title              string
    Synopsis           string
    Outline            string
    Content            string          // 章节正文内容（由章节写作工作流生成）
    WordCount          int             // 实际字数
    WordCountEstimate  int             // 预估字数
    KeyEvents          StringArray     // 关键事件列表（JSON 数组）
    KeyCharacters      StringArray     // 关键角色列表（JSON 数组）
    PlotlineProgress   string          // 剧情线进度（JSON 对象）
    Status             string
    CreatedAt          time.Time
    UpdatedAt          time.Time
    DeletedAt          gorm.DeletedAt
}
```

## 工作流

当前统一章节工作流支持三种模式。

### 完整工作流模式

触发条件：只有 `NovelID` 和 `VolumeID`

执行流程：

```text
START -> init -> generate -> polish -> review -> revise -> saveToDB -> END
```

说明：为指定分卷重新进行完整章节划分。章节号按当前分卷从 1 开始连续编号，保存时以本次最终章节列表完全覆盖该分卷旧章节数据。

### 快速新增模式

触发条件：`NovelID + VolumeID + UserSuggestion`（无 `ChapterID`）

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

说明：根据用户建议直接新增章节，跳过生成节点，从润色节点开始。

### 快速修改模式

触发条件：`NovelID + VolumeID + ChapterID + UserSuggestion`

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

说明：修改指定的已有章节，跳过生成节点，从润色节点开始。

## 节点语义

### init

- 加载 `model.Novel`（小说大纲、世界观、修行体系）
- 加载 `model.Volume`（目标分卷的大纲、主题、简介）
- 加载当前小说的全部 `model.Plotline`（剧情线列表）
- 加载当前小说的关键 `model.Character`（主要角色列表）
- 加载当前分卷已有的 `model.Chapter`（已有章节列表）
- 若提供 `ChapterID`，校验目标章节存在且属于当前分卷
- 初始化工作流状态
- 记录小说、分卷、剧情线、角色、已有章节和当前目标章节的上下文

### generate

- 调用章节生成 Agent
- 输入为小说大纲、分卷大纲、剧情线列表、角色列表和已有章节上下文
- 输出初始 `chapters` JSON
- 重新章节划分时，已有章节仅作为上下文参考，不要求保留旧章节的 `id`、章节号或内容
- 支持阶段文件读取，避免重复生成
- 生成结果保存到 `data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/章节规划.json`
- 仅在完整工作流模式下执行

### polish

- 调用章节润色 Agent
- **职责**：
  - 优化章节质量（标题、简介、大纲、节奏）
  - 根据用户建议（`user_suggestion`）修改内容
- **工作模式**：
  - 完整模式下基于生成结果润色
  - 快速新增模式下基于已有章节、分卷大纲和 `user_suggestion` 生成新章节
  - 快速修改模式下基于目标章节、已有章节和 `user_suggestion` 润色和修改
- 润色结果保存到 `data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/润色.json`

### review

- 调用章节审核 Agent
- 检查章节之间的连续性、节奏、剧情承接关系和设定一致性
- 检查章节与剧情线的对应关系
- 检查章节与角色设定的一致性
- **输出格式为 `suggestions` 数组**，包含修改建议、修改目标类型（outline/plotline/character/volume/chapter）和目标ID
- 如果所有章节都合理，输出空数组：`{"suggestions": []}`
- 审核结果保存到 `data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/审核.json`

### revise

- 调用章节修改 Agent
- **职责**：只根据审核建议（`review_suggestions`）修改章节内容
- **不处理用户建议**：用户建议由 polish 节点处理
- 修改范围以目标章节为主，必要时可联动少量相关章节
- 确保章节号连续且唯一
- 修改结果保存到 `data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/修改.json`

### saveToDB

- 对最终章节 JSON 做统一校验
- 校验 `chapter_number` 在当前卷内从 1 开始连续且唯一
- 有 `id` 的章节执行更新
- 无 `id` 的章节执行创建
- 完整工作流模式下，视为重新章节划分：保存前删除当前分卷下的全部旧章节，再按本次最终结果重新创建章节，实现完全覆盖旧内容
- 快速修改模式下，仅更新本次输出涉及的章节，不删除未参与修改的其他章节
- 保存成功后返回章节ID列表

## 上下文提取要求

为避免只看分卷大纲而忽略已有铺垫，章节规划前必须先提取数据库上下文。必要上下文包括：

- `model.Novel`（小说大纲、世界观、修行体系）
- `model.Volume`（目标分卷的大纲、主题、简介）
- `model.Plotline` 列表（全部剧情线）
- `model.Character` 列表（关键角色）
- 已存在的 `model.Chapter` 列表（当前分卷的已有章节）

完整工作流重新章节划分时，已有章节只用于帮助 Agent 理解既有铺垫和避免重复设计，不作为必须保留的数据源；最终保存以新章节列表为准，旧章节会被完全覆盖。

这样可以保证：

- 章节推进与剧情线对应
- 章节内容与角色设定一致
- 章节之间连续且不重复
- 既有章节信息可作为重跑参考

## 路由规则

`init` 之后的路由如下：

```go
func routeAfterInit(ctx context.Context, state *ChapterWorkflowState) (string, error) {
    // 快速新增模式：有小说ID、分卷ID和用户建议，无章节ID（用于新增章节）
    if state.NovelID > 0 && state.VolumeID > 0 && state.UserSuggestion != "" && state.ChapterID == 0 {
        return "polish", nil
    }
    // 快速修改模式：有小说ID、分卷ID、章节ID和用户建议（用于修改已有章节）
    if state.NovelID > 0 && state.VolumeID > 0 && state.ChapterID > 0 && state.UserSuggestion != "" {
        return "polish", nil
    }
    // 完整生成模式：有小说ID和分卷ID，从生成节点开始
    if state.NovelID > 0 && state.VolumeID > 0 {
        return "generate", nil
    }
    return compose.END, fmt.Errorf("输入参数不足，无法确定章节工作流模式")
}
```

其余节点统一采用以下语义：

- 当前节点成功则进入下一节点
- 当前节点失败且未达到最大重试次数，则重试当前节点
- 达到最大重试次数后终止工作流并返回带上下文的错误信息

## 日志输出

所有节点都包含详细的日志输出，参考大纲 outline 工作流的日志格式：

- `workflow.input`: 工作流输入日志
- `workflow.output`: 工作流输出日志
- `node.input`: 节点输入日志（包含当前状态）
- `node.output`: 节点输出日志（包含执行结果）

日志字段包括：
- `workflow`: 工作流名称（chapter）
- `node`: 节点名称（init/generate/polish/review/revise/saveToDB）
- `novel_id`: 小说ID
- `volume_id`: 分卷ID
- `chapter_id`: 章节ID
- `current_node`: 当前节点状态
- `generated_count`: 生成的章节数量
- `polished_count`: 润色的章节数量
- `reviewed_count`: 审核的章节数量
- `revised_count`: 修改的章节数量
- `saved_chapter_ids`: 保存的章节ID列表
- `retry_count`: 重试次数
- `error_msg`: 错误信息

## 阶段文件

阶段文件统一保存到：

```text
data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/
├── 章节规划.json
├── 润色.json
├── 审核.json
└── 修改.json
```

文件结构统一为：

```json
{
  "content": "{...payload json...}",
  "stage": "章节规划",
  "file_type": "chapter"
}
```

## 目录结构

重构后的目标目录结构如下：

```text
internal/chapter/
├── README.md
├── agents/
│   ├── common.go
│   ├── chapter_generator_agent.go
│   ├── chapter_polisher_agent.go
│   ├── chapter_reviewer_agent.go
│   └── chapter_reviser_agent.go
├── compose/
│   ├── agent_executor.go
│   ├── logger.go
│   ├── nodes.go
│   ├── router.go
│   └── chapter_workflow.go
├── contracts/
│   └── chapter_payload.go
├── prompts/
│   ├── chapter_generator_prompt.go
│   ├── chapter_polisher_prompt.go
│   ├── chapter_reviewer_prompt.go
│   └── chapter_reviser_prompt.go
└── repository/
    └── chapter_repository.go
```

## 仓储接口

章节仓储接口至少应包含：

- `CreateBatch(chapters []*model.Chapter) error`
- `GetByID(id uint) (*model.Chapter, error)`
- `GetByVolumeID(volumeID uint) ([]*model.Chapter, error)`
- `GetByNovelID(novelID uint) ([]*model.Chapter, error)`
- `Update(chapter *model.Chapter) error`
- `Delete(id uint) error`
- `DeleteByVolumeID(volumeID uint) error`
- `DeleteByNovelID(novelID uint) error`

## 测试

统一工作流测试至少应覆盖以下场景：

- 完整章节规划工作流模式
- 指定章节修改工作流模式
- 章节上下文提取测试
- JSON 契约校验测试
- 已存在章节时的重跑测试
- 章节号连续性校验测试

至少应覆盖三个核心场景：

- 仅 `NovelID` 和 `VolumeID` 时完整生成章节列表
- 已有部分章节时，基于数据库上下文重跑并保持结果自洽
- 提供 `ChapterID` 和 `UserSuggestion` 时，仅修改目标章节及必要关联章节

## 章节规划原则

### 1. 章节数量规划

- 每卷建议 8-15 章
- 根据分卷大纲的内容量和复杂度调整
- 避免单章内容过多或过少

### 2. 章节节奏控制

- 开篇章节：快节奏，吸引读者
- 中间章节：张弛有度，推进剧情
- 高潮章节：密集冲突，情绪饱满
- 收尾章节：留有余韵，埋下伏笔

### 3. 剧情线分配

- 每章至少推进 1 条剧情线
- 主线剧情线应在每章都有体现
- 支线剧情线可以间隔出现
- 避免单章推进过多剧情线导致混乱

### 4. 角色出场安排

- 主角应在每章都有戏份
- 重要配角应定期出场
- 新角色出场应有铺垫
- 避免角色突然消失或突然出现

### 5. 章节标题规范

- 标题应简洁有力，4-8 字为宜
- 标题应体现本章核心事件或主题
- 避免剧透关键情节
- 保持标题风格统一

### 6. 字数预估

- 常规章节：2500-3500 字
- 重要章节：3500-5000 字
- 过渡章节：2000-2500 字
- 高潮章节：4000-6000 字

## 与其他模块的关系

### 依赖关系

- **依赖 Outline 模块**：需要小说大纲、世界观、修行体系
- **依赖 Volume 模块**：需要分卷大纲、主题、简介
- **依赖 Plotline 模块**：需要剧情线列表和进度
- **依赖 Character 模块**：需要角色列表和设定

### 被依赖关系

- **被 Content 模块依赖**：章节写作需要章节大纲
- **被 Review 模块依赖**：内容审核需要章节结构

## 注意事项

1. **章节号必须连续**：章节号从 1 开始，在当前卷内连续且唯一；每个分卷独立编号，不沿用其他分卷的章节号
2. **剧情线进度追踪**：每章必须记录推进的剧情线和进度
3. **角色出场记录**：每章必须记录出场的关键角色
4. **字数预估合理**：字数预估应符合实际写作习惯
5. **上下文完整**：生成章节前必须加载完整的上下文信息
6. **修改范围控制**：快速修改模式下，仅修改必要的章节，避免大范围改动
7. **完整划分覆盖**：完整工作流属于重新章节划分，会以本次最终结果完全覆盖当前分卷旧章节
8. **文件路径规范**：阶段文件保存到 `volume_{volumeNumber}/chapter/` 目录下
9. **JSON 格式严格**：所有 Agent 输出必须符合契约定义的 JSON 格式
