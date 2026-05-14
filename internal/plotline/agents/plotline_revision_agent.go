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

// PlotlineRevisionInput 剧情线优化器的输入参数
type PlotlineRevisionInput struct {
	// Outline 小说大纲
	Outline string
	// Genre 小说类型
	Genre string
	// ExistingPlotlines 现有剧情线列表（JSON 格式）
	ExistingPlotlines string
	// UserSuggestion 用户的修改建议
	UserSuggestion string
}

// PlotlineRevisionItem 单条优化后的剧情线
type PlotlineRevisionItem struct {
	// ID 剧情线ID（修改现有剧情线时保留原ID，新增时为空）
	ID string `json:"id"`
	// Title 剧情线标题
	Title string `json:"title"`
	// Type 剧情线类型（main-主线、sub-支线）
	Type string `json:"type"`
	// Description 剧情线描述
	Description string `json:"description"`
	// StartChapter 起始章节序号
	StartChapter int `json:"start_chapter"`
	// EndChapter 结束章节序号（0 表示未结束）
	EndChapter int `json:"end_chapter"`
}

// PlotlineRevisionOutput 剧情线优化器的输出结构
type PlotlineRevisionOutput struct {
	// Plotlines 优化后的剧情线列表
	Plotlines []PlotlineRevisionItem `json:"plotlines"`
}

// NewPlotlineRevisionAgent 创建剧情线优化 Agent
//
// 功能：根据用户建议、大纲和现有剧情线，优化剧情线（可以修改、删除、新增）
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 剧情线优化 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewPlotlineRevisionAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := RevisePlotlines(ctx, agent, input)
func NewPlotlineRevisionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlineRevision",
		Description: "剧情线优化专家，根据用户建议优化剧情线（可以修改、删除、新增）",
		Instruction: prompts.PlotlineRevisionPrompt,
		Model:       chatModel,
	})
}

// RevisePlotlines 优化剧情线（使用 JSON parser 严格校验）
//
// 参数：
//   - ctx: 上下文
//   - agent: 剧情线优化 Agent
//   - input: 输入参数（大纲、现有剧情线、用户建议）
//
// 返回值：
//   - *PlotlineRevisionOutput: 优化后的剧情线列表
//   - error: 优化失败时返回错误
func RevisePlotlines(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlineRevisionInput) (*PlotlineRevisionOutput, error) {
	if input.Outline == "" {
		return nil, fmt.Errorf("大纲不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.ExistingPlotlines == "" {
		return nil, fmt.Errorf("现有剧情线不能为空")
	}
	if input.UserSuggestion == "" {
		return nil, fmt.Errorf("用户建议不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据用户建议优化以下剧情线，并以 JSON 格式输出：

**小说类型**：%s

**小说大纲**：
%s

**现有剧情线**：
%s

**用户修改建议**：
%s

请严格按照 JSON 格式输出，包含 plotlines 数组，每条剧情线包含 id、title、type、description、start_chapter、end_chapter 字段。
- 修改现有剧情线时，必须保留原有的 id
- 新增剧情线时，id 字段留空（""）
- 删除剧情线时，直接在输出中省略该剧情线`, input.Genre, input.Outline, input.ExistingPlotlines, input.UserSuggestion)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[PlotlineRevision] 开始优化剧情线...")

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("优化剧情线失败: %w", event.Err)
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
	parser := schema.NewMessageJSONParser[PlotlineRevisionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if len(output.Plotlines) == 0 {
		return nil, fmt.Errorf("未输出任何剧情线")
	}

	// 验证每条剧情线的字段
	for i, plotline := range output.Plotlines {
		if plotline.Title == "" {
			return nil, fmt.Errorf("第 %d 条剧情线的标题为空", i+1)
		}
		if plotline.Type != "main" && plotline.Type != "sub" {
			return nil, fmt.Errorf("第 %d 条剧情线的类型无效: %s（只能是 main 或 sub）", i+1, plotline.Type)
		}
		if plotline.Description == "" {
			return nil, fmt.Errorf("第 %d 条剧情线的描述为空", i+1)
		}
		if plotline.StartChapter < 0 {
			return nil, fmt.Errorf("第 %d 条剧情线的起始章节无效: %d", i+1, plotline.StartChapter)
		}
		if plotline.EndChapter < 0 {
			return nil, fmt.Errorf("第 %d 条剧情线的结束章节无效: %d", i+1, plotline.EndChapter)
		}
		if plotline.EndChapter > 0 && plotline.StartChapter > plotline.EndChapter {
			return nil, fmt.Errorf("第 %d 条剧情线的章节范围无效: start=%d, end=%d", i+1, plotline.StartChapter, plotline.EndChapter)
		}
	}

	slog.Info(fmt.Sprintf("[PlotlineRevision] 剧情线优化完成，共输出 %d 条剧情线", len(output.Plotlines)))

	return &output, nil
}
