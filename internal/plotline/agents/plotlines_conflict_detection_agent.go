package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// PlotlinesConflictDetectionInput 剧情线冲突检测输入（单个剧情线）
type PlotlinesConflictDetectionInput struct {
	Genre           string `json:"genre"`             // 小说类型
	TargetPlotline  string `json:"target_plotline"`   // 目标剧情线（JSON 格式）
	OtherPlotline   string `json:"other_plotline"`    // 待检测的单个剧情线（JSON 格式）
	OtherPlotlineID string `json:"other_plotline_id"` // 待检测的剧情线ID
	UserSuggestion  string `json:"user_suggestion"`   // 用户修改建议
}

// PlotlinesConflictDetectionOutput 剧情线冲突检测输出（单个剧情线）
type PlotlinesConflictDetectionOutput struct {
	PlotlineID  string         `json:"plotline_id"`   // 被检测的剧情线ID
	HasConflict bool           `json:"has_conflicts"` // 是否有冲突
	Conflicts   []ConflictItem `json:"conflicts"`     // 冲突列表
}

// NewPlotlinesConflictDetectionAgent 创建剧情线冲突检测 Agent
func NewPlotlinesConflictDetectionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlinesConflictDetection",
		Description: "剧情线冲突检测专家，检测剧情线修改与其他剧情线之间的冲突",
		Instruction: prompts.PlotlinesConflictDetectionPrompt,
		Model:       chatModel,
	})
}

// DetectPlotlinesConflicts 检测与其他剧情线的冲突
func DetectPlotlinesConflicts(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlinesConflictDetectionInput) (*PlotlinesConflictDetectionOutput, error) {
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.TargetPlotline == "" {
		return nil, fmt.Errorf("目标剧情线不能为空")
	}
	if input.OtherPlotline == "" {
		return nil, fmt.Errorf("待检测的剧情线不能为空")
	}
	if input.OtherPlotlineID == "" {
		return nil, fmt.Errorf("待检测的剧情线ID不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请检测以下剧情线修改与指定剧情线之间的冲突：

**小说类型**: %s

**目标剧情线（已修改）**:
%s

**待检测的剧情线**:
%s

**用户修改建议**:
%s

请严格按照 JSON 格式输出，包含 plotline_id、has_conflicts、conflicts 数组。`,
		input.Genre,
		input.TargetPlotline,
		input.OtherPlotline,
		input.UserSuggestion,
	)

	// 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[PlotlinesConflictDetection] 开始检测与其他剧情线的冲突...")

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("检测剧情线冲突失败: %w", event.Err)
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				if msg.Content != "" {
					slog.Info(msg.Content)
				}
				lastMessage = msg
			}
		}
	}

	if lastMessage == nil {
		return nil, fmt.Errorf("未获取到 Agent 响应")
	}

	// 解析 JSON 输出
	parserConfig := &schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	}
	parser := schema.NewMessageJSONParser[PlotlinesConflictDetectionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if output.PlotlineID == "" {
		return nil, fmt.Errorf("输出中缺少 plotline_id 字段")
	}
	if output.PlotlineID != input.OtherPlotlineID {
		return nil, fmt.Errorf("输出的 plotline_id (%s) 与输入不匹配 (%s)", output.PlotlineID, input.OtherPlotlineID)
	}
	if output.HasConflict && len(output.Conflicts) == 0 {
		return nil, fmt.Errorf("has_conflicts 为 true 但 conflicts 数组为空")
	}

	if output.HasConflict {
		slog.Info(fmt.Sprintf("[PlotlinesConflictDetection] 剧情线 %s 检测到 %d 个冲突", output.PlotlineID, len(output.Conflicts)))
	} else {
		slog.Info(fmt.Sprintf("[PlotlinesConflictDetection] 剧情线 %s 未检测到冲突", output.PlotlineID))
	}

	return &output, nil
}
