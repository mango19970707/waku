package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"waku/internal/character/contracts"
	"waku/internal/character/repository"
	"waku/internal/constant"
	"waku/internal/model"

	einoModel "github.com/cloudwego/eino/components/model"
)

type characterFilePayload struct {
	Content  string `json:"content"`
	Stage    string `json:"stage"`
	FileType string `json:"file_type"`
}

func initNode(novelRepo repository.Repository, characterRepo repository.CharacterRepository) func(context.Context, *CharacterWorkflowInput) (*CharacterWorkflowState, error) {
	return func(ctx context.Context, input *CharacterWorkflowInput) (*CharacterWorkflowState, error) {
		state := newWorkflowState(input)

		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("加载小说失败: %w", err)
		}

		state.NovelName = novel.Title
		state.Author = novel.Author
		state.Genre = novel.Genre
		state.Outline = extractOutlineContent(novel.Outline)

		existingCharacters, err := loadCharactersFromDB(characterRepo, novel.ID)
		if err != nil {
			return nil, err
		}
		state.ExistingCharacters = existingCharacters

		if input.CharacterID > 0 {
			character, err := characterRepo.GetByID(input.CharacterID)
			if err != nil {
				return nil, fmt.Errorf("加载目标角色失败: %w", err)
			}
			if character.NovelID != input.NovelID {
				return nil, fmt.Errorf("角色 %d 不属于小说 %d", input.CharacterID, input.NovelID)
			}
		}

		return state, nil
	}
}

func extractNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterWorkflowState) (*CharacterWorkflowState, error) {
	return func(ctx context.Context, state *CharacterWorkflowState) (*CharacterWorkflowState, error) {
		if !isEmptyPayload(state.ExtractedCharacters) {
			state.CurrentNode = "extract_success"
			return state, nil
		}

		content, err := loadStageFile(state.Author, state.NovelName, constant.StageExtract)
		if err != nil {
			state.CurrentNode = "extract_failed"
			state.ErrorMsg = fmt.Sprintf("加载提取阶段文件失败: %v", err)
			return state, nil
		}
		if content != "" {
			payload, err := parseCharacterPayloadMap(content)
			if err != nil {
				state.CurrentNode = "extract_failed"
				state.ErrorMsg = fmt.Sprintf("解析提取阶段数据失败: %v", err)
				return state, nil
			}
			state.ExtractedCharacters = payload
			state.CurrentNode = "extract_success"
			state.RetryCount = 0
			return state, nil
		}

		existingJSON, err := marshalPayload(state.ExistingCharacters)
		if err != nil {
			state.CurrentNode = "extract_failed"
			state.ErrorMsg = fmt.Sprintf("序列化已有角色失败: %v", err)
			return state, nil
		}

		extracted, err := agentExecutor.ExtractCharacters(ctx, chatModel, state.Outline, state.Genre, existingJSON)
		if err != nil {
			state.CurrentNode = "extract_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		payload, err := parseCharacterPayloadMap(extracted)
		if err != nil {
			state.CurrentNode = "extract_failed"
			state.ErrorMsg = fmt.Sprintf("解析提取结果失败: %v", err)
			return state, nil
		}

		state.ExtractedCharacters = payload
		if err := saveStageMap(state.Author, state.NovelName, constant.StageExtract, payload); err != nil {
			state.CurrentNode = "extract_failed"
			state.ErrorMsg = fmt.Sprintf("保存提取阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "extract_success"
		state.RetryCount = 0
		return state, nil
	}
}

func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterWorkflowState) (*CharacterWorkflowState, error) {
	return func(ctx context.Context, state *CharacterWorkflowState) (*CharacterWorkflowState, error) {
		if !isEmptyPayload(state.PolishedCharacters) {
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
			payload, err := parseCharacterPayloadMap(content)
			if err != nil {
				state.CurrentNode = "polish_failed"
				state.ErrorMsg = fmt.Sprintf("解析润色阶段数据失败: %v", err)
				return state, nil
			}
			state.PolishedCharacters = payload
			state.CurrentNode = "polish_success"
			state.RetryCount = 0
			return state, nil
		}

		source := state.ExtractedCharacters
		if isEmptyPayload(source) {
			source = state.ExistingCharacters
		}
		if isEmptyPayload(source) {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = "缺少待润色的角色数据"
			return state, nil
		}

		sourceJSON, err := marshalPayload(source)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("序列化润色输入失败: %v", err)
			return state, nil
		}

		polished, err := agentExecutor.PolishCharacters(ctx, chatModel, state.Outline, state.Genre, sourceJSON, state.UserSuggestion)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		payload, err := parseCharacterPayloadMap(polished)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("解析润色结果失败: %v", err)
			return state, nil
		}

		state.PolishedCharacters = payload
		if err := saveStageMap(state.Author, state.NovelName, constant.StagePolish, payload); err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("保存润色阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterWorkflowState) (*CharacterWorkflowState, error) {
	return func(ctx context.Context, state *CharacterWorkflowState) (*CharacterWorkflowState, error) {
		if !isEmptyPayload(state.ReviewedCharacters) {
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
			payload, err := parseReviewPayloadMap(content)
			if err != nil {
				state.CurrentNode = "review_failed"
				state.ErrorMsg = fmt.Sprintf("解析审核阶段数据失败: %v", err)
				return state, nil
			}
			state.ReviewedCharacters = payload
			state.CurrentNode = "review_success"
			state.RetryCount = 0
			return state, nil
		}

		if isEmptyPayload(state.PolishedCharacters) {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = "缺少待审核的角色数据"
			return state, nil
		}

		polishedJSON, err := marshalPayload(state.PolishedCharacters)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("序列化审核输入失败: %v", err)
			return state, nil
		}

		reviewed, err := agentExecutor.ReviewCharacters(ctx, chatModel, state.Outline, state.Genre, polishedJSON)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		payload, err := parseReviewPayloadMap(reviewed)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("解析审核结果失败: %v", err)
			return state, nil
		}

		state.ReviewedCharacters = payload
		if err := saveStageMap(state.Author, state.NovelName, constant.StageReview, payload); err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("保存审核阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "review_success"
		state.RetryCount = 0
		return state, nil
	}
}

func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterWorkflowState) (*CharacterWorkflowState, error) {
	return func(ctx context.Context, state *CharacterWorkflowState) (*CharacterWorkflowState, error) {
		if !isEmptyPayload(state.RevisedCharacters) {
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
			payload, err := parseCharacterPayloadMap(content)
			if err != nil {
				state.CurrentNode = "revise_failed"
				state.ErrorMsg = fmt.Sprintf("解析修改阶段数据失败: %v", err)
				return state, nil
			}
			state.RevisedCharacters = payload
			state.CurrentNode = "revise_success"
			state.RetryCount = 0
			return state, nil
		}

		if isEmptyPayload(state.PolishedCharacters) {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "缺少待修改的角色数据"
			return state, nil
		}
		if isEmptyPayload(state.ReviewedCharacters) {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "缺少审核修改意见"
			return state, nil
		}

		polishedJSON, err := marshalPayload(state.PolishedCharacters)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("序列化修改输入失败: %v", err)
			return state, nil
		}
		reviewJSON, err := marshalPayload(state.ReviewedCharacters)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("序列化审核意见失败: %v", err)
			return state, nil
		}

		revised, err := agentExecutor.ReviseCharacters(ctx, chatModel, state.Outline, state.Genre, polishedJSON, reviewJSON)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		payload, err := parseCharacterPayloadMap(revised)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("解析修改结果失败: %v", err)
			return state, nil
		}

		state.RevisedCharacters = payload
		if err := saveStageMap(state.Author, state.NovelName, constant.StageRevise, payload); err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("保存修改阶段文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "revise_success"
		state.RetryCount = 0
		return state, nil
	}
}

func saveToDBNode(characterRepo repository.CharacterRepository) func(context.Context, *CharacterWorkflowState) (*CharacterWorkflowState, error) {
	return func(ctx context.Context, state *CharacterWorkflowState) (*CharacterWorkflowState, error) {
		if len(state.SavedCharacterIDs) > 0 {
			state.CurrentNode = "save_success"
			return state, nil
		}

		revisedJSON, err := marshalPayload(state.RevisedCharacters)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("序列化最终角色失败: %v", err)
			return state, nil
		}

		normalized, payload, err := contracts.NormalizeCharacterPayload(revisedJSON)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("最终角色格式无效: %v", err)
			return state, nil
		}

		revisedMap, err := parseCharacterPayloadMap(normalized)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("解析最终角色失败: %v", err)
			return state, nil
		}
		state.RevisedCharacters = revisedMap

		existing, err := characterRepo.GetByNovelID(state.NovelID)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("加载现有角色失败: %v", err)
			return state, nil
		}

		existingByID := make(map[uint]*model.Character, len(existing))
		for _, item := range existing {
			existingByID[item.ID] = item
		}

		seen := make(map[uint]struct{}, len(payload.Characters))
		var created []*model.Character
		savedIDs := make([]uint, 0, len(payload.Characters))

		for _, item := range payload.Characters {
			characterModel, err := contracts.CharacterMapToModel(item, state.NovelID)
			if err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = err.Error()
				return state, nil
			}

			if characterModel.ID > 0 {
				existingItem, ok := existingByID[characterModel.ID]
				if !ok {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("角色不存在: %d", characterModel.ID)
					return state, nil
				}
				existingItem.NovelID = state.NovelID
				existingItem.Name = characterModel.Name
				existingItem.Role = characterModel.Role
				existingItem.Age = characterModel.Age
				existingItem.Gender = characterModel.Gender
				existingItem.Appearance = characterModel.Appearance
				existingItem.Personality = characterModel.Personality
				existingItem.Background = characterModel.Background
				existingItem.PlotlineIDs = characterModel.PlotlineIDs
				existingItem.Relationships = characterModel.Relationships
				if err := characterRepo.Update(existingItem); err != nil {
					state.CurrentNode = "save_failed"
					state.ErrorMsg = fmt.Sprintf("更新角色失败: %v", err)
					return state, nil
				}
				seen[characterModel.ID] = struct{}{}
				savedIDs = append(savedIDs, characterModel.ID)
				continue
			}

			created = append(created, characterModel)
		}

		if len(created) > 0 {
			if err := characterRepo.CreateBatch(created); err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("创建角色失败: %v", err)
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
			if err := characterRepo.Delete(item.ID); err != nil {
				state.CurrentNode = "save_failed"
				state.ErrorMsg = fmt.Sprintf("删除旧角色失败: %v", err)
				return state, nil
			}
		}

		state.SavedCharacterIDs = savedIDs
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

func loadCharactersFromDB(characterRepo repository.CharacterRepository, novelID uint) (map[string]any, error) {
	characters, err := characterRepo.GetByNovelID(novelID)
	if err != nil {
		return nil, fmt.Errorf("加载角色失败: %w", err)
	}
	if len(characters) == 0 {
		return nil, nil
	}

	items := make([]map[string]any, 0, len(characters))
	for _, character := range characters {
		items = append(items, contracts.CharacterModelToMap(character))
	}

	payload, _, err := contracts.BuildCharacterPayload(items)
	if err != nil {
		return nil, fmt.Errorf("构造角色 JSON 失败: %w", err)
	}
	return parseCharacterPayloadMap(payload)
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

	var payload characterFilePayload
	if err := json.Unmarshal(content, &payload); err == nil && payload.Content != "" {
		return payload.Content, nil
	}
	return string(content), nil
}

func saveStageMap(author, novelName, stage string, payload map[string]any) error {
	content, err := marshalPayload(payload)
	if err != nil {
		return err
	}
	return saveStageFile(author, novelName, stage, content)
}

func saveStageFile(author, novelName, stage, content string) error {
	filePath := stageFilePath(author, novelName, stage)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("创建阶段目录失败: %w", err)
	}

	payload := characterFilePayload{
		Content:  content,
		Stage:    stage,
		FileType: "character",
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
	return filepath.Join(projectRootDir(), "data", "content", author, novelName, "character", stage+".json")
}

func projectRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func isEmptyPayload(payload map[string]any) bool {
	return len(payload) == 0
}

func marshalPayload(payload map[string]any) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseCharacterPayloadMap(raw string) (map[string]any, error) {
	normalized, _, err := contracts.NormalizeCharacterPayload(raw)
	if err != nil {
		return nil, err
	}
	return unmarshalPayloadMap(normalized)
}

func parseReviewPayloadMap(raw string) (map[string]any, error) {
	normalized, _, err := contracts.NormalizeCharacterReviewPayload(raw)
	if err != nil {
		return nil, err
	}
	return unmarshalPayloadMap(normalized)
}

func unmarshalPayloadMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
