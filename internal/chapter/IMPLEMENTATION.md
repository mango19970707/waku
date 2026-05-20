# 章节划分工作流实现总结

## 已完成的工作

### 1. 数据模型扩展

#### 1.1 扩展 Chapter 模型 (`internal/model/chapter.go`)
- ✅ 添加 `WordCountEstimate` 字段 - 预估字数
- ✅ 添加 `KeyEvents` 字段 - 关键事件列表（StringArray）
- ✅ 添加 `KeyCharacters` 字段 - 关键角色列表（StringArray）
- ✅ 添加 `PlotlineProgress` 字段 - 剧情线进度（PlotlineProgressMap）

#### 1.2 新增自定义类型 (`internal/model/types.go`)
- ✅ 添加 `PlotlineProgressMap` 类型 - 用于存储剧情线进度（JSON 对象）
- ✅ 实现 `Scan` 和 `Value` 方法，支持数据库读写

#### 1.3 新增 Event 模型 (`internal/model/event.go`)
- ✅ 事件唯一标识 ID
- ✅ 所属小说 ID 和章节 ID
- ✅ 事件名称、描述、类型
- ✅ 关联的剧情线 ID 列表
- ✅ 关联的角色 ID 列表
- ✅ 事件影响程度（low/medium/high/critical）

#### 1.4 新增 PlotlineProgress 模型 (`internal/model/plotline_progress.go`)
- ✅ 进度记录唯一标识 ID
- ✅ 所属小说 ID、剧情线 ID、章节 ID
- ✅ 进度描述和进度百分比
- ✅ 状态（active/paused/completed）

### 2. 数据契约层 (`internal/chapter/contracts/`)

#### 2.1 章节数据契约 (`chapter_payload.go`)
- ✅ `ChapterPayload` - 章节生成/润色/修改 Agent 的输出格式
- ✅ `ChapterItem` - 单个章节的数据结构
- ✅ `NormalizeChapterPayload` - 校验并规范化 Agent 输出
- ✅ `ToModel` / `FromModel` - 数据转换方法

#### 2.2 审核数据契约
- ✅ `ReviewPayload` - 章节审核 Agent 的输出格式
- ✅ `ReviewSuggestion` - 单条审核建议
- ✅ `NormalizeChapterReviewPayload` - 校验并规范化审核输出

### 3. 数据仓储层 (`internal/chapter/repository/`)

#### 3.1 章节仓储接口 (`chapter_repository.go`)
- ✅ `CreateBatch` - 批量创建章节
- ✅ `GetByID` - 根据 ID 获取章节
- ✅ `GetByVolumeID` - 根据分卷 ID 获取所有章节
- ✅ `GetByNovelID` - 根据小说 ID 获取所有章节
- ✅ `Update` - 更新章节
- ✅ `Delete` - 删除章节
- ✅ `DeleteByVolumeID` - 删除分卷下的所有章节
- ✅ `DeleteByNovelID` - 删除小说下的所有章节
- ✅ `GetMaxChapterNumber` - 获取分卷下的最大章节号

### 4. Agent 提示词层 (`internal/chapter/prompts/`)

#### 4.1 章节生成提示词 (`chapter_generator_prompt.go`)
- ✅ 定义章节规划师角色
- ✅ 章节规划原则（数量、节奏、剧情线分配、角色出场、标题规范、字数预估）
- ✅ 输出格式要求（严格 JSON）

#### 4.2 章节润色提示词 (`chapter_polisher_prompt.go`)
- ✅ 定义章节优化师角色
- ✅ 支持三种工作模式（完整模式、快速新增模式、快速修改模式）
- ✅ 优化原则（标题、简介、大纲、节奏、剧情线、角色出场）

#### 4.3 章节审核提示词 (`chapter_reviewer_prompt.go`)
- ✅ 定义章节审核师角色
- ✅ 审核重点（逻辑问题、节奏问题、剧情线问题、设定冲突、角色问题）
- ✅ 输出格式（suggestions 数组，支持空数组）

#### 4.4 章节修改提示词 (`chapter_reviser_prompt.go`)
- ✅ 定义章节修改师角色
- ✅ 修改原则（针对性修改、最小化改动、保持连续性、优先级排序）
- ✅ 常见修改场景（过渡问题、节奏问题、剧情线问题、设定冲突、角色出场问题）

### 5. Agent 实现层 (`internal/chapter/agents/`)

#### 5.1 公共函数 (`common.go`)
- ✅ `runChapterJSONQuery` - 执行章节相关的 JSON 查询
- ✅ `runChapterReviewJSONQuery` - 执行章节审核的 JSON 查询
- ✅ 流式输出处理
- ✅ JSON 格式校验

#### 5.2 章节生成 Agent (`chapter_generator_agent.go`)
- ✅ `NewChapterGeneratorAgent` - 创建章节生成 Agent
- ✅ `GenerateChapters` - 执行章节生成
- ✅ 输入包含：小说大纲、世界观、修行体系、分卷大纲、剧情线、角色、已有章节

#### 5.3 章节润色 Agent (`chapter_polisher_agent.go`)
- ✅ `NewChapterPolisherAgent` - 创建章节润色 Agent
- ✅ `PolishChapters` - 执行章节润色
- ✅ 支持用户建议（user_suggestion）

#### 5.4 章节审核 Agent (`chapter_reviewer_agent.go`)
- ✅ `NewChapterReviewerAgent` - 创建章节审核 Agent
- ✅ `ReviewChapters` - 执行章节审核
- ✅ 输出审核建议列表

#### 5.5 章节修改 Agent (`chapter_reviser_agent.go`)
- ✅ `NewChapterReviserAgent` - 创建章节修改 Agent
- ✅ `ReviseChapters` - 执行章节修改
- ✅ 根据审核建议修改章节

### 6. 数据库迁移 (`internal/database/migrate.go`)
- ✅ 添加 `Event` 模型到迁移列表
- ✅ 添加 `PlotlineProgress` 模型到迁移列表
- ✅ 更新删除顺序（按依赖关系逆序）

## 待完成的工作

### 7. 工作流编排层 (`internal/chapter/compose/`)

需要创建以下文件（参考 `internal/volume/compose/` 的实现）：

#### 7.1 工作流主文件 (`chapter_workflow.go`)
```go
// ChapterWorkflowInput - 工作流输入
type ChapterWorkflowInput struct {
    NovelID        uint
    VolumeID       uint
    ChapterID      uint
    UserSuggestion string
}

// ChapterWorkflowState - 工作流状态
type ChapterWorkflowState struct {
    CurrentNode          string
    NovelID              uint
    VolumeID             uint
    ChapterID            uint
    UserSuggestion       string
    GeneratedChapters    string
    PolishedChapters     string
    ReviewedChapters     string
    RevisedChapters      string
    SavedChapterIDs      []uint
    ErrorMsg             string
    RetryCount           int
    MaxRetries           int
}

// BuildChapterWorkflowGraph - 构建工作流图
// NewChapterWorkflow - 创建工作流
// ExecuteChapterWorkflow - 执行工作流
```

#### 7.2 节点实现 (`nodes.go`)
```go
// initNode - 初始化节点
//   - 加载 Novel、Volume、Plotline、Character、Chapter
//   - 校验输入参数
//   - 初始化工作流状态

// generateNode - 生成节点
//   - 调用 ChapterGeneratorAgent
//   - 支持断点续传
//   - 保存到 data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/章节规划.json

// polishNode - 润色节点
//   - 调用 ChapterPolisherAgent
//   - 支持用户建议
//   - 保存到 data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/润色.json

// reviewNode - 审核节点
//   - 调用 ChapterReviewerAgent
//   - 输出审核建议
//   - 保存到 data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/审核.json

// reviseNode - 修改节点
//   - 调用 ChapterReviserAgent
//   - 根据审核建议修改
//   - 保存到 data/content/{author}/{novelName}/volume_{volumeNumber}/chapter/修改.json

// saveToDBNode - 保存节点
//   - 校验章节号连续性
//   - 批量创建/更新章节
//   - 删除未参与修改的章节（完整模式）
```

#### 7.3 路由逻辑 (`router.go`)
```go
// routeAfterInit - 初始化后的路由
//   - 快速新增模式：NovelID + VolumeID + UserSuggestion（无 ChapterID）→ polish
//   - 快速修改模式：NovelID + VolumeID + ChapterID + UserSuggestion → polish
//   - 完整生成模式：NovelID + VolumeID → generate

// routeAfterGenerate - 生成后的路由
//   - 成功 → polish
//   - 失败且未达到最大重试次数 → generate（重试）
//   - 失败且达到最大重试次数 → END

// routeAfterPolish - 润色后的路由
//   - 成功 → review
//   - 失败且未达到最大重试次数 → polish（重试）
//   - 失败且达到最大重试次数 → END

// routeAfterReview - 审核后的路由
//   - 成功 → revise
//   - 失败且未达到最大重试次数 → review（重试）
//   - 失败且达到最大重试次数 → END

// routeAfterRevise - 修改后的路由
//   - 成功 → saveToDB
//   - 失败且未达到最大重试次数 → revise（重试）
//   - 失败且达到最大重试次数 → END

// routeAfterSave - 保存后的路由
//   - 成功 → END
//   - 失败且未达到最大重试次数 → saveToDB（重试）
//   - 失败且达到最大重试次数 → END
```

#### 7.4 日志处理 (`logger.go`)
```go
// logWorkflowInput - 记录工作流输入
// logWorkflowOutput - 记录工作流输出
// logNodeInput - 记录节点输入
// logNodeOutput - 记录节点输出
```

#### 7.5 Agent 执行器 (`agent_executor.go`)
```go
// ensureAgentExecutor - 确保 Agent 执行器已初始化
// executeGenerateAgent - 执行生成 Agent
// executePolishAgent - 执行润色 Agent
// executeReviewAgent - 执行审核 Agent
// executeReviseAgent - 执行修改 Agent
```

### 8. 测试文件

需要创建以下测试文件：

#### 8.1 单元测试
- `internal/chapter/agents/chapter_generator_agent_test.go`
- `internal/chapter/agents/chapter_polisher_agent_test.go`
- `internal/chapter/agents/chapter_reviewer_agent_test.go`
- `internal/chapter/agents/chapter_reviser_agent_test.go`
- `internal/chapter/contracts/chapter_payload_test.go`
- `internal/chapter/repository/chapter_repository_test.go`

#### 8.2 集成测试
- `internal/chapter/compose/chapter_workflow_test.go`

## 工作流执行流程

### 完整生成模式
```
输入：NovelID + VolumeID
流程：START → init → generate → polish → review → revise → saveToDB → END
```

### 快速新增模式
```
输入：NovelID + VolumeID + UserSuggestion（无 ChapterID）
流程：START → init → polish → review → revise → saveToDB → END
```

### 快速修改模式
```
输入：NovelID + VolumeID + ChapterID + UserSuggestion
流程：START → init → polish → review → revise → saveToDB → END
```

## 使用示例

```go
// 1. 创建工作流
ctx := context.Background()
chatModel := utils.NewChatModel(ctx, config)
novelRepo := novelrepository.NewRepository(db)
volumeRepo := volumerepository.NewVolumeRepository(db)
plotlineRepo := plotlinerepository.NewPlotlineRepository(db)
characterRepo := characterrepository.NewCharacterRepository(db)
chapterRepo := chapterrepository.NewChapterRepository(db)

workflow, err := compose.NewChapterWorkflow(
    ctx,
    chatModel,
    novelRepo,
    volumeRepo,
    plotlineRepo,
    characterRepo,
    chapterRepo,
)

// 2. 执行完整生成模式
input := &compose.ChapterWorkflowInput{
    NovelID:  1,
    VolumeID: 2,
}
state, err := compose.ExecuteChapterWorkflow(ctx, workflow, input)

// 3. 执行快速新增模式
input := &compose.ChapterWorkflowInput{
    NovelID:        1,
    VolumeID:       2,
    UserSuggestion: "增加主角与师姐的互动，强化感情线铺垫",
}
state, err := compose.ExecuteChapterWorkflow(ctx, workflow, input)

// 4. 执行快速修改模式
input := &compose.ChapterWorkflowInput{
    NovelID:        1,
    VolumeID:       2,
    ChapterID:      15,
    UserSuggestion: "增加主角与师姐的互动，强化感情线铺垫",
}
state, err := compose.ExecuteChapterWorkflow(ctx, workflow, input)
```

## 数据库表结构

### chapters 表（已扩展）
```sql
CREATE TABLE chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    novel_id INTEGER NOT NULL,
    volume_id INTEGER NOT NULL,
    chapter_number INTEGER NOT NULL,
    title VARCHAR(200) NOT NULL,
    synopsis TEXT,
    outline TEXT,
    content TEXT,
    word_count INTEGER DEFAULT 0,
    word_count_estimate INTEGER DEFAULT 0,
    key_events TEXT,  -- JSON 数组
    key_characters TEXT,  -- JSON 数组
    plotline_progress TEXT,  -- JSON 对象
    status VARCHAR(20) DEFAULT 'draft',
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    INDEX idx_novel_id (novel_id),
    INDEX idx_volume_id (volume_id)
);
```

### events 表（新增）
```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    novel_id INTEGER NOT NULL,
    chapter_id INTEGER,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(50),
    plotline_ids TEXT,  -- JSON 数组
    character_ids TEXT,  -- JSON 数组
    impact VARCHAR(20) DEFAULT 'medium',
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    INDEX idx_novel_id (novel_id),
    INDEX idx_chapter_id (chapter_id)
);
```

### plotline_progresses 表（新增）
```sql
CREATE TABLE plotline_progresses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    novel_id INTEGER NOT NULL,
    plotline_id INTEGER NOT NULL,
    chapter_id INTEGER NOT NULL,
    progress TEXT NOT NULL,
    progress_percentage INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    INDEX idx_novel_id (novel_id),
    INDEX idx_plotline_id (plotline_id),
    INDEX idx_chapter_id (chapter_id)
);
```

## 注意事项

1. **章节号连续性**：章节号从 1 开始，在当前卷内连续且唯一
2. **剧情线进度追踪**：每章必须记录推进的剧情线和进度
3. **角色出场记录**：每章必须记录出场的关键角色
4. **字数预估合理**：字数预估应符合实际写作习惯（2000-6000 字）
5. **上下文完整**：生成章节前必须加载完整的上下文信息
6. **修改范围控制**：快速修改模式下，仅修改必要的章节，避免大范围改动
7. **文件路径规范**：阶段文件保存到 `volume_{volumeNumber}/chapter/` 目录下
8. **JSON 格式严格**：所有 Agent 输出必须符合契约定义的 JSON 格式

## 下一步工作

1. **实现工作流编排层**：参考 `internal/volume/compose/` 的实现，创建完整的工作流编排文件
2. **编写单元测试**：为所有 Agent、Repository 和 Contracts 编写单元测试
3. **编写集成测试**：测试完整的工作流执行流程
4. **性能优化**：优化 Agent 执行效率，减少 API 调用次数
5. **错误处理**：完善错误处理机制，提供更友好的错误信息
6. **日志优化**：添加详细的日志输出，便于调试和监控

## 参考文档

- [Chapter 模块 README](D:/code/waku/internal/chapter/README.md)
- [Volume 模块 README](D:/code/waku/internal/volume/README.md)
- [项目主 README](D:/code/waku/CLAUDE.md)
