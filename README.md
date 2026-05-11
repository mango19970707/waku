# Waku - AI 小说生成 Agent

一个基于 AI 的智能小说创作系统，通过 Agent 架构实现小说的自动化生成和管理。

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
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password  # 修改为你的数据库密码
DB_NAME=waku
DB_SSLMODE=disable
DB_LOG_LEVEL=info

# 应用配置
APP_PORT=8080
APP_ENV=development

# AI 模型配置
MODEL_PROVIDER=openai
MODEL_API_KEY=your_openai_api_key  # 修改为你的 API 密钥
MODEL_NAME=gpt-4
MODEL_TIMEOUT=60s
MODEL_MAX_TOKENS=2000
MODEL_TEMPERATURE=0.7
```

#### 方式二：使用 config.yaml 文件（推荐用于生产环境）

编辑 `config/config.yaml` 文件：

```yaml
# 数据库配置
database:
  host: localhost
  port: 5432
  user: postgres
  password: your_password  # 修改为你的数据库密码
  dbname: waku
  sslmode: disable
  log_level: info

# 应用配置
app:
  port: 8080
  env: development

# AI 模型配置
model:
  provider: openai
  api_key: your_api_key  # 修改为你的 API 密钥
  model: gpt-4
  timeout: 60s
  max_tokens: 2000
  temperature: 0.7
```

**注意**：环境变量的优先级高于 config.yaml 文件。如果同时设置了环境变量和 YAML 配置，将使用环境变量的值。

### 3. 创建数据库

在 PostgreSQL 中创建数据库：

```sql
CREATE DATABASE waku;
```

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
├── main.go                  # 程序入口
├── go.mod                   # Go 模块依赖
├── .env                     # 环境变量配置（不提交到 Git）
├── .env.example             # 环境变量配置示例
├── .gitignore               # Git 忽略文件
├── config/                  # 配置文件目录
│   ├── config.yaml          # 应用配置
│   └── config.yaml.example  # 配置示例
├── internal/                # 内部业务逻辑
│   ├── config/              # 配置加载
│   │   └── config.go        # 配置结构和加载逻辑
│   ├── database/            # 数据库连接和迁移
│   │   ├── database.go      # 数据库连接
│   │   └── migrate.go       # 数据库迁移
│   ├── model/               # 数据模型定义
│   │   ├── types.go         # 自定义类型（JSON 数组支持）
│   │   ├── novel.go         # 小说模型
│   │   ├── volume.go        # 分卷模型
│   │   ├── chapter.go       # 章节模型
│   │   ├── abstract.go      # 大纲模型
│   │   ├── character.go     # 角色模型
│   │   ├── plotline.go      # 剧情线模型
│   │   └── memory.go        # 记忆模型
│   ├── utils/               # 工具函数
│   │   ├── chat_model.go    # ChatModel 封装
│   │   └── config_helper.go # 配置辅助函数
│   ├── abstract/            # 章节大纲模块
│   ├── chapter/             # 章节内容模块
│   ├── character/           # 角色管理模块
│   ├── memory/              # 记忆系统模块
│   ├── plotline/            # 剧情管理模块
│   └── volume/              # 分卷管理模块
└── docs/                    # 项目文档
```

## 数据模型

### Novel（小说）
- 小说基本信息：标题、作者、类型、简介、状态

### Volume（分卷）
- 分卷信息：标题、序号、简介

### Chapter（章节）
- 章节信息：标题、内容、字数、状态

### Abstract（章节大纲）
- 大纲信息：摘要、关键要点、涉及角色和剧情线

### Character（角色）
- 角色信息：名称、定位、外貌、性格、背景

### Plotline（剧情线）
- 剧情线信息：标题、类型（主线/支线）、描述、状态

### Memory（记忆）
- 记忆信息：类型（事件/状态/设定）、内容、重要程度

## 技术栈

- **语言**: Go 1.25
- **数据库**: PostgreSQL
- **ORM**: GORM
- **配置**: YAML
- **AI 模型**: 支持 OpenAI、Azure OpenAI、Ollama、Ark、Anthropic

## 开发规范

详见 [CLAUDE.md](./CLAUDE.md)

## License

MIT
