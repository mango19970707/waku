# Volume 模块

## 概述

Volume 模块负责小说分卷的生成、润色、审核和修改。围绕 4 个 Agent 展开：

- 分卷生成 Agent：基于小说大纲和全部剧情线生成初始分卷列表
- 分卷润色 Agent：优化分卷标题、简介、主题和结构表达
- 分卷审核 Agent：检查分卷之间的逻辑连续性、节奏分布和设定冲突
- 分卷修改 Agent：根据审核结果和用户建议输出最终分卷列表

## 输入契约

统一工作流输入结构：

```json
{
  "novel_id": 1,
  "volume_id": 2,
  "user_suggestion": "强化第二卷的宗门冲突，压缩铺垫篇幅"
}
```

约束如下：

- `novel_id` 为必填，类型为 `uint`
- `volume_id` 为可选，类型为 `uint`
- `user_suggestion` 为可选，类型为 `string`
- 当输入只有 `novel_id` 时，走完整工作流
- 当输入同时具备 `novel_id`、`volume_id`、`user_suggestion` 时，从 `init` 直接跳到 `polish`
- `volume_id` 指向的卷必须属于当前 `novel_id`

对应代码结构：

```go
type VolumeWorkflowInput struct {
    NovelID        uint
    VolumeID       uint
    UserSuggestion string
}
```

## 输出契约

### 分卷生成/润色/修改 Agent 输出格式

所有分卷 Agent（生成、润色、修改）的输出都必须收敛为统一 JSON，顶层只允许 `volumes`：

```json
{
  "volumes": [
    {
      "id": 1,
      "novel_id": 1,
      "volume_number": 1,
      "title": "少年出山",
      "synopsis": "主角离开边陲小镇，正式卷入宗门与王朝的冲突。",
      "theme": "成长与入局",
      "outline": "主角完成世界观进入、初次结识核心势力，并埋下全书主线伏笔。",
      "status": "draft"
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `volumes`
- `volumes` 必须是数组，且不能为空
- 每个卷对象最终都必须能映射到 `model.Volume`
- 至少应包含 `volume_number`、`title`、`outline`
- 已有卷在修改流程中必须保留原有 `id`
- 新增卷可以省略 `id`，保存时由数据库自增生成
- `novel_id` 在保存阶段统一写入当前工作流的 `NovelID`
- `status` 建议使用 `draft`、`writing`、`completed`
- `created_at`、`updated_at` 不要求 Agent 输出，保存层自动维护
- `deleted_at` 不出现在 Agent 输出中

### 分卷审核 Agent 输出格式

分卷审核 Agent 输出格式为 `suggestions` 数组，每条建议包含修改建议、修改目标类型和目标ID：

```json
{
  "suggestions": [
    {
      "suggestion": "第一卷容量过大，建议拆分为两卷，第一卷聚焦主角成长，第二卷聚焦势力冲突",
      "target": "volume",
      "id": 1
    },
    {
      "suggestion": "第二卷与第三卷之间缺少过渡，建议在第二卷结尾增加伏笔",
      "target": "volume",
      "id": 2
    },
    {
      "suggestion": "分卷划分与剧情线3的发展节奏不匹配，建议调整第三卷的大纲",
      "target": "plotline",
      "id": 3
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `suggestions`
- `suggestions` 必须是数组，可以为空（表示所有分卷都合理）
- 每条建议必须包含 `suggestion`（修改建议）、`target`（修改目标类型）、`id`（目标ID）
- `target` 必须是 `outline`、`plotline`、`character`、`volume` 之一
- `id` 必须是对应目标的真实ID
- 如果所有分卷都合理，输出空数组：`{"suggestions": []}`

## 数据模型

当前分卷模型核心字段如下：

```go
type Volume struct {
    ID           uint
    NovelID      uint
    VolumeNumber int
    Title        string
    Synopsis     string
    Theme        string
    Outline      string
    Status       string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

## 工作流

当前统一分卷工作流支持三种模式。

### 完整工作流模式

触发条件：只有 `NovelID`

执行流程：

```text
START -> init -> generate -> polish -> review -> revise -> saveToDB -> END
```

### 快速新增模式

触发条件：`NovelID + UserSuggestion`（无 `VolumeID`）

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

说明：根据用户建议直接新增分卷，跳过生成节点，从润色节点开始。

### 快速修改模式

触发条件：`NovelID + VolumeID + UserSuggestion`

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

说明：修改指定的已有分卷，跳过生成节点，从润色节点开始。

## 节点语义

### init

- 加载 `model.Novel`
- 加载当前小说已有 `model.Plotline`
- 加载当前小说已有 `model.Volume`
- 若提供 `VolumeID`，校验目标卷存在且属于当前小说
- 初始化工作流状态
- 记录大纲、剧情线、已有卷和当前目标卷的上下文

### generate

- 调用分卷生成 Agent
- 输入为小说大纲、全部剧情线和已有卷上下文
- 输出初始 `volumes` JSON
- 支持阶段文件读取，避免重复生成
- 生成结果保存到 `data/content/{author}/{novelName}/volume/分卷.json`
- 仅在完整工作流模式下执行

### polish

- 调用分卷润色 Agent
- 完整模式下基于生成结果润色
- 快速修改模式下基于目标卷、已有卷和 `user_suggestion` 润色

### review

- 调用分卷审核 Agent
- 检查卷与卷之间的连续性、节奏、主题一致性和剧情承接关系
- **输出格式为 `suggestions` 数组**，包含修改建议、修改目标类型（outline/plotline/character/volume）和目标ID
- 如果所有分卷都合理，输出空数组：`{"suggestions": []}`
- 审核结果保存到 `data/content/{author}/{novelName}/volume/审核.json`

### revise

- 调用分卷修改 Agent
- 基于审核后的分卷数据和用户建议输出最终分卷列表
- 修改范围以目标卷为主，必要时可联动少量相关卷

### saveToDB

- 对最终分卷 JSON 做统一校验
- 有 `id` 的卷执行更新
- 无 `id` 的卷执行创建
- 完整工作流模式下，数据库中已存在但不在最终结果里的卷会被删除
- 快速修改模式下，仅更新本次输出涉及的卷，不删除未参与修改的其他卷

## 上下文提取要求

为避免只看大纲而忽略已有铺垫，分卷前必须先提取数据库上下文。必要上下文包括：

- `model.Novel`
- `model.Plotline` 列表
- 已存在的 `model.Volume` 列表

这样可以保证：

- 各卷主题推进连续
- 分卷不会脱离剧情线
- 既有分卷信息可作为重跑参考

## 路由规则

`init` 之后的路由如下：

```go
func routeAfterInit(ctx context.Context, state *VolumeWorkflowState) (string, error) {
    // 快速修改模式：只有小说ID和用户建议，直接跳到润色节点（用于新增卷）
    if state.NovelID > 0 && state.UserSuggestion != "" && state.VolumeID == 0 {
        return "polish", nil
    }
    // 修订模式：有小说ID、分卷ID和用户建议（用于修改已有卷）
    if state.RevisionMode {
        return "polish", nil
    }
    // 完整生成模式：有小说ID，从生成节点开始
    if state.NovelID > 0 {
        return "generate", nil
    }
    return compose.END, fmt.Errorf("输入参数不足，无法确定分卷工作流模式")
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
- `workflow`: 工作流名称（volume）
- `node`: 节点名称（init/generate/polish/review/revise/saveToDB）
- `novel_id`: 小说ID
- `volume_id`: 分卷ID
- `current_node`: 当前节点状态
- `generated_chars`: 生成阶段字符数
- `polished_chars`: 润色阶段字符数
- `reviewed_chars`: 审核阶段字符数
- `revised_chars`: 修改阶段字符数
- `saved_volume_ids`: 保存的分卷ID列表
- `retry_count`: 重试次数
- `error_msg`: 错误信息

## 阶段文件

阶段文件统一保存到：

```text
data/content/{author}/{novelName}/volume/
├── 分卷.json
├── 润色.json
├── 审核.json
└── 修改.json
```

文件结构统一为：

```json
{
  "content": "{...payload json...}",
  "stage": "分卷",
  "file_type": "volume"
}
```

## 目录结构

重构后的目标目录结构如下：

```text
internal/volume/
├── README.md
├── agents/
│   ├── common.go
│   ├── volume_generator_agent.go
│   ├── volume_polisher_agent.go
│   ├── volume_reviewer_agent.go
│   └── volume_reviser_agent.go
├── compose/
│   ├── agent_executor.go
│   ├── logger.go
│   ├── nodes.go
│   ├── router.go
│   └── volume_workflow.go
├── contracts/
│   └── volume_payload.go
├── prompts/
│   ├── volume_generator_prompt.go
│   ├── volume_polisher_prompt.go
│   ├── volume_reviewer_prompt.go
│   └── volume_reviser_prompt.go
└── repository/
    └── volume_repository.go
```

## 仓储接口

卷仓储接口至少应包含：

- `CreateBatch(volumes []*model.Volume) error`
- `GetByID(id uint) (*model.Volume, error)`
- `GetByNovelID(novelID uint) ([]*model.Volume, error)`
- `Update(volume *model.Volume) error`
- `Delete(id uint) error`
- `DeleteByNovelID(novelID uint) error`

## 测试

统一工作流测试至少应覆盖以下场景：

- 完整分卷工作流模式
- 指定卷修改工作流模式
- 分卷上下文提取测试
- JSON 契约校验测试
- 已存在分卷时的重跑测试

至少应覆盖三个核心场景：

- 仅 `NovelID` 时完整生成分卷
- 已有部分分卷时，基于数据库上下文重跑并保持结果自洽
- 提供 `VolumeID` 和 `UserSuggestion` 时，仅修改目标卷及必要关联卷
