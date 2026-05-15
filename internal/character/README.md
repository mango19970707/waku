# Character 模块 - 小说人物提取优化工作流

## 概述

Character 模块是 Waku 项目中负责小说人物信息提取和优化的核心模块。通过 AI Agent 和工作流编排，实现从小说大纲中自动提取人物信息、根据用户建议修改人物等功能。

## 目录结构

```
internal/character/
├── agents/                                    # Agent 实现层（2个Agent）
│   ├── character_extractor_agent.go           # 人物提取 Agent
│   └── character_revision_agent.go            # 人物优化 Agent
├── compose/                                   # 工作流编排层
│   ├── character_extract_workflow.go          # 人物提取工作流
│   ├── character_revision_workflow.go         # 人物修改工作流
│   ├── nodes.go                               # 工作流节点实现
│   ├── router.go                              # 工作流路由逻辑
│   └── agent_executor.go                      # Agent 执行器（工厂模式）
├── prompts/                                   # 提示词层（2个提示词）
│   ├── character_extractor_prompt.go          # 提取提示词
│   └── character_revision_prompt.go           # 优化提示词
└── repository/                                # 数据仓储层
    ├── character_repository.go                # 人物数据访问
    └── repository.go                          # 仓储接口定义
```

## 核心功能

### 1. 人物提取（CharacterExtractorAgent）

从小说大纲中自动提取所有重要人物的详细信息。

**提取字段：**
- `name`：人物姓名
- `role`：角色定位（protagonist/supporting/antagonist/mentor/love_interest）
- `age`：年龄（选填）
- `gender`：性别（选填）
- `appearance`：外貌描述（100-500字）
- `personality`：性格特征（100-500字）
- `background`：背景故事（200-800字）

**输出格式：** JSON

### 2. 人物优化（CharacterRevisionAgent）

根据用户建议对人物信息进行修改、删除和新增。

**支持操作：**
- **修改现有人物**：保留原有 ID，修改部分或全部字段
- **删除人物**：直接在输出中省略该人物
- **新增人物**：ID 设置为空字符串，系统自动生成 UUID

## 工作流

### 人物提取工作流（CharacterExtractWorkflow）

**流程图：**

```
START → init → extract → saveToDB → END
         ↓       ↓         ↓
      初始化   提取      保存
      (1步)  (3重试)   (3重试)
```

**节点说明：**

1. **initNode**：初始化工作流状态
   - 从数据库加载小说信息
   - 验证大纲是否存在
   - 初始化工作流状态

2. **extractNode**：提取人物信息
   - 支持断点续传（从文件加载已有内容）
   - 调用 CharacterExtractorAgent 执行提取
   - 保存到 `data/content/{novelID-novelName}/character/提取.json`
   - 失败自动重试 3 次

3. **saveToDBNode**：保存到数据库
   - 解析 JSON 数据
   - 转换为 `model.Character` 对象
   - 批量插入到数据库
   - 收集保存后的 ID 列表

### 人物修改工作流（CharacterRevisionWorkflow）

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
   - 查询该小说的所有人物
   - 将人物转换为 JSON 格式

2. **reviseNode**：优化人物信息
   - 调用 CharacterRevisionAgent 执行优化
   - 保存到 `data/content/{novelID-novelName}/character/修改.json`
   - 失败自动重试 3 次

3. **saveOrUpdateDBNode**：保存或更新到数据库
   - 解析 JSON 数据
   - 分类处理：有 ID 的更新，无 ID 的新增
   - 批量执行更新和插入操作
