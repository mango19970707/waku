package compose

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"waku/internal/constant"
	"waku/internal/model"
	"waku/internal/plotline/repository"
	"waku/internal/utils"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"
)

// ==================== 提取工作流节点 ====================

// initExtractNode 初始化提取工作流节点（将输入转换为状态）
func initExtractNode(ctx context.Context, input *PlotlineExtractWorkflowInput) (*PlotlineExtractWorkflowState, error) {
	fmt.Printf("\n🤖 [初始化] 准备工作流状态...\n")

	state := &PlotlineExtractWorkflowState{
		CurrentNode: "init",
		NovelID:     input.NovelID,
		NovelName:   input.NovelName,
		Author:      input.Author,
		Genre:       input.Genre,
		Outline:     input.Outline,
		MaxRetries:  constant.MaxRetries,
	}

	fmt.Printf("  ✓ 状态初始化完成\n")
	return state, nil
}

// extractNode 提取剧情线节点
func extractNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
		fmt.Printf("\n🤖 [提取] 开始从大纲提取剧情线...\n")

		// 如果已经提取过，跳过
		if state.ExtractedPlotline != "" {
			fmt.Printf("  ✓ 已有提取的剧情线，跳过\n")
			state.CurrentNode = "extract_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传）
		if state.NovelID > 0 && state.NovelName != "" {
			existingContent, err := utils.LoadFromFile(state.Author, constant.FileTypePlotline, constant.StageExtract)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("加载已有内容失败: %v", err)
				state.CurrentNode = "extract_failed"
				return state, nil
			}
			if existingContent != "" {
				fmt.Printf("  ✓ 从文件加载已有内容\n")
				state.ExtractedPlotline = existingContent
				state.CurrentNode = "extract_success"
				return state, nil
			}
		}

		// 执行提取
		result, err := agentExecutor.ExtractPlotline(ctx, chatModel, state.Outline, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("提取剧情线失败: %v", err)
			state.CurrentNode = "extract_failed"
			return state, nil
		}

		// 保存结果到状态
		state.ExtractedPlotline = result

		// 保存到文件
		if state.NovelName != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypePlotline, constant.StageExtract, result); err != nil {
				fmt.Printf("  ⚠️  保存文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 已保存到文件\n")
			}
		}

		fmt.Printf("  ✓ 剧情线提取完成\n")
		state.CurrentNode = "extract_success"
		state.RetryCount = 0
		return state, nil
	}
}

// polishNode 润色剧情线节点
func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
		fmt.Printf("\n🤖 [润色] 开始润色剧情线...\n")

		// 如果已经润色过，跳过
		if state.PolishedPlotline != "" {
			fmt.Printf("  ✓ 已有润色的剧情线，跳过\n")
			state.CurrentNode = "polish_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传）
		if state.NovelID > 0 && state.NovelName != "" {
			existingContent, err := utils.LoadFromFile(state.Author, constant.FileTypePlotline, constant.StagePolish)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("加载已有内容失败: %v", err)
				state.CurrentNode = "polish_failed"
				return state, nil
			}
			if existingContent != "" {
				fmt.Printf("  ✓ 从文件加载已有内容\n")
				state.PolishedPlotline = existingContent
				state.CurrentNode = "polish_success"
				return state, nil
			}
		}

		// 执行润色
		result, err := agentExecutor.PolishPlotline(ctx, chatModel, state.ExtractedPlotline, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("润色剧情线失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		// 保存结果到状态
		state.PolishedPlotline = result

		// 保存到文件
		if state.NovelName != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypePlotline, constant.StagePolish, result); err != nil {
				fmt.Printf("  ⚠️  保存文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 已保存到文件\n")
			}
		}

		fmt.Printf("  ✓ 剧情线润色完成\n")
		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

// getUserApproval 获取用户审批决定（从终端控制台输入）
func getUserApproval(state *PlotlineExtractWorkflowState) (bool, string) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("⚠️  需要用户确认\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("操作: 保存剧情线到数据库\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fmt.Printf("小说名称: %s\n", state.NovelName)
	fmt.Printf("作者: %s\n", state.Author)
	fmt.Printf("小说类型: %s\n", state.Genre)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if state.NovelID > 0 {
		fmt.Printf("⚠️  这将更新已有小说 (ID: %d)\n", state.NovelID)
	} else {
		fmt.Printf("📝 这将创建新小说\n")
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("是否继续保存? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		return false, "读取输入失败"
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		fmt.Printf("✓ 用户批准，继续保存\n")
		return true, ""
	}

	fmt.Printf("✗ 用户拒绝，取消保存\n")
	fmt.Printf("拒绝原因 (可选，直接回车跳过): ")
	reason, _ := reader.ReadString('\n')
	reason = strings.TrimSpace(reason)

	if reason == "" {
		reason = "用户取消操作"
	}

	return false, reason
}

// saveToDBNode 保存到数据库节点
func saveToDBNode(db *gorm.DB, repo repository.Repository) func(context.Context, *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
		fmt.Printf("\n🤖 [保存] 准备保存到数据库...\n")

		// 如果已经保存过，跳过
		if state.SavedNovelID > 0 {
			fmt.Printf("  ✓ 已保存，小说ID: %d\n", state.SavedNovelID)
			state.CurrentNode = "save_success"
			return state, nil
		}

		// 中断恢复逻辑：首次调用时触发中断，等待用户审批
		if !state.WaitingForApproval {
			// 首次调用：触发中断，等待用户审批
			state.WaitingForApproval = true
			state.CurrentNode = "save_waiting_approval"

			// 获取用户审批
			approved, reason := getUserApproval(state)
			state.UserApproved = approved
			state.DisapproveReason = reason

			// 如果用户拒绝，直接返回失败
			if !approved {
				state.ErrorMsg = fmt.Sprintf("用户拒绝保存: %s", reason)
				state.CurrentNode = "save_rejected"
				state.WaitingForApproval = false
				return state, nil
			}

			// 用户批准，继续执行保存逻辑
			state.WaitingForApproval = false
		}

		// 执行保存操作
		fmt.Printf("\n🤖 [保存] 开始保存到数据库...\n")

		// 判断是新增还是更新
		if state.NovelID > 0 {
			// 更新已有小说
			fmt.Printf("  → 更新已有小说 (ID: %d)\n", state.NovelID)

			// 查询已有小说
			var existingNovel model.Novel
			if err := db.First(&existingNovel, state.NovelID).Error; err != nil {
				state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}

			// 更新字段
			existingNovel.Title = state.NovelName
			existingNovel.Author = state.Author
			existingNovel.Genre = state.Genre

			// 保存更新
			if err := db.Save(&existingNovel).Error; err != nil {
				state.ErrorMsg = fmt.Sprintf("更新小说失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}

			state.SavedNovelID = existingNovel.ID
			fmt.Printf("  ✓ 更新成功，小说ID: %d\n", state.SavedNovelID)
		} else {
			// 新增小说
			fmt.Printf("  → 创建新小说\n")

			// 检查作者是否已有同名小说
			isDuplicate, err := repo.CheckDuplicateByAuthorAndTitle(state.Author, state.NovelName)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("检查重名失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}

			if isDuplicate {
				state.ErrorMsg = fmt.Sprintf("作者 %s 已有同名小说《%s》，请修改小说名称后重试", state.Author, state.NovelName)
				state.CurrentNode = "save_failed"
				fmt.Printf("  ❌ %s\n", state.ErrorMsg)
				return state, nil
			}

			novel := &model.Novel{
				Title:  state.NovelName,
				Author: state.Author,
				Genre:  state.Genre,
				Status: "draft",
			}

			// 保存到数据库
			if err := repo.Create(novel); err != nil {
				state.ErrorMsg = fmt.Sprintf("保存到数据库失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}

			state.SavedNovelID = novel.ID
			state.NovelID = novel.ID // 更新状态中的 NovelID
			fmt.Printf("  ✓ 保存成功，小说ID: %d\n", state.SavedNovelID)
		}

		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

// ==================== 修改工作流节点 ====================

// initRevisionNode 初始化修改工作流节点（从数据库加载小说信息）
func initRevisionNode(db *gorm.DB) func(context.Context, *PlotlineRevisionWorkflowInput) (*PlotlineRevisionWorkflowState, error) {
	return func(ctx context.Context, input *PlotlineRevisionWorkflowInput) (*PlotlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [初始化] 从数据库加载小说信息...\n")

		// 查询小说信息
		var novel model.Novel
		if err := db.First(&novel, input.NovelID).Error; err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		// 尝试从文件加载剧情线
		plotline := ""
		// 优先从润色阶段加载
		if content, err := utils.LoadFromFile(novel.Author, constant.FileTypePlotline, constant.StagePolish); err == nil && content != "" {
			plotline = content
		} else if content, err := utils.LoadFromFile(novel.Author, constant.FileTypePlotline, constant.StageExtract); err == nil && content != "" {
			plotline = content
		}

		if plotline == "" {
			return nil, fmt.Errorf("未找到剧情线文件")
		}

		state := &PlotlineRevisionWorkflowState{
			CurrentNode:      "init",
			NovelID:          novel.ID,
			NovelName:        novel.Title,
			Author:           novel.Author,
			Genre:            novel.Genre,
			OriginalPlotline: plotline,
			UserSuggestion:   input.UserSuggestion,
			MaxRetries:       constant.MaxRetries,
		}

		fmt.Printf("  ✓ 小说信息加载完成\n")
		fmt.Printf("    - 标题: %s\n", novel.Title)
		fmt.Printf("    - 作者: %s\n", novel.Author)
		fmt.Printf("    - 类型: %s\n", novel.Genre)
		fmt.Printf("    - 剧情线长度: %d 字符\n", len(plotline))

		return state, nil
	}
}

// polishRevisionNode 润色修改后的剧情线节点
func polishRevisionNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [润色] 开始润色剧情线（根据用户建议）...\n")

		// 执行润色（将用户建议融入润色过程）
		// 构建包含用户建议的剧情线
		plotlineWithSuggestion := fmt.Sprintf(`原始剧情线：
%s

用户修改建议：
%s

请根据用户的修改建议，对剧情线进行润色和优化。`, state.OriginalPlotline, state.UserSuggestion)

		result, err := agentExecutor.PolishPlotline(ctx, chatModel, plotlineWithSuggestion, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("润色剧情线失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		// 保存结果
		state.RevisedPlotline = result

		// 保存到文件
		if err := utils.SaveToFile(state.Author, constant.FileTypePlotline, constant.StageRevise, result); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存文件失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 剧情线润色完成，已保存到文件\n")
		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

// getUserApprovalForRevision 获取用户审批决定（修改工作流）
func getUserApprovalForRevision(state *PlotlineRevisionWorkflowState) (bool, string) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("⚠️  需要用户确认\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("操作: 更新剧情线到数据库\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fmt.Printf("小说名称: %s\n", state.NovelName)
	fmt.Printf("作者: %s\n", state.Author)
	fmt.Printf("小说类型: %s\n", state.Genre)
	fmt.Printf("小说ID: %d\n", state.NovelID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("是否继续更新? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		return false, "读取输入失败"
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		fmt.Printf("✓ 用户批准，继续更新\n")
		return true, ""
	}

	fmt.Printf("✗ 用户拒绝，取消更新\n")
	fmt.Printf("拒绝原因 (可选，直接回车跳过): ")
	reason, _ := reader.ReadString('\n')
	reason = strings.TrimSpace(reason)

	if reason == "" {
		reason = "用户取消操作"
	}

	return false, reason
}

// updateDBNode 更新数据库节点（修改工作流）
func updateDBNode(db *gorm.DB) func(context.Context, *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [更新] 准备更新数据库...\n")

		// 中断恢复逻辑：首次调用时触发中断，等待用户审批
		if !state.WaitingForApproval {
			// 首次调用：触发中断，等待用户审批
			state.WaitingForApproval = true
			state.CurrentNode = "update_waiting_approval"

			// 获取用户审批
			approved, reason := getUserApprovalForRevision(state)
			state.UserApproved = approved
			state.DisapproveReason = reason

			// 如果用户拒绝，直接返回失败
			if !approved {
				state.ErrorMsg = fmt.Sprintf("用户拒绝更新: %s", reason)
				state.CurrentNode = "update_rejected"
				state.WaitingForApproval = false
				return state, nil
			}

			// 用户批准，继续执行更新逻辑
			state.WaitingForApproval = false
		}

		// 执行更新操作
		fmt.Printf("\n🤖 [更新] 开始更新数据库...\n")

		// 查询已有小说
		var existingNovel model.Novel
		if err := db.First(&existingNovel, state.NovelID).Error; err != nil {
			state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
			state.CurrentNode = "update_failed"
			return state, nil
		}

		// 更新字段
		existingNovel.Title = state.NovelName
		existingNovel.Author = state.Author
		existingNovel.Genre = state.Genre

		// 保存更新
		if err := db.Save(&existingNovel).Error; err != nil {
			state.ErrorMsg = fmt.Sprintf("更新小说失败: %v", err)
			state.CurrentNode = "update_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 更新成功，小说ID: %d\n", existingNovel.ID)
		state.CurrentNode = "update_success"
		state.RetryCount = 0
		return state, nil
	}
}
