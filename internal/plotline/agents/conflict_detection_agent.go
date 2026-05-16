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

// ConflictDetectionInput 冲突检测的输入参数
type ConflictDetectionInput struct {
	// Outline 小说大纲
	Outline string
	// Genre 小说类型
	Genre string
	// TargetPlotline 修改后的目标剧情线（JSON 格式）
	TargetPlotline string
	// OtherPlotlines 其他剧情线列表（JSON 格式）
	OtherPlotlines string
}

// ConflictItem 单个冲突项
type ConflictItem struct {
	// Type 冲突类型（timeline, character, logic, outline）
	Type string `json:"type"`
	// Severity 严重程度（high, medium, low）
	Severity string `json:"severity"`
	// Target 冲突目标（"outline" 或剧情线ID）
	Target string `json:"target"`
	// Description 冲突描述
	Description string `json:"description"`
	// Suggestion 修改建议
	Suggestion string `json:"suggestion"`
}

// ConflictDetectionOutput 冲突检测的输出结构
type ConflictDetectionOutput struct {
	// HasConflict 是否存在冲突
	HasConflict bool `json:"has_conflict"`
	// Conflicts 冲突列表
	Conflicts []ConflictItem `json:"conflicts"`
	// Summary 冲突总结
	Summary string `json:"summary"`
}

// NewConflictDetectionAgent 创建冲突检测 Agent
//
// 功能：检测修改后的目标剧情线与大纲、其他剧情线之间的冲突
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 冲突检测 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewConflictDetectionAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := DetectConflicts(ctx, agent, input)
func NewConflictDetectionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ConflictDetection",
		Description: "冲突检测专家，检测剧情线之间、剧情线与大纲之间的逻辑冲突",
		Instruction: prompts.ConflictDetectionPrompt,
		Model:       chatModel,
	})
}

// DetectConflicts 检测冲突（使用 JSON parser 严格校验）
//
// 参数：
//   - ctx: 上下文
//   - agent: 冲突检测 Agent
//   - input: 输入参数（大纲、目标剧情线、其他剧情线）
//
// 返回值：
//   - *ConflictDetectionOutput: 冲突检测结果
//   - error: 检测失败时返回错误
func DetectConflicts(ctx context.Context, agent *adk.ChatModelAgent, input *ConflictDetectionInput) (*ConflictDetectionOutput, error) {
	if input.Outline == "" {
		return nil, fmt.Errorf("大纲不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.TargetPlotline == "" {
		return nil, fmt.Errorf("目标剧情线不能为空")
	}
	if input.OtherPlotlines == "" {
		return nil, fmt.Errorf("其他剧情线不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请检测以下剧情线是否存在冲突，并以 JSON 格式输出：

**小说类型**：%s

**小说大纲**：
%s

**修改后的目标剧情线**：
%s

**其他剧情线**：
%s

请严格按照 JSON 格式输出，包含 has_conflict、conflicts 数组、summary 字段。
- 如果存在冲突，conflicts 数组中每个冲突项包含 type、severity、target、description、suggestion 字段
- 如果不存在冲突，has_conflict 为 false，conflicts 为空数组`, input.Genre, input.Outline, input.TargetPlotline, input.OtherPlotlines)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[ConflictDetection] 开始检测冲突...")

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("检测冲突失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				// 打印流式输出的内容
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

	// 使用 MessageJSONParser 解析 JSON 输出
	parserConfig := &schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	}
	parser := schema.NewMessageJSONParser[ConflictDetectionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if output.Summary == "" {
		return nil, fmt.Errorf("冲突总结为空")
	}

	// 如果有冲突，验证每个冲突项的字段
	if output.HasConflict {
		if len(output.Conflicts) == 0 {
			return nil, fmt.Errorf("has_conflict 为 true 但 conflicts 数组为空")
		}

		for i, conflict := range output.Conflicts {
			if conflict.Type == "" {
				return nil, fmt.Errorf("第 %d 个冲突的类型为空", i+1)
			}
			if conflict.Type != "timeline" && conflict.Type != "character" && conflict.Type != "logic" && conflict.Type != "outline" {
				return nil, fmt.Errorf("第 %d 个冲突的类型无效: %s（只能是 timeline、character、logic、outline）", i+1, conflict.Type)
			}
			if conflict.Severity == "" {
				return nil, fmt.Errorf("第 %d 个冲突的严重程度为空", i+1)
			}
			if conflict.Severity != "high" && conflict.Severity != "medium" && conflict.Severity != "low" {
				return nil, fmt.Errorf("第 %d 个冲突的严重程度无效: %s（只能是 high、medium、low）", i+1, conflict.Severity)
			}
			if conflict.Target == "" {
				return nil, fmt.Errorf("第 %d 个冲突的目标为空", i+1)
			}
			if conflict.Description == "" {
				return nil, fmt.Errorf("第 %d 个冲突的描述为空", i+1)
			}
			if conflict.Suggestion == "" {
				return nil, fmt.Errorf("第 %d 个冲突的修改建议为空", i+1)
			}
		}
	}

	if output.HasConflict {
		slog.Info(fmt.Sprintf("[ConflictDetection] 检测到 %d 个冲突", len(output.Conflicts)))
	} else {
		slog.Info("[ConflictDetection] 未检测到冲突")
	}

	return &output, nil
}
