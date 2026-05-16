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

// PlotlinesConflictResolverInput 剧情线冲突解决的输入参数（单个剧情线）
type PlotlinesConflictResolverInput struct {
	// Outline 修改后的大纲
	Outline string
	// Genre 小说类型
	Genre string
	// TargetPlotline 修改后的目标剧情线（JSON 格式）
	TargetPlotline string
	// OtherPlotline 待修改的单个剧情线（JSON 格式）
	OtherPlotline string
	// OtherPlotlineID 待修改的剧情线ID
	OtherPlotlineID string
	// Conflicts 冲突列表（JSON 格式）
	Conflicts string
}

// PlotlinesConflictResolverOutput 剧情线冲突解决的输出结构（单个剧情线）
type PlotlinesConflictResolverOutput struct {
	// Plotline 修改后的剧情线
	Plotline PlotlineRevisionItem `json:"plotline"`
}

// NewPlotlinesConflictResolverAgent 创建剧情线冲突解决 Agent
//
// 功能：根据大纲、目标剧情线和冲突信息，修改其他剧情线以解决冲突
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 剧情线冲突解决 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewPlotlinesConflictResolverAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ResolvePlotlinesConflicts(ctx, agent, input)
func NewPlotlinesConflictResolverAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlinesConflictResolver",
		Description: "剧情线冲突解决专家，调整其他剧情线以解决与目标剧情线的冲突",
		Instruction: prompts.PlotlinesConflictResolverPrompt,
		Model:       chatModel,
	})
}

// ResolvePlotlinesConflicts 解决剧情线冲突（使用 JSON parser 严格校验）
//
// 参数：
//   - ctx: 上下文
//   - agent: 剧情线冲突解决 Agent
//   - input: 输入参数（大纲、目标剧情线、其他剧情线、冲突信息）
//
// 返回值：
//   - *PlotlinesConflictResolverOutput: 修改后的剧情线列表
//   - error: 解决失败时返回错误
func ResolvePlotlinesConflicts(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlinesConflictResolverInput) (*PlotlinesConflictResolverOutput, error) {
	if input.Outline == "" {
		return nil, fmt.Errorf("大纲不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.TargetPlotline == "" {
		return nil, fmt.Errorf("目标剧情线不能为空")
	}
	if input.OtherPlotline == "" {
		return nil, fmt.Errorf("待修改的剧情线不能为空")
	}
	if input.OtherPlotlineID == "" {
		return nil, fmt.Errorf("待修改的剧情线ID不能为空")
	}
	if input.Conflicts == "" {
		return nil, fmt.Errorf("冲突信息不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息修改指定剧情线，解决冲突，并以 JSON 格式输出：

**小说类型**：%s

**修改后的大纲**：
%s

**修改后的目标剧情线**：
%s

**待修改的剧情线**：
%s

**检测到的冲突**：
%s

请严格按照 JSON 格式输出，包含 plotline 对象，包含 id、title、type、description、start_chapter、end_chapter 字段。
- 必须保留剧情线的原有 ID
- 根据冲突信息调整剧情线内容`, input.Genre, input.Outline, input.TargetPlotline, input.OtherPlotline, input.Conflicts)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[PlotlinesConflictResolver] 开始修改其他剧情线以解决冲突...")

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("修改剧情线失败: %w", event.Err)
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
	parser := schema.NewMessageJSONParser[PlotlinesConflictResolverOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if output.Plotline.ID == "" {
		return nil, fmt.Errorf("剧情线的 ID 为空")
	}
	if output.Plotline.ID != input.OtherPlotlineID {
		return nil, fmt.Errorf("输出的剧情线 ID (%s) 与输入不匹配 (%s)", output.Plotline.ID, input.OtherPlotlineID)
	}
	if output.Plotline.Title == "" {
		return nil, fmt.Errorf("剧情线的标题为空")
	}
	if output.Plotline.Type != "main" && output.Plotline.Type != "sub" {
		return nil, fmt.Errorf("剧情线的类型无效: %s（只能是 main 或 sub）", output.Plotline.Type)
	}
	if output.Plotline.Description == "" {
		return nil, fmt.Errorf("剧情线的描述为空")
	}
	if output.Plotline.StartChapter < 0 {
		return nil, fmt.Errorf("剧情线的起始章节无效: %d", output.Plotline.StartChapter)
	}
	if output.Plotline.EndChapter < 0 {
		return nil, fmt.Errorf("剧情线的结束章节无效: %d", output.Plotline.EndChapter)
	}
	if output.Plotline.EndChapter > 0 && output.Plotline.StartChapter > output.Plotline.EndChapter {
		return nil, fmt.Errorf("剧情线的章节范围无效: start=%d, end=%d", output.Plotline.StartChapter, output.Plotline.EndChapter)
	}

	slog.Info(fmt.Sprintf("[PlotlinesConflictResolver] 剧情线 %s 修改完成", output.Plotline.ID))

	return &output, nil
}
