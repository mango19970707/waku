# Waku - AI 小说生成 Agent

## 项目简介

Waku 是一个基于 AI 的智能小说创作系统，通过 Agent 架构实现小说的自动化生成和管理。系统能够处理小说创作的各个环节，包括角色设定、剧情规划、章节大纲、内容生成等，为作者提供智能化的创作辅助工具。

### 核心功能

- **角色管理**：创建和管理小说角色，包括角色属性、性格特征、背景故事等
- **剧情规划**：设计和管理故事主线、支线剧情，确保情节连贯性
- **章节大纲**：自动生成或辅助创建章节大纲，规划故事结构
- **内容生成**：基于大纲和剧情自动生成章节内容
- **记忆系统**：维护故事上下文，确保前后一致性
- **分卷管理**：支持长篇小说的分卷组织

## 项目结构

```
waku/
├── main.go              # 程序入口
├── go.mod               # Go 模块依赖
├── config/              # 配置文件目录
│   └── config.yaml      # 应用配置
└── internal/            # 内部业务逻辑（核心代码目录）
    ├── model/           # 数据模型定义
    ├── config/          # 配置管理模块
    ├── database/        # 数据库管理模块
    ├── novel/           # 小说管理模块
    ├── abstract/        # 章节大纲模块
    ├── chapter/         # 章节内容模块
    ├── character/       # 角色管理模块
    ├── memory/          # 记忆系统模块
    ├── plotline/        # 剧情管理模块
    ├── volume/          # 分卷管理模块
    ├── utils/           # 工具函数模块
    └── version/         # 版本管理模块
```

## Commit 规范

### Commit Message 格式

```
<type>: <subject>

<body>

<footer>
```

### Commit Message 示例

```
feat: 实现角色性格分析功能

- 添加角色性格特征提取算法
- 实现基于 MBTI 的性格分类
- 支持角色性格一致性检查

Closes #123
```

## 代码规范

### 1. 注释规范

- **包注释**：每个包必须有包级别的注释，说明包的用途和主要功能
- **函数注释**：所有导出函数必须有详细的中文注释，包括参数说明、返回值说明、注意事项
- **结构体注释**：结构体和字段都需要清晰的注释，说明用途和约束条件
- **复杂逻辑注释**：对于复杂的业务逻辑，添加详细的行内注释，说明每个步骤的目的

### 2. 代码复用原则

- **提取公共函数**：避免重复代码，将通用逻辑提取为独立函数
- **使用接口实现多态**：通过接口实现代码复用和解耦，不同的实现策略可以实现同一接口
- **使用组合而非继承**：Go 推荐使用组合模式，通过嵌入结构体复用功能

### 3. 减少冗余

- **避免不必要的中间变量**：如果可以直接返回，就不要创建中间变量
- **使用表驱动测试**：减少测试代码冗余，用数据结构描述测试用例
- **使用常量避免魔法数字**：所有硬编码的数字都应该定义为有意义的常量

### 4. 错误处理规范

- **使用 errors.Is 和 errors.As**：定义明确的错误类型，使用标准库函数判断错误
- **错误信息要清晰具体**：错误信息应包含足够的上下文信息，便于定位问题
- **包装错误保留原始信息**：使用 `%w` 格式化动词包装错误，保留错误链

### 5. 性能优化建议

- **使用 sync.Pool 复用对象**：为频繁创建的对象使用对象池，减少 GC 压力
- **避免不必要的内存分配**：使用 strings.Builder 进行字符串拼接，预分配切片容量
- **合理使用缓存**：对于计算密集或 I/O 密集的操作，使用缓存提高性能

### 6. 并发安全

- **使用互斥锁保护共享资源**：对于共享数据结构，使用 sync.Mutex 或 sync.RWMutex 保护
- **优先使用 channel 通信**：Go 推荐使用 channel 在 goroutine 之间传递数据
- **避免数据竞争**：使用 `go run -race` 检测数据竞争问题

## 测试规范

### 测试覆盖率要求

- 单元测试覆盖率：≥ 80%
- 核心业务逻辑：≥ 90%
- 每个导出函数都应有对应的测试

### 测试文件命名

- 测试文件与源文件同目录
- 命名格式：`xxx_test.go`
- 示例：`character.go` 对应 `character_test.go`

### 测试函数命名

```go
// 格式：Test + 函数名 + 测试场景
func TestCreateCharacter_Success(t *testing.T) { }
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
```

Behavioral guidelines to reduce common LLM coding mistakes. Use when writing, reviewing, or refactoring code to avoid overcomplication, make surgical changes, surface assumptions, and define verifiable success criteria.
## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

