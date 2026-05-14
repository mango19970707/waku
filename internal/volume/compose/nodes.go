package compose

import (
	"context"
	"fmt"
	"waku/internal/constant"
	"waku/internal/model"
	"waku/internal/utils"
	"waku/internal/volume/agents"
	"waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"
)

// initVolumePartitionNode 初始化分卷工作流节点（从数据库加载小说信息）
func initVolumePartitionNode(db *gorm.DB) func(context.Context, *VolumePartitionWorkflowInput) (*VolumePartitionWorkflowState, error) {
	return func(ctx context.Context, input *VolumePartitionWorkflowInput) (*VolumePartitionWorkflowState, error) {
		fmt.Printf("\n🤖 [初始化] 从数据库加载小说信息...\n")

		// 查询小说信息
		var novel model.Novel
		if err := db.First(&novel, input.NovelID).Error; err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		state := &VolumePartitionWorkflowState{
			CurrentNode:  "init",
			NovelID:      novel.ID,
			NovelName:    novel.Title,
			Genre:        novel.Genre,
			NovelOutline: novel.Outline,
			MaxRetries:   constant.MaxRetries,
		}

		fmt.Printf("  ✓ 小说信息加载完成\n")
		fmt.Printf("    - 标题: %s\n", novel.Title)
		fmt.Printf("    - 类型: %s\n", novel.Genre)
		fmt.Printf("    - 大纲长度: %d 字符\n", len(novel.Outline))

		return state, nil
	}
}

// partitionVolumesNode 分卷节点
func partitionVolumesNode(ctx context.Context, chatModel einoModel.BaseChatModel, db *gorm.DB) func(context.Context, *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
	return func(ctx context.Context, state *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
		fmt.Printf("\n🤖 [分卷] 开始划分卷...\n")

		// 尝试从文件加载（断点续传）
		if state.PartitionedVolumes == "" {
			// 需要获取作者信息，从数据库查询
			var novel model.Novel
			if err := db.First(&novel, state.NovelID).Error; err == nil {
				existingContent, err := utils.LoadFromFile(novel.Author, constant.FileTypeVolume, constant.StagePartition)
				if err != nil {
					state.ErrorMsg = fmt.Sprintf("加载已有内容失败: %v", err)
					state.CurrentNode = "partition_failed"
					return state, nil
				}
				if existingContent != "" {
					fmt.Printf("  ✓ 从文件加载已有内容\n")
					state.PartitionedVolumes = existingContent
					state.CurrentNode = "partition_success"
					return state, nil
				}
			}
		}

		// 执行分卷
		result, err := agentExecutor.PartitionVolumes(ctx, chatModel, state.NovelOutline, state.Genre, 0)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("分卷失败: %v", err)
			state.CurrentNode = "partition_failed"
			return state, nil
		}

		// 保存结果
		state.PartitionedVolumes = result

		// 保存到文件
		var novel model.Novel
		if err := db.First(&novel, state.NovelID).Error; err != nil {
			state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
			state.CurrentNode = "partition_failed"
			return state, nil
		}

		if err := utils.SaveToFile(novel.Author, constant.FileTypeVolume, constant.StagePartition, result); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存文件失败: %v", err)
			state.CurrentNode = "partition_failed"
			return state, nil
		}

		fmt.Printf("  ✓ 分卷完成，已保存到文件\n")
		state.CurrentNode = "partition_success"
		state.RetryCount = 0
		return state, nil
	}
}

// extractVolumesNode 提取卷信息节点
func extractVolumesNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
	return func(ctx context.Context, state *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
		fmt.Printf("\n🤖 [提取] 开始提取卷信息...\n")

		// 如果已经提取过，跳过
		if state.ExtractedVolumes != nil {
			fmt.Printf("  ✓ 已提取信息\n")
			state.CurrentNode = "extract_success"
			return state, nil
		}

		// 执行提取
		result, err := agentExecutor.ExtractVolumeInfo(ctx, chatModel, state.PartitionedVolumes, state.Genre)
		if err != nil {
			state.ErrorMsg = fmt.Sprintf("提取卷信息失败: %v", err)
			state.CurrentNode = "extract_failed"
			return state, nil
		}

		// 保存结果
		state.ExtractedVolumes = result

		fmt.Printf("  ✓ 信息提取完成\n")
		fmt.Printf("    - 卷数: %d\n", len(result.Volumes))
		state.CurrentNode = "extract_success"
		state.RetryCount = 0
		return state, nil
	}
}

// saveVolumesToDBNode 保存卷到数据库节点
func saveVolumesToDBNode(db *gorm.DB, volumeRepo repository.VolumeRepository) func(context.Context, *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
	return func(ctx context.Context, state *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
		fmt.Printf("\n🤖 [保存] 准备保存卷到数据库...\n")

		// 如果已经保存过，跳过
		if len(state.SavedVolumeIDs) > 0 {
			fmt.Printf("  ✓ 已保存，卷数: %d\n", len(state.SavedVolumeIDs))
			state.CurrentNode = "save_success"
			return state, nil
		}

		// 检查是否已提取信息
		if state.ExtractedVolumes == nil {
			state.ErrorMsg = "未提取卷信息，无法保存到数据库"
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 中断恢复逻辑：首次调用时触发中断，等待用户审批
		if !state.WaitingForApproval {
			// 首次调用：触发中断，等待用户审批
			state.WaitingForApproval = true
			state.CurrentNode = "save_waiting_approval"

			// 获取用户审批
			approved, reason := getUserVolumeApproval(state)
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
		fmt.Printf("\n🤖 [保存] 开始保存卷到数据库...\n")

		// 构建卷对象列表
		volumes := make([]*model.Volume, 0, len(state.ExtractedVolumes.Volumes))
		for _, volumeInfo := range state.ExtractedVolumes.Volumes {
			volume := &model.Volume{
				NovelID:      state.NovelID,
				VolumeNumber: volumeInfo.VolumeNumber,
				Title:        volumeInfo.Title,
				Synopsis:     volumeInfo.Synopsis,
				Theme:        volumeInfo.Theme,
				Outline:      volumeInfo.Outline,
				Status:       "draft",
			}
			volumes = append(volumes, volume)
		}

		// 批量保存到数据库
		if err := volumeRepo.CreateBatch(volumes); err != nil {
			state.ErrorMsg = fmt.Sprintf("保存卷到数据库失败: %v", err)
			state.CurrentNode = "save_failed"
			return state, nil
		}

		// 收集保存后的卷ID
		volumeIDs := make([]uint, 0, len(volumes))
		for _, volume := range volumes {
			volumeIDs = append(volumeIDs, volume.ID)
		}
		state.SavedVolumeIDs = volumeIDs

		fmt.Printf("  ✓ 保存成功，共 %d 个卷\n", len(volumeIDs))
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

// partitionChaptersNode 为每个卷划分章节节点
func partitionChaptersNode(ctx context.Context, chatModel einoModel.BaseChatModel, chapterRepo repository.ChapterRepository, db *gorm.DB) func(context.Context, *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
	return func(ctx context.Context, state *VolumePartitionWorkflowState) (*VolumePartitionWorkflowState, error) {
		fmt.Printf("\n🤖 [分章] 开始为每个卷划分章节...\n")

		// 如果已经划分过章节，跳过
		if len(state.ChapterPartitionMap) > 0 {
			fmt.Printf("  ✓ 已划分章节\n")
			state.CurrentNode = "partition_chapters_success"
			return state, nil
		}

		// 初始化章节划分映射
		state.ChapterPartitionMap = make(map[uint][]agents.ChapterInfo)

		// 获取作者信息
		var novel model.Novel
		if err := db.First(&novel, state.NovelID).Error; err != nil {
			state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
			state.CurrentNode = "partition_chapters_failed"
			return state, nil
		}

		// 遍历每个卷，为其划分章节
		for i, volumeInfo := range state.ExtractedVolumes.Volumes {
			volumeID := state.SavedVolumeIDs[i]
			volumeNumber := volumeInfo.VolumeNumber

			fmt.Printf("\n  → 为第 %d 卷划分章节...\n", volumeNumber)

			// 尝试从文件加载（断点续传）
			stageName := constant.FormatChapterStage(volumeNumber)
			existingContent, err := utils.LoadFromFile(novel.Author, constant.FileTypeVolume, stageName)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("加载第 %d 卷章节失败: %v", volumeNumber, err)
				state.CurrentNode = "partition_chapters_failed"
				return state, nil
			}

			var chapterText string
			if existingContent != "" {
				fmt.Printf("    ✓ 从文件加载已有章节划分\n")
				chapterText = existingContent
			} else {
				// 执行分章
				result, err := agentExecutor.PartitionChapters(ctx, chatModel, volumeInfo.Outline, state.Genre, volumeNumber, 0)
				if err != nil {
					state.ErrorMsg = fmt.Sprintf("第 %d 卷分章失败: %v", volumeNumber, err)
					state.CurrentNode = "partition_chapters_failed"
					return state, nil
				}

				chapterText = result

				// 保存到文件
				if err := utils.SaveToFile(novel.Author, constant.FileTypeVolume, stageName, result); err != nil {
					state.ErrorMsg = fmt.Sprintf("保存第 %d 卷章节失败: %v", volumeNumber, err)
					state.CurrentNode = "partition_chapters_failed"
					return state, nil
				}

				fmt.Printf("    ✓ 第 %d 卷章节划分完成\n", volumeNumber)
			}

			// 提取章节信息
			chapterOutput, err := agentExecutor.ExtractChapterInfo(ctx, chatModel, chapterText, state.Genre)
			if err != nil {
				state.ErrorMsg = fmt.Sprintf("提取第 %d 卷章节信息失败: %v", volumeNumber, err)
				state.CurrentNode = "partition_chapters_failed"
				return state, nil
			}

			// 保存章节到数据库
			chapters := make([]*model.Chapter, 0, len(chapterOutput.Chapters))
			for _, chapterInfo := range chapterOutput.Chapters {
				chapter := &model.Chapter{
					VolumeID:      volumeID,
					NovelID:       state.NovelID,
					ChapterNumber: chapterInfo.ChapterNumber,
					Title:         chapterInfo.Title,
					Synopsis:      chapterInfo.Synopsis,
					Outline:       chapterInfo.Outline,
					Status:        "draft",
				}
				chapters = append(chapters, chapter)
			}

			// 批量保存章节
			if err := chapterRepo.CreateBatch(chapters); err != nil {
				state.ErrorMsg = fmt.Sprintf("保存第 %d 卷章节到数据库失败: %v", volumeNumber, err)
				state.CurrentNode = "partition_chapters_failed"
				return state, nil
			}

			// 保存到状态
			state.ChapterPartitionMap[volumeID] = chapterOutput.Chapters

			fmt.Printf("    ✓ 第 %d 卷共 %d 个章节已保存到数据库\n", volumeNumber, len(chapters))
		}

		fmt.Printf("\n  ✓ 所有卷的章节划分完成\n")
		state.CurrentNode = "partition_chapters_success"
		state.RetryCount = 0
		return state, nil
	}
}

// getUserVolumeApproval 获取用户审批决定（从终端控制台输入）
func getUserVolumeApproval(state *VolumePartitionWorkflowState) (bool, string) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("⚠️  需要用户确认\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("操作: 保存卷到数据库\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if state.ExtractedVolumes != nil {
		fmt.Printf("小说标题: %s\n", state.NovelName)
		fmt.Printf("小说类型: %s\n", state.Genre)
		fmt.Printf("卷数: %d\n", len(state.ExtractedVolumes.Volumes))
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for _, volume := range state.ExtractedVolumes.Volumes {
			fmt.Printf("  第 %d 卷: %s\n", volume.VolumeNumber, volume.Title)
		}
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	fmt.Printf("📝 这将创建 %d 个卷\n", len(state.ExtractedVolumes.Volumes))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("是否继续保存? (y/n): ")

	var input string
	fmt.Scanln(&input)

	if input == "y" || input == "yes" || input == "Y" {
		fmt.Printf("✓ 用户批准，继续保存\n")
		return true, ""
	}

	fmt.Printf("✗ 用户拒绝，取消保存\n")
	return false, "用户取消操作"
}
