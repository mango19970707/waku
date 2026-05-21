package backend

import (
	"context"
	"fmt"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"

	chaptercompose "waku/internal/chapter/compose"
	chapterrepo "waku/internal/chapter/repository"
	charactercompose "waku/internal/character/compose"
	characterrepo "waku/internal/character/repository"
	contentcompose "waku/internal/content/compose"
	contentrepo "waku/internal/content/repository"
	outlinecompose "waku/internal/outline/compose"
	outlinerepo "waku/internal/outline/repository"
	plotlinecompose "waku/internal/plotline/compose"
	plotlinerepo "waku/internal/plotline/repository"
	volumecompose "waku/internal/volume/compose"
	volumerepo "waku/internal/volume/repository"
)

type DefaultWorkflowRunner struct {
	db        *gorm.DB
	chatModel einoModel.BaseChatModel
}

func NewDefaultWorkflowRunner(db *gorm.DB, chatModel einoModel.BaseChatModel) *DefaultWorkflowRunner {
	return &DefaultWorkflowRunner{db: db, chatModel: chatModel}
}

func (r *DefaultWorkflowRunner) Run(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("后端工作流未初始化")
	}
	switch req.Step {
	case StepOutline:
		return r.runOutline(ctx, req)
	case StepPlotlines:
		return r.runPlotlines(ctx, req)
	case StepCharacters:
		return r.runCharacters(ctx, req)
	case StepVolumes:
		return r.runVolumes(ctx, req)
	case StepChapters:
		return r.runChapters(ctx, req)
	case StepContent:
		return r.runContent(ctx, req)
	default:
		return nil, fmt.Errorf("不支持的创作步骤: %s", req.Step)
	}
}

func (r *DefaultWorkflowRunner) runOutline(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	// HTTP 层只组装工作流输入，具体生成、审核、保存逻辑全部委托 internal/outline 工作流。
	repo := outlinerepo.NewRepository(r.db)
	graph, err := outlinecompose.BuildOutlineWorkflowGraph(ctx, r.chatModel, repo)
	if err != nil {
		return nil, err
	}
	state, err := outlinecompose.ExecuteOutlineWorkflow(ctx, graph, &outlinecompose.OutlineWorkflowInput{
		NovelID:         req.NovelID,
		NovelName:       req.NovelName,
		Author:          req.Author,
		Genre:           req.Genre,
		OriginalOutline: req.OriginalOutline,
		UserSuggestion:  req.UserSuggestion,
	})
	if err != nil {
		return nil, err
	}
	return &WorkflowRunResult{Step: StepOutline, NovelID: state.SavedNovelID, SavedIDs: []uint{state.SavedNovelID}, CurrentNode: state.CurrentNode, Message: "小说大纲工作流执行完成", Data: map[string]any{"title": state.NovelTitle, "synopsis": state.Synopsis, "outline": state.Outline}}, nil
}

func (r *DefaultWorkflowRunner) runPlotlines(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	// 剧情线依赖小说大纲，仓储和上下文提取由 internal/plotline 工作流负责。
	novelRepo := plotlinerepo.NewRepository(r.db)
	plotlineRepository := plotlinerepo.NewPlotlineRepository(r.db)
	graph, err := plotlinecompose.NewPlotlineWorkflow(ctx, r.chatModel, novelRepo, plotlineRepository)
	if err != nil {
		return nil, err
	}
	state, err := plotlinecompose.ExecutePlotlineWorkflow(ctx, graph, &plotlinecompose.PlotlineWorkflowInput{NovelID: req.NovelID, PlotlineID: req.PlotlineID, UserSuggestion: req.UserSuggestion})
	if err != nil {
		return nil, err
	}
	return &WorkflowRunResult{Step: StepPlotlines, NovelID: state.NovelID, SavedIDs: state.SavedPlotlineIDs, CurrentNode: state.CurrentNode, Message: "剧情线工作流执行完成", Data: map[string]any{"plotlines": state.RevisedPlotlines}}, nil
}

func (r *DefaultWorkflowRunner) runCharacters(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	// 关键人物依赖剧情线，HTTP 层仅传递角色 ID 与用户建议，避免复制 internal 规则。
	novelRepo := characterrepo.NewRepository(r.db)
	characterRepository := characterrepo.NewCharacterRepository(r.db)
	plotlineRepository := plotlinerepo.NewPlotlineRepository(r.db)
	graph, err := charactercompose.NewCharacterWorkflow(ctx, r.chatModel, novelRepo, characterRepository, plotlineRepository)
	if err != nil {
		return nil, err
	}
	state, err := charactercompose.ExecuteCharacterWorkflow(ctx, graph, &charactercompose.CharacterWorkflowInput{NovelID: req.NovelID, CharacterID: req.CharacterID, UserSuggestion: req.UserSuggestion})
	if err != nil {
		return nil, err
	}
	return &WorkflowRunResult{Step: StepCharacters, NovelID: state.NovelID, SavedIDs: state.SavedCharacterIDs, CurrentNode: state.CurrentNode, Message: "关键人物工作流执行完成", Data: map[string]any{"characters": state.RevisedCharacters}}, nil
}

func (r *DefaultWorkflowRunner) runVolumes(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	// 分卷工作流支持完整生成、新增和修改，模式选择由 internal/volume 根据输入判断。
	novelRepo := outlinerepo.NewRepository(r.db)
	plotlineRepository := plotlinerepo.NewPlotlineRepository(r.db)
	volumeRepository := volumerepo.NewVolumeRepository(r.db)
	graph, err := volumecompose.NewVolumeWorkflow(ctx, r.chatModel, novelRepo, plotlineRepository, volumeRepository)
	if err != nil {
		return nil, err
	}
	state, err := volumecompose.ExecuteVolumeWorkflow(ctx, graph, &volumecompose.VolumeWorkflowInput{NovelID: req.NovelID, VolumeID: req.VolumeID, UserSuggestion: req.UserSuggestion})
	if err != nil {
		return nil, err
	}
	return &WorkflowRunResult{Step: StepVolumes, NovelID: state.NovelID, SavedIDs: state.SavedVolumeIDs, CurrentNode: state.CurrentNode, Message: "分卷工作流执行完成", Data: map[string]any{"volumes": state.RevisedVolumes}}, nil
}

func (r *DefaultWorkflowRunner) runChapters(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	// 章节划分以分卷为边界，章节连续性和覆盖策略由 internal/chapter 负责。
	novelRepo := outlinerepo.NewRepository(r.db)
	volumeRepository := volumerepo.NewVolumeRepository(r.db)
	plotlineRepository := plotlinerepo.NewPlotlineRepository(r.db)
	characterRepository := characterrepo.NewCharacterRepository(r.db)
	chapterRepository := chapterrepo.NewChapterRepository(r.db)
	graph, err := chaptercompose.NewChapterWorkflow(ctx, r.chatModel, novelRepo, volumeRepository, plotlineRepository, characterRepository, chapterRepository)
	if err != nil {
		return nil, err
	}
	state, err := chaptercompose.ExecuteChapterWorkflow(ctx, graph, &chaptercompose.ChapterWorkflowInput{NovelID: req.NovelID, VolumeID: req.VolumeID, ChapterID: req.ChapterID, UserSuggestion: req.UserSuggestion})
	if err != nil {
		return nil, err
	}
	return &WorkflowRunResult{Step: StepChapters, NovelID: state.NovelID, SavedIDs: state.SavedChapterIDs, CurrentNode: state.CurrentNode, Message: "章节划分工作流执行完成", Data: map[string]any{"chapters": state.RevisedChapters}}, nil
}

func (r *DefaultWorkflowRunner) runContent(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	// 正文写作可以通过章节 ID 入口触发；缺失的小说和分卷 ID 从章节仓储补齐，保证前端调用简单。
	contentRepository := contentrepo.NewContentRepository(r.db)
	if req.ChapterID > 0 && (req.NovelID == 0 || req.VolumeID == 0) {
		chapter, err := contentRepository.GetChapterByID(req.ChapterID)
		if err != nil {
			return nil, err
		}
		req.NovelID = chapter.NovelID
		req.VolumeID = chapter.VolumeID
	}
	novelRepo := outlinerepo.NewRepository(r.db)
	volumeRepository := volumerepo.NewVolumeRepository(r.db)
	plotlineRepository := plotlinerepo.NewPlotlineRepository(r.db)
	characterRepository := characterrepo.NewCharacterRepository(r.db)
	graph, err := contentcompose.NewContentWorkflow(ctx, r.chatModel, novelRepo, volumeRepository, plotlineRepository, characterRepository, contentRepository)
	if err != nil {
		return nil, err
	}
	state, err := contentcompose.ExecuteContentWorkflow(ctx, graph, &contentcompose.ContentWorkflowInput{NovelID: req.NovelID, VolumeID: req.VolumeID, ChapterID: req.ChapterID, ChapterIDs: req.ChapterIDs, UserSuggestion: req.UserSuggestion})
	if err != nil {
		return nil, err
	}
	return &WorkflowRunResult{Step: StepContent, NovelID: state.NovelID, SavedIDs: state.SavedChapterIDs, CurrentNode: state.CurrentNode, Message: "正文撰写工作流执行完成", Data: map[string]any{"content": state.RevisedContent}}, nil
}
