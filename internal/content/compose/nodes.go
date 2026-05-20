package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	characterrepository "waku/internal/character/repository"
	"waku/internal/constant"
	"waku/internal/content/agents"
	"waku/internal/content/contracts"
	contentrepository "waku/internal/content/repository"
	"waku/internal/model"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	"waku/internal/utils"
	volumerepository "waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
)

// initNode 加载小说、分卷、章节和压缩上下文，并确定当前要处理的章节。
func initNode(
	novelRepo novelrepository.Repository,
	volumeRepo volumerepository.VolumeRepository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	characterRepo characterrepository.CharacterRepository,
	contentRepo contentrepository.ContentRepository,
) func(context.Context, *ContentWorkflowInput) (*ContentWorkflowState, error) {
	return func(ctx context.Context, input *ContentWorkflowInput) (state *ContentWorkflowState, err error) {
		state = newWorkflowState(input)
		logNodeInput("init", state)
		defer func() { logNodeOutput("init", state, err) }()

		// 加载小说和分卷基础信息，后续所有章节共用这部分高层上下文。
		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("加载小说失败: %w", err)
		}
		volume, err := volumeRepo.GetByID(input.VolumeID)
		if err != nil {
			return nil, fmt.Errorf("加载分卷失败: %w", err)
		}
		if volume.NovelID != input.NovelID {
			return nil, fmt.Errorf("分卷 %d 不属于小说 %d", input.VolumeID, input.NovelID)
		}

		state.NovelName = novel.Title
		state.Author = novel.Author
		state.Genre = novel.Genre
		state.NovelContext = buildNovelContext(novel)
		state.VolumeContext = buildVolumeContext(volume)
		state.Plotlines = loadPlotlinesContext(plotlineRepo, input.NovelID, nil)
		state.Characters = loadCharactersContext(characterRepo, input.NovelID, nil)

		// 未指定章节时，默认收集当前卷正文为空的章节，按章节号串行补齐。
		if state.ChapterID == 0 {
			if len(state.PendingChapterIDs) == 0 {
				chapters, err := contentRepo.GetEmptyContentChapters(input.VolumeID)
				if err != nil {
					return nil, err
				}
				for _, chapter := range chapters {
					state.PendingChapterIDs = append(state.PendingChapterIDs, chapter.ID)
				}
			}
			if len(state.PendingChapterIDs) == 0 {
				return nil, fmt.Errorf("当前分卷没有需要生成正文的章节")
			}
			state.ChapterID = state.PendingChapterIDs[0]
			state.PendingChapterIDs = state.PendingChapterIDs[1:]
		}

		if err := loadTargetChapterContext(contentRepo, state); err != nil {
			return nil, err
		}
		state.FastPolishMode = strings.TrimSpace(state.UserSuggestion) != "" && strings.TrimSpace(state.TargetContent) != ""
		logWorkflowInput(state)
		return state, nil
	}
}

// prepareNextNode 在批量模式下切换到下一章，并清理上一章阶段产物。
func prepareNextNode(contentRepo contentrepository.ContentRepository) func(context.Context, *ContentWorkflowState) (*ContentWorkflowState, error) {
	return func(ctx context.Context, state *ContentWorkflowState) (result *ContentWorkflowState, err error) {
		logNodeInput("prepareNext", state)
		result = state
		defer func() { logNodeOutput("prepareNext", result, err) }()
		if len(state.PendingChapterIDs) == 0 {
			state.CurrentNode = "prepare_next_failed"
			state.ErrorMsg = "没有待处理章节"
			return state, nil
		}

		// 批量串行时重置阶段字段，确保下一章不会复用上一章正文。
		state.ChapterID = state.PendingChapterIDs[0]
		state.PendingChapterIDs = state.PendingChapterIDs[1:]
		state.TargetContent = ""
		state.GeneratedContent = ""
		state.PolishedContent = ""
		state.ReviewedContent = ""
		state.RevisedContent = ""
		state.NeedRevise = false
		state.FastPolishMode = false
		if err := loadTargetChapterContext(contentRepo, state); err != nil {
			state.CurrentNode = "prepare_next_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}
		state.CurrentNode = "prepare_next_success"
		return state, nil
	}
}

// generateNode 生成单章初稿，并在节点内保存阶段文件。
func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ContentWorkflowState) (*ContentWorkflowState, error) {
	return func(ctx context.Context, state *ContentWorkflowState) (result *ContentWorkflowState, err error) {
		logNodeInput("generate", state)
		result = state
		defer func() { logNodeOutput("generate", result, err) }()
		if state.GeneratedContent != "" {
			state.CurrentNode = "generate_success"
			return state, nil
		}

		if content, err := loadContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StageGenerate); err == nil && content != "" {
			if normalized, _, err := contracts.NormalizeContentPayload(content); err == nil {
				state.GeneratedContent = normalized
				state.CurrentNode = "generate_success"
				return state, nil
			}
		}

		generated, err := executeGenerateAgent(ctx, chatModel, &agents.ContentGeneratorInput{NovelContext: state.NovelContext, VolumeContext: state.VolumeContext, ChapterContext: state.ChapterContext, PreviousContext: state.PreviousContext, NextContext: state.NextContext, Characters: state.Characters, Plotlines: state.Plotlines, UserSuggestion: state.UserSuggestion})
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}
		state.GeneratedContent = generated
		if err := saveContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StageGenerate, generated); err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("保存生成文件失败: %v", err)
			return state, nil
		}
		state.CurrentNode = "generate_success"
		return state, nil
	}
}

// polishNode 润色正文；快速修改模式下，用户建议也在本节点优先处理。
func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ContentWorkflowState) (*ContentWorkflowState, error) {
	return func(ctx context.Context, state *ContentWorkflowState) (result *ContentWorkflowState, err error) {
		logNodeInput("polish", state)
		result = state
		defer func() { logNodeOutput("polish", result, err) }()
		if state.PolishedContent != "" {
			state.CurrentNode = "polish_success"
			return state, nil
		}

		if content, err := loadContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StagePolish); err == nil && content != "" {
			if normalized, _, err := contracts.NormalizeContentPayload(content); err == nil {
				state.PolishedContent = normalized
				state.CurrentNode = "polish_success"
				return state, nil
			}
		}

		source := state.GeneratedContent
		if source == "" {
			source = state.TargetContent
		}
		if source == "" {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = "缺少待润色的正文数据"
			return state, nil
		}

		polished, err := executePolishAgent(ctx, chatModel, &agents.ContentPolisherInput{Chapter: source, UserSuggestion: state.UserSuggestion, NovelContext: state.NovelContext, VolumeContext: state.VolumeContext, ChapterContext: state.ChapterContext, PreviousContext: state.PreviousContext, NextContext: state.NextContext, Characters: state.Characters, Plotlines: state.Plotlines})
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}
		state.PolishedContent = polished
		if err := saveContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StagePolish, polished); err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("保存润色文件失败: %v", err)
			return state, nil
		}
		state.CurrentNode = "polish_success"
		return state, nil
	}
}

// reviewNode 审核润色后的正文，空建议会在路由阶段直接进入保存。
func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ContentWorkflowState) (*ContentWorkflowState, error) {
	return func(ctx context.Context, state *ContentWorkflowState) (result *ContentWorkflowState, err error) {
		logNodeInput("review", state)
		result = state
		defer func() { logNodeOutput("review", result, err) }()
		if state.ReviewedContent != "" {
			state.CurrentNode = "review_success"
			return state, nil
		}

		if content, err := loadContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StageReview); err == nil && content != "" {
			if normalized, payload, err := contracts.NormalizeContentReviewPayload(content); err == nil {
				state.ReviewedContent = normalized
				state.NeedRevise = len(payload.Suggestions) > 0
				state.CurrentNode = "review_success"
				return state, nil
			}
		}
		if state.PolishedContent == "" {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = "缺少待审核的正文数据"
			return state, nil
		}

		reviewed, err := executeReviewAgent(ctx, chatModel, &agents.ContentReviewerInput{Chapter: state.PolishedContent, NovelContext: state.NovelContext, VolumeContext: state.VolumeContext, ChapterContext: state.ChapterContext, PreviousContext: state.PreviousContext, NextContext: state.NextContext, Characters: state.Characters, Plotlines: state.Plotlines})
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}
		_, payload, err := contracts.NormalizeContentReviewPayload(reviewed)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}
		state.ReviewedContent = reviewed
		state.NeedRevise = len(payload.Suggestions) > 0
		if err := saveContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StageReview, reviewed); err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("保存审核文件失败: %v", err)
			return state, nil
		}
		state.CurrentNode = "review_success"
		return state, nil
	}
}

// reviseNode 根据审核建议修改正文；审核无建议时不会进入本节点。
func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ContentWorkflowState) (*ContentWorkflowState, error) {
	return func(ctx context.Context, state *ContentWorkflowState) (result *ContentWorkflowState, err error) {
		logNodeInput("revise", state)
		result = state
		defer func() { logNodeOutput("revise", result, err) }()
		if state.RevisedContent != "" {
			state.CurrentNode = "revise_success"
			return state, nil
		}

		if content, err := loadContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StageRevise); err == nil && content != "" {
			if normalized, _, err := contracts.NormalizeContentPayload(content); err == nil {
				state.RevisedContent = normalized
				state.CurrentNode = "revise_success"
				return state, nil
			}
		}
		if state.PolishedContent == "" || state.ReviewedContent == "" {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "缺少正文或审核建议"
			return state, nil
		}

		revised, err := executeReviseAgent(ctx, chatModel, &agents.ContentReviserInput{Chapter: state.PolishedContent, ReviewSuggestions: state.ReviewedContent, NovelContext: state.NovelContext, VolumeContext: state.VolumeContext, ChapterContext: state.ChapterContext, PreviousContext: state.PreviousContext, NextContext: state.NextContext, Characters: state.Characters, Plotlines: state.Plotlines})
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}
		state.RevisedContent = revised
		if err := saveContentStageFile(state.Author, state.NovelName, state.VolumeID, state.ChapterID, constant.StageRevise, revised); err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("保存修改文件失败: %v", err)
			return state, nil
		}
		state.CurrentNode = "revise_success"
		return state, nil
	}
}

// saveToDBNode 将最终正文保存到数据库，且只更新正文相关字段。
func saveToDBNode(contentRepo contentrepository.ContentRepository) func(context.Context, *ContentWorkflowState) (*ContentWorkflowState, error) {
	return func(ctx context.Context, state *ContentWorkflowState) (result *ContentWorkflowState, err error) {
		logNodeInput("saveToDB", state)
		result = state
		defer func() {
			logNodeOutput("saveToDB", result, err)
			if err == nil {
				logWorkflowOutput(state)
			}
		}()

		finalData := state.RevisedContent
		if finalData == "" {
			finalData = state.PolishedContent
		}
		if finalData == "" {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = "缺少待保存的正文数据"
			return state, nil
		}
		_, payload, err := contracts.NormalizeContentPayload(finalData)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("解析正文数据失败: %v", err)
			return state, nil
		}
		if payload.Chapter.ID != state.ChapterID || payload.Chapter.NovelID != state.NovelID || payload.Chapter.VolumeID != state.VolumeID {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = "正文归属与当前工作流不一致"
			return state, nil
		}
		if err := contentRepo.UpdateContent(payload.Chapter.ID, payload.Chapter.Content, payload.Chapter.WordCount, payload.Chapter.Status); err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("保存正文到数据库失败: %v", err)
			return state, nil
		}
		state.SavedChapterIDs = append(state.SavedChapterIDs, payload.Chapter.ID)
		state.CurrentNode = "save_success"
		return state, nil
	}
}

func loadTargetChapterContext(contentRepo contentrepository.ContentRepository, state *ContentWorkflowState) error {
	chapter, err := contentRepo.GetChapterByID(state.ChapterID)
	if err != nil {
		return fmt.Errorf("加载目标章节失败: %w", err)
	}
	if chapter.NovelID != state.NovelID || chapter.VolumeID != state.VolumeID {
		return fmt.Errorf("章节 %d 不属于当前小说或分卷", state.ChapterID)
	}
	payload, err := contracts.BuildContentPayload(chapter)
	if err != nil {
		return err
	}
	state.TargetContent = payload
	state.ChapterContext = buildChapterContext(chapter)

	previous, err := contentRepo.GetPreviousChapter(chapter.ID)
	if err != nil {
		return err
	}
	state.PreviousContext = buildAdjacentContext("上一章", previous, true)
	next, err := contentRepo.GetNextChapter(chapter.ID)
	if err != nil {
		return err
	}
	state.NextContext = buildAdjacentContext("下一章", next, false)
	return nil
}

func buildNovelContext(novel *model.Novel) string {
	data := map[string]any{"id": novel.ID, "title": novel.Title, "author": novel.Author, "genre": novel.Genre, "synopsis": novel.Synopsis, "outline": novel.Outline}
	return marshalContext(data)
}

func buildVolumeContext(volume *model.Volume) string {
	data := map[string]any{"id": volume.ID, "novel_id": volume.NovelID, "volume_number": volume.VolumeNumber, "title": volume.Title, "synopsis": volume.Synopsis, "theme": volume.Theme, "outline": volume.Outline}
	return marshalContext(data)
}

func buildChapterContext(chapter *model.Chapter) string {
	data := map[string]any{"id": chapter.ID, "novel_id": chapter.NovelID, "volume_id": chapter.VolumeID, "chapter_number": chapter.ChapterNumber, "title": chapter.Title, "synopsis": chapter.Synopsis, "outline": chapter.Outline, "word_count_estimate": chapter.WordCountEstimate, "key_events": []string(chapter.KeyEvents), "key_characters": []string(chapter.KeyCharacters), "plotline_progress": map[string]string(chapter.PlotlineProgress)}
	return marshalContext(data)
}

func buildAdjacentContext(label string, chapter *model.Chapter, includeTail bool) string {
	if chapter == nil {
		return label + "：无"
	}
	data := map[string]any{"id": chapter.ID, "chapter_number": chapter.ChapterNumber, "title": chapter.Title, "synopsis": chapter.Synopsis, "outline": chapter.Outline}
	if includeTail {
		data["content_tail"] = tailText(chapter.Content, 600)
	}
	return marshalContext(data)
}

func loadPlotlinesContext(repo plotlinerepository.PlotlineRepository, novelID uint, allowed map[string]struct{}) string {
	plotlines, err := repo.GetByNovelID(novelID)
	if err != nil || len(plotlines) == 0 {
		return "暂无剧情线"
	}
	items := make([]map[string]any, 0, len(plotlines))
	for _, item := range plotlines {
		key := fmt.Sprintf("%d", item.ID)
		if len(allowed) > 0 {
			if _, ok := allowed[key]; !ok {
				continue
			}
		}
		items = append(items, map[string]any{"id": item.ID, "title": item.Title, "type": item.Type, "description": item.Description, "status": item.Status})
	}
	return marshalContext(map[string]any{"plotlines": items})
}

func loadCharactersContext(repo characterrepository.CharacterRepository, novelID uint, allowed map[string]struct{}) string {
	characters, err := repo.GetByNovelID(novelID)
	if err != nil || len(characters) == 0 {
		return "暂无角色"
	}
	items := make([]map[string]any, 0, len(characters))
	for _, item := range characters {
		if len(allowed) > 0 {
			if _, ok := allowed[item.Name]; !ok {
				continue
			}
		}
		items = append(items, map[string]any{"id": item.ID, "name": item.Name, "role": item.Role, "personality": item.Personality, "background": item.Background, "relationships": map[string]string(item.Relationships)})
	}
	return marshalContext(map[string]any{"characters": items})
}

func marshalContext(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func tailText(text string, limit int) string {
	trimmed := strings.TrimSpace(text)
	if utf8.RuneCountInString(trimmed) <= limit {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[len(runes)-limit:])
}

func loadContentStageFile(author, novelName string, volumeID, chapterID uint, stage string) (string, error) {
	fileType := filepath.Join(novelName, fmt.Sprintf("volume_%d", volumeID), constant.FileTypeContent, fmt.Sprintf("chapter_%d", chapterID))
	return utils.LoadFromFile(author, fileType, stage)
}

func saveContentStageFile(author, novelName string, volumeID, chapterID uint, stage, content string) error {
	fileType := filepath.Join(novelName, fmt.Sprintf("volume_%d", volumeID), constant.FileTypeContent, fmt.Sprintf("chapter_%d", chapterID))
	return utils.SaveToFile(author, fileType, stage, content)
}
