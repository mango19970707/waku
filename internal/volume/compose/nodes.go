package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"waku/internal/constant"
	"waku/internal/model"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	"waku/internal/volume/contracts"
	volumerepository "waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
)

type volumeFilePayload struct {
	Content  string `json:"content"`
	Stage    string `json:"stage"`
	FileType string `json:"file_type"`
}

func initNode(novelRepo novelrepository.Repository, plotlineRepo plotlinerepository.PlotlineRepository, volumeRepo volumerepository.VolumeRepository) func(context.Context, *VolumeWorkflowInput) (*VolumeWorkflowState, error) {
	return func(ctx context.Context, input *VolumeWorkflowInput) (state *VolumeWorkflowState, err error) {
		state = newWorkflowState(input)

		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("加载小说失败: %w", err)
		}

		state.NovelName = novel.Title
		state.Author = novel.Author
		state.Genre = novel.Genre
		state.Outline = strings.TrimSpace(novel.Outline)

		plotlines, err := loadPlotlinesFromDB(plotlineRepo, novel.ID)
		if err != nil {
			return nil, err
		}
		state.ExistingPlotlines = plotlines

		volumes, err := loadVolumesFromDB(volumeRepo, novel.ID)
		if err != nil {
			return nil, err
		}
		state.ExistingVolumes = volumes

		if input.VolumeID > 0 {
			targetVolume, err := volumeRepo.GetByID(input.VolumeID)
			if err != nil {
				return nil, fmt.Errorf("加载目标分卷失败: %w", err)
			}
			if targetVolume.NovelID != input.NovelID {
				return nil, fmt.Errorf("分卷 %d 不属于小说 %d", input.VolumeID, input.NovelID)
			}

			targetJSON, err := json.Marshal(contracts.VolumeModelToItem(targetVolume))
			if err != nil {
				return nil, fmt.Errorf("序列化目标分卷失败: %w", err)
			}
			state.TargetVolumeJSON = string(targetJSON)
		}

		return state, nil
	}
}

func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *VolumeWorkflowState) (*VolumeWorkflowState, error) {
	return func(ctx context.Context, state *VolumeWorkflowState) (*VolumeWorkflowState, error) {
		if state.GeneratedVolumes != "" {
			state.CurrentNode = "generate_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StagePartition)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("加载分卷阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, _, err := contracts.NormalizeVolumePayload(content)
			if err != nil {
				state.CurrentNode = "generate_failed"
				state.ErrorMsg = fmt.Sprintf("解析分卷阶段数据失败: %v", err)
				return state, nil
			}
			state.GeneratedVolumes = normalized
			state.CurrentNode = "generate_success"
			state.RetryCount = 0
			return state, nil
		}

		generated, err := agentExecutor.GenerateVolumes(ctx, chatModel, state.Outline, state.Genre, state.ExistingPlotlines, state.ExistingVolumes)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.GeneratedVolumes = generated
		if err := saveStageFile(state.Author, state.NovelName, constant.StagePartition, generated); err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("保存分卷阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "generate_success"
		state.RetryCount = 0
		return state, nil
	}
}

func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *VolumeWorkflowState) (*VolumeWorkflowState, error) {
	return func(ctx context.Context, state *VolumeWorkflowState) (*VolumeWorkflowState, error) {
		if state.PolishedVolumes != "" {
			state.CurrentNode = "polish_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StagePolish)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("加载润色阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, _, err := contracts.NormalizeVolumePayload(content)
			if err != nil {
				state.CurrentNode = "polish_failed"
				state.ErrorMsg = fmt.Sprintf("解析润色阶段数据失败: %v", err)
				return state, nil
			}
			state.PolishedVolumes = normalized
			state.CurrentNode = "polish_success"
			state.RetryCount = 0
			return state, nil
		}

		source := state.GeneratedVolumes
		if source == "" {
			source = state.ExistingVolumes
		}
		if source == "" {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = "缺少待润色的分卷数据"
			return state, nil
		}

		polished, err := agentExecutor.PolishVolumes(ctx, chatModel, state.Outline, state.Genre, state.ExistingPlotlines, source, state.TargetVolumeJSON, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.PolishedVolumes = polished
		if err := saveStageFile(state.Author, state.NovelName, constant.StagePolish, polished); err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("保存润色阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *VolumeWorkflowState) (*VolumeWorkflowState, error) {
	return func(ctx context.Context, state *VolumeWorkflowState) (*VolumeWorkflowState, error) {
		if state.ReviewedVolumes != "" {
			state.CurrentNode = "review_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageReview)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("加载审核阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, _, err := contracts.NormalizeVolumePayload(content)
			if err != nil {
				state.CurrentNode = "review_failed"
				state.ErrorMsg = fmt.Sprintf("解析审核阶段数据失败: %v", err)
				return state, nil
			}
			state.ReviewedVolumes = normalized
			state.CurrentNode = "review_success"
			state.RetryCount = 0
			return state, nil
		}

		source := state.PolishedVolumes
		if source == "" {
			source = state.GeneratedVolumes
		}
		if source == "" {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = "缺少待审核的分卷数据"
			return state, nil
		}

		reviewed, err := agentExecutor.ReviewVolumes(ctx, chatModel, state.Outline, state.Genre, state.ExistingPlotlines, source, state.TargetVolumeJSON, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.ReviewedVolumes = reviewed
		if err := saveStageFile(state.Author, state.NovelName, constant.StageReview, reviewed); err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("保存审核阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "review_success"
		state.RetryCount = 0
		return state, nil
	}
}

func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *VolumeWorkflowState) (*VolumeWorkflowState, error) {
	return func(ctx context.Context, state *VolumeWorkflowState) (*VolumeWorkflowState, error) {
		if state.RevisedVolumes != "" {
			state.CurrentNode = "revise_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageRevise)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("加载修改阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, _, err := contracts.NormalizeVolumePayload(content)
			if err != nil {
				state.CurrentNode = "revise_failed"
				state.ErrorMsg = fmt.Sprintf("解析修改阶段数据失败: %v", err)
				return state, nil
			}
			state.RevisedVolumes = normalized
			state.CurrentNode = "revise_success"
			state.RetryCount = 0
			return state, nil
		}

		source := state.ReviewedVolumes
		if source == "" {
			source = state.PolishedVolumes
		}
		if source == "" {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "缺少待修改的分卷数据"
			return state, nil
		}

		revised, err := agentExecutor.ReviseVolumes(ctx, chatModel, state.Outline, state.Genre, state.ExistingPlotlines, source, state.TargetVolumeJSON, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.RevisedVolumes = revised
		if err := saveStageFile(state.Author, state.NovelName, constant.StageRevise, revised); err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("保存修改阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "revise_success"
		state.RetryCount = 0
		return state, nil
	}
}

func saveToDBNode(volumeRepo volumerepository.VolumeRepository) func(context.Context, *VolumeWorkflowState) (*VolumeWorkflowState, error) {
	return func(ctx context.Context, state *VolumeWorkflowState) (*VolumeWorkflowState, error) {
		if len(state.SavedVolumeIDs) > 0 {
			state.CurrentNode = "save_success"
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeVolumePayload(state.RevisedVolumes)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("最终分卷格式无效: %v", err)
			return state, nil
		}
		state.RevisedVolumes = normalized

		existing, err := volumeRepo.GetByNovelID(state.NovelID)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("加载现有分卷失败: %v", err)
			return state, nil
		}

		existingByID := make(map[uint]*model.Volume, len(existing))
		existingByNumber := make(map[int]*model.Volume, len(existing))
		for _, item := range existing {
			existingByID[item.ID] = item
			if _, ok := existingByNumber[item.VolumeNumber]; !ok {
				existingByNumber[item.VolumeNumber] = item
			}
		}

		seen := make(map[uint]struct{}, len(payload.Volumes))
		savedIDs := make([]uint, 0, len(payload.Volumes))
		created := make([]*model.Volume, 0)

		for _, item := range payload.Volumes {
			volumeModel, err := contracts.VolumeItemToModel(item, state.NovelID)
			if err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = err.Error()
				return state, nil
			}

			if volumeModel.ID > 0 {
				existingItem, ok := existingByID[volumeModel.ID]
				if !ok {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("分卷不存在: %d", volumeModel.ID)
					return state, nil
				}
				existingItem.NovelID = state.NovelID
				existingItem.VolumeNumber = volumeModel.VolumeNumber
				existingItem.Title = volumeModel.Title
				existingItem.Synopsis = volumeModel.Synopsis
				existingItem.Theme = volumeModel.Theme
				existingItem.Outline = volumeModel.Outline
				existingItem.Status = volumeModel.Status
				if err := volumeRepo.Update(existingItem); err != nil {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("更新分卷失败: %v", err)
					return state, nil
				}
				seen[existingItem.ID] = struct{}{}
				savedIDs = append(savedIDs, existingItem.ID)
				continue
			}

			if existingItem, ok := existingByNumber[volumeModel.VolumeNumber]; ok {
				existingItem.NovelID = state.NovelID
				existingItem.VolumeNumber = volumeModel.VolumeNumber
				existingItem.Title = volumeModel.Title
				existingItem.Synopsis = volumeModel.Synopsis
				existingItem.Theme = volumeModel.Theme
				existingItem.Outline = volumeModel.Outline
				existingItem.Status = volumeModel.Status
				if err := volumeRepo.Update(existingItem); err != nil {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("更新分卷失败: %v", err)
					return state, nil
				}
				seen[existingItem.ID] = struct{}{}
				savedIDs = append(savedIDs, existingItem.ID)
				continue
			}

			created = append(created, volumeModel)
		}

		if len(created) > 0 {
			if err := volumeRepo.CreateBatch(created); err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("创建分卷失败: %v", err)
				return state, nil
			}
			for _, item := range created {
				seen[item.ID] = struct{}{}
				savedIDs = append(savedIDs, item.ID)
			}
		}

		if !state.RevisionMode {
			for _, item := range existing {
				if _, ok := seen[item.ID]; ok {
					continue
				}
				if err := volumeRepo.Delete(item.ID); err != nil {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("删除旧分卷失败: %v", err)
					return state, nil
				}
			}
		}

		state.SavedVolumeIDs = savedIDs
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

func loadPlotlinesFromDB(plotlineRepo plotlinerepository.PlotlineRepository, novelID uint) (string, error) {
	plotlines, err := plotlineRepo.GetByNovelID(novelID)
	if err != nil {
		return "", fmt.Errorf("加载剧情线失败: %w", err)
	}
	if len(plotlines) == 0 {
		return "", nil
	}

	items := make([]map[string]any, 0, len(plotlines))
	for _, item := range plotlines {
		items = append(items, map[string]any{
			"id":          item.ID,
			"novel_id":    item.NovelID,
			"title":       item.Title,
			"type":        item.Type,
			"description": item.Description,
			"status":      item.Status,
		})
	}

	data := map[string]any{"plotlines": items}
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("构造剧情线上下文失败: %w", err)
	}
	return string(encoded), nil
}

func loadVolumesFromDB(volumeRepo volumerepository.VolumeRepository, novelID uint) (string, error) {
	volumes, err := volumeRepo.GetByNovelID(novelID)
	if err != nil {
		return "", fmt.Errorf("加载现有分卷失败: %w", err)
	}
	if len(volumes) == 0 {
		return "", nil
	}

	items := make([]contracts.VolumeItem, 0, len(volumes))
	for _, item := range volumes {
		items = append(items, contracts.VolumeModelToItem(item))
	}

	payload, _, err := contracts.BuildVolumePayload(items)
	if err != nil {
		return "", fmt.Errorf("构造分卷上下文失败: %w", err)
	}
	return payload, nil
}

func loadStageFile(author, novelName, stage string) (string, error) {
	filePath := stageFilePath(author, novelName, stage)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("读取阶段文件失败: %w", err)
	}

	var payload volumeFilePayload
	if err := json.Unmarshal(content, &payload); err == nil && strings.TrimSpace(payload.Content) != "" {
		return payload.Content, nil
	}
	return string(content), nil
}

func saveStageFile(author, novelName, stage, content string) error {
	filePath := stageFilePath(author, novelName, stage)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("创建阶段目录失败: %w", err)
	}

	payload := volumeFilePayload{
		Content:  content,
		Stage:    stage,
		FileType: constant.FileTypeVolume,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化阶段文件失败: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("写入阶段文件失败: %w", err)
	}
	return nil
}

func stageFilePath(author, novelName, stage string) string {
	return filepath.Join(projectRootDir(), "data", "content", author, novelName, "volume", stage+".json")
}

func projectRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
