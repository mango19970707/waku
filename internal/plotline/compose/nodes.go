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

		if input.PlotlineID > 0 && input.NovelID == 0 {
			plotline, err := plotlineRepo.GetByID(input.PlotlineID)
			if err != nil {
				return nil, fmt.Errorf("load plotline failed: %w", err)
			}
			state.NovelID = plotline.NovelID
			state.PlotlineID = plotline.ID
		}

		novel, err := novelRepo.GetByID(state.NovelID)
		if err != nil {
			return nil, fmt.Errorf("load novel failed: %w", err)
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

		if !isEmptyStateData(state.GeneratedPlotlines) {
			state.CurrentNode = "generate_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageGenerate)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("load generate stage file failed: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, err := plotlineStateFromRaw(content)
			if err != nil {
				state.CurrentNode = "generate_failed"
				state.ErrorMsg = fmt.Sprintf("parse generate stage data failed: %v", err)
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

		generatedState, err := plotlineStateFromRaw(generated)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("parse generate stage data failed: %v", err)
			return state, nil
		}

		state.GeneratedPlotlines = generatedState
		if err := saveStageFile(state.Author, state.NovelName, constant.StageGenerate, generated); err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("save generate stage file failed: %v", err)
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

		if !isEmptyStateData(state.PolishedPlotlines) {
			state.CurrentNode = "polish_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StagePolish)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("load polish stage file failed: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, err := plotlineStateFromRaw(content)
			if err != nil {
				state.CurrentNode = "polish_failed"
				state.ErrorMsg = fmt.Sprintf("parse polish stage data failed: %v", err)
				return state, nil
			}
			state.PolishedPlotlines = normalized
			state.CurrentNode = "polish_success"
			state.RetryCount = 0
			return state, nil
		}

		source := state.GeneratedPlotlines
		if isEmptyStateData(source) {
			source = state.ExistingPlotlines
		}
		if isEmptyStateData(source) {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = "missing plotlines to polish"
			return state, nil
		}

		sourceRaw, err := plotlineStateToRaw(source)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("marshal plotlines to polish failed: %v", err)
			return state, nil
		}

		polished, err := agentExecutor.PolishPlotlines(ctx, chatModel, sourceRaw, state.Genre, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		polishedState, err := plotlineStateFromRaw(polished)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("parse polish stage data failed: %v", err)
			return state, nil
		}

		state.PolishedPlotlines = polishedState
		if err := saveStageFile(state.Author, state.NovelName, constant.StagePolish, polished); err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("save polish stage file failed: %v", err)
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

		if !isEmptyStateData(state.ReviewedPlotlines) {
			state.CurrentNode = "review_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageReview)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("load review stage file failed: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, err := plotlineReviewStateFromRaw(content)
			if err != nil {
				state.CurrentNode = "review_failed"
				state.ErrorMsg = fmt.Sprintf("parse review stage data failed: %v", err)
				return state, nil
			}
			state.ReviewedPlotlines = normalized
			state.CurrentNode = "review_success"
			state.RetryCount = 0
			return state, nil
		}

		polishedRaw, err := plotlineStateToRaw(state.PolishedPlotlines)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("marshal plotlines to review failed: %v", err)
			return state, nil
		}

		reviewed, err := agentExecutor.ReviewPlotlines(ctx, chatModel, state.Outline, state.Genre, polishedRaw, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		reviewedState, err := plotlineReviewStateFromRaw(reviewed)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("parse review stage data failed: %v", err)
			return state, nil
		}

		state.ReviewedPlotlines = reviewedState
		if err := saveStageFile(state.Author, state.NovelName, constant.StageReview, reviewed); err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("save review stage file failed: %v", err)
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

		if !isEmptyStateData(state.RevisedPlotlines) {
			state.CurrentNode = "revise_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageRevise)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("load revise stage file failed: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, err := plotlineStateFromRaw(content)
			if err != nil {
				state.CurrentNode = "revise_failed"
				state.ErrorMsg = fmt.Sprintf("parse revise stage data failed: %v", err)
				return state, nil
			}
			state.RevisedPlotlines = normalized
			state.CurrentNode = "revise_success"
			state.RetryCount = 0
			return state, nil
		}

		polishedRaw, err := plotlineStateToRaw(state.PolishedPlotlines)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("marshal polished plotlines failed: %v", err)
			return state, nil
		}
		reviewedRaw, err := plotlineReviewStateToRaw(state.ReviewedPlotlines)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("marshal review suggestions failed: %v", err)
			return state, nil
		}

		revised, err := agentExecutor.RevisePlotlines(ctx, chatModel, state.Outline, state.Genre, polishedRaw, reviewedRaw)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		revisedState, err := plotlineStateFromRaw(revised)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("parse revise stage data failed: %v", err)
			return state, nil
		}

		state.RevisedPlotlines = revisedState
		if err := saveStageFile(state.Author, state.NovelName, constant.StageRevise, revised); err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("save revise stage file failed: %v", err)
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

		normalizedState, payload, err := plotlineStateToPayload(state.RevisedPlotlines)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("final plotlines format is invalid: %v", err)
			return state, nil
		}
		state.RevisedPlotlines = normalizedState

		existing, err := plotlineRepo.GetByNovelID(state.NovelID)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("load existing plotlines failed: %v", err)
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
					state.ErrorMsg = fmt.Sprintf("plotline not found: %d", plotlineModel.ID)
					return state, nil
				}

				existingItem.Title = plotlineModel.Title
				existingItem.Type = plotlineModel.Type
				existingItem.Description = plotlineModel.Description
				existingItem.Status = plotlineModel.Status
				if err := plotlineRepo.Update(existingItem); err != nil {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("update plotline failed: %v", err)
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
				state.ErrorMsg = fmt.Sprintf("create plotlines failed: %v", err)
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
				state.ErrorMsg = fmt.Sprintf("delete stale plotline failed: %v", err)
				return state, nil
			}
		}

		state.SavedPlotlineIDs = savedIDs
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

func loadPlotlinesFromDB(plotlineRepo repository.PlotlineRepository, novelID uint) (map[string]any, error) {
	plotlines, err := plotlineRepo.GetByNovelID(novelID)
	if err != nil {
		return nil, fmt.Errorf("load plotlines failed: %w", err)
	}
	if len(plotlines) == 0 {
		return nil, nil
	}

	items := make([]map[string]any, 0, len(plotlines))
	for _, plotline := range plotlines {
		items = append(items, modelToPlotlineMap(plotline))
	}

	payload, _, err := contracts.BuildPlotlinePayload(items)
	if err != nil {
		return nil, fmt.Errorf("build plotlines JSON failed: %w", err)
	}
	return plotlineStateFromRaw(payload)
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
		return "", fmt.Errorf("read stage file failed: %w", err)
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
		return fmt.Errorf("create stage directory failed: %w", err)
	}

	payload := plotlineFilePayload{
		Content:  content,
		Stage:    stage,
		FileType: constant.FileTypePlotline,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal stage file failed: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("write stage file failed: %w", err)
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

func isEmptyStateData(data map[string]any) bool {
	return len(data) == 0
}

func plotlineStateFromRaw(raw string) (map[string]any, error) {
	normalized, _, err := contracts.NormalizePlotlinePayload(raw)
	if err != nil {
		return nil, err
	}

	var state map[string]any
	if err := json.Unmarshal([]byte(normalized), &state); err != nil {
		return nil, fmt.Errorf("unmarshal plotline state failed: %w", err)
	}
	return state, nil
}

func plotlineStateToRaw(data map[string]any) (string, error) {
	if isEmptyStateData(data) {
		return "", nil
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal plotline state failed: %w", err)
	}

	normalized, _, err := contracts.NormalizePlotlinePayload(string(raw))
	if err != nil {
		return "", err
	}
	return normalized, nil
}

func plotlineStateToPayload(data map[string]any) (map[string]any, *contracts.PlotlinePayload, error) {
	raw, err := plotlineStateToRaw(data)
	if err != nil {
		return nil, nil, err
	}

	var normalized map[string]any
	if err := json.Unmarshal([]byte(raw), &normalized); err != nil {
		return nil, nil, fmt.Errorf("unmarshal plotline state failed: %w", err)
	}

	_, payload, err := contracts.NormalizePlotlinePayload(raw)
	if err != nil {
		return nil, nil, err
	}
	return normalized, payload, nil
}

func plotlineReviewStateFromRaw(raw string) (map[string]any, error) {
	normalized, _, err := contracts.NormalizePlotlineReviewPayload(raw)
	if err != nil {
		return nil, err
	}

	var state map[string]any
	if err := json.Unmarshal([]byte(normalized), &state); err != nil {
		return nil, fmt.Errorf("unmarshal plotline review state failed: %w", err)
	}
	return state, nil
}

func plotlineReviewStateToRaw(data map[string]any) (string, error) {
	if isEmptyStateData(data) {
		return "", nil
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal plotline review state failed: %w", err)
	}

	normalized, _, err := contracts.NormalizePlotlineReviewPayload(string(raw))
	if err != nil {
		return "", err
	}
	return normalized, nil
}
