# Waku - AI 小说生成 Agent

## 项目简介

Waku 是一个基于 AI 的智能小说创作系统，通过 Agent 架构和工作流编排实现小说大纲的自动化生成、优化和管理。系统采用 Cloudwego Eino 框架构建，使用 Graph-based Workflow 实现复杂的多阶段处理流程。

### 核心功能

- **智能大纲生成**：基于用户输入自动生成完整的小说大纲
- **多阶段优化**：生成 → 扩写 → 润色 → 不合理性审核 → 修改，逐步提升大纲质量
- **结构化提取**：自动提取标题、简介、世界观、修行体系等关键信息
- **智能路由**：根据输入自动选择完整生成流程或快速修改流程
- **不合理性审核**：自动检测大纲中的逻辑漏洞、设定冲突等问题
- **大纲修改**：根据审核建议或用户建议智能修改已有大纲
- **断点续传**：支持工作流中断后从上次位置继续
- **中断恢复**：保存前需用户审批，防止误操作
- **自动重试**：每个节点失败自动重试 3 次
- **多模型支持**：支持 OpenAI、Azure OpenAI、Ollama、Ark、Anthropic Claude

## 技术栈

- **语言**：Go 1.25.0
- **核心框架**：Cloudwego Eino v0.8.13（Agent 和 Workflow 编排）
- **数据库**：GORM + SQLite/PostgreSQL
- **配置管理**：YAML + 环境变量
- **AI 模型支持**：OpenAI、Azure OpenAI、Ollama、Ark、Anthropic Claude

## 项目架构

### 目录结构

```
waku/
├── main.go                          # 程序入口
├── go.mod / go.sum                  # 依赖管理
├── config/                          # 配置文件
│   └── config.yaml                  # 应用配置
├── data/                            # 数据存储目录
│   ├── waku.db                      # SQLite 数据库文件
│   └── content/                     # 小说内容文件
│       └── {novelID-novelName}/     # 每个小说的目录
│           └── abstract/            # 大纲文件（JSON 格式）
│               ├── 生成.json
│               ├── 扩写.json
│               ├── 润色.json
│               └── 修改.json
└── internal/                        # 核心业务逻辑
    ├── model/                       # 数据模型（GORM）
    │   ├── types.go                 # 自定义类型（StringArray）
    │   └── novel.go                 # 小说模型
    ├── config/                      # 配置管理
    │   └── config.go                # 配置加载逻辑
    ├── database/                    # 数据库管理
    │   ├── database.go              # 数据库连接
    │   └── migrate.go               # 数据库迁移
    ├── utils/                       # 通用工具函数
    │   ├── chat_model.go            # ChatModel 封装
    │   ├── file.go                  # 文件操作
    │   ├── agent.go                 # Agent 创建工厂
    │   └── callback.go              # 回调处理
    └── novel/                       # 小说管理核心模块
        ├── agents/                  # AI Agent 实现（6个）
        │   ├── outline_generator_agent.go      # 大纲生成 Agent
        │   ├── outline_expander_agent.go       # 大纲扩写 Agent
        │   ├── outline_polisher_agent.go       # 大纲润色 Agent
        │   ├── outline_reviewer_agent.go       # 不合理性审核 Agent
        │   ├── outline_reviser_agent.go        # 大纲修改 Agent
        │   └── outline_extractor_agent.go      # 信息提取 Agent
        ├── compose/                 # 工作流编排
        │   ├── outline_workflow.go             # 统一大纲工作流
        │   ├── nodes.go                        # 工作流节点实现
        │   ├── router.go                       # 工作流路由逻辑
        │   └── agent_executor.go               # Agent 执行器
        ├── prompts/                 # Agent 系统提示词（6个）
        │   ├── outline_generator_prompt.go
        │   ├── outline_expander_prompt.go
        │   ├── outline_polisher_prompt.go
        │   ├── outline_reviewer_prompt.go
        │   ├── outline_reviser_prompt.go
        │   └── outline_extractor_prompt.go
        ├── tools/                   # 工具定义
        │   └── save_outline_tool.go            # 保存大纲工具
        ├── repository/              # 数据仓储
        │   └── novel_repository.go             # 小说数据访问
        └── utils/                   # 工具函数
            └── file.go                         # 文件操作
```

### 模块说明

#### 1. 数据模型层 (`internal/model/`)

**Novel 模型结构：**

```go
type Novel struct {
    ID                uint              // 主键
    Title             string            // 小说标题
    Author            string            // 作者
    Genre             string            // 类型（玄幻、都市等）
    Synopsis          string            // 简介
    Outline           string            // 完整大纲（JSON 格式）
    Worldview         string            // 世界观设定
    CultivationSystem string            // 修行体系
    Status            string            // 状态（draft/writing/completed）
    CreatedAt         time.Time         // 创建时间
    UpdatedAt         time.Time         // 更新时间
    DeletedAt         gorm.DeletedAt    // 软删除时间
}
```

**特点：**
- 使用 GORM 的软删除机制
- 支持 JSON 序列化
- 包含完整的小说创作信息
- **Outline 字段存储 JSON 格式**：便于结构化存储章节、卷等信息

#### 2. Agent 层 (`internal/novel/agents/`)

**6 个专业 Agent：**

| Agent | 功能 | 输入 | 输出 |
|-------|------|------|------|
| **OutlineGeneratorAgent** | 生成初始大纲 | 原始大纲 + 类型 | 完整大纲 |
| **OutlineExpanderAgent** | 扩写大纲细节 | 简要大纲 + 类型 | 详细大纲 |
| **OutlinePolisherAgent** | 润色和优化 | 大纲 + 类型 | 优化后大纲 |
| **OutlineReviewerAgent** | 不合理性审核 | 大纲 + 类型 | 审核建议列表 |
| **OutlineReviserAgent** | 根据建议修改 | 原大纲 + 建议 + 类型 | 修改后大纲 |
| **OutlineExtractorAgent** | 提取结构化信息 | 大纲 + 类型 | JSON（标题、简介、世界观等） |

**Agent 实现模式：**

```go
// 基于 Cloudwego Eino 的 ChatModelAgent
agent := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "AgentName",
    Description: "Agent 描述",
    Instruction: systemPrompt,  // 系统提示词
    Model:       chatModel,
})

// 使用 Runner 执行查询（支持流式输出）
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    EnableStreaming: true,
    Agent:           agent,
})
iter := runner.Query(ctx, userQuery)
```

#### 3. 工作流编排层 (`internal/novel/compose/`)

**核心概念：Graph-based Workflow**

使用 Cloudwego Eino 的 `compose.Graph` 实现有向无环图（DAG）工作流。

##### 统一大纲工作流 (`outline_workflow.go`)

**流程图：**

```
完整生成模式（输入：小说类型 + 大纲构思）：
START → init → generate → expand → polish → review → revise → extract → saveToDB → END
         ↓        ↓         ↓        ↓        ↓       ↓        ↓         ↓
      初始化    生成      扩写     润色     审核    修改     提取      保存
      (1步)    (3重试)   (3重试)  (3重试)  (3重试) (3重试)  (3重试)   (3重试)

快速修改模式（输入：小说类型 + 小说ID + 用户建议）：
START → init → polish → review → revise → extract → saveToDB → END
         ↓       ↓        ↓       ↓        ↓         ↓
      初始化   润色     审核    修改     提取      保存
      (从DB)  (3重试)  (3重试) (3重试)  (3重试)   (3重试)
```

**节点说明：**

1. **initNode** - 初始化工作流状态
   - 输入：`OutlineWorkflowInput`（小说类型、大纲构思、小说ID、用户建议）
   - 输出：`OutlineWorkflowState`
   - **智能路由**：
     - 如果只有小说类型和大纲构思 → 进入完整生成模式
     - 如果有小说ID和用户建议 → 从数据库加载，进入快速修改模式

2. **generateNode** - 生成大纲
   - 调用 `OutlineGeneratorAgent`
   - 支持断点续传（从文件加载已有内容）
   - 失败自动重试 3 次
   - 保存到 `data/content/{name}/abstract/生成.json`
   - **仅在完整生成模式下执行**

3. **expandNode** - 扩写大纲
   - 调用 `OutlineExpanderAgent`
   - 基于生成的大纲进行扩写
   - 保存到 `data/content/{name}/abstract/扩写.json`
   - **仅在完整生成模式下执行**

4. **polishNode** - 润色大纲
   - 调用 `OutlinePolisherAgent`
   - 优化大纲质量
   - 保存到 `data/content/{name}/abstract/润色.json`
   - **两种模式都会执行**

5. **reviewNode** - 不合理性审核
   - 调用 `OutlineReviewerAgent`
   - 检测逻辑漏洞、设定冲突、情节不合理等问题
   - 输出审核建议列表（JSON 格式）
   - 保存到 `data/content/{name}/abstract/审核.json`
   - **两种模式都会执行**

6. **reviseNode** - 修改大纲
   - 调用 `OutlineReviserAgent`
   - 根据审核建议或用户建议修改大纲
   - 保存到 `data/content/{name}/abstract/修改.json`
   - **两种模式都会执行**

7. **extractNode** - 提取结构化信息
   - 调用 `OutlineExtractorAgent`
   - 输出 JSON 格式的小说信息
   - 使用 `MessageJSONParser` 强制 JSON 输出
   - **两种模式都会执行**

8. **saveToDBNode** - 保存到数据库
   - **中断恢复机制**：首次调用时触发用户审批
   - 支持新增或更新小说
   - 用户可以拒绝保存
   - **Outline 字段保存为 JSON 格式**
   - **两种模式都会执行**

**路由逻辑 (`router.go`)：**

```go
// 初始化后的路由决策
func routeAfterInit(ctx context.Context, state *OutlineWorkflowState) (string, error) {
    // 快速修改模式：有小说ID和用户建议
    if state.NovelID > 0 && state.UserSuggestion != "" {
        return "polish", nil  // 直接跳到润色节点
    }
    // 完整生成模式：有大纲构思
    if state.OriginalOutline != "" {
        return "generate", nil  // 从生成节点开始
    }
    return compose.END, fmt.Errorf("输入参数不足")
}

// 每个节点完成后，根据状态决定下一步
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

#### 4. 提示词层 (`internal/novel/prompts/`)

**6 个专业提示词文件：**

1. **outline_generator_prompt.go** - 生成大纲的系统提示
2. **outline_expander_prompt.go** - 扩写大纲的系统提示
3. **outline_polisher_prompt.go** - 润色大纲的系统提示
4. **outline_reviewer_prompt.go** - 不合理性审核的系统提示（检测逻辑漏洞、设定冲突等）
5. **outline_reviser_prompt.go** - 修改大纲的系统提示
6. **outline_extractor_prompt.go** - 提取信息的系统提示（强制 JSON 输出）

**提示词特点：**
- 中文编写，针对玄幻小说优化
- 包含详细的角色定义和职责说明
- 提供输出格式示例
- 强制结构化输出（JSON）

#### 5. 工具层 (`internal/novel/tools/`)

**SaveOutlineTool** - 保存大纲工具

```go
type SaveOutlineToolInput struct {
    Title             string  // 小说标题
    Author            string  // 作者
    Genre             string  // 类型
    Synopsis          string  // 简介
    Outline           string  // 大纲
    Worldview         string  // 世界观
    CultivationSystem string  // 修行体系
}

type SaveOutlineToolOutput struct {
    Success bool   // 是否成功
    NovelID uint   // 保存后的小说ID
    Message string // 提示信息
}
```

#### 6. 工具函数层 (`internal/utils/`)

**关键文件：**

1. **agent.go** - Agent 创建工厂函数
   - `NewChatModelAgent()` - 创建基础 Agent
   - `NewSupervisorAgent()` - 创建 Supervisor 协调 Agent
   - `NewPlanExecuteReplanAgent()` - 创建 Plan-Execute-Replan Agent
   - `NewDeepAgent()` - 创建深度任务编排 Agent

2. **chat_model.go** - 模型初始化
   - `NewChatModel()` - 根据配置创建聊天模型
   - `GenerateOnce()` - 一次性生成
   - `GenerateStream()` - 流式生成

3. **file.go** - 文件操作
   - `SaveToFile()` - 保存内容到文件
   - `LoadFromFile()` - 从文件加载（断点续传）

## 核心设计模式

### 1. 中断恢复机制

在 `saveToDBNode` 中实现：

```go
// 首次调用时设置等待审批标志
if !state.WaitingForApproval {
    state.WaitingForApproval = true
    return state, nil  // 返回，等待用户审批
}

// 获取用户审批决定
approved := getUserApproval()
if !approved {
    return state, fmt.Errorf("用户拒绝保存")
}

// 用户批准，继续执行保存
```

### 2. 断点续传

在每个处理节点中：

```go
// 尝试从文件加载（断点续传）
if state.GeneratedOutline == "" {
    existingContent, err := utils.LoadFromFile(filePath)
    if existingContent != "" {
        state.GeneratedOutline = existingContent
        state.CurrentNode = "generate_success"
        return state, nil  // 跳过重复生成
    }
}

// 文件不存在，执行生成
```

### 3. 自动重试机制

在路由函数中：

```go
if state.RetryCount >= state.MaxRetries {
    return compose.END, fmt.Errorf("失败: %s", state.ErrorMsg)
}
state.RetryCount++
return "generate", nil  // 重试当前节点
```

### 4. JSON 强制输出

在 `OutlineExtractorAgent` 中：

```go
// 使用 MessageJSONParser 强制 JSON 输出
parserConfig := &schema.MessageJSONParseConfig{
    ParseFrom: schema.MessageParseFromContent,
}
parser := schema.NewMessageJSONParser[OutlineExtractorOutput](parserConfig)
output, err := parser.Parse(ctx, lastMessage)
```

## 文件存储格式

### JSON 文件格式

所有阶段的文件（大纲、剧情线等）都以 JSON 格式保存，便于结构化存储和解析。

**文件路径格式：**
```
data/{作者名称}/{文件类型}/{阶段}.json
```

**JSON 文件结构：**
```json
{
  "content": "实际内容（大纲、剧情线等）",
  "stage": "阶段名称（生成、扩写、润色、修改等）",
  "file_type": "文件类型（outline、plotline、volume、chapter）"
}
```

**示例：**
```json
{
  "content": "第一章：穿越异界\n主角林凡意外穿越到修仙世界...",
  "stage": "生成",
  "file_type": "outline"
}
```

**优势：**
- **结构化存储**：便于添加元数据（阶段、类型、时间戳等）
- **易于解析**：标准 JSON 格式，支持各种工具和语言
- **向后兼容**：`LoadFromFile` 函数支持读取旧的纯文本格式
- **便于扩展**：未来可以添加更多字段（如版本号、作者、创建时间等）

**相关函数：**
- `utils.SaveToFile()` - 保存内容到 JSON 文件
- `utils.LoadFromFile()` - 从 JSON 文件加载内容（自动解析）

## 数据流向

```
用户输入
  │
  ▼
OutlineWorkflowInput
  │
  ├─ NovelID (可选，用于快速修改模式)
  ├─ NovelName
  ├─ Genre
  ├─ OriginalOutline (可选，用于完整生成模式)
  └─ UserSuggestion (可选，用于快速修改模式)
  │
  ▼
[智能路由决策]
  │
  ├─ 完整生成模式：有大纲构思 → 走完整流程
  └─ 快速修改模式：有小说ID和用户建议 → 跳到润色节点
  │
  ▼
[工作流执行]
  │
  ├─ Agent 处理 (调用 AI 模型)
  ├─ 文件保存 (断点续传)
  └─ 状态转移 (路由)
  │
  ▼
OutlineWorkflowState
  │
  ├─ CurrentNode (当前节点)
  ├─ GeneratedOutline (生成的大纲)
  ├─ ExpandedOutline (扩写的大纲)
  ├─ PolishedOutline (润色的大纲)
  ├─ ReviewSuggestions (审核建议列表)
  ├─ RevisedOutline (修改后的大纲)
  ├─ ExtractedInfo (提取的信息)
  │  ├─ NovelTitle
  │  ├─ Synopsis
  │  ├─ Outline (JSON 格式)
  │  ├─ Worldview
  │  └─ CultivationSystem
  ├─ SavedNovelID (保存后的ID)
  └─ ErrorMsg (错误信息)
  │
  ▼
数据库 (Novel 表)
  │
  └─ Outline 字段存储为 JSON 格式
```

## 代码规范

### 1. 注释规范

- **包注释**：每个包必须有包级别的注释，说明包的用途和主要功能
- **函数注释**：所有导出函数必须有详细的中文注释，包括参数说明、返回值说明、注意事项
- **结构体注释**：结构体和字段都需要清晰的注释，说明用途和约束条件
- **复杂逻辑注释**：对于复杂的业务逻辑，添加详细的行内注释，说明每个步骤的目的

**示例：**

```go
// OutlineGeneratorAgent 是一个专门用于生成小说大纲的 AI Agent
// 它接收用户提供的原始大纲和小说类型，生成完整的小说大纲
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型实例
//   - originalOutline: 用户提供的原始大纲
//   - genre: 小说类型（如：玄幻、都市等）
//
// 返回：
//   - string: 生成的完整大纲
//   - error: 错误信息
func OutlineGeneratorAgent(ctx context.Context, chatModel chat_model.ChatModel, originalOutline, genre string) (string, error) {
    // 实现代码...
}
```

### 2. 代码复用原则

- **提取公共函数**：避免重复代码，将通用逻辑提取为独立函数
- **使用接口实现多态**：通过接口实现代码复用和解耦，不同的实现策略可以实现同一接口
- **使用组合而非继承**：Go 推荐使用组合模式，通过嵌入结构体复用功能

**示例：**

```go
// 不好的做法：重复代码
func generateOutline() {
    content, _ := utils.LoadFromFile(path1)
    if content != "" {
        return content
    }
    // 生成逻辑...
}

func expandOutline() {
    content, _ := utils.LoadFromFile(path2)
    if content != "" {
        return content
    }
    // 扩写逻辑...
}

// 好的做法：提取公共函数
func loadOrProcess(filePath string, processFn func() (string, error)) (string, error) {
    content, _ := utils.LoadFromFile(filePath)
    if content != "" {
        return content, nil
    }
    return processFn()
}
```

### 3. 减少冗余

- **避免不必要的中间变量**：如果可以直接返回，就不要创建中间变量
- **使用表驱动测试**：减少测试代码冗余，用数据结构描述测试用例
- **使用常量避免魔法数字**：所有硬编码的数字都应该定义为有意义的常量

**示例：**

```go
// 不好的做法：魔法数字
if state.RetryCount >= 3 {
    return compose.END, fmt.Errorf("失败")
}

// 好的做法：使用常量
const MaxRetries = 3

if state.RetryCount >= MaxRetries {
    return compose.END, fmt.Errorf("失败")
}
```

### 4. 错误处理规范

- **使用 errors.Is 和 errors.As**：定义明确的错误类型，使用标准库函数判断错误
- **错误信息要清晰具体**：错误信息应包含足够的上下文信息，便于定位问题
- **包装错误保留原始信息**：使用 `%w` 格式化动词包装错误，保留错误链

**示例：**

```go
// 不好的做法：丢失错误上下文
if err != nil {
    return "", fmt.Errorf("生成失败")
}

// 好的做法：包装错误保留上下文
if err != nil {
    return "", fmt.Errorf("生成大纲失败，小说类型: %s: %w", genre, err)
}
```

### 5. 性能优化建议

- **使用 sync.Pool 复用对象**：为频繁创建的对象使用对象池，减少 GC 压力
- **避免不必要的内存分配**：使用 strings.Builder 进行字符串拼接，预分配切片容量
- **合理使用缓存**：对于计算密集或 I/O 密集的操作，使用缓存提高性能

**示例：**

```go
// 不好的做法：频繁字符串拼接
var result string
for _, line := range lines {
    result += line + "\n"
}

// 好的做法：使用 strings.Builder
var builder strings.Builder
builder.Grow(len(lines) * 100)  // 预分配容量
for _, line := range lines {
    builder.WriteString(line)
    builder.WriteString("\n")
}
result := builder.String()
```

### 6. 并发安全

- **使用互斥锁保护共享资源**：对于共享数据结构，使用 sync.Mutex 或 sync.RWMutex 保护
- **优先使用 channel 通信**：Go 推荐使用 channel 在 goroutine 之间传递数据
- **避免数据竞争**：使用 `go run -race` 检测数据竞争问题

**示例：**

```go
// 不好的做法：未保护的共享变量
var counter int
go func() { counter++ }()
go func() { counter++ }()

// 好的做法：使用互斥锁
var (
    counter int
    mu      sync.Mutex
)
go func() {
    mu.Lock()
    counter++
    mu.Unlock()
}()
```

## 测试规范

### 测试覆盖率要求

- 单元测试覆盖率：≥ 80%
- 核心业务逻辑：≥ 90%
- 每个导出函数都应有对应的测试

### 测试文件命名

- 测试文件与源文件同目录
- 命名格式：`xxx_test.go`
- 示例：`outline_generator_agent.go` 对应 `outline_generator_agent_test.go`

### 测试函数命名

```go
// 格式：Test + 函数名 + 测试场景
func TestOutlineGeneratorAgent_Success(t *testing.T) { }
func TestOutlineGeneratorAgent_EmptyInput(t *testing.T) { }
func TestOutlineGeneratorAgent_ModelError(t *testing.T) { }
```

### 表驱动测试

```go
func TestOutlineGeneratorAgent(t *testing.T) {
    tests := []struct {
        name           string
        originalOutline string
        genre          string
        wantErr        bool
    }{
        {
            name:           "成功生成大纲",
            originalOutline: "主角穿越到修仙世界",
            genre:          "玄幻",
            wantErr:        false,
        },
        {
            name:           "空输入",
            originalOutline: "",
            genre:          "",
            wantErr:        true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑...
        })
    }
}
```

## Commit 规范

### Commit Message 格式

```
<type>: <subject>

<body>

<footer>
```

### Type 类型

- **feat**: 新功能
- **fix**: 修复 bug
- **refactor**: 重构（不改变功能）
- **docs**: 文档更新
- **test**: 测试相关
- **chore**: 构建/工具链相关
- **perf**: 性能优化
- **ci**: CI/CD 相关

### Commit Message 示例

```
feat: 添加大纲修改工作流

- 实现 OutlineReviserAgent
- 添加大纲修改工作流
- 支持根据用户建议修改大纲
- 添加修改后的信息提取

Closes #123
```

```
fix: 修复断点续传文件路径错误

修复在 Windows 系统下文件路径分隔符错误导致断点续传失败的问题

Fixes #456
```

```
refactor: 优化 SaveOutlineTool，移除冗余的提取逻辑

将提取逻辑从 SaveOutlineTool 移到工作流的 extractNode 中，
避免重复提取，提高代码复用性
```

## 开发工具

### 推荐的 Go 工具

- **gofmt**: 代码格式化
- **golangci-lint**: 代码静态检查
- **go test**: 运行测试
- **go mod**: 依赖管理
- **delve**: 调试工具

### 代码检查命令

```bash
# 格式化代码
gofmt -w .

# 运行 linter
golangci-lint run

# 运行测试
go test ./... -v -cover

# 查看测试覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 检测数据竞争
go test ./... -race
```

## 常见问题

### 1. 如何添加新的 Agent？

1. 在 `internal/novel/agents/` 创建新的 Agent 文件
2. 在 `internal/novel/prompts/` 创建对应的提示词文件
3. 在工作流中添加新的节点调用该 Agent
4. 更新路由逻辑
5. 编写单元测试

### 2. 如何修改工作流？

1. 在 `internal/novel/compose/nodes.go` 中添加或修改节点
2. 在 `internal/novel/compose/router.go` 中更新路由逻辑
3. 在 `internal/novel/compose/outline_workflow.go` 中更新 Graph 定义
4. 测试新的工作流

**注意**：统一工作流支持两种模式：
- **完整生成模式**：输入小说类型和大纲构思，走完整流程
- **快速修改模式**：输入小说类型、小说ID和用户建议，从润色节点开始

### 3. 如何支持新的 AI 模型提供商？

1. 在 `internal/utils/chat_model.go` 中添加新的模型初始化逻辑
2. 更新配置文件支持新的 provider
3. 测试新模型的兼容性

### 4. 如何调试工作流？

1. 启用详细日志：设置 `DB_LOG_LEVEL=debug`
2. 查看文件输出：检查 `data/content/` 目录下的中间文件
3. 使用断点续传：修改中间文件后重新运行
4. 使用 delve 调试器：`dlv debug main.go`

## 项目入口

**main.go** 支持三种模式：

```bash
# 1. 数据库迁移
go run main.go -migrate

# 2. 重置数据库（危险操作）
go run main.go -reset

# 3. 启动应用（默认）
go run main.go -config config/config.yaml
```

## License

MIT
