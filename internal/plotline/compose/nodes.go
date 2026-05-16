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
	"waku/internal/plotline/contracts"
	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
)

type plotlineFilePayload struct {
	Content  string `json:"content"`
	Stage    string `json:"stage"`
	FileType string `json:"file_type"`
}

func initNode(novelRepo repository.Repository, plotlineRepo repository.PlotlineRepository) func(context.Context, *PlotlineWorkflowInput) (*PlotlineWorkflowState, error) {
	return func(ctx context.Context, input *PlotlineWorkflowInput) (state *PlotlineWorkflowState, err error) {
		logInitNodeInput(ctx, input)
		defer func() {
			logNodeOutput(ctx, "init", state, err)
		}()

		state = &PlotlineWorkflowState{
			CurrentNode:    "init_success",
			NovelID:        input.NovelID,
			PlotlineID:     input.PlotlineID,
			UserSuggestion: strings.TrimSpace(input.UserSuggestion),
			MaxRetries:     constant.MaxRetries,
		}

		if input.PlotlineID > 0 && input.UserSuggestion != "" {
			plotline, err := plotlineRepo.GetByID(input.PlotlineID)
			if err != nil {
				err = fmt.Errorf("加载剧情线失败: %w", err)
				return nil, err
			}

			novel, err := novelRepo.GetByID(plotline.NovelID)
			if err != nil {
				err = fmt.Errorf("加载所属小说失败: %w", err)
				return nil, err
			}

			existingPlotlines, err := loadPlotlinesFromDB(plotlineRepo, novel.ID)
			if err != nil {
				return nil, err
			}

			state.NovelID = novel.ID
			state.PlotlineID = plotline.ID
			state.NovelName = novel.Title
			state.Author = novel.Author
			state.Genre = novel.Genre
			state.Outline = extractOutlineContent(novel.Outline)
			state.ExistingPlotlines = existingPlotlines
			return state, nil
		}

		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			err = fmt.Errorf("加载小说失败: %w", err)
			return nil, err
		}

		state.NovelID = novel.ID
		state.NovelName = novel.Title
		state.Author = novel.Author
		state.Genre = novel.Genre
		state.Outline = extractOutlineContent(novel.Outline)

		existingPlotlines, err := loadPlotlinesFromDB(plotlineRepo, novel.ID)
		if err != nil {
			return nil, err
		}
		state.ExistingPlotlines = existingPlotlines
		return state, nil
	}
}

func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineWorkflowState) (*PlotlineWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineWorkflowState) (result *PlotlineWorkflowState, err error) {
		logStateNodeInput(ctx, "generate", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "generate", result, err)
		}()

		if state.GeneratedPlotlines != "" {
			state.CurrentNode = "generate_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageGenerate)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("加载生成阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, _, err := contracts.NormalizePlotlinePayload(content)
			if err != nil {
				state.CurrentNode = "generate_failed"
				state.ErrorMsg = fmt.Sprintf("解析生成阶段数据失败: %v", err)
				return state, nil
			}
			state.GeneratedPlotlines = normalized
			state.CurrentNode = "generate_success"
			state.RetryCount = 0
			return state, nil
		}

		generated, err := agentExecutor.GeneratePlotlines(ctx, chatModel, state.Outline, state.Genre)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.GeneratedPlotlines = generated
		if err := saveStageFile(state.Author, state.NovelName, constant.StageGenerate, generated); err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("保存生成阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "generate_success"
		state.RetryCount = 0
		return state, nil
	}
}

func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineWorkflowState) (*PlotlineWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineWorkflowState) (result *PlotlineWorkflowState, err error) {
		logStateNodeInput(ctx, "polish", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "polish", result, err)
		}()

		if state.PolishedPlotlines != "" {
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
			normalized, _, err := contracts.NormalizePlotlinePayload(content)
			if err != nil {
				state.CurrentNode = "polish_failed"
				state.ErrorMsg = fmt.Sprintf("解析润色阶段数据失败: %v", err)
				return state, nil
			}
			state.PolishedPlotlines = normalized
			state.CurrentNode = "polish_success"
			state.RetryCount = 0
			return state, nil
		}

		source := state.GeneratedPlotlines
		if source == "" {
			source = state.ExistingPlotlines
		}
		if source == "" {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = "缺少待润色的剧情线数据"
			return state, nil
		}

		polished, err := agentExecutor.PolishPlotlines(ctx, chatModel, source, state.Genre)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.PolishedPlotlines = polished
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

func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineWorkflowState) (*PlotlineWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineWorkflowState) (result *PlotlineWorkflowState, err error) {
		logStateNodeInput(ctx, "review", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "review", result, err)
		}()

		if state.ReviewedPlotlines != "" {
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
			normalized, _, err := contracts.NormalizePlotlinePayload(content)
			if err != nil {
				state.CurrentNode = "review_failed"
				state.ErrorMsg = fmt.Sprintf("解析审核阶段数据失败: %v", err)
				return state, nil
			}
			state.ReviewedPlotlines = normalized
			state.CurrentNode = "review_success"
			state.RetryCount = 0
			return state, nil
		}

		reviewed, err := agentExecutor.ReviewPlotlines(ctx, chatModel, state.Outline, state.Genre, state.PolishedPlotlines, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.ReviewedPlotlines = reviewed
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

func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *PlotlineWorkflowState) (*PlotlineWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineWorkflowState) (result *PlotlineWorkflowState, err error) {
		logStateNodeInput(ctx, "revise", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "revise", result, err)
		}()

		if state.RevisedPlotlines != "" {
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
			normalized, _, err := contracts.NormalizePlotlinePayload(content)
			if err != nil {
				state.CurrentNode = "revise_failed"
				state.ErrorMsg = fmt.Sprintf("解析修改阶段数据失败: %v", err)
				return state, nil
			}
			state.RevisedPlotlines = normalized
			state.CurrentNode = "revise_success"
			state.RetryCount = 0
			return state, nil
		}

		revised, err := agentExecutor.RevisePlotlines(ctx, chatModel, state.Outline, state.Genre, state.ReviewedPlotlines, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.RevisedPlotlines = revised
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

func saveToDBNode(plotlineRepo repository.PlotlineRepository) func(context.Context, *PlotlineWorkflowState) (*PlotlineWorkflowState, error) {
	return func(ctx context.Context, state *PlotlineWorkflowState) (result *PlotlineWorkflowState, err error) {
		logStateNodeInput(ctx, "saveToDB", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "saveToDB", result, err)
		}()

		if len(state.SavedPlotlineIDs) > 0 {
			state.CurrentNode = "save_success"
			return state, nil
		}

		normalized, payload, err := contracts.NormalizePlotlinePayload(state.RevisedPlotlines)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("最终剧情线格式无效: %v", err)
			return state, nil
		}
		state.RevisedPlotlines = normalized

		existing, err := plotlineRepo.GetByNovelID(state.NovelID)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("加载现有剧情线失败: %v", err)
			return state, nil
		}

		existingByID := make(map[uint]*model.Plotline, len(existing))
		for _, item := range existing {
			existingByID[item.ID] = item
		}

		seen := make(map[uint]struct{}, len(payload.Plotlines))
		var created []*model.Plotline
		savedIDs := make([]uint, 0, len(payload.Plotlines))

		for _, item := range payload.Plotlines {
			plotlineModel, err := plotlineMapToModel(item, state.NovelID)
			if err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = err.Error()
				return state, nil
			}

			if plotlineModel.ID > 0 {
				existingItem, ok := existingByID[plotlineModel.ID]
				if !ok {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("剧情线不存在: %d", plotlineModel.ID)
					return state, nil
				}

				existingItem.Title = plotlineModel.Title
				existingItem.Type = plotlineModel.Type
				existingItem.Description = plotlineModel.Description
				existingItem.Status = plotlineModel.Status
				if err := plotlineRepo.Update(existingItem); err != nil {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("更新剧情线失败: %v", err)
					return state, nil
				}
				seen[plotlineModel.ID] = struct{}{}
				savedIDs = append(savedIDs, plotlineModel.ID)
				continue
			}

			created = append(created, plotlineModel)
		}

		if len(created) > 0 {
			if err := plotlineRepo.CreateBatch(created); err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("创建剧情线失败: %v", err)
				return state, nil
			}
			for _, item := range created {
				seen[item.ID] = struct{}{}
				savedIDs = append(savedIDs, item.ID)
			}
		}

		for _, item := range existing {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			if err := plotlineRepo.Delete(item.ID); err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("删除旧剧情线失败: %v", err)
				return state, nil
			}
		}

		state.SavedPlotlineIDs = savedIDs
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

func loadPlotlinesFromDB(plotlineRepo repository.PlotlineRepository, novelID uint) (string, error) {
	plotlines, err := plotlineRepo.GetByNovelID(novelID)
	if err != nil {
		return "", fmt.Errorf("加载剧情线失败: %w", err)
	}
	if len(plotlines) == 0 {
		return "", nil
	}

	items := make([]map[string]any, 0, len(plotlines))
	for _, plotline := range plotlines {
		items = append(items, modelToPlotlineMap(plotline))
	}

	payload, _, err := contracts.BuildPlotlinePayload(items)
	if err != nil {
		return "", fmt.Errorf("构造剧情线 JSON 失败: %w", err)
	}
	return payload, nil
}

func plotlineMapToModel(item map[string]any, novelID uint) (*model.Plotline, error) {
	title, _ := item["title"].(string)
	plotlineType, _ := item["type"].(string)
	description, _ := item["description"].(string)
	status, _ := item["status"].(string)

	plotline := &model.Plotline{
		NovelID:     novelID,
		Title:       title,
		Type:        plotlineType,
		Description: description,
		Status:      status,
	}

	if rawID, ok := item["id"]; ok {
		switch v := rawID.(type) {
		case float64:
			plotline.ID = uint(v)
		case uint:
			plotline.ID = v
		case int:
			plotline.ID = uint(v)
		}
	}

	return plotline, nil
}

func modelToPlotlineMap(plotline *model.Plotline) map[string]any {
	return map[string]any{
		"id":          plotline.ID,
		"title":       plotline.Title,
		"type":        plotline.Type,
		"description": plotline.Description,
		"status":      plotline.Status,
	}
}

func extractOutlineContent(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	var outlineData map[string]any
	if err := json.Unmarshal([]byte(raw), &outlineData); err == nil {
		if outline, ok := outlineData["outline"].(string); ok && strings.TrimSpace(outline) != "" {
			return outline
		}
	}
	return raw
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

	var payload plotlineFilePayload
	if err := json.Unmarshal(content, &payload); err == nil && payload.Content != "" {
		return payload.Content, nil
	}
	return string(content), nil
}

func saveStageFile(author, novelName, stage, content string) error {
	filePath := stageFilePath(author, novelName, stage)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("创建阶段目录失败: %w", err)
	}

	payload := plotlineFilePayload{
		Content:  content,
		Stage:    stage,
		FileType: constant.FileTypePlotline,
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
	return filepath.Join(projectRootDir(), "data", "content", author, novelName, "plotline", stage+".json")
}

func projectRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
