# Outline 模块 - 小说大纲生成优化工作流

## 概述

Outline 模块是 Waku 项目中负责小说大纲生成、扩写、润色和修改的核心模块。通过 AI Agent 和工作流编排，实现从原始创意到完整大纲的自动化生成，以及根据用户建议修改大纲等功能。

## 目录结构

```
internal/outline/
├── agents/                                    # Agent 实现层（5个Agent）
│   ├── outline_generator_agent.go             # 大纲生成 Agent
│   ├── outline_expander_agent.go              # 大纲扩写 Agent
│   ├── outline_polisher_agent.go              # 大纲润色 Agent
│   ├── outline_extractor_agent.go             # 信息提取 Agent
│   └── outline_reviser_agent.go               # 大纲修改 Agent
├── compose/                                   # 工作流编排层
│   ├── outline_generate_workflow.go           # 大纲生成工作流
│   ├── outline_revision_workflow.go           # 大纲修改工作流
│   ├── nodes.go                               # 工作流节点实现
│   ├── router.go                              # 工作流路由逻辑
│   └── agent_executor.go                      # Agent 执行器（工厂模式）
├── prompts/                                   # 提示词层（5个提示词）
│   ├── outline_generator_prompt.go            # 生成提示词
│   ├── outline_expander_prompt.go             # 扩写提示词
│   ├── outline_polisher_prompt.go             # 润色提示词
│   ├── outline_extractor_prompt.go            # 提取提示词
│   └── outline_reviser_prompt.go              # 修改提示词
├── tools/                                     # 工具层
│   └── save_outline_tool.go                   # 保存大纲工具
└── repository/                                # 数据仓储层
    └── repository.go                          # 小说数据访问接口
```

## 核心功能

### 1. 大纲生成（OutlineGeneratorAgent）

将用户提供的简单创意扩展为完整的小说大纲。

**输入参数：**
- `original_outline`：用户提供的原始大纲或创意
- `genre`：小说类型（玄幻、都市、科幻、武侠、历史、悬疑推理、言情、奇幻等）

**输出：** 完整的小说大纲（Markdown 格式）

### 2. 大纲扩写（OutlineExpanderAgent）

将简要大纲扩展为详细的故事框架，增加更多细节和情节。

**输入参数：**
- `brief_outline`：简要大纲
- `genre`：小说类型

**输出：** 详细的小说大纲（Markdown 格式）

### 3. 大纲润色（OutlinePolisherAgent）

对大纲进行深度优化和提升，改善叙事结构、人物塑造和情节连贯性。

**输入参数：**
- `outline`：待润色的大纲
- `genre`：小说类型

**输出：** 优化后的小说大纲（Markdown 格式）

### 4. 信息提取（OutlineExtractorAgent）

从完整大纲中提取结构化信息，用于数据库存储。

**提取字段：**
- `novel_title`：小说标题
- `synopsis`：简介
- `outline`：大纲内容
- `worldview`：世界观设定
- `cultivation_system`：修行体系（玄幻类小说）

**输出格式：** JSON

### 5. 大纲修改（OutlineReviserAgent）

根据用户建议对已有大纲进行修改和优化。

**输入参数：**
- `original_outline`：原始大纲
- `user_suggestion`：用户的修改建议
- `genre`：小说类型

**输出：** 修改后的小说大纲（Markdown 格式）

## 工作流

### 大纲生成工作流（OutlineGenerateWorkflow）

**流程图：**

```
START → init → generate → expand → polish → extract → saveToDB → END
         ↓        ↓         ↓        ↓        ↓         ↓
      初始化    生成      扩写     润色     提取      保存
      (1步)   (3重试)   (3重试)  (3重试)  (3重试)   (3重试)
```

**节点说明：**

1. **initNode**：初始化工作流状态
   - 设置最大重试次数为 3
   - 初始化工作流状态对象

2. **generateNode**：生成大纲
   - 支持断点续传（从文件加载已有内容）
   - 调用 OutlineGeneratorAgent 执行生成
   - 保存到 `data/content/{novelID-novelName}/abstract/生成.md`
   - 失败自动重试 3 次

3. **expandNode**：扩写大纲
   - 支持断点续传（从文件加载已有内容）
   - 调用 OutlineExpanderAgent 执行扩写
   - 保存到 `data/content/{novelID-novelName}/abstract/扩写.md`
   - 失败自动重试 3 次

4. **polishNode**：润色大纲
   - 支持断点续传（从文件加载已有内容）
   - 调用 OutlinePolisherAgent 执行润色
   - 保存到 `data/content/{novelID-novelName}/abstract/润色.md`
   - 失败自动重试 3 次

5. **extractNode**：提取结构化信息
   - 调用 OutlineExtractorAgent 执行提取
   - 使用 MessageJSONParser 强制 JSON 输出
   - 失败自动重试 3 次

6. **saveToDBNode**：保存到数据库
   - 将提取的信息保存到 Novel 表
   - 支持新增或更新小说
   - 失败自动重试 3 次

### 大纲修改工作流（OutlineRevisionWorkflow）

**流程图：**

```
START → init → revise → extract → update → END
         ↓       ↓        ↓         ↓
      加载    修改      提取      更新
      (从DB) (3重试)   (3重试)   (3重试)
```

**节点说明：**

1. **initRevisionNode**：初始化修改工作流
   - 从数据库加载小说信息
   - 验证小说是否存在
   - 加载现有大纲

2. **reviseNode**：修改大纲
   - 调用 OutlineReviserAgent 执行修改
   - 保存到 `data/content/{novelID-novelName}/abstract/修改.md`
   - 失败自动重试 3 次

3. **extractRevisionNode**：提取修改后的信息
   - 调用 OutlineExtractorAgent 执行提取
   - 使用 MessageJSONParser 强制 JSON 输出
   - 失败自动重试 3 次

4. **updateDBNode**：更新数据库
   - 更新 Novel 表中的小说信息
   - 失败自动重试 3 次

## 数据模型

### OutlineWorkflowInput（生成工作流输入）

```go
type OutlineWorkflowInput struct {
    NovelID         uint   // 小说ID（用于更新已有小说，非必填）
    NovelName       string // 小说名称（非必填，第一次生成时可为空）
    Author          string // 作者名称（必填）
    Genre           string // 小说类型（必填）
    OriginalOutline string // 原始大纲（必填）
}
```

### OutlineWorkflowState（生成工作流状态）

```go
type OutlineWorkflowState struct {
    CurrentNode      string                  // 当前节点名称
    NovelID          uint                    // 小说ID
    NovelName        string                  // 小说名称
    Author           string                  // 作者名称
    Genre            string                  // 小说类型
    OriginalOutline  string                  // 原始大纲
    GeneratedOutline string                  // 生成的大纲
    ExpandedOutline  string                  // 扩写后的大纲
    PolishedOutline  string                  // 润色后的大纲
    ExtractedInfo    *OutlineExtractorOutput // 提取的结构化信息
    SavedNovelID     uint                    // 保存到数据库后的小说ID
    ErrorMsg         string                  // 错误信息
    RetryCount       int                     // 当前节点的重试次数
    MaxRetries       int                     // 最大重试次数
}
```

### OutlineRevisionWorkflowInput（修改工作流输入）

```go
type OutlineRevisionWorkflowInput struct {
    NovelID        uint   // 小说ID（必填）
    UserSuggestion string // 用户的修改建议（必填）
}
```

### OutlineExtractorOutput（提取器输出）

```go
type OutlineExtractorOutput struct {
    NovelTitle        string `json:"novel_title"`        // 小说标题
    Synopsis          string `json:"synopsis"`           // 简介
    Outline           string `json:"outline"`            // 大纲内容
    Worldview         string `json:"worldview"`          // 世界观设定
    CultivationSystem string `json:"cultivation_system"` // 修行体系
}
```

## 使用示例

### 生成新大纲

```go
// 1. 创建工作流
graph, err := compose.BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
if err != nil {
    return err
}

// 2. 准备输入
input := &compose.OutlineWorkflowInput{
    NovelName:       "修仙传奇",
    Author:          "张三",
    Genre:           "玄幻",
    OriginalOutline: "一个少年意外获得修仙功法，踏上修仙之路...",
}

// 3. 执行工作流
result, err := compose.ExecuteOutlineWorkflow(ctx, graph, input)
if err != nil {
    return err
}

// 4. 获取结果
fmt.Printf("保存的小说ID: %d\n", result.SavedNovelID)
```

### 修改已有大纲

```go
// 1. 创建修改工作流
graph, err := compose.BuildOutlineRevisionWorkflowGraph(ctx, chatModel, db, repo)
if err != nil {
    return err
}

// 2. 准备输入
input := &compose.OutlineRevisionWorkflowInput{
    NovelID:        1,
    UserSuggestion: "增加一个反派角色，让故事更有冲突",
}

// 3. 执行工作流
result, err := compose.ExecuteOutlineRevisionWorkflow(ctx, graph, input)
if err != nil {
    return err
}

// 4. 获取结果
fmt.Printf("修改后的小说ID: %d\n", result.NovelID)
```

## 特性

### 1. 断点续传

每个处理节点都支持断点续传。如果文件已存在，直接加载而不重新生成：

```go
// 尝试从文件加载（断点续传）
existingContent, err := utils.LoadFromFile(filePath)
if existingContent != "" {
    state.GeneratedOutline = existingContent
    state.CurrentNode = "generate_success"
    return state, nil
}
```

### 2. 自动重试机制

每个节点失败后自动重试 3 次：

```go
if state.RetryCount >= state.MaxRetries {
    return compose.END, fmt.Errorf("生成失败: %s", state.ErrorMsg)
}
state.RetryCount++
return "generate", nil  // 重试当前节点
```

### 3. 强制 JSON 输出

在信息提取节点中，使用 MessageJSONParser 强制 JSON 输出：

```go
parserConfig := &schema.MessageJSONParseConfig{
    ParseFrom: schema.MessageParseFromContent,
}
parser := schema.NewMessageJSONParser[OutlineExtractorOutput](parserConfig)
output, err := parser.Parse(ctx, lastMessage)
```

### 4. 文件持久化

每个阶段的输出都保存到文件系统：

```
data/content/{novelID-novelName}/abstract/
├── 生成.md      # 生成阶段的输出
├── 扩写.md      # 扩写阶段的输出
├── 润色.md      # 润色阶段的输出
└── 修改.md      # 修改阶段的输出
```

## 路由逻辑

工作流使用条件分支实现自动重试和状态转移：

```go
// 生成节点后的路由
func routeAfterGenerate(ctx context.Context, state *OutlineWorkflowState) (string, error) {
    if state.CurrentNode == "generate_success" {
        return "expand", nil  // 成功，进入下一步
    }
    if state.RetryCount >= state.MaxRetries {
        return compose.END, fmt.Errorf("生成失败: %s", state.ErrorMsg)
    }
    state.RetryCount++
    return "generate", nil  // 重试
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
go test ./internal/outline/... -v

# 运行特定测试
go test ./internal/outline/compose -v -run TestOutlineGenerateWorkflow
```

## 注意事项

1. **文件路径**：所有文件操作使用绝对路径，确保跨平台兼容性
2. **错误处理**：每个节点都有详细的错误信息，便于调试
3. **重试策略**：默认重试 3 次，可通过 `MaxRetries` 字段调整
4. **断点续传**：修改中间文件后重新运行，可以跳过已完成的步骤
5. **JSON 解析**：提取节点使用严格的 JSON 解析，确保数据格式正确

## 扩展指南

### 添加新的处理阶段

1. 在 `agents/` 目录创建新的 Agent
2. 在 `prompts/` 目录创建对应的提示词
3. 在 `compose/nodes.go` 中添加新的节点函数
4. 在 `compose/router.go` 中添加路由逻辑
5. 在工作流中注册新节点和边

### 自定义 Agent 执行器

通过 `RegisterAgentExecutor` 注册自定义执行器：

```go
type CustomAgentExecutor struct{}

func (e *CustomAgentExecutor) GenerateOutline(...) (string, error) {
    // 自定义实现
}

// 注册
compose.RegisterAgentExecutor(&CustomAgentExecutor{})
```

## 相关模块

- **Character 模块**：从大纲中提取人物信息
- **Plotline 模块**：从大纲中提取剧情线
- **Novel 模块**：小说数据管理
