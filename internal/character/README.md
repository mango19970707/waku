# Character 模块

## 概述

Character 模块负责小说角色的提取、润色、审核和修改。当前实现参考 `internal/plotline` 的统一工作流架构，已经删除旧的“角色提取工作流 / 角色修改工作流”双轨实现，收敛为一个统一的角色工作流，并围绕 4 个 Agent 展开：

- 角色提取 Agent：从小说大纲和已有角色上下文中提取或补全角色集合
- 角色润色 Agent：优化角色设定表达、信息完整性和人物辨识度
- 角色审核 Agent：审查角色设定是否与大纲、关系和用户建议冲突
- 角色修改 Agent：根据审核结果和用户建议输出最终角色集合

## 输入契约

统一工作流输入结构：

```json
{
  "novel_id": 1,
  "character_id": 2,
  "user_suggestion": "强化反派压迫感"
}
```

约束如下：

- `novel_id` 为必填，类型为 `uint`
- `character_id` 为可选，类型为 `uint`
- `user_suggestion` 为可选，类型为 `string`
- 当输入只有 `novel_id` 时，走完整工作流
- 当输入同时具备 `novel_id`、`character_id`、`user_suggestion` 时，从 `init` 直接跳到 `polish`

对应代码结构：

```go
type CharacterWorkflowInput struct {
    NovelID        uint
    CharacterID    uint
    UserSuggestion string
}
```

## 输出契约

所有角色 Agent 的输出都必须是统一 JSON，顶层只允许 `characters`：

```json
{
  "characters": [
    {
      "id": 2,
      "novel_id": 1,
      "name": "沈夜",
      "role": "antagonist",
      "age": 28,
      "gender": "男",
      "appearance": "身形清瘦，常着玄色长袍，目光阴冷。",
      "personality": "隐忍、多疑、掌控欲强。",
      "background": "出身没落世家，为振兴家族不择手段。",
      "plotline_ids": [1, 3],
      "relationships": {
        "1": "与主角敌对",
        "4": "与宗门长老互相利用"
      }
    }
  ]
}
```

约束如下：

- 顶层字段只能是 `characters`
- `characters` 必须是数组，且不能为空
- 每个角色对象最终要能映射到 [character.go]的 `model.Character`
- 至少应包含 `name`、`personality`、`background`
- 已有角色在修改流程中必须保留原有 `id`
- 新增角色可以省略 `id`，保存时由数据库自增生成
- `novel_id` 在保存阶段统一写入当前工作流的 `NovelID`
- `plotline_ids` 对应 `model.UintArray`
- `relationships` 对应 `model.RelationshipMap`，键为字符串形式的角色 ID，值为关系描述
- `created_at`、`updated_at` 不要求 Agent 输出，保存层自动维护
- `deleted_at` 不出现在 Agent 输出中

## 数据模型

当前角色模型核心字段如下：

```go
type Character struct {
    ID            uint
    NovelID       uint
    Name          string
    Role          string
    Age           int
    Gender        string
    Appearance    string
    Personality   string
    Background    string
    PlotlineIDs   UintArray
    Relationships RelationshipMap
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

## 工作流

当前统一角色工作流保留两种模式。

### 完整工作流模式

触发条件：只有 `NovelID`

执行流程：

```text
START -> init -> extract -> polish -> review -> revise -> saveToDB -> END
```

### 快速修改模式

触发条件：`NovelID + CharacterID + UserSuggestion`

执行流程：

```text
START -> init -> polish -> review -> revise -> saveToDB -> END
```

## 节点语义

### init

- 加载 `model.Novel`
- 加载当前小说已有角色集合
- 若提供 `CharacterID`，校验目标角色存在且属于当前小说
- 初始化工作流状态

### extract

- 调用角色提取 Agent
- 输入为大纲、类型和已有角色上下文
- 输出统一 `characters` JSON
- 支持阶段文件读取，避免重复生成

### polish

- 调用角色润色 Agent
- 完整模式下基于提取结果润色
- 快速修改模式下基于已有角色集合和 `user_suggestion` 润色

### review

- 调用角色审核 Agent
- 输出仍然是统一 `characters` JSON
- 当前实现不单独保存“问题列表”结构

### revise

- 调用角色修改 Agent
- 基于审核后的角色数据和用户建议输出最终角色集合

### saveToDB

- 对最终角色 JSON 做统一校验
- 有 `id` 的角色执行更新
- 无 `id` 的角色执行创建
- 数据库中已存在但不在最终结果里的角色会被删除

## 路由规则

当前路由逻辑与代码一致：

```go
func routeAfterInit(ctx context.Context, state *CharacterWorkflowState) (string, error) {
    if state.CharacterID > 0 && state.UserSuggestion != "" {
        return "polish", nil
    }
    if state.NovelID > 0 {
        return "extract", nil
    }
    return compose.END, fmt.Errorf("输入参数不足，无法确定角色工作流模式")
}
```

其余节点统一使用“成功进入下一节点，失败按最大重试次数重试当前节点”的语义。

## 阶段文件

阶段文件保存在：

```text
data/content/{author}/{novelName}/character/
├── 提取.json
├── 润色.json
├── 审核.json
└── 修改.json
```

文件结构：

```json
{
  "content": "{...characters json...}",
  "stage": "提取",
  "file_type": "character"
}
```

## 目录结构

当前实际目录结构：

```text
internal/character/
├── README.md
├── charater_test.go
├── agents/
│   ├── common.go
│   ├── character_extractor_agent.go
│   ├── character_polisher_agent.go
│   ├── character_reviewer_agent.go
│   └── character_reviser_agent.go
├── compose/
│   ├── agent_executor.go
│   ├── character_workflow.go
│   ├── character_workflow_test.go
│   ├── nodes.go
│   └── router.go
├── contracts/
│   └── character_payload.go
├── prompts/
│   ├── character_extractor_prompt.go
│   ├── character_polisher_prompt.go
│   ├── character_reviewer_prompt.go
│   └── character_reviser_prompt.go
└── repository/
    ├── character_repository.go
    └── repository.go
```

## 仓储接口

小说仓储接口：

- `Create(*model.Novel) error`
- `GetByID(id uint) (*model.Novel, error)`

角色仓储接口：

- `CreateBatch(characters []*model.Character) error`
- `GetByNovelID(novelID uint) ([]*model.Character, error)`
- `GetByID(id uint) (*model.Character, error)`
- `Update(character *model.Character) error`
- `Delete(id uint) error`
- `DeleteByNovelID(novelID uint) error`

## 测试

统一工作流测试位于：

- [character_workflow_test.go]

当前覆盖两个核心场景：

- 完整工作流模式
- 快速修改模式

