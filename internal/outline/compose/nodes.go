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
	"waku/internal/outline/contracts"
	"waku/internal/outline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
)

type outlineFilePayload struct {
	Content  string `json:"content"`
	Stage    string `json:"stage"`
	FileType string `json:"file_type"`
}

func initNode(repo repository.Repository) func(context.Context, *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, input *OutlineWorkflowInput) (state *OutlineWorkflowState, err error) {
		logInitNodeInput(ctx, input)
		defer func() {
			logNodeOutput(ctx, "init", state, err)
		}()

		state = &OutlineWorkflowState{
			CurrentNode:     "init_success",
			NovelID:         input.NovelID,
			NovelName:       strings.TrimSpace(input.NovelName),
			Author:          strings.TrimSpace(input.Author),
			Genre:           strings.TrimSpace(input.Genre),
			OriginalOutline: strings.TrimSpace(input.OriginalOutline),
			UserSuggestion:  strings.TrimSpace(input.UserSuggestion),
			MaxRetries:      constant.MaxRetries,
		}

		if input.NovelID > 0 && input.UserSuggestion != "" {
			novel, err := repo.GetByID(input.NovelID)
			if err != nil {
				return nil, fmt.Errorf("加载小说失败: %w", err)
			}

			state.NovelID = novel.ID
			state.NovelName = firstNonEmpty(state.NovelName, novel.Title)
			state.Author = firstNonEmpty(state.Author, novel.Author)
			state.Genre = firstNonEmpty(state.Genre, novel.Genre)

			novelJSON, payload, err := buildNovelJSONFromModel(novel)
			if err != nil {
				return nil, fmt.Errorf("构造小说 JSON 失败: %w", err)
			}

			state.OriginalOutline = novelJSON
			state.NovelTitle = payload.Title
			state.Synopsis = payload.Synopsis
			state.Outline = payload.Outline
			return state, nil
		}

		state.NovelName = firstNonEmpty(state.NovelName, "未命名小说")
		return state, nil
	}
}

func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (result *OutlineWorkflowState, err error) {
		logStateNodeInput(ctx, "generate", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "generate", result, err)
		}()

		if state.GeneratedOutline != "" {
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
			normalized, payload, err := normalizeStageOutput(content, state.Genre, state.NovelName, state.Synopsis)
			if err != nil {
				state.CurrentNode = "generate_failed"
				state.ErrorMsg = fmt.Sprintf("解析生成阶段数据失败: %v", err)
				return state, nil
			}
			state.GeneratedOutline = normalized
			applyPayloadToState(state, payload)
			state.CurrentNode = "generate_success"
			state.RetryCount = 0
			return state, nil
		}
		generated, err := agentExecutor.GenerateOutline(ctx, chatModel, state.OriginalOutline, state.Genre)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("生成大纲失败: %v", err)
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeNovelPayload(generated, state.Genre)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("生成阶段输出格式无效: %v", err)
			return state, nil
		}

		state.GeneratedOutline = normalized
		applyPayloadToState(state, payload)
		if err := saveStageFile(state.Author, state.NovelName, constant.StageGenerate, normalized); err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("保存生成阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "generate_success"
		state.RetryCount = 0
		return state, nil
	}
}

func expandNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (result *OutlineWorkflowState, err error) {
		logStateNodeInput(ctx, "expand", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "expand", result, err)
		}()

		if state.ExpandedOutline != "" {
			state.CurrentNode = "expand_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageExpand)
		if err != nil {
			state.CurrentNode = "expand_failed"
			state.ErrorMsg = fmt.Sprintf("加载扩写阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, payload, err := normalizeStageOutput(content, state.Genre, state.NovelName, state.Synopsis)
			if err != nil {
				state.CurrentNode = "expand_failed"
				state.ErrorMsg = fmt.Sprintf("解析扩写阶段数据失败: %v", err)
				return state, nil
			}
			state.ExpandedOutline = normalized
			applyPayloadToState(state, payload)
			state.CurrentNode = "expand_success"
			state.RetryCount = 0
			return state, nil
		}
		expanded, err := agentExecutor.ExpandOutline(ctx, chatModel, state.GeneratedOutline, state.Genre)
		if err != nil {
			state.CurrentNode = "expand_failed"
			state.ErrorMsg = fmt.Sprintf("扩写大纲失败: %v", err)
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeNovelPayload(expanded, state.Genre)
		if err != nil {
			state.CurrentNode = "expand_failed"
			state.ErrorMsg = fmt.Sprintf("扩写阶段输出格式无效: %v", err)
			return state, nil
		}

		state.ExpandedOutline = normalized
		applyPayloadToState(state, payload)
		if err := saveStageFile(state.Author, state.NovelName, constant.StageExpand, normalized); err != nil {
			state.CurrentNode = "expand_failed"
			state.ErrorMsg = fmt.Sprintf("保存扩写阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "expand_success"
		state.RetryCount = 0
		return state, nil
	}
}

func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (result *OutlineWorkflowState, err error) {
		logStateNodeInput(ctx, "polish", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "polish", result, err)
		}()

		if state.PolishedOutline != "" {
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
			normalized, payload, err := normalizeStageOutput(content, state.Genre, state.NovelName, state.Synopsis)
			if err != nil {
				state.CurrentNode = "polish_failed"
				state.ErrorMsg = fmt.Sprintf("解析润色阶段数据失败: %v", err)
				return state, nil
			}
			state.PolishedOutline = normalized
			applyPayloadToState(state, payload)
			state.CurrentNode = "polish_success"
			state.RetryCount = 0
			return state, nil
		}
		// 在 quick revision 链路中，polish 节点承担“按用户建议预修改 + 润色”的职责。
		source := state.ExpandedOutline
		if source == "" {
			source = state.OriginalOutline
		}

		polished, err := agentExecutor.PolishOutline(ctx, chatModel, source, state.UserSuggestion, state.Genre)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("润色大纲失败: %v", err)
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeNovelPayload(polished, state.Genre)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("润色阶段输出格式无效: %v", err)
			return state, nil
		}

		state.PolishedOutline = normalized
		applyPayloadToState(state, payload)
		if err := saveStageFile(state.Author, state.NovelName, constant.StagePolish, normalized); err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("保存润色阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (result *OutlineWorkflowState, err error) {
		logStateNodeInput(ctx, "review", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "review", result, err)
		}()

		if state.ReviewSuggestions != "" {
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
			normalized, err := normalizeReviewSuggestions(content)
			if err != nil {
				state.CurrentNode = "review_failed"
				state.ErrorMsg = fmt.Sprintf("解析审核阶段数据失败: %v", err)
				return state, nil
			}
			state.ReviewSuggestions = normalized
			state.CurrentNode = "review_success"
			state.RetryCount = 0
			return state, nil
		}

		reviewed, err := agentExecutor.ReviewOutline(ctx, chatModel, latestOutlineForRevision(state), state.UserSuggestion, state.Genre)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("审核大纲失败: %v", err)
			return state, nil
		}

		normalized, err := normalizeReviewSuggestions(reviewed)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("审核阶段输出格式无效: %v", err)
			return state, nil
		}

		state.ReviewSuggestions = normalized
		if err := saveStageFile(state.Author, state.NovelName, constant.StageReview, normalized); err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("保存审核阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "review_success"
		state.RetryCount = 0
		return state, nil
	}
}

func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (result *OutlineWorkflowState, err error) {
		logStateNodeInput(ctx, "revise", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "revise", result, err)
		}()

		if state.RevisedOutline != "" {
			state.CurrentNode = "revise_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageRevise)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("???????????????: %v", err)
			return state, nil
		}
		if content != "" {
			normalized, payload, err := normalizeStageOutput(content, state.Genre, state.NovelName, state.Synopsis)
			if err != nil {
				state.CurrentNode = "revise_failed"
				state.ErrorMsg = fmt.Sprintf("???????????????: %v", err)
				return state, nil
			}
			state.RevisedOutline = normalized
			applyPayloadToState(state, payload)
			state.CurrentNode = "revise_success"
			state.RetryCount = 0
			return state, nil
		}

		source := latestOutlineForRevision(state)
		if source == "" {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "latest outline is empty"
			return state, nil
		}

		suggestions := strings.TrimSpace(state.ReviewSuggestions)
		if suggestions == "" {
			suggestions = strings.TrimSpace(state.UserSuggestion)
		}
		if suggestions == "" {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "revision suggestions are empty"
			return state, nil
		}

		revised, err := agentExecutor.ReviseOutline(ctx, chatModel, source, suggestions, state.Genre)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("?????????: %v", err)
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeNovelPayload(revised, state.Genre)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("???????????????: %v", err)
			return state, nil
		}

		state.RevisedOutline = normalized
		applyPayloadToState(state, payload)
		if err := saveStageFile(state.Author, state.NovelName, constant.StageRevise, normalized); err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("???????????????: %v", err)
			return state, nil
		}

		state.CurrentNode = "revise_success"
		state.RetryCount = 0
		return state, nil
	}
}

func saveToDBNode(repo repository.Repository) func(context.Context, *OutlineWorkflowState) (*OutlineWorkflowState, error) {
	return func(ctx context.Context, state *OutlineWorkflowState) (result *OutlineWorkflowState, err error) {
		logStateNodeInput(ctx, "saveToDB", state)
		result = state
		defer func() {
			logNodeOutput(ctx, "saveToDB", result, err)
		}()

		if state.SavedNovelID > 0 {
			state.CurrentNode = "save_success"
			return state, nil
		}

		if state.RevisedOutline == "" {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = "修改后的大纲为空，无法保存"
			return state, nil
		}
		if state.Author == "" {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = "作者名称为空，无法保存"
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeNovelPayload(state.RevisedOutline, state.Genre)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("最终大纲格式无效: %v", err)
			return state, nil
		}
		state.RevisedOutline = normalized
		applyPayloadToState(state, payload)

		if state.NovelID > 0 {
			novel, err := repo.GetByID(state.NovelID)
			if err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("查询小说失败: %v", err)
				return state, nil
			}

			novel.Title = payload.Title
			novel.Author = state.Author
			novel.Genre = state.Genre
			novel.Synopsis = payload.Synopsis
			novel.Outline = payload.Outline

			if err := repo.Update(novel); err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("更新小说失败: %v", err)
				return state, nil
			}

			state.SavedNovelID = novel.ID
			state.CurrentNode = "save_success"
			state.RetryCount = 0
			return state, nil
		}

		duplicated, err := repo.CheckDuplicateByAuthorAndTitle(state.Author, payload.Title)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("检查重名失败: %v", err)
			return state, nil
		}
		if duplicated {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("作者 %s 已存在同名小说《%s》", state.Author, payload.Title)
			return state, nil
		}

		novel := &model.Novel{
			Title:    payload.Title,
			Author:   state.Author,
			Genre:    state.Genre,
			Synopsis: payload.Synopsis,
			Outline:  payload.Outline,
			Status:   constant.StatusDraft,
		}
		if err := repo.Create(novel); err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("创建小说失败: %v", err)
			return state, nil
		}

		state.NovelID = novel.ID
		state.SavedNovelID = novel.ID
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

func applyPayloadToState(state *OutlineWorkflowState, payload *contracts.NovelPayload) {
	if payload == nil {
		return
	}
	state.NovelTitle = payload.Title
	state.Synopsis = payload.Synopsis
	state.Outline = payload.Outline
	if state.NovelName == "" {
		state.NovelName = payload.Title
	}
}

func normalizeStageOutput(raw, genre, fallbackTitle, fallbackSynopsis string) (string, *contracts.NovelPayload, error) {
	normalized, payload, err := contracts.NormalizeNovelPayload(raw, genre)
	if err == nil {
		return normalized, payload, nil
	}

	legacyOutline := extractLegacyOutlineContent(raw)
	outlineMap := map[string]any{
		"outline": legacyOutline,
	}
	if requiresCultivationFields(genre) {
		outlineMap["worldview"] = "待补充"
		outlineMap["cultivation_system"] = "待补充"
	}

	return contracts.BuildNovelPayload(
		firstNonEmpty(fallbackTitle, extractLegacyTitle(legacyOutline), "未命名小说"),
		firstNonEmpty(fallbackSynopsis, summarizeLegacyOutline(legacyOutline)),
		outlineMap,
		genre,
	)
}

func buildNovelJSONFromModel(novel *model.Novel) (string, *contracts.NovelPayload, error) {
	outlineJSON, _, err := contracts.NormalizeOutlineJSON(novel.Outline, novel.Genre)
	if err == nil {
		return contracts.BuildNovelPayload(novel.Title, novel.Synopsis, mustParseOutlineMap(outlineJSON), novel.Genre)
	}

	outlineMap := map[string]any{
		"outline": novel.Outline,
	}
	if requiresCultivationFields(novel.Genre) {
		outlineMap["worldview"] = "待补充"
		outlineMap["cultivation_system"] = "待补充"
	}
	return contracts.BuildNovelPayload(novel.Title, novel.Synopsis, outlineMap, novel.Genre)
}

func mustParseOutlineMap(raw string) map[string]any {
	var outlineMap map[string]any
	_ = json.Unmarshal([]byte(raw), &outlineMap)
	return outlineMap
}

func extractLegacyOutlineContent(raw string) string {
	if raw == "" {
		return ""
	}

	var payload outlineFilePayload
	if err := json.Unmarshal([]byte(raw), &payload); err == nil && payload.Content != "" {
		return payload.Content
	}

	return raw
}

func extractLegacyTitle(outline string) string {
	lines := strings.Split(outline, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimLeft(line, "#*- "))
		if line != "" {
			return line
		}
	}
	return ""
}

func summarizeLegacyOutline(outline string) string {
	plain := strings.TrimSpace(strings.ReplaceAll(outline, "\n", " "))
	runes := []rune(plain)
	if len(runes) <= 120 {
		return plain
	}
	return string(runes[:120]) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func requiresCultivationFields(genre string) bool {
	return strings.Contains(genre, "玄幻") || strings.Contains(genre, "修真")
}

func normalizeReviewSuggestions(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("review suggestions cannot be empty")
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		switch value := decoded.(type) {
		case []any:
			return marshalNormalizedJSON(value)
		case map[string]any:
			if suggestions, ok := value["review_suggestions"]; ok {
				return marshalNormalizedJSON(suggestions)
			}
		}
	}

	_, payload, err := contracts.NormalizeNovelPayload(trimmed, "")
	if err == nil {
		return extractReviewSuggestionsFromNovelPayload(payload)
	}

	return "", fmt.Errorf("review output must be valid JSON suggestions")
}

func extractReviewSuggestionsFromNovelPayload(payload *contracts.NovelPayload) (string, error) {
	if payload == nil {
		return "", fmt.Errorf("review result is empty")
	}

	var outlineMap map[string]any
	if err := json.Unmarshal([]byte(payload.Outline), &outlineMap); err != nil {
		return "", fmt.Errorf("parse outline from review result: %w", err)
	}

	suggestions, ok := outlineMap["review_suggestions"]
	if !ok {
		return "", fmt.Errorf("review result missing review_suggestions")
	}

	return marshalNormalizedJSON(suggestions)
}

func marshalNormalizedJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(data), nil
}

func latestOutlineForRevision(state *OutlineWorkflowState) string {
	switch {
	case strings.TrimSpace(state.PolishedOutline) != "":
		return state.PolishedOutline
	case strings.TrimSpace(state.ExpandedOutline) != "":
		return state.ExpandedOutline
	case strings.TrimSpace(state.GeneratedOutline) != "":
		return state.GeneratedOutline
	default:
		return strings.TrimSpace(state.OriginalOutline)
	}
}

func combineRevisionSuggestions(reviewSuggestions, userSuggestion string) string {
	reviewSuggestions = strings.TrimSpace(reviewSuggestions)
	userSuggestion = strings.TrimSpace(userSuggestion)

	switch {
	case reviewSuggestions != "" && userSuggestion != "":
		combined := map[string]any{
			"review_suggestions": json.RawMessage(reviewSuggestions),
			"user_suggestion":    userSuggestion,
		}
		data, err := json.Marshal(combined)
		if err != nil {
			return reviewSuggestions + "\n\nUser suggestion:\n" + userSuggestion
		}
		return string(data)
	case reviewSuggestions != "":
		return reviewSuggestions
	default:
		return userSuggestion
	}
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

	var payload outlineFilePayload
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

	payload := outlineFilePayload{
		Content:  content,
		Stage:    stage,
		FileType: constant.FileTypeOutline,
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
	return filepath.Join(projectRootDir(), "data", "content", author, novelName, "abstract_bak", stage+".json")
}

func projectRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
