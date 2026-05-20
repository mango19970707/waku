package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"waku/internal/chapter/agents"
	"waku/internal/chapter/contracts"
	chapterrepository "waku/internal/chapter/repository"
	characterrepository "waku/internal/character/repository"
	"waku/internal/model"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	"waku/internal/utils"
	volumerepository "waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
)

// initNode 初始化节点
func initNode(
	novelRepo novelrepository.Repository,
	volumeRepo volumerepository.VolumeRepository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	characterRepo characterrepository.CharacterRepository,
	chapterRepo chapterrepository.ChapterRepository,
) func(context.Context, *ChapterWorkflowInput) (*ChapterWorkflowState, error) {
	return func(ctx context.Context, input *ChapterWorkflowInput) (state *ChapterWorkflowState, err error) {
		state = newWorkflowState(input)
		logNodeInput("init", state)
		defer func() {
			logNodeOutput("init", state, err)
		}()

		// 加载小说信息
		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("加载小说失败: %w", err)
		}
		state.NovelName = novel.Title
		state.Author = novel.Author
		state.Genre = novel.Genre
		state.NovelOutline = strings.TrimSpace(novel.Outline)
		// Worldview 和 CultivationSystem 存储在 Outline JSON 中，这里暂时留空
		// 如果需要，可以从 Outline JSON 中解析
		state.NovelWorldview = ""
		state.CultivationSystem = ""

		// 加载分卷信息
		volume, err := volumeRepo.GetByID(input.VolumeID)
		if err != nil {
			return nil, fmt.Errorf("加载分卷失败: %w", err)
		}
		if volume.NovelID != input.NovelID {
			return nil, fmt.Errorf("分卷 %d 不属于小说 %d", input.VolumeID, input.NovelID)
		}
		state.VolumeOutline = strings.TrimSpace(volume.Outline)
		state.VolumeTheme = strings.TrimSpace(volume.Theme)

		// 加载剧情线列表
		plotlines, err := loadPlotlinesFromDB(plotlineRepo, novel.ID)
		if err != nil {
			return nil, err
		}
		state.ExistingPlotlines = plotlines

		// 加载角色列表
		characters, err := loadCharactersFromDB(characterRepo, novel.ID)
		if err != nil {
			return nil, err
		}
		state.ExistingCharacters = characters

		// 加载已有章节列表
		chapters, err := loadChaptersFromDB(chapterRepo, input.VolumeID)
		if err != nil {
			return nil, err
		}
		state.ExistingChapters = chapters

		// 如果提供了章节ID，加载目标章节
		if input.ChapterID > 0 {
			targetChapter, err := chapterRepo.GetByID(input.ChapterID)
			if err != nil {
				return nil, fmt.Errorf("加载目标章节失败: %w", err)
			}
			if targetChapter.VolumeID != input.VolumeID {
				return nil, fmt.Errorf("章节 %d 不属于分卷 %d", input.ChapterID, input.VolumeID)
			}

			targetJSON, err := json.Marshal(contracts.FromModel(targetChapter))
			if err != nil {
				return nil, fmt.Errorf("序列化目标章节失败: %w", err)
			}
			state.TargetChapterJSON = string(targetJSON)
		}

		logWorkflowInput(state)
		return state, nil
	}
}

// generateNode 生成节点
func generateNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ChapterWorkflowState) (*ChapterWorkflowState, error) {
	return func(ctx context.Context, state *ChapterWorkflowState) (result *ChapterWorkflowState, err error) {
		logNodeInput("generate", state)
		result = state
		defer func() {
			logNodeOutput("generate", result, err)
		}()

		if state.GeneratedChapters != "" {
			state.CurrentNode = "generate_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传）
		content, err := loadChapterStageFile(state.Author, state.NovelName, state.VolumeID, "章节规划")
		if err == nil && content != "" {
			normalized, _, err := contracts.NormalizeChapterPayload(content)
			if err == nil {
				state.GeneratedChapters = normalized
				state.CurrentNode = "generate_success"
				state.RetryCount = 0
				return state, nil
			}
		}

		// 执行生成
		input := &agents.ChapterGeneratorInput{
			NovelOutline:     state.NovelOutline,
			NovelWorldview:   state.NovelWorldview,
			CultivationSys:   state.CultivationSystem,
			VolumeOutline:    state.VolumeOutline,
			VolumeTheme:      state.VolumeTheme,
			Plotlines:        state.ExistingPlotlines,
			Characters:       state.ExistingCharacters,
			ExistingChapters: state.ExistingChapters,
		}

		generated, err := executeGenerateAgent(ctx, chatModel, input)
		if err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.GeneratedChapters = generated
		if err := saveChapterStageFile(state.Author, state.NovelName, state.VolumeID, "章节规划", generated); err != nil {
			state.CurrentNode = "generate_failed"
			state.ErrorMsg = fmt.Sprintf("保存章节规划文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "generate_success"
		state.RetryCount = 0
		return state, nil
	}
}

// polishNode 润色节点
func polishNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ChapterWorkflowState) (*ChapterWorkflowState, error) {
	return func(ctx context.Context, state *ChapterWorkflowState) (result *ChapterWorkflowState, err error) {
		logNodeInput("polish", state)
		result = state
		defer func() {
			logNodeOutput("polish", result, err)
		}()

		if state.PolishedChapters != "" {
			state.CurrentNode = "polish_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传）
		content, err := loadChapterStageFile(state.Author, state.NovelName, state.VolumeID, "润色")
		if err == nil && content != "" {
			normalized, _, err := contracts.NormalizeChapterPayload(content)
			if err == nil {
				state.PolishedChapters = normalized
				state.CurrentNode = "polish_success"
				state.RetryCount = 0
				return state, nil
			}
		}

		// 确定输入源
		source := state.GeneratedChapters
		if source == "" {
			source = state.ExistingChapters
		}
		if source == "" {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = "缺少待润色的章节数据"
			return state, nil
		}

		// 执行润色
		input := &agents.ChapterPolisherInput{
			Chapters:         source,
			UserSuggestion:   state.UserSuggestion,
			NovelOutline:     state.NovelOutline,
			VolumeOutline:    state.VolumeOutline,
			Plotlines:        state.ExistingPlotlines,
			Characters:       state.ExistingCharacters,
			ExistingChapters: state.ExistingChapters,
		}

		polished, err := executePolishAgent(ctx, chatModel, input)
		if err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.PolishedChapters = polished
		if err := saveChapterStageFile(state.Author, state.NovelName, state.VolumeID, "润色", polished); err != nil {
			state.CurrentNode = "polish_failed"
			state.ErrorMsg = fmt.Sprintf("保存润色文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "polish_success"
		state.RetryCount = 0
		return state, nil
	}
}

// reviewNode 审核节点
func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ChapterWorkflowState) (*ChapterWorkflowState, error) {
	return func(ctx context.Context, state *ChapterWorkflowState) (result *ChapterWorkflowState, err error) {
		logNodeInput("review", state)
		result = state
		defer func() {
			logNodeOutput("review", result, err)
		}()

		if state.ReviewedChapters != "" {
			state.CurrentNode = "review_success"
			return state, nil
		}

		// 尝试从文件加载（断点续传）
		content, err := loadChapterStageFile(state.Author, state.NovelName, state.VolumeID, "审核")
		if err == nil && content != "" {
			normalized, _, err := contracts.NormalizeChapterReviewPayload(content)
			if err == nil {
				state.ReviewedChapters = normalized
				state.CurrentNode = "review_success"
				state.RetryCount = 0
				return state, nil
			}
		}

		// 确定输入源
		source := state.PolishedChapters
		if source == "" {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = "缺少待审核的章节数据"
			return state, nil
		}

		// 执行审核
		input := &agents.ChapterReviewerInput{
			Chapters:       source,
			NovelOutline:   state.NovelOutline,
			NovelWorldview: state.NovelWorldview,
			CultivationSys: state.CultivationSystem,
			VolumeOutline:  state.VolumeOutline,
			Plotlines:      state.ExistingPlotlines,
			Characters:     state.ExistingCharacters,
		}

		reviewed, err := executeReviewAgent(ctx, chatModel, input)
		if err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		state.ReviewedChapters = reviewed
		if err := saveChapterStageFile(state.Author, state.NovelName, state.VolumeID, "审核", reviewed); err != nil {
			state.CurrentNode = "review_failed"
			state.ErrorMsg = fmt.Sprintf("保存审核文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "review_success"
		state.RetryCount = 0
		return state, nil
	}
}

// reviseNode 修改节点
func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *ChapterWorkflowState) (*ChapterWorkflowState, error) {
	return func(ctx context.Context, state *ChapterWorkflowState) (result *ChapterWorkflowState, err error) {
		logNodeInput("revise", state)
		result = state
		defer func() {
			logNodeOutput("revise", result, err)
		}()

		if state.RevisedChapters != "" {
			state.CurrentNode = "revise_success"
			return state, nil
		}

		// 确定输入源
		source := state.PolishedChapters
		if source == "" {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = "缺少待修改的章节数据"
			return state, nil
		}

		// 尝试从文件加载（断点续传）
		content, err := loadChapterStageFile(state.Author, state.NovelName, state.VolumeID, "修改")
		if err == nil && content != "" {
			normalized, _, err := normalizeRevisedChapters(source, content)
			if err == nil {
				state.RevisedChapters = normalized
				if err := saveChapterStageFile(state.Author, state.NovelName, state.VolumeID, "修改", normalized); err != nil {
					state.CurrentNode = "revise_failed"
					state.ErrorMsg = fmt.Sprintf("保存修改文件失败: %v", err)
					return state, nil
				}
				state.CurrentNode = "revise_success"
				state.RetryCount = 0
				return state, nil
			}
		}
		_, reviewPayload, err := contracts.NormalizeChapterReviewPayload(state.ReviewedChapters)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("解析审核建议失败: %v", err)
			return state, nil
		}
		if len(reviewPayload.Suggestions) == 0 {
			state.RevisedChapters = source
			if err := saveChapterStageFile(state.Author, state.NovelName, state.VolumeID, "修改", source); err != nil {
				state.CurrentNode = "revise_failed"
				state.ErrorMsg = fmt.Sprintf("保存修改文件失败: %v", err)
				return state, nil
			}
			state.CurrentNode = "revise_success"
			state.RetryCount = 0
			return state, nil
		}

		// 执行修改
		input := &agents.ChapterReviserInput{
			Chapters:          source,
			ReviewSuggestions: state.ReviewedChapters,
			NovelOutline:      state.NovelOutline,
			VolumeOutline:     state.VolumeOutline,
			Plotlines:         state.ExistingPlotlines,
			Characters:        state.ExistingCharacters,
		}

		revisedPatch, err := executeReviseAgent(ctx, chatModel, input)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = err.Error()
			return state, nil
		}

		revised, _, err := mergeRevisedChapters(source, revisedPatch)
		if err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("合并修改章节失败: %v", err)
			return state, nil
		}

		state.RevisedChapters = revised
		if err := saveChapterStageFile(state.Author, state.NovelName, state.VolumeID, "修改", revised); err != nil {
			state.CurrentNode = "revise_failed"
			state.ErrorMsg = fmt.Sprintf("保存修改文件失败: %v", err)
			return state, nil
		}

		state.CurrentNode = "revise_success"
		state.RetryCount = 0
		return state, nil
	}
}

// saveToDBNode 保存到数据库节点
func saveToDBNode(chapterRepo chapterrepository.ChapterRepository) func(context.Context, *ChapterWorkflowState) (*ChapterWorkflowState, error) {
	return func(ctx context.Context, state *ChapterWorkflowState) (result *ChapterWorkflowState, err error) {
		logNodeInput("saveToDB", state)
		result = state
		defer func() {
			logNodeOutput("saveToDB", result, err)
			if err == nil {
				logWorkflowOutput(state)
			}
		}()

		// 确定最终数据源
		finalData := state.RevisedChapters
		if finalData == "" {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = "缺少待保存的章节数据"
			return state, nil
		}

		// 解析章节数据
		_, payload, err := contracts.NormalizeChapterPayload(finalData)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("解析章节数据失败: %v", err)
			return state, nil
		}

		// 校验章节号连续性
		if err := validateChapterNumbers(payload.Chapters); err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("章节号校验失败: %v", err)
			return state, nil
		}

		// 保存章节
		savedIDs, err := saveChaptersToDB(chapterRepo, payload.Chapters, state.NovelID, state.VolumeID, state.RevisionMode)
		if err != nil {
			state.CurrentNode = "save_failed"
			state.ErrorMsg = fmt.Sprintf("保存章节到数据库失败: %v", err)
			return state, nil
		}

		state.SavedChapterIDs = savedIDs
		state.CurrentNode = "save_success"
		state.RetryCount = 0
		return state, nil
	}
}

// 辅助函数

func loadPlotlinesFromDB(repo plotlinerepository.PlotlineRepository, novelID uint) (string, error) {
	plotlines, err := repo.GetByNovelID(novelID)
	if err != nil {
		return "", fmt.Errorf("加载剧情线失败: %w", err)
	}
	if len(plotlines) == 0 {
		return "暂无剧情线", nil
	}

	var builder strings.Builder
	for i, p := range plotlines {
		builder.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, p.Title, p.Type))
		builder.WriteString(fmt.Sprintf("   描述：%s\n", p.Description))
	}
	return builder.String(), nil
}

func loadCharactersFromDB(repo characterrepository.CharacterRepository, novelID uint) (string, error) {
	characters, err := repo.GetByNovelID(novelID)
	if err != nil {
		return "", fmt.Errorf("加载角色失败: %w", err)
	}
	if len(characters) == 0 {
		return "暂无角色", nil
	}

	var builder strings.Builder
	for i, c := range characters {
		builder.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, c.Name, c.Role))
		builder.WriteString(fmt.Sprintf("   性格：%s\n", c.Personality))
		builder.WriteString(fmt.Sprintf("   背景：%s\n", c.Background))
	}
	return builder.String(), nil
}

func loadChaptersFromDB(repo chapterrepository.ChapterRepository, volumeID uint) (string, error) {
	chapters, err := repo.GetByVolumeID(volumeID)
	if err != nil {
		return "", fmt.Errorf("加载章节失败: %w", err)
	}
	if len(chapters) == 0 {
		return "暂无章节", nil
	}

	items := make([]contracts.ChapterItem, len(chapters))
	for i, c := range chapters {
		items[i] = *contracts.FromModel(c)
	}

	data, err := json.Marshal(contracts.ChapterPayload{Chapters: items})
	if err != nil {
		return "", fmt.Errorf("序列化章节失败: %w", err)
	}
	return string(data), nil
}

func loadChapterStageFile(author, novelName string, volumeID uint, stage string) (string, error) {
	// 构建文件类型路径：{novelName}/volume_{volumeID}/chapter
	fileType := filepath.Join(novelName, fmt.Sprintf("volume_%d", volumeID), "chapter")

	content, err := utils.LoadFromFile(author, fileType, stage)
	if err != nil {
		return "", err
	}
	return content, nil
}

func saveChapterStageFile(author, novelName string, volumeID uint, stage, content string) error {
	// 构建文件类型路径：{novelName}/volume_{volumeID}/chapter
	fileType := filepath.Join(novelName, fmt.Sprintf("volume_%d", volumeID), "chapter")

	return utils.SaveToFile(author, fileType, stage, content)
}

func validateChapterNumbers(chapters []contracts.ChapterItem) error {
	if len(chapters) == 0 {
		return fmt.Errorf("章节列表不能为空")
	}

	seen := make(map[int]bool)
	for _, ch := range chapters {
		if ch.ChapterNumber <= 0 {
			return fmt.Errorf("章节号必须大于 0")
		}
		if seen[ch.ChapterNumber] {
			return fmt.Errorf("章节号 %d 重复", ch.ChapterNumber)
		}
		seen[ch.ChapterNumber] = true
	}

	// 检查连续性
	for i := 1; i <= len(chapters); i++ {
		if !seen[i] {
			return fmt.Errorf("章节号不连续，缺少第 %d 章", i)
		}
	}

	return nil
}

func mergeRevisedChapters(baseJSON, patchJSON string) (string, *contracts.ChapterPayload, error) {
	_, basePayload, err := contracts.NormalizeChapterPayload(baseJSON)
	if err != nil {
		return "", nil, fmt.Errorf("解析原章节失败: %w", err)
	}
	_, patchPayload, err := contracts.NormalizeChapterPayload(patchJSON)
	if err != nil {
		return "", nil, fmt.Errorf("解析修改章节失败: %w", err)
	}

	merged := make([]contracts.ChapterItem, len(basePayload.Chapters))
	copy(merged, basePayload.Chapters)

	indexByID := make(map[uint]int, len(merged))
	indexByNumber := make(map[int]int, len(merged))
	for i, chapter := range merged {
		if chapter.ID > 0 {
			indexByID[chapter.ID] = i
		}
		indexByNumber[chapter.ChapterNumber] = i
	}

	for _, patch := range patchPayload.Chapters {
		index, ok := -1, false
		if patch.ID > 0 {
			index, ok = indexByID[patch.ID]
		}
		if !ok {
			index, ok = indexByNumber[patch.ChapterNumber]
		}
		if !ok {
			return "", nil, fmt.Errorf("未匹配到原章节: id=%d chapter_number=%d", patch.ID, patch.ChapterNumber)
		}

		base := merged[index]
		patch.ID = base.ID
		patch.NovelID = base.NovelID
		patch.VolumeID = base.VolumeID
		merged[index] = patch
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].ChapterNumber < merged[j].ChapterNumber
	})
	if err := validateChapterNumbers(merged); err != nil {
		return "", nil, fmt.Errorf("合并后章节号校验失败: %w", err)
	}

	mergedData, err := json.Marshal(contracts.ChapterPayload{Chapters: merged})
	if err != nil {
		return "", nil, fmt.Errorf("序列化合并章节失败: %w", err)
	}
	normalized, payload, err := contracts.NormalizeChapterPayload(string(mergedData))
	if err != nil {
		return "", nil, fmt.Errorf("规范化合并章节失败: %w", err)
	}
	return normalized, payload, nil
}

func normalizeRevisedChapters(baseJSON, revisedJSON string) (string, *contracts.ChapterPayload, error) {
	normalized, payload, err := contracts.NormalizeChapterPayload(revisedJSON)
	if err != nil {
		return "", nil, err
	}
	if err := validateChapterNumbers(payload.Chapters); err == nil {
		return normalized, payload, nil
	}
	return mergeRevisedChapters(baseJSON, revisedJSON)
}

func saveChaptersToDB(repo chapterrepository.ChapterRepository, items []contracts.ChapterItem, novelID, volumeID uint, revisionMode bool) ([]uint, error) {
	// 转换为模型
	chapters := make([]*model.Chapter, len(items))
	for i, item := range items {
		item.NovelID = novelID
		item.VolumeID = volumeID
		if !revisionMode {
			item.ID = 0
		}
		chapters[i] = item.ToModel()
	}

	// 批量保存
	if err := repo.CreateBatch(chapters); err != nil {
		return nil, err
	}

	// 收集保存的ID
	ids := make([]uint, len(chapters))
	for i, ch := range chapters {
		ids[i] = ch.ID
	}

	return ids, nil
}
