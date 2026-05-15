# Plotline 模块 - 小说剧情线提取优化工作流

## 概述

Plotline 模块是 Waku 项目中负责小说剧情线提取和优化的核心模块。通过 AI Agent 和工作流编排，实现从小说大纲中自动提取主线和支线剧情、根据用户建议修改剧情线等功能。

## 目录结构

```
internal/plotline/
├── agents/                                    # Agent 实现层（3个Agent）
│   ├── plotline_extractor_agent.go            # 剧情线提取 Agent
│   ├── plotline_polisher_agent.go             # 剧情线润色 Agent
│   └── plotline_revision_agent.go             # 剧情线修改 Agent
├── compose/                                   # 工作流编排层
│   ├── plotline_extract_workflow.go           # 剧情线提取工作流
│   ├── plotline_revision_workflow.go          # 剧情线修改工作流
│   ├── nodes.go                               # 工作流节点实现
│   ├── router.go                              # 工作流路由逻辑
│   └── agent_executor.go                      # Agent 执行器（工厂模式）
├── prompts/                                   # 提示词层（3个提示词）
│   ├── plotline_extractor_prompt.go           # 提取提示词
│   ├── plotline_polisher_prompt.go            # 润色提示词
│   └── plotline_revision_prompt.go            # 修改提示词
└── repository/                                # 数据仓储层
    ├── plotline_repository.go                 # 剧情线数据访问
    └── repository.go                          # 仓储接口定义
```

## 核心功能

### 1. 剧情线提取（PlotlineExtractorAgent）

从小说大纲中自动提取所有主线和支线剧情。

**提取字段：**
- `title`：剧情线标题
- `type`：剧情线类型（main-主线、sub-支线）
- `description`：剧情线描述（200-800字）
- `start_chapter`：起始章节序号
- `end_chapter`：结束章节序号（0 表示未结束）

**输出格式：** JSON

**示例输出：**
```json
{
  "plotlines": [
    {
      "title": "修仙之路",
      "type": "main",
      "description": "主角从凡人开始修炼，经历筑基、金丹、元婴等境界...",
      "start_chapter": 1,
      "end_chapter": 0
    },
    {
      "title": "宗门争斗",
      "type": "sub",
      "description": "主角所在宗门与其他宗门的矛盾冲突...",
      "start_chapter": 10,
      "end_chapter": 25
    }
  ]
}
```

### 2. 剧情线润色（PlotlinePolisherAgent）

对提取的剧情线进行优化，改善描述质量和逻辑连贯性。

**输入参数：**
- `plotlines`：待润色的剧情线列表（JSON 格式）
- `genre`：小说类型

**输出：** 优化后的剧情线列表（JSON 格式）

### 3. 剧情线修改（PlotlineRevisionAgent）

根据用户建议对剧情线进行修改、删除和新增。

**支持操作：**
- **修改现有剧情线**：保留原有 ID，修改部分或全部字段
- **删除剧情线**：直接在输出中省略该剧情线
- **新增剧情线**：ID 设置为空字符串，系统自动生成 UUID

**输入参数：**
- `existing_plotlines`：现有剧情线列表（JSON 格式）
- `user_suggestion`：用户的修改建议
- `outline`：小说大纲（提供上下文）
- `genre`：小说类型

**输出：** 修改后的剧情线列表（JSON 格式）

## 工作流

### 剧情线提取工作流（PlotlineExtractWorkflow）

**流程图：**

```
START → init → extract → saveToDB → END
         ↓       ↓         ↓
      初始化   提取      保存
      (1步)  (3重试)   (3重试)
```

**节点说明：**

1. **initExtractNode**：初始化工作流状态
   - 从数据库加载小说信息
   - 验证大纲是否存在
   - 初始化工作流状态

2. **extractNode**：提取剧情线信息
   - 支持断点续传（从文件加载已有内容）
   - 调用 PlotlineExtractorAgent 执行提取
   - 使用 MessageJSONParser 强制 JSON 输出
   - 保存到 `data/content/{novelID-novelName}/plotline/提取.json`
   - 失败自动重试 3 次

3. **saveToDBNode**：保存到数据库
   - 解析 JSON 数据
   - 转换为 `model.Plotline` 对象
   - 批量插入到数据库
   - 收集保存后的 ID 列表

### 剧情线修改工作流（PlotlineRevisionWorkflow）

**流程图：**

```
START → init → revise → saveOrUpdateDB → END
         ↓       ↓           ↓
      加载    优化        保存/更新
      (从DB) (3重试)     (3重试)
```

**节点说明：**

1. **initRevisionNode**：初始化修改工作流
   - 从数据库加载小说信息
   - 查询该小说的所有剧情线
   - 将剧情线转换为 JSON 格式

2. **reviseNode**：优化剧情线信息
   - 调用 PlotlineRevisionAgent 执行优化
   - 使用 MessageJSONParser 强制 JSON 输出
   - 保存到 `data/content/{novelID-novelName}/plotline/修改.json`
   - 失败自动重试 3 次

3. **saveOrUpdateDBNode**：保存或更新到数据库
   - 解析 JSON 数据
   - 分类处理：有 ID 的更新，无 ID 的新增
   - 批量执行更新和插入操作
   - 收集保存后的 ID 列表

## 数据模型

### PlotlineExtractWorkflowInput（提取工作流输入）

```go
type PlotlineExtractWorkflowInput struct {
    NovelID uint // 小说ID（必填）
}
```

### PlotlineExtractWorkflowState（提取工作流状态）

```go
type PlotlineExtractWorkflowState struct {
    CurrentNode        string   // 当前节点名称
    NovelID            uint     // 小说ID
    NovelName          string   // 小说名称
    Author             string   // 作者
    Genre              string   // 小说类型
    Outline            string   // 小说大纲
    ExtractedPlotlines string   // 提取的剧情线列表（JSON 格式）
    SavedPlotlineIDs   []string // 保存到数据库后的剧情线 ID 列表
    ErrorMsg           string   // 错误信息
    RetryCount         int      // 重试次数
    MaxRetries         int      // 最大重试次数
}
```

### PlotlineRevisionWorkflowInput（修改工作流输入）

```go
type PlotlineRevisionWorkflowInput struct {
    NovelID        uint   // 小说ID（必填）
    UserSuggestion string // 用户的修改建议（必填）
}
```

### PlotlineRevisionWorkflowState（修改工作流状态）

```go
type PlotlineRevisionWorkflowState struct {
    CurrentNode       string   // 当前节点名称
    NovelID           uint     // 小说ID
    NovelName         string   // 小说名称
    Author            string   // 作者名称
    Genre             string   // 小说类型
    Outline           string   // 小说大纲
    ExistingPlotlines string   // 现有剧情线列表（JSON 格式）
    UserSuggestion    string   // 用户的修改建议
    RevisedPlotlines  string   // 优化后的剧情线列表（JSON 格式）
    SavedPlotlineIDs  []string // 保存到数据库后的剧情线 ID 列表
    ErrorMsg          string   // 错误信息
    RetryCount        int      // 当前节点的重试次数
    MaxRetries        int      // 最大重试次数
}
```

### PlotlineItem（剧情线数据结构）

```go
type PlotlineItem struct {
    Title        string `json:"title"`         // 剧情线标题
    Type         string `json:"type"`          // 剧情线类型（main-主线、sub-支线）
    Description  string `json:"description"`   // 剧情线描述
    StartChapter int    `json:"start_chapter"` // 起始章节序号
    EndChapter   int    `json:"end_chapter"`   // 结束章节序号（0 表示未结束）
}
```

### PlotlineExtractorOutput（提取器输出）

```go
type PlotlineExtractorOutput struct {
    Plotlines []PlotlineItem `json:"plotlines"` // 剧情线列表
}
```

## 使用示例

### 提取剧情线

```go
// 1. 创建提取工作流
graph, err := compose.NewPlotlineExtractWorkflow(ctx, chatModel, db)
if err != nil {
    return err
}

// 2. 准备输入
input := &compose.PlotlineExtractWorkflowInput{
    NovelID: 1,
}

// 3. 执行工作流
result, err := compose.ExecutePlotlineExtractWorkflow(ctx, graph, input)
if err != nil {
    return err
}

// 4. 获取结果
fmt.Printf("提取的剧情线数量: %d\n", len(result.SavedPlotlineIDs))
fmt.Printf("剧情线ID列表: %v\n", result.SavedPlotlineIDs)
```

### 修改剧情线

```go
// 1. 创建修改工作流
graph, err := compose.NewPlotlineRevisionWorkflow(ctx, chatModel, db, plotlineRepo)
if err != nil {
    return err
}

// 2. 准备输入
input := &compose.PlotlineRevisionWorkflowInput{
    NovelID:        1,
    UserSuggestion: "增加一条关于主角寻找神器的支线剧情",
}

// 3. 执行工作流
result, err := compose.ExecutePlotlineRevisionWorkflow(ctx, graph, input)
if err != nil {
    return err
}

// 4. 获取结果
fmt.Printf("保存的剧情线数量: %d\n", len(result.SavedPlotlineIDs))
```

## 特性

### 1. 断点续传

提取节点支持断点续传。如果文件已存在，直接加载而不重新提取：

```go
// 尝试从文件加载（断点续传）
existingContent, err := utils.LoadFromFile(filePath)
if existingContent != "" {
    state.ExtractedPlotlines = existingContent
    state.CurrentNode = "extract_success"
    return state, nil
}
```

### 2. 自动重试机制

每个节点失败后自动重试 3 次：

```go
if state.RetryCount >= state.MaxRetries {
    return compose.END, fmt.Errorf("提取失败: %s", state.ErrorMsg)
}
state.RetryCount++
return "extract", nil  // 重试当前节点
```

### 3. 强制 JSON 输出

使用 MessageJSONParser 强制 JSON 输出，确保数据格式正确：

```go
parserConfig := &schema.MessageJSONParseConfig{
    ParseFrom: schema.MessageParseFromContent,
}
parser := schema.NewMessageJSONParser[PlotlineExtractorOutput](parserConfig)
output, err := parser.Parse(ctx, lastMessage)
```

### 4. 文件持久化

每个阶段的输出都保存到文件系统：

```
data/content/{novelID-novelName}/plotline/
├── 提取.json    # 提取阶段的输出
└── 修改.json    # 修改阶段的输出
```

### 5. 批量数据库操作

支持批量插入和更新剧情线，提高性能：

```go
// 批量插入新剧情线
if len(newPlotlines) > 0 {
    err := plotlineRepo.BatchCreate(ctx, newPlotlines)
}

// 批量更新现有剧情线
for _, plotline := range existingPlotlines {
    err := plotlineRepo.Update(ctx, &plotline)
}
```

## 路由逻辑

工作流使用条件分支实现自动重试和状态转移：

```go
// 提取节点后的路由
func routeAfterExtract(ctx context.Context, state *PlotlineExtractWorkflowState) (string, error) {
    if state.CurrentNode == "extract_success" {
        return "saveToDB", nil  // 成功，进入下一步
    }
    if state.RetryCount >= state.MaxRetries {
        return compose.END, fmt.Errorf("提取失败: %s", state.ErrorMsg)
    }
    state.RetryCount++
    return "extract", nil  // 重试
}
```

## 数据库模型

剧情线在数据库中的存储结构（参考 `internal/model/plotline.go`）：

```go
type Plotline struct {
    ID           string         // 剧情线唯一标识（UUID）
    NovelID      string         // 所属小说 ID
    Title        string         // 剧情线标题
    Type         string         // 剧情线类型（main-主线、sub-支线）
    Description  string         // 剧情线描述
    StartChapter int            // 起始章节序号
    EndChapter   int            // 结束章节序号（0 表示未结束）
    CharacterIDs StringArray    // 关联的角色ID列表
    CreatedAt    time.Time      // 创建时间
    UpdatedAt    time.Time      // 更新时间
    DeletedAt    gorm.DeletedAt // 软删除时间
}
```

## 依赖关系

- **Cloudwego Eino**：Agent 和 Workflow 编排框架
- **GORM**：数据库 ORM
- **internal/utils**：通用工具函数（ChatModel、文件操作等）
- **internal/model**：数据模型定义

## 测试

模块包含完整的单元测试：

```bash
# 运行所有测试
go test ./internal/plotline/... -v

# 运行特定测试
go test ./internal/plotline/compose -v -run TestPlotlineExtractWorkflow
```

## 注意事项

1. **前置条件**：提取剧情线前，必须先生成小说大纲（Outline 模块）
2. **JSON 格式**：所有剧情线数据使用严格的 JSON 格式，确保数据一致性
3. **章节序号**：`start_chapter` 和 `end_chapter` 必须是正整数，0 表示未结束
4. **剧情线类型**：只支持 `main`（主线）和 `sub`（支线）两种类型
5. **文件路径**：所有文件操作使用绝对路径，确保跨平台兼容性
6. **重试策略**：默认重试 3 次，可通过 `MaxRetries` 字段调整

## 扩展指南

### 添加新的剧情线类型

1. 在 `model.Plotline` 中扩展 `Type` 字段的枚举值
2. 更新 `prompts/plotline_extractor_prompt.go` 中的提示词
3. 更新数据库迁移脚本

### 自定义剧情线字段

1. 在 `PlotlineItem` 结构体中添加新字段
2. 更新提取提示词，要求 AI 输出新字段
3. 更新数据库模型和迁移脚本
4. 更新仓储层的 CRUD 操作

### 添加剧情线关系

剧情线可以关联角色（通过 `CharacterIDs` 字段）：

```go
// 在修改工作流中关联角色
plotline.CharacterIDs = []string{"character-uuid-1", "character-uuid-2"}
```

## 工作流执行示例

### 完整的剧情线提取流程

```go
package main

import (
    "context"
    "fmt"
    "waku/internal/plotline/compose"
    "waku/internal/utils"
)

func main() {
    ctx := context.Background()

    // 1. 初始化 ChatModel
    chatModel, err := utils.NewChatModel(ctx, config)
    if err != nil {
        panic(err)
    }

    // 2. 创建工作流
    graph, err := compose.NewPlotlineExtractWorkflow(ctx, chatModel, db)
    if err != nil {
        panic(err)
    }

    // 3. 执行工作流
    input := &compose.PlotlineExtractWorkflowInput{
        NovelID: 1,
    }

    result, err := compose.ExecutePlotlineExtractWorkflow(ctx, graph, input)
    if err != nil {
        panic(err)
    }

    // 4. 输出结果
    fmt.Printf("✅ 提取完成\n")
    fmt.Printf("小说ID: %d\n", result.NovelID)
    fmt.Printf("小说名称: %s\n", result.NovelName)
    fmt.Printf("提取的剧情线数量: %d\n", len(result.SavedPlotlineIDs))
    fmt.Printf("剧情线ID列表: %v\n", result.SavedPlotlineIDs)
}
```

## 相关模块

- **Outline 模块**：生成小说大纲（剧情线提取的前置条件）
- **Character 模块**：提取人物信息（可与剧情线关联）
- **Novel 模块**：小说数据管理

## 常见问题

### 1. 提取的剧情线数量不符合预期？

检查大纲内容是否足够详细。如果大纲过于简单，AI 可能无法提取足够的剧情线。建议先使用 Outline 模块的扩写和润色功能优化大纲。

### 2. JSON 解析失败？

确保 AI 模型输出的是有效的 JSON 格式。如果持续失败，可以检查 `data/content/{novelID-novelName}/plotline/提取.json` 文件内容，手动修正后重新运行。

### 3. 如何删除剧情线？

使用修改工作流，在用户建议中明确说明要删除的剧情线。AI 会在输出中省略该剧情线，系统会自动从数据库中删除。

### 4. 章节序号如何确定？

AI 会根据大纲中的章节信息自动推断。如果大纲中没有明确的章节划分，可能需要手动调整。

## 性能优化

1. **批量操作**：使用 `BatchCreate` 批量插入剧情线，减少数据库交互次数
2. **断点续传**：利用文件缓存，避免重复提取
3. **并发处理**：多个小说的剧情线提取可以并发执行
4. **索引优化**：在 `novel_id` 字段上建立索引，加速查询

## 未来规划

- [ ] 支持剧情线之间的依赖关系
- [ ] 支持剧情线的时间线可视化
- [ ] 支持剧情线的冲突检测
- [ ] 支持剧情线的自动补全
- [ ] 支持剧情线的版本管理
