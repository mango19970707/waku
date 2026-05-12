package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"waku/internal/model"
	"waku/internal/novel/tools"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// OutlineWorkflowState 大纲工作流状态
type OutlineWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string
	// NovelName 小说名称（用于文件保存路径）
	NovelName string
	// Genre 小说类型
	Genre string
	// OriginalOutline 原始大纲
	OriginalOutline string
	// GeneratedOutline 生成的大纲
	GeneratedOutline string
	// ExpandedOutline 扩写后的大纲
	ExpandedOutline string
	// PolishedOutline 润色后的大纲
	PolishedOutline string
	// NovelID 保存到数据库后的小说ID
	NovelID string
	// ErrorMsg 错误信息
	ErrorMsg string
	// RetryCount 当前节点的重试次数
	RetryCount int
	// MaxRetries 最大重试次数
	MaxRetries int
}

// OutlineWorkflowInput 大纲工作流输入
type OutlineWorkflowInput struct {
	// NovelName 小说名称
	NovelName string
	// Genre 小说类型
	Genre string
	// OriginalOutline 原始大纲
	OriginalOutline string
}

// ==================== 文件保存辅助函数 ====================

// saveToFile 保存内容到文件
func saveToFile(novelName, stage, content string) error {
	// 构建文件路径：data/content/{小说名称}/abstract/{阶段}.md
	dir := filepath.Join("data", "content", novelName, "abstract")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(dir, fmt.Sprintf("%s.md", stage))

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// loadFromFile 从文件加载内容（用于断点续传）
func loadFromFile(novelName, stage string) (string, error) {
	filePath := filepath.Join("data", "content", novelName, "abstract", fmt.Sprintf("%s.md", stage))

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空字符串
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	return string(content), nil
}

// ==================== 节点函数 ====================

// initNode 初始化节点（将输入转换为状态）
func initNode(ctx context.Context, input *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	fmt.Printf("\n🤖 [初始化] 准备工作流状态...\n")

	state := &OutlineWorkflowState{
		CurrentNode:     "init",
		NovelName:       input.NovelName,
		Genre:           input.Genre,
		OriginalOutline: input.OriginalOutline,
		MaxRetries:      3, // 默认最大重试次数为3
	}

	fmt.Printf("  ✓ 状态初始化完成\n")
	return state, nil
}

// generateNode 生成大纲节点
func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [生成] 开始生成大纲...\n")

		// 尝试从文件加载（断点续传）
		if state.GeneratedOutline == "" {
			existingContent, err := loadFromFile(state.NovelName, "生成")
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("加载已有内容失败: %v", err)
				state.CurrentNode = "generate_failed"
				return state, nil
			}
			if existingContent != "" {
				fmt.Printf("  ✓ 从文件加载已有内容\n")
				state.GeneratedOutline = existingContent
				state.CurrentNode = "generate_success"
				return state, nil
			}
		}

		// 创建生成 Agent
		agent, err := NewOutlineGeneratorAgent(ctx, chatModel)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("创建生成 Agent 失败: %v", err)
			state.CurrentNode = "generate_failed"
			return state, nil
		}

		// 执行生成
		input := &OutlineGeneratorInput{
			OriginalOutline: state.OriginalOutline,
			Genre:           state.Genre,
		}

		result, err := GenerateOutline(ctx, agent, input)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("生成大纲失败: %v", err)
			state.CurrentNode = "generate_failed"
			return state, nil
		}

		// 保存结果
		state.GeneratedOutline = result

		// 保存到文件
		if err := saveToFile(state.NovelName, "生成", result); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存文件失败: %v", err)
			state.CurrentNode = "generate_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 大纲生成完成，已保存到文件\n")
		state.CurrentNode = "generate_success"
		state.RetryCount = 0
		return state, nil
	}
}

// expandNode 扩写大纲节点
func expandNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [扩写] 开始扩写大纲...\n")

		// 尝试从文件加载（断点续传）
		if state.ExpandedOutline == "" {
			existingContent, err := loadFromFile(state.NovelName, "扩写")
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("加载已有内容失败: %v", err)
				state.CurrentNode = "expand_failed"
				return state, nil
			}
			if existingContent != "" {
				fmt.Printf("  ✓ 从文件加载已有内容\n")
				state.ExpandedOutline = existingContent
				state.CurrentNode = "expand_success"
				return state, nil
			}
		}

		// 创建扩写 Agent
		agent, err := NewOutlineExpanderAgent(ctx, chatModel)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("创建扩写 Agent 失败: %v", err)
			state.CurrentNode = "expand_failed"
			return state, nil
		}

		// 执行扩写
		input := &OutlineExpanderInput{
			BriefOutline: state.GeneratedOutline,
			Genre:        state.Genre,
		}

		result, err := ExpandOutline(ctx, agent, input)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("扩写大纲失败: %v", err)
			state.CurrentNode = "expand_failed"
			return state, nil
		}

		// 保存结果
		state.ExpandedOutline = result

		// 保存到文件
		if err := saveToFile(state.NovelName, "扩写", result); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存文件失败: %v", err)
			state.CurrentNode = "expand_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 大纲扩写完成，已保存到文件\n")
		state.CurrentNode = "expand_success"
		state.RetryCount = 0
		return state, nil
	}
}

// polishNode 润色大纲节点
func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [润色] 开始润色大纲...\n")

		// 尝试从文件加载（断点续传）
		if state.PolishedOutline == "" {
			existingContent, err := loadFromFile(state.NovelName, "润色")
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("加载已有内容失败: %v", err)
				state.CurrentNode = "polish_failed"
				return state, nil
			}
			if existingContent != "" {
				fmt.Printf("  ✓ 从文件加载已有内容\n")
				state.PolishedOutline = existingContent
				state.CurrentNode = "polish_success"
				return state, nil
			}
		}

		// 创建润色 Agent（不带数据库工具，因为我们在单独的节点中保存）
		agent, err := NewOutlinePolisherAgent(ctx, chatModel, nil, nil)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("创建润色 Agent 失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		// 执行润色
		input := &OutlinePolisherInput{
			Outline: state.ExpandedOutline,
			Genre:   state.Genre,
		}

		result, err := PolishOutline(ctx, agent, input)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("润色大纲失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		// 保存结果
		state.PolishedOutline = result

		// 保存到文件
		if err := saveToFile(state.NovelName, "润色", result); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存文件失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 大纲润色完成，已保存到文件\n")
		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

// saveToDBNode 保存到数据库节点
func saveToDBNode(db *gorm.DB, repo tools.Repository) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [保存] 开始保存到数据库...\n")

		// 如果已经保存过，跳过
		if state.NovelID != "" {
			fmt.Printf("  ✓ 已保存，小说ID: %s\n", state.NovelID)
			state.CurrentNode = "save_success"
			return state, nil
		}

		// 从大纲中提取世界观和修行体系
		worldview := extractWorldview(state.PolishedOutline)
		cultivationSystem := extractCultivationSystem(state.PolishedOutline)

		// 创建小说对象
		novel := &model.Novel{
			Title:             state.NovelName,
			Author:            "AI创作",
			Genre:             state.Genre,
			Synopsis:          fmt.Sprintf("这是一部%s类型的小说", state.Genre),
			Worldview:         worldview,
			CultivationSystem: cultivationSystem,
			Status:            "draft",
		}

		// 保存到数据库
		if err := repo.Create(novel); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存到数据库失败: %v", err)
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 保存结果
		state.NovelID = novel.ID

		fmt.Printf("  ✓ 保存成功，小说ID: %s\n", state.NovelID)
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

// extractWorldview 从大纲中提取世界观内容（简化版本）
func extractWorldview(outline string) string {
	// 简化实现：查找包含"世界观"的段落
	if idx := strings.Index(outline, "世界观"); idx != -1 {
		// 提取该段落
		end := strings.Index(outline[idx:], "\n\n")
		if end == -1 {
			end = len(outline) - idx
		}
		return strings.TrimSpace(outline[idx : idx+end])
	}
	return "暂无详细世界观设定"
}

// extractCultivationSystem 从大纲中提取修行体系内容（简化版本）
func extractCultivationSystem(outline string) string {
	// 简化实现：查找包含"修行体系"或"修炼体系"的段落
	keywords := []string{"修行体系", "修炼体系", "境界"}
	for _, keyword := range keywords {
		if idx := strings.Index(outline, keyword); idx != -1 {
			// 提取该段落
			end := strings.Index(outline[idx:], "\n\n")
			if end == -1 {
				end = len(outline) - idx
			}
			return strings.TrimSpace(outline[idx : idx+end])
		}
	}
	return "暂无详细修行体系设定"
}

// ==================== 路由函数 ====================

// routeAfterGenerate 生成后的路由
func routeAfterGenerate(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "generate_success" {
		return "expand", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 生成失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("生成失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  生成失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "generate", nil
}

// routeAfterExpand 扩写后的路由
func routeAfterExpand(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "expand_success" {
		return "polish", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 扩写失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("扩写失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  扩写失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "expand", nil
}

// routeAfterPolish 润色后的路由
func routeAfterPolish(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "polish_success" {
		return "saveToDB", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 润色失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("润色失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  润色失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "polish", nil
}

// routeAfterSave 保存后的路由
func routeAfterSave(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 保存失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("保存失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  保存失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "saveToDB", nil
}

// ==================== 构建 Graph ====================

// BuildOutlineWorkflowGraph 构建大纲工作流 Graph
//
// 功能：创建一个完整的大纲生成工作流，包括生成、扩写、润色、保存四个步骤
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接
//   - repo: 小说仓储实现
//
// 返回值：
//   - compose.Runnable: 可执行的 Graph
//   - error: 创建失败时返回错误
//
// 工作流程：
//  1. 生成：将原始创意扩展为完整大纲（失败重试3次）
//  2. 扩写：将大纲扩展为详细的故事框架（失败重试3次）
//  3. 润色：对大纲进行深度优化和提升（失败重试3次）
//  4. 保存：将大纲保存到数据库（失败重试3次）
//
// 特性：
//  - 每个步骤的输出保存到 data/content/{小说名称}/abstract/{阶段}.md
//  - 支持断点续传：如果文件已存在，直接加载而不重新生成
//  - 工具调用失败自动重试3次
func BuildOutlineWorkflowGraph(ctx context.Context, chatModel einoModel.BaseChatModel, db *gorm.DB, repo tools.Repository) (compose.Runnable[*OutlineWorkflowInput, *OutlineWorkflowState], error) {
	graph := compose.NewGraph[*OutlineWorkflowInput, *OutlineWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("expand", compose.InvokableLambda(expandNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(db, repo)))

	// 添加边
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "generate")

	// 添加条件分支
	graph.AddBranch("generate", compose.NewGraphBranch(
		routeAfterGenerate,
		map[string]bool{
			"generate": true,
			"expand":   true,
		},
	))

	graph.AddBranch("expand", compose.NewGraphBranch(
		routeAfterExpand,
		map[string]bool{
			"expand": true,
			"polish": true,
		},
	))

	graph.AddBranch("polish", compose.NewGraphBranch(
		routeAfterPolish,
		map[string]bool{
			"polish":   true,
			"saveToDB": true,
		},
	))

	graph.AddBranch("saveToDB", compose.NewGraphBranch(
		routeAfterSave,
		map[string]bool{
			"saveToDB": true,
			compose.END: true,
		},
	))

	// 编译 Graph，设置最大步数以支持重试
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// ExecuteOutlineWorkflow 执行大纲工作流
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *OutlineWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecuteOutlineWorkflow(ctx context.Context, graph compose.Runnable[*OutlineWorkflowInput, *OutlineWorkflowState], input *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	if input.NovelName == "" {
		return nil, fmt.Errorf("小说名称不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.OriginalOutline == "" {
		return nil, fmt.Errorf("原始大纲不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📚 开始执行大纲工作流\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说名称: %s\n", input.NovelName)
	fmt.Printf("小说类型: %s\n", input.Genre)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 执行 Graph
	result, err := graph.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("工作流执行失败: %w", err)
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ 工作流执行完成\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("最终状态: %s\n", result.CurrentNode)
	if result.NovelID != "" {
		fmt.Printf("小说ID: %s\n", result.NovelID)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}

// ==================== 兼容旧接口 ====================

// NewOutlineWorkflowAgent 创建小说大纲工作流 Agent（兼容旧接口）
//
// 注意：此函数保留用于向后兼容，新代码应使用 BuildOutlineWorkflowGraph
func NewOutlineWorkflowAgent(ctx context.Context, chatModel einoModel.BaseChatModel, db *gorm.DB, repo tools.Repository) (adk.ResumableAgent, error) {
	// 创建三个子 Agent
	generatorAgent, err := NewOutlineGeneratorAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建大纲生成 Agent 失败: %w", err)
	}

	expanderAgent, err := NewOutlineExpanderAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建大纲扩写 Agent 失败: %w", err)
	}

	polisherAgent, err := NewOutlinePolisherAgent(ctx, chatModel, db, repo)
	if err != nil {
		return nil, fmt.Errorf("创建大纲润色 Agent 失败: %w", err)
	}

	// 创建串行工作流 Agent
	return utils.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "OutlineWorkflow",
		Description: "小说大纲完整工作流：生成 -> 扩写 -> 润色",
		SubAgents: []adk.Agent{
			generatorAgent,
			expanderAgent,
			polisherAgent,
		},
	})
}

// GenerateOutlineWithWorkflow 使用工作流生成完整的小说大纲（兼容旧接口）
//
// 注意：此函数保留用于向后兼容，新代码应使用 ExecuteOutlineWorkflow
func GenerateOutlineWithWorkflow(ctx context.Context, agent adk.ResumableAgent, input *OutlineGeneratorInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息生成、扩写并润色小说大纲：

**小说类型**：%s

**原始大纲**：
%s

请完成完整的大纲创作流程。`, input.Genre, input.OriginalOutline)

	// 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("工作流执行失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				result += msg.Content
			}
		}
	}

	return result, nil
}
