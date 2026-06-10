# Waku - AI 小说生成 Agent

一个基于 AI 的智能小说创作系统，通过 Agent 架构和工作流编排实现小说大纲的自动化生成、优化和管理。

## 核心特性

- **智能大纲生成**：基于用户输入自动生成完整的小说大纲
- **多阶段优化**：生成 → 扩写 → 润色，逐步提升大纲质量
- **结构化提取**：自动提取标题、简介、世界观、修行体系等关键信息
- **大纲修改**：根据用户建议智能修改已有大纲
- **断点续传**：支持工作流中断后从上次位置继续
- **中断恢复**：保存前需用户审批，防止误操作
- **多模型支持**：支持 OpenAI、Azure OpenAI、Ollama、Ark、Anthropic Claude

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置数据库和 AI 模型

有两种方式配置应用：

#### 方式一：使用 .env 文件（推荐用于开发环境）

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```bash
# 数据库配置
DB_DRIVER=sqlite                    # 或 postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password           # 修改为你的数据库密码
DB_NAME=waku
DB_SSLMODE=disable
DB_LOG_LEVEL=info
DB_DATA_PATH=./data/waku.db         # SQLite 数据库文件路径

# 应用配置
APP_PORT=6060
APP_ENV=development

# AI 模型配置
MODEL_PROVIDER=openai               # openai/azure/ollama/ark/anthropic
MODEL_API_KEY=your_openai_api_key   # 修改为你的 API 密钥
MODEL_BASE_URL=                     # 可选，自定义 API 地址
MODEL_NAME=gpt-4                    # 模型名称
MODEL_TIMEOUT=60s
MODEL_MAX_TOKENS=2000
MODEL_TEMPERATURE=0.7
```

#### 方式二：使用 config.yaml 文件（推荐用于生产环境）

编辑 `config/config.yaml` 文件：

```yaml
# 数据库配置
database:
  driver: sqlite                    # 或 postgres
  host: localhost
  port: 5432
  user: postgres
  password: your_password           # 修改为你的数据库密码
  dbname: waku
  sslmode: disable
  log_level: info
  data_path: ./data/waku.db         # SQLite 数据库文件路径

# 应用配置
app:
  port: 6060
  env: development

# AI 模型配置
model:
  provider: openai                  # openai/azure/ollama/ark/anthropic
  api_key: your_api_key             # 修改为你的 API 密钥
  base_url: ""                      # 可选，自定义 API 地址
  model: gpt-4                      # 模型名称
  timeout: 60s
  max_tokens: 2000
  temperature: 0.7
```

**注意**：环境变量的优先级高于 config.yaml 文件。如果同时设置了环境变量和 YAML 配置，将使用环境变量的值。

### 3. 创建数据库（仅 PostgreSQL）

如果使用 PostgreSQL，需要先创建数据库：

```sql
CREATE DATABASE waku;
```

如果使用 SQLite，程序会自动创建数据库文件。

### 4. 执行数据库迁移

```bash
# 执行迁移（创建或更新表结构）
go run main.go -migrate

# 使用自定义配置文件
go run main.go -config path/to/config.yaml -migrate

# 重置数据库（删除所有表后重新创建，危险操作）
go run main.go -reset
```

### 5. 运行应用

```bash
go run main.go
```

## 项目结构

```
waku/
├── main.go                          # 程序入口
├── go.mod                           # Go 模块依赖
├── .env                             # 环境变量配置（不提交到 Git）
├── .env.example                     # 环境变量配置示例
├── .gitignore                       # Git 忽略文件
├── config/                          # 配置文件目录
│   ├── config.yaml                  # 应用配置
│   └── config.yaml.example          # 配置示例
├── data/                            # 数据存储目录
│   ├── waku.db                      # SQLite 数据库文件
│   └── content/                     # 小说内容文件
│       └── {novelID-novelName}/     # 每个小说的目录
│           └── abstract/            # 大纲文件
│               ├── 生成.md
│               ├── 扩写.md
│               ├── 润色.md
│               └── 修改.md
└── internal/                        # 内部业务逻辑
    ├── config/                      # 配置加载
    │   └── config.go                # 配置结构和加载逻辑
    ├── database/                    # 数据库连接和迁移
    │   ├── database.go              # 数据库连接
    │   └── migrate.go               # 数据库迁移
    ├── model/                       # 数据模型定义
    │   ├── types.go                 # 自定义类型（StringArray）
    │   └── novel.go                 # 小说模型
    ├── utils/                       # 通用工具函数
    │   ├── chat_model.go            # ChatModel 封装
    │   ├── file.go                  # 文件操作
    │   └── agent.go                 # Agent 创建工厂
    └── novel/                       # 小说管理核心模块
        ├── agents/                  # AI Agent 实现
        │   ├── outline_generator_agent.go      # 大纲生成 Agent
        │   ├── outline_expander_agent.go       # 大纲扩写 Agent
        │   ├── outline_polisher_agent.go       # 大纲润色 Agent
        │   ├── outline_extractor_agent.go      # 信息提取 Agent
        │   └── outline_reviser_agent.go        # 大纲修改 Agent
        ├── compose/                 # 工作流编排
        │   ├── outline_generate_workflow.go    # 大纲生成工作流
        │   ├── outline_revision_workflow.go    # 大纲修改工作流
        │   ├── nodes.go                        # 工作流节点实现
        │   ├── router.go                       # 工作流路由逻辑
        │   └── agent_executor.go               # Agent 执行器
        ├── prompts/                 # Agent 系统提示词
        │   ├── outline_generator_prompt.go
        │   ├── outline_expander_prompt.go
        │   ├── outline_polisher_prompt.go
        │   ├── outline_extractor_prompt.go
        │   ├── outline_reviser_prompt.go
        │   └── outline_revision_prompt.go
        ├── tools/                   # 工具定义
        │   └── save_outline_tool.go            # 保存大纲工具
        ├── repository/              # 数据仓储
        │   └── novel_repository.go             # 小说数据访问
        └── utils/                   # 工具函数
            └── file.go                         # 文件操作
```

## 核心工作流

### 大纲生成工作流

```
用户输入（小说名称、类型、原始大纲）
    ↓
[初始化] 创建工作流状态
    ↓
[生成] OutlineGeneratorAgent 生成完整大纲
    ↓ 保存到 data/content/{name}/abstract/生成.md
[扩写] OutlineExpanderAgent 扩写大纲细节
    ↓ 保存到 data/content/{name}/abstract/扩写.md
[润色] OutlinePolisherAgent 润色和优化大纲
    ↓ 保存到 data/content/{name}/abstract/润色.md
[提取] OutlineExtractorAgent 提取结构化信息
    ↓ 输出 JSON（标题、简介、世界观、修行体系等）
[保存] 保存到数据库（需用户审批）
    ↓
完成（返回小说 ID）
```

**特性：**
- 每个节点失败自动重试 3 次
- 支持断点续传（文件已存在则跳过）
- 保存前需用户审批（中断恢复机制）

### 大纲修改工作流

```
用户输入（小说 ID、修改建议）
    ↓
[加载] 从数据库加载现有小说
    ↓
[修改] OutlineReviserAgent 根据建议修改大纲
    ↓ 保存到 data/content/{name}/abstract/修改.md
[提取] OutlineExtractorAgent 提取修改后的信息
    ↓
[更新] 更新数据库
    ↓
完成
```

## 技术栈

- **语言**: Go 1.25.0
- **核心框架**: Cloudwego Eino v0.8.13（Agent 和 Workflow 编排）
- **数据库**: GORM + SQLite/PostgreSQL
- **配置**: YAML + 环境变量
- **AI 模型**: 支持 OpenAI、Azure OpenAI、Ollama、Ark、Anthropic Claude

## 开发规范

详见 [CLAUDE.md](./CLAUDE.md)

## License

MIT
