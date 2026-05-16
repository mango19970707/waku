package compose

import (
	"context"

	"waku/internal/plotline/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// AgentExecutor Agent 执行器接口
type AgentExecutor interface {
	// ExtractPlotline 从大纲提取剧情线（兼容旧接口）
	ExtractPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error)

	// ExtractPlotlinesJSON 从大纲提取剧情线（返回 JSON 结构）
	ExtractPlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (*agents.PlotlineExtractorOutput, error)

	// RevisePlotlinesJSON 优化剧情线（返回 JSON 结构）
	RevisePlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingPlotlines, userSuggestion string) (*agents.PlotlineRevisionOutput, error)

	// PolishPlotline 润色剧情线
	PolishPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, plotline, genre string) (string, error)

	// DetectConflicts 检测冲突（旧版本，保持兼容）
	DetectConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, targetPlotline, otherPlotlines string) (*agents.ConflictDetectionOutput, error)

	// DetectOutlineConflicts 检测与大纲的冲突
	DetectOutlineConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, targetPlotline, userSuggestion string) (*agents.OutlineConflictDetectionOutput, error)

	// DetectPlotlinesConflicts 检测与其他剧情线的冲突（单个剧情线）
	DetectPlotlinesConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, genre, targetPlotline, otherPlotline, otherPlotlineID, userSuggestion string) (*agents.PlotlinesConflictDetectionOutput, error)

	// ResolveOutlineConflicts 解决大纲冲突
	ResolveOutlineConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre, targetPlotline, conflicts string) (string, error)

	// ResolvePlotlinesConflicts 解决其他剧情线冲突（单个剧情线）
	ResolvePlotlinesConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, targetPlotline, otherPlotline, otherPlotlineID, conflicts string) (*agents.PlotlinesConflictResolverOutput, error)
}

// agentExecutor 全局 Agent 执行器实例
var agentExecutor AgentExecutor

// RegisterAgentExecutor 注册 Agent 执行器
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// realAgentExecutor 真实的 Agent 执行器实现
type realAgentExecutor struct{}

// ExtractPlotline 从大纲提取剧情线（兼容旧接口）
func (e *realAgentExecutor) ExtractPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	agent, err := agents.NewPlotlineExtractorAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}

	return agents.ExtractPlotline(ctx, agent, &agents.PlotlineExtractorInput{
		Outline: outline,
		Genre:   genre,
	})
}

// ExtractPlotlinesJSON 从大纲提取剧情线（返回 JSON 结构）
func (e *realAgentExecutor) ExtractPlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (*agents.PlotlineExtractorOutput, error) {
	agent, err := agents.NewPlotlineExtractorAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.ExtractPlotlines(ctx, agent, &agents.PlotlineExtractorInput{
		Outline: outline,
		Genre:   genre,
	})
}

// RevisePlotlinesJSON 优化剧情线（返回 JSON 结构）
func (e *realAgentExecutor) RevisePlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingPlotlines, userSuggestion string) (*agents.PlotlineRevisionOutput, error) {
	agent, err := agents.NewPlotlineRevisionAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.RevisePlotlines(ctx, agent, &agents.PlotlineRevisionInput{
		Outline:           outline,
		Genre:             genre,
		ExistingPlotlines: existingPlotlines,
		UserSuggestion:    userSuggestion,
	})
}

// PolishPlotline 润色剧情线
func (e *realAgentExecutor) PolishPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, plotline, genre string) (string, error) {
	agent, err := agents.NewPlotlinePolisherAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}

	return agents.PolishPlotline(ctx, agent, &agents.PlotlinePolisherInput{
		Plotline: plotline,
		Genre:    genre,
	})
}

// DetectConflicts 检测冲突
func (e *realAgentExecutor) DetectConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, targetPlotline, otherPlotlines string) (*agents.ConflictDetectionOutput, error) {
	agent, err := agents.NewConflictDetectionAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.DetectConflicts(ctx, agent, &agents.ConflictDetectionInput{
		Outline:        outline,
		Genre:          genre,
		TargetPlotline: targetPlotline,
		OtherPlotlines: otherPlotlines,
	})
}

// ResolveOutlineConflicts 解决大纲冲突
func (e *realAgentExecutor) ResolveOutlineConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre, targetPlotline, conflicts string) (string, error) {
	agent, err := agents.NewOutlineConflictResolverAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}

	return agents.ResolveOutlineConflicts(ctx, agent, &agents.OutlineConflictResolverInput{
		OriginalOutline: originalOutline,
		Genre:           genre,
		TargetPlotline:  targetPlotline,
		Conflicts:       conflicts,
	})
}

// ResolvePlotlinesConflicts 解决其他剧情线冲突（单个剧情线）
func (e *realAgentExecutor) ResolvePlotlinesConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, targetPlotline, otherPlotline, otherPlotlineID, conflicts string) (*agents.PlotlinesConflictResolverOutput, error) {
	agent, err := agents.NewPlotlinesConflictResolverAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.ResolvePlotlinesConflicts(ctx, agent, &agents.PlotlinesConflictResolverInput{
		Outline:         outline,
		Genre:           genre,
		TargetPlotline:  targetPlotline,
		OtherPlotline:   otherPlotline,
		OtherPlotlineID: otherPlotlineID,
		Conflicts:       conflicts,
	})
}

// DetectOutlineConflicts 检测与大纲的冲突
func (e *realAgentExecutor) DetectOutlineConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, targetPlotline, userSuggestion string) (*agents.OutlineConflictDetectionOutput, error) {
	agent, err := agents.NewOutlineConflictDetectionAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.DetectOutlineConflicts(ctx, agent, &agents.OutlineConflictDetectionInput{
		Outline:        outline,
		Genre:          genre,
		TargetPlotline: targetPlotline,
		UserSuggestion: userSuggestion,
	})
}

// DetectPlotlinesConflicts 检测与其他剧情线的冲突（单个剧情线）
func (e *realAgentExecutor) DetectPlotlinesConflicts(ctx context.Context, chatModel einoModel.BaseChatModel, genre, targetPlotline, otherPlotline, otherPlotlineID, userSuggestion string) (*agents.PlotlinesConflictDetectionOutput, error) {
	agent, err := agents.NewPlotlinesConflictDetectionAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.DetectPlotlinesConflicts(ctx, agent, &agents.PlotlinesConflictDetectionInput{
		Genre:           genre,
		TargetPlotline:  targetPlotline,
		OtherPlotline:   otherPlotline,
		OtherPlotlineID: otherPlotlineID,
		UserSuggestion:  userSuggestion,
	})
}

func init() {
	// 注册真实的 Agent 执行器
	RegisterAgentExecutor(&realAgentExecutor{})
}
