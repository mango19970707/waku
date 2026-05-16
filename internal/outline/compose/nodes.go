package compose

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"waku/internal/constant"
	"waku/internal/model"
	"waku/internal/outline/repository"
	"waku/internal/utils"

	einoModel "github.com/cloudwego/eino/components/model"
)

// initNode 初始化节点（将输入转换为状态）
func initNode(ctx context.Context, input *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	fmt.Printf("\n🤖 [初始化] 准备工作流状态...\n")

	state := &OutlineWorkflowState{
		CurrentNode:     "init",
		NovelID:         input.NovelID,
		NovelName:       input.NovelName,
		Author:          input.Author,
		Genre:           input.Genre,
		OriginalOutline: input.OriginalOutline,
		MaxRetries:      constant.MaxRetries,
	}

	fmt.Printf("  ✓ 状态初始化完成\n")
	return state, nil
}

// generateNode 生成大纲节点
func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [生成] 开始生成大纲...\n")

		// 如果已经生成过，跳过
		if state.GeneratedOutline != "" {
			fmt.Printf("  ✓ 已有生成的大纲，跳过\n")
			state.CurrentNode = "generate_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传，仅在有 NovelID 时）
		if state.NovelID > 0 {
			existingContent, err := utils.LoadFromFile(state.Author, constant.FileTypeOutline, constant.StageGenerate)
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

		// 执行生成
		result, err := agentExecutor.GenerateOutline(ctx, chatModel, state.OriginalOutline, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("生成大纲失败: %v", err)
			state.CurrentNode = "generate_failed"
			return state, nil
		}

		// 保存结果到状态（不立即写文件）
		state.GeneratedOutline = result

		// 保存生成阶段
		if state.GeneratedOutline != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypeOutline, constant.StageGenerate, state.GeneratedOutline); err != nil {
				fmt.Printf("  ⚠️  保存生成阶段文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 生成阶段文档已保存\n")
			}
		}

		fmt.Printf("  ✓ 大纲生成完成\n")
		state.CurrentNode = "generate_success"
		state.RetryCount = 0
		return state, nil
	}
}

// expandNode 扩写大纲节点
func expandNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [扩写] 开始扩写大纲...\n")

		// 如果已经扩写过，跳过
		if state.ExpandedOutline != "" {
			fmt.Printf("  ✓ 已有扩写的大纲，跳过\n")
			state.CurrentNode = "expand_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传，仅在有 NovelID 时）
		if state.NovelID > 0 {
			existingContent, err := utils.LoadFromFile(state.Author, constant.FileTypeOutline, constant.StageExpand)
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

		// 执行扩写
		result, err := agentExecutor.ExpandOutline(ctx, chatModel, state.GeneratedOutline, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("扩写大纲失败: %v", err)
			state.CurrentNode = "expand_failed"
			return state, nil
		}

		// 保存结果到状态（不立即写文件）
		state.ExpandedOutline = result

		// 保存扩写阶段
		if state.ExpandedOutline != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypeOutline, constant.StageExpand, state.ExpandedOutline); err != nil {
				fmt.Printf("  ⚠️  保存扩写阶段文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 扩写阶段文档已保存\n")
			}
		}

		fmt.Printf("  ✓ 大纲扩写完成\n")
		state.CurrentNode = "expand_success"
		state.RetryCount = 0
		return state, nil
	}
}

// polishNode 润色大纲节点
func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [润色] 开始润色大纲...\n")

		// 如果已经润色过，跳过
		if state.PolishedOutline != "" {
			fmt.Printf("  ✓ 已有润色的大纲，跳过\n")
			state.CurrentNode = "polish_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传，仅在有 NovelID 时）
		if state.NovelID > 0 {
			existingContent, err := utils.LoadFromFile(state.Author, constant.FileTypeOutline, constant.StagePolish)
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

		// 执行润色
		result, err := agentExecutor.PolishOutline(ctx, chatModel, state.ExpandedOutline, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("润色大纲失败: %v", err)
			state.CurrentNode = "polish_failed"
			return state, nil
		}

		// 保存结果到状态（不立即写文件）
		state.PolishedOutline = result

		// 保存润色阶段
		if state.PolishedOutline != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypeOutline, constant.StagePolish, state.PolishedOutline); err != nil {
				fmt.Printf("  ⚠️  保存润色阶段文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 润色阶段文档已保存\n")
			}
		}

		fmt.Printf("  ✓ 大纲润色完成\n")
		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

// extractNode 提取大纲信息节点
func extractNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [提取] 开始提取大纲信息...\n")

		// 如果已经提取过，跳过
		if state.ExtractedInfo != nil {
			fmt.Printf("  ✓ 已提取信息\n")
			state.CurrentNode = "extract_success"
			return state, nil
		}

		// 执行提取
		result, err := agentExecutor.ExtractOutlineInfo(ctx, chatModel, state.PolishedOutline, state.Genre, state.NovelName)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("提取大纲信息失败: %v", err)
			state.CurrentNode = "extract_failed"
			return state, nil
		}

		// 保存结果
		state.ExtractedInfo = result

		// 如果工作流输入中没有提供小说名称，使用提取的标题作为名称
		if state.NovelName == "" && result.NovelTitle != "" {
			state.NovelName = result.NovelTitle
		}

		fmt.Printf("  ✓ 信息提取完成\n")
		fmt.Printf("    - 标题: %s\n", result.NovelTitle)
		fmt.Printf("    - 简介: %s\n", result.Synopsis[:utils.Min(50, len(result.Synopsis))])
		state.CurrentNode = "extract_success"
		state.RetryCount = 0
		return state, nil
	}
}

// getUserApproval 获取用户审批决定（从终端控制台输入）
func getUserApproval(state *OutlineWorkflowState) (bool, string) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("⚠️  需要用户确认\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("操作: 保存小说到数据库\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if state.ExtractedInfo != nil {
		fmt.Printf("小说标题: %s\n", state.ExtractedInfo.NovelTitle)
		fmt.Printf("小说类型: %s\n", state.Genre)
		fmt.Printf("简介: %s\n", state.ExtractedInfo.Synopsis)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

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
func saveToDBNode(repo repository.Repository) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (*OutlineWorkflowState, error) {
		fmt.Printf("\n🤖 [保存] 准备保存到数据库...\n")

		// 如果已经保存过，跳过
		if state.SavedNovelID > 0 {
			fmt.Printf("  ✓ 已保存，小说ID: %d\n", state.SavedNovelID)
			state.CurrentNode = "save_success"
			return state, nil
		}

		// 检查是否已提取信息
		if state.ExtractedInfo == nil {
			state.ErrorMsg = "未提取大纲信息，无法保存到数据库"
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 执行保存操作
		fmt.Printf("\n🤖 [保存] 开始保存到数据库...\n")

		// 判断是新增还是更新
		if state.NovelID > 0 {
			// 更新已有小说
			fmt.Printf("  → 更新已有小说 (ID: %d)\n", state.NovelID)

			// 查询已有小说
			existingNovel, err := repo.GetByID(state.NovelID)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}

			// 更新字段
			existingNovel.Title = state.ExtractedInfo.NovelTitle
			existingNovel.Author = state.Author
			existingNovel.Genre = state.Genre
			existingNovel.Synopsis = state.ExtractedInfo.Synopsis
			existingNovel.Outline = state.ExtractedInfo.Outline
			existingNovel.Worldview = state.ExtractedInfo.Worldview
			existingNovel.CultivationSystem = state.ExtractedInfo.CultivationSystem

			// 保存更新
			if err := repo.Update(existingNovel); err != nil {
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
			isDuplicate, err := repo.CheckDuplicateByAuthorAndTitle(state.Author, state.ExtractedInfo.NovelTitle)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("检查重名失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}

			if isDuplicate {
				state.ErrorMsg = fmt.Sprintf("作者 %s 已有同名小说《%s》，请修改小说名称后重试", state.Author, state.ExtractedInfo.NovelTitle)
				state.CurrentNode = "save_failed"
				fmt.Printf("  ❌ %s\n", state.ErrorMsg)
				return state, nil
			}

			novel := &model.Novel{
				Title:             state.ExtractedInfo.NovelTitle,
				Author:            state.Author,
				Genre:             state.Genre,
				Synopsis:          state.ExtractedInfo.Synopsis,
				Outline:           state.ExtractedInfo.Outline,
				Worldview:         state.ExtractedInfo.Worldview,
				CultivationSystem: state.ExtractedInfo.CultivationSystem,
				Status:            "draft",
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

// ==================== revision节点实现 ====================

// initRevisionNode 初始化修改工作流节点（从数据库加载小说信息）
func initRevisionNode(repo repository.Repository) func(context.Context, *OutlineRevisionWorkflowInput) (*OutlineRevisionWorkflowState, error) {
	return func(ctx context.Context, input *OutlineRevisionWorkflowInput) (*OutlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [初始化] 从数据库加载小说信息...\n")

		// 查询小说信息
		novel, err := repo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		state := &OutlineRevisionWorkflowState{
			CurrentNode:     "init",
			NovelID:         novel.ID,
			NovelName:       novel.Title,
			Author:          novel.Author,
			Genre:           novel.Genre,
			OriginalOutline: novel.Outline,
			UserSuggestion:  input.UserSuggestion,
			MaxRetries:      constant.MaxRetries,
		}

		fmt.Printf("  ✓ 小说信息加载完成\n")
		fmt.Printf("    - 标题: %s\n", novel.Title)
		fmt.Printf("    - 作者: %s\n", novel.Author)
		fmt.Printf("    - 类型: %s\n", novel.Genre)
		fmt.Printf("    - 大纲长度: %d 字符\n", len(novel.Outline))

		return state, nil
	}
}

// reviseNode 修改大纲节点
func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineRevisionWorkflowState) (*OutlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *OutlineRevisionWorkflowState) (*OutlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [修改] 开始根据用户建议修改大纲...\n")

		// 执行修改
		result, err := agentExecutor.ReviseOutline(ctx, chatModel, state.OriginalOutline, state.UserSuggestion, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("修改大纲失败: %v", err)
			state.CurrentNode = "revise_failed"
			return state, nil
		}

		// 保存结果
		state.RevisedOutline = result

		// 保存到文件
		if err := utils.SaveToFile(state.Author, constant.FileTypeOutline, constant.StageRevise, result); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存文件失败: %v", err)
			state.CurrentNode = "revise_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 大纲修改完成，已保存到文件\n")
		state.CurrentNode = "revise_success"
		state.RetryCount = 0
		return state, nil
	}
}

// extractRevisionNode 提取修改后的大纲信息节点
func extractRevisionNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineRevisionWorkflowState) (*OutlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *OutlineRevisionWorkflowState) (*OutlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [提取] 开始提取大纲信息...\n")

		// 如果已经提取过，跳过
		if state.ExtractedInfo != nil {
			fmt.Printf("  ✓ 已提取信息\n")
			state.CurrentNode = "extract_success"
			return state, nil
		}

		// 执行提取
		result, err := agentExecutor.ExtractOutlineInfo(ctx, chatModel, state.RevisedOutline, state.Genre, state.NovelName)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("提取大纲信息失败: %v", err)
			state.CurrentNode = "extract_failed"
			return state, nil
		}

		// 保存结果
		state.ExtractedInfo = result

		fmt.Printf("  ✓ 信息提取完成\n")
		fmt.Printf("    - 标题: %s\n", result.NovelTitle)
		fmt.Printf("    - 简介: %s\n", result.Synopsis[:utils.Min(50, len(result.Synopsis))])
		state.CurrentNode = "extract_success"
		state.RetryCount = 0
		return state, nil
	}
}

// updateDBNode 更新数据库节点
func updateDBNode(repo repository.Repository) func(context.Context, *OutlineRevisionWorkflowState) (*OutlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *OutlineRevisionWorkflowState) (*OutlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [更新] 开始更新数据库...\n")

		// 检查是否已提取信息
		if state.ExtractedInfo == nil {
			state.ErrorMsg = "未提取大纲信息，无法更新数据库"
			state.CurrentNode = "update_failed"
			return state, nil
		}

		// 查询已有小说
		existingNovel, err := repo.GetByID(state.NovelID)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
			state.CurrentNode = "update_failed"
			return state, nil
		}

		// 更新字段
		existingNovel.Title = state.ExtractedInfo.NovelTitle
		existingNovel.Author = state.Author
		existingNovel.Genre = state.Genre
		existingNovel.Synopsis = state.ExtractedInfo.Synopsis
		existingNovel.Outline = state.ExtractedInfo.Outline
		existingNovel.Worldview = state.ExtractedInfo.Worldview
		existingNovel.CultivationSystem = state.ExtractedInfo.CultivationSystem

		// 保存更新
		if err := repo.Update(existingNovel); err != nil {
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
