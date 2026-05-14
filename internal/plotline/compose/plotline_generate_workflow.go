package compose

import (
	"context"
	"fmt"

	einoModel "github.com/cloudwego/eino/components/model"
)

// PlotlineExtractWorkflowState 剧情线提取工作流状态
type PlotlineExtractWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string
	// NovelID 小说ID（用于更新已有小说，非必填）
	NovelID uint
	// NovelName 小说名称（用于文件保存路径）
	NovelName string
	// Author 作者名称（用于文件保存路径）
	Author string
	// Genre 小说类型
	Genre string
	// Outline 小说大纲
	Outline string
	// ExtractedPlotline 提取的剧情线
	ExtractedPlotline string
	// PolishedPlotline 润色后的剧情线
	PolishedPlotline string
	// SavedNovelID 保存到数据库后的小说ID
	SavedNovelID uint
	// ErrorMsg 错误信息
	ErrorMsg string
	// RetryCount 当前节点的重试次数
	RetryCount int
	// MaxRetries 最大重试次数
	MaxRetries int
	// WaitingForApproval 是否正在等待用户审批
	WaitingForApproval bool
	// UserApproved 用户是否批准（true=批准，false=拒绝）
	UserApproved bool
	// DisapproveReason 拒绝原因
	DisapproveReason string
}

// PlotlineExtractWorkflowInput 剧情线提取工作流输入
type PlotlineExtractWorkflowInput struct {
	// NovelID 小说ID（用于更新已有小说，非必填）
	NovelID uint
	// NovelName 小说名称（必填）
	NovelName string
	// Author 作者名称（必填）
	Author string
	// Genre 小说类型（必填）
	Genre string
	// Outline 小说大纲（必填）
	Outline string
}

// AgentExecutor 定义 Agent 执行接口，用于解耦
type AgentExecutor interface {
	ExtractPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error)
	PolishPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, plotline, genre string) (string, error)
}

// defaultAgentExecutor 默认的 Agent 执行器实现
type defaultAgentExecutor struct{}

// SetAgentExecutor 设置自定义的 Agent 执行器（用于测试或扩展）
var agentExecutor AgentExecutor = &defaultAgentExecutor{}

func (e *defaultAgentExecutor) ExtractPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) PolishPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, plotline, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

// RegisterAgentExecutor 注册 Agent 执行器（由 agents 包调用）
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// PlotlineRevisionWorkflowState 剧情线修改工作流状态
type PlotlineRevisionWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string
	// NovelID 小说ID（必填）
	NovelID uint
	// NovelName 小说名称
	NovelName string
	// Author 作者名称
	Author string
	// Genre 小说类型
	Genre string
	// OriginalPlotline 原始剧情线（从数据库加载）
	OriginalPlotline string
	// UserSuggestion 用户的修改建议
	UserSuggestion string
	// RevisedPlotline 修改后的剧情线
	RevisedPlotline string
	// ErrorMsg 错误信息
	ErrorMsg string
	// RetryCount 当前节点的重试次数
	RetryCount int
	// MaxRetries 最大重试次数
	MaxRetries int
	// WaitingForApproval 是否正在等待用户审批
	WaitingForApproval bool
	// UserApproved 用户是否批准（true=批准，false=拒绝）
	UserApproved bool
	// DisapproveReason 拒绝原因
	DisapproveReason string
}

// PlotlineRevisionWorkflowInput 剧情线修改工作流输入
type PlotlineRevisionWorkflowInput struct {
	// NovelID 小说ID（必填）
	NovelID uint
	// UserSuggestion 用户的修改建议（必填）
	UserSuggestion string
}
