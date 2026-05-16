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

// OutlineConflictDetectionInput 大纲冲突检测输入
type OutlineConflictDetectionInput struct {
	Outline        string `json:"outline"`         // 小说大纲
	Genre          string `json:"genre"`           // 小说类型
	TargetPlotline string `json:"target_plotline"` // 目标剧情线（JSON 格式）
	UserSuggestion string `json:"user_suggestion"` // 用户修改建议
}

// OutlineConflictDetectionOutput 大纲冲突检测输出
type OutlineConflictDetectionOutput struct {
	HasConflict bool           `json:"has_conflicts"` // 是否有冲突
	Conflicts   []ConflictItem `json:"conflicts"`     // 冲突列表
}

// NewOutlineConflictDetectionAgent 创建大纲冲突检测 Agent
func NewOutlineConflictDetectionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineConflictDetection",
		Description: "大纲冲突检测专家，检测剧情线修改与小说大纲之间的冲突",
		Instruction: prompts.OutlineConflictDetectionPrompt,
		Model:       chatModel,
	})
}

// DetectOutlineConflicts 检测与大纲的冲突
func DetectOutlineConflicts(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineConflictDetectionInput) (*OutlineConflictDetectionOutput, error) {
	if input.Outline == "" {
		return nil, fmt.Errorf("大纲不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.TargetPlotline == "" {
		return nil, fmt.Errorf("目标剧情线不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请检测以下剧情线修改与小说大纲之间的冲突：

**小说类型**: %s

**小说大纲**:
%s

**目标剧情线**:
%s

**用户修改建议**:
%s

请严格按照 JSON 格式输出，包含 has_conflicts、conflicts 数组。`,
		input.Genre,
		input.Outline,
		input.TargetPlotline,
		input.UserSuggestion,
	)

	// 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlineConflictDetection] 开始检测与大纲的冲突...")

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("检测大纲冲突失败: %w", event.Err)
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
	parser := schema.NewMessageJSONParser[OutlineConflictDetectionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if output.HasConflict && len(output.Conflicts) == 0 {
		return nil, fmt.Errorf("has_conflicts 为 true 但 conflicts 数组为空")
	}

	if output.HasConflict {
		slog.Info(fmt.Sprintf("[OutlineConflictDetection] 检测到 %d 个与大纲的冲突", len(output.Conflicts)))
	} else {
		slog.Info("[OutlineConflictDetection] 未检测到与大纲的冲突")
	}

	return &output, nil
}
