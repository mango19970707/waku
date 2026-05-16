package compose

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"waku/internal/constant"
	"waku/internal/model"
	"waku/internal/plotline/repository"
	"waku/internal/utils"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/google/uuid"
)

// ==================== 提取工作流节点 ====================

// initExtractNode 初始化提取工作流节点（从数据库加载小说信息）
func initExtractNode(novelRepo repository.Repository) func(context.Context, *PlotlineExtractWorkflowInput) (*PlotlineExtractWorkflowState, error) {
	return func(ctx context.Context, input *PlotlineExtractWorkflowInput) (*PlotlineExtractWorkflowState, error) {
		fmt.Printf("\n🤖 [初始化] 从数据库加载小说信息...\n")

		// 查询小说信息
		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		// 检查大纲是否存在
		if novel.Outline == "" {
			return nil, fmt.Errorf("小说大纲为空，请先生成大纲")
		}

		state := &PlotlineExtractWorkflowState{
			CurrentNode: "init",
			NovelID:     novel.ID,
			NovelName:   novel.Title,
			Author:      novel.Author,
			Genre:       novel.Genre,
			Outline:     novel.Outline,
			MaxRetries:  constant.MaxRetries,
		}

		fmt.Printf("  ✓ 小说信息加载完成\n")
		fmt.Printf("    - 标题: %s\n", novel.Title)
		fmt.Printf("    - 作者: %s\n", novel.Author)
		fmt.Printf("    - 类型: %s\n", novel.Genre)
		fmt.Printf("    - 大纲长度: %d 字符\n", len(novel.Outline))

		return state, nil
	}
}

// extractNode 提取剧情线节点（使用 JSON parser 严格校验）
func extractNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
		fmt.Printf("\n🤖 [提取] 开始从大纲提取剧情线（JSON 格式）...\n")

		// 如果已经提取过，跳过
		if state.ExtractedPlotlines != "" {
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
				state.ExtractedPlotlines = existingContent
				state.CurrentNode = "extract_success"
				return state, nil
			}
		}

		// 执行提取（使用新的 JSON parser 方法）
		output, err := agentExecutor.ExtractPlotlinesJSON(ctx, chatModel, state.Outline, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("提取剧情线失败: %v", err)
			state.CurrentNode = "extract_failed"
			return state, nil
		}

		// 将结果转换为 JSON 字符串
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("序列化 JSON 失败: %v", err)
			state.CurrentNode = "extract_failed"
			return state, nil
		}

		// 保存结果到状态
		state.ExtractedPlotlines = string(jsonBytes)

		// 保存到文件
		if state.NovelName != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypePlotline, constant.StageExtract, state.ExtractedPlotlines); err != nil {
				fmt.Printf("  ⚠️  保存文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 已保存到文件\n")
			}
		}

		fmt.Printf("  ✓ 剧情线提取完成，共提取 %d 条剧情线\n", len(output.Plotlines))
		state.CurrentNode = "extract_success"
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

	// 解析剧情线数量
	var output struct {
		Plotlines []interface{} `json:"plotlines"`
	}
	if err := json.Unmarshal([]byte(state.ExtractedPlotlines), &output); err == nil {
		fmt.Printf("提取的剧情线数量: %d\n", len(output.Plotlines))
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

// saveToDBNode 保存到数据库节点（批量插入多条剧情线）
func saveToDBNode(plotlineRepo repository.PlotlineRepository) func(context.Context, *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineExtractWorkflowState) (*PlotlineExtractWorkflowState, error) {
		fmt.Printf("\n🤖 [保存] 准备保存到数据库...\n")

		// 如果已经保存过，跳过
		if len(state.SavedPlotlineIDs) > 0 {
			fmt.Printf("  ✓ 已保存，剧情线数量: %d\n", len(state.SavedPlotlineIDs))
			state.CurrentNode = "save_success"
			return state, nil
		}

		// 解析 JSON 数据
		var output struct {
			Plotlines []struct {
				Title       string `json:"title"`
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"plotlines"`
		}

		if err := json.Unmarshal([]byte(state.ExtractedPlotlines), &output); err != nil {
			state.ErrorMsg = fmt.Sprintf("解析 JSON 失败: %v", err)
			state.CurrentNode = "save_failed"
			return state, nil
		}

		if len(output.Plotlines) == 0 {
			state.ErrorMsg = "未提取到任何剧情线"
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 转换为 Plotline 模型
		plotlines := make([]*model.Plotline, 0, len(output.Plotlines))
		for _, item := range output.Plotlines {
			plotline := &model.Plotline{
				ID:          uuid.New().String(),
				NovelID:     fmt.Sprintf("%d", state.NovelID),
				Title:       item.Title,
				Type:        item.Type,
				Description: item.Description,
				Status:      "active",
			}
			plotlines = append(plotlines, plotline)
		}

		fmt.Printf("  → 准备保存 %d 条剧情线\n", len(plotlines))

		// 批量插入
		if err := plotlineRepo.CreateBatch(plotlines); err != nil {
			state.ErrorMsg = fmt.Sprintf("批量插入剧情线失败: %v", err)
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 收集 ID 列表
		state.SavedPlotlineIDs = make([]string, 0, len(plotlines))
		for _, plotline := range plotlines {
			state.SavedPlotlineIDs = append(state.SavedPlotlineIDs, plotline.ID)
		}

		fmt.Printf("  ✓ 保存成功，剧情线数量: %d\n", len(state.SavedPlotlineIDs))
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

// ==================== 修改工作流节点 ====================

// initRevisionNode 初始化修改工作流节点（从数据库加载小说信息和剧情线）
func initRevisionNode(novelRepo repository.Repository, plotlineRepo repository.PlotlineRepository) func(context.Context, *PlotlineRevisionWorkflowInput) (*PlotlineRevisionWorkflowState, error) {
	return func(ctx context.Context, input *PlotlineRevisionWorkflowInput) (*PlotlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [初始化] 从数据库加载小说信息和剧情线...\n")

		// 查询小说信息
		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		// 检查大纲是否存在
		if novel.Outline == "" {
			return nil, fmt.Errorf("小说大纲为空，请先生成大纲")
		}

		// 查询该小说的所有剧情线
		plotlines, err := plotlineRepo.GetByNovelID(fmt.Sprintf("%d", novel.ID))
		if err != nil {
			return nil, fmt.Errorf("查询剧情线失败: %w", err)
		}

		if len(plotlines) == 0 {
			return nil, fmt.Errorf("未找到任何剧情线，请先提取剧情线")
		}

		// 将剧情线转换为 JSON 格式
		type PlotlineJSON struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Type        string `json:"type"`
			Description string `json:"description"`
		}

		plotlinesJSON := make([]PlotlineJSON, 0, len(plotlines))
		for _, p := range plotlines {
			plotlinesJSON = append(plotlinesJSON, PlotlineJSON{
				ID:          p.ID,
				Title:       p.Title,
				Type:        p.Type,
				Description: p.Description,
			})
		}

		jsonBytes, err := json.MarshalIndent(map[string]interface{}{
			"plotlines": plotlinesJSON,
		}, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("序列化剧情线失败: %w", err)
		}

		state := &PlotlineRevisionWorkflowState{
			CurrentNode:       "init",
			NovelID:           novel.ID,
			NovelName:         novel.Title,
			Author:            novel.Author,
			Genre:             novel.Genre,
			Outline:           novel.Outline,
			ExistingPlotlines: string(jsonBytes),
			UserSuggestion:    input.UserSuggestion,
			MaxRetries:        constant.MaxRetries,
		}

		fmt.Printf("  ✓ 小说信息加载完成\n")
		fmt.Printf("    - 标题: %s\n", novel.Title)
		fmt.Printf("    - 作者: %s\n", novel.Author)
		fmt.Printf("    - 类型: %s\n", novel.Genre)
		fmt.Printf("    - 大纲长度: %d 字符\n", len(novel.Outline))
		fmt.Printf("    - 现有剧情线数量: %d\n", len(plotlines))

		return state, nil
	}
}

// reviseNode 优化剧情线节点（使用 JSON parser 严格校验）
func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [优化] 开始优化剧情线（根据用户建议）...\n")

		// 如果已经优化过，跳过
		if state.RevisedPlotlines != "" {
			fmt.Printf("  ✓ 已有优化的剧情线，跳过\n")
			state.CurrentNode = "revise_success"
			return state, nil
		}

		// 执行优化（使用新的 JSON parser 方法）
		output, err := agentExecutor.RevisePlotlinesJSON(ctx, chatModel, state.Outline, state.Genre, state.ExistingPlotlines, state.UserSuggestion)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("优化剧情线失败: %v", err)
			state.CurrentNode = "revise_failed"
			return state, nil
		}

		// 将结果转换为 JSON 字符串
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("序列化 JSON 失败: %v", err)
			state.CurrentNode = "revise_failed"
			return state, nil
		}

		// 保存结果到状态
		state.RevisedPlotlines = string(jsonBytes)

		// 保存到文件
		if state.NovelName != "" {
			if err := utils.SaveToFile(state.Author, constant.FileTypePlotline, constant.StageRevise, state.RevisedPlotlines); err != nil {
				fmt.Printf("  ⚠️  保存文件失败: %v\n", err)
			} else {
				fmt.Printf("  ✓ 已保存到文件\n")
			}
		}

		fmt.Printf("  ✓ 剧情线优化完成，共输出 %d 条剧情线\n", len(output.Plotlines))
		state.CurrentNode = "revise_success"
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

// saveOrUpdateDBNode 保存或更新数据库节点（修改工作流）
func saveOrUpdateDBNode(plotlineRepo repository.PlotlineRepository) func(context.Context, *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineRevisionWorkflowState) (*PlotlineRevisionWorkflowState, error) {
		fmt.Printf("\n🤖 [保存] 准备保存或更新到数据库...\n")

		// 如果已经保存过，跳过
		if len(state.SavedPlotlineIDs) > 0 {
			fmt.Printf("  ✓ 已保存，剧情线数量: %d\n", len(state.SavedPlotlineIDs))
			state.CurrentNode = "save_success"
			return state, nil
		}

		// 解析 JSON 数据
		var output struct {
			Plotlines []struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"plotlines"`
		}

		if err := json.Unmarshal([]byte(state.RevisedPlotlines), &output); err != nil {
			state.ErrorMsg = fmt.Sprintf("解析 JSON 失败: %v", err)
			state.CurrentNode = "save_failed"
			return state, nil
		}

		if len(output.Plotlines) == 0 {
			state.ErrorMsg = "未输出任何剧情线"
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 分类处理：更新现有剧情线 vs 插入新剧情线
		var toUpdate []*model.Plotline
		var toInsert []*model.Plotline
		savedIDs := make([]string, 0, len(output.Plotlines))

		for _, item := range output.Plotlines {
			plotline := &model.Plotline{
				ID:          item.ID,
				NovelID:     fmt.Sprintf("%d", state.NovelID),
				Title:       item.Title,
				Type:        item.Type,
				Description: item.Description,
				Status:      "active",
			}

			if item.ID != "" {
				// 有 ID，尝试更新
				toUpdate = append(toUpdate, plotline)
				savedIDs = append(savedIDs, item.ID)
			} else {
				// 没有 ID，插入新记录
				plotline.ID = uuid.New().String()
				toInsert = append(toInsert, plotline)
				savedIDs = append(savedIDs, plotline.ID)
			}
		}

		fmt.Printf("  → 准备更新 %d 条剧情线\n", len(toUpdate))
		fmt.Printf("  → 准备插入 %d 条剧情线\n", len(toInsert))

		// 更新现有剧情线
		for _, plotline := range toUpdate {
			if err := plotlineRepo.Update(plotline); err != nil {
				state.ErrorMsg = fmt.Sprintf("更新剧情线失败 (ID=%s): %v", plotline.ID, err)
				state.CurrentNode = "save_failed"
				return state, nil
			}
		}

		// 批量插入新剧情线
		if len(toInsert) > 0 {
			if err := plotlineRepo.CreateBatch(toInsert); err != nil {
				state.ErrorMsg = fmt.Sprintf("批量插入剧情线失败: %v", err)
				state.CurrentNode = "save_failed"
				return state, nil
			}
		}

		// 保存 ID 列表
		state.SavedPlotlineIDs = savedIDs

		fmt.Printf("  ✓ 保存成功，剧情线数量: %d\n", len(state.SavedPlotlineIDs))
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}
