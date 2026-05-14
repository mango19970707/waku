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

// PlotlineExtractorInput 剧情线提取器的输入参数
type PlotlineExtractorInput struct {
	// Outline 小说大纲
	Outline string
	// Genre 小说类型
	Genre string
}

// PlotlineItem 单条剧情线
type PlotlineItem struct {
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

// PlotlineExtractorOutput 剧情线提取器的输出结构
type PlotlineExtractorOutput struct {
	// Plotlines 剧情线列表
	Plotlines []PlotlineItem `json:"plotlines"`
}

// NewPlotlineExtractorAgent 创建剧情线提取 Agent
//
// 功能：从小说大纲中提取多条剧情线（主线和支线）
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 剧情线提取 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewPlotlineExtractorAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ExtractPlotlines(ctx, agent, input)
func NewPlotlineExtractorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlineExtractor",
		Description: "剧情线提取专家，从小说大纲中提取多条剧情线（主线和支线）",
		Instruction: prompts.PlotlineExtractorPrompt,
		Model:       chatModel,
	})
}

// ExtractPlotlines 从大纲中提取多条剧情线（使用 JSON parser 严格校验）
//
// 参数：
//   - ctx: 上下文
//   - agent: 剧情线提取 Agent
//   - input: 输入参数（大纲和类型）
//
// 返回值：
//   - *PlotlineExtractorOutput: 提取的剧情线列表
//   - error: 提取失败时返回错误
func ExtractPlotlines(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlineExtractorInput) (*PlotlineExtractorOutput, error) {
	if input.Outline == "" {
		return nil, fmt.Errorf("大纲不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请从以下小说大纲中提取剧情线（包括主线和支线），并以 JSON 格式输出：

**小说类型**：%s

**小说大纲**：
%s

请严格按照 JSON 格式输出，包含 plotlines 数组，每条剧情线包含 title、type、description、start_chapter、end_chapter 字段。`, input.Genre, input.Outline)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[PlotlineExtractor] 开始从大纲提取剧情线...")

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("提取剧情线失败: %w", event.Err)
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
	parser := schema.NewMessageJSONParser[PlotlineExtractorOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if len(output.Plotlines) == 0 {
		return nil, fmt.Errorf("未提取到任何剧情线")
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

	slog.Info(fmt.Sprintf("[PlotlineExtractor] 剧情线提取完成，共提取 %d 条剧情线", len(output.Plotlines)))

	return &output, nil
}

// ExtractPlotline 从大纲中提取剧情线（兼容旧接口，返回字符串）
//
// 已废弃：请使用 ExtractPlotlines 代替
//
// 参数：
//   - ctx: 上下文
//   - agent: 剧情线提取 Agent
//   - input: 输入参数（大纲和类型）
//
// 返回值：
//   - string: 提取的剧情线（JSON 字符串）
//   - error: 提取失败时返回错误
func ExtractPlotline(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlineExtractorInput) (string, error) {
	output, err := ExtractPlotlines(ctx, agent, input)
	if err != nil {
		return "", err
	}

	// 将结果转换为字符串（用于兼容旧代码）
	result := fmt.Sprintf("提取到 %d 条剧情线：\n\n", len(output.Plotlines))
	for i, plotline := range output.Plotlines {
		result += fmt.Sprintf("## %d. %s（%s）\n\n", i+1, plotline.Title, plotline.Type)
		result += fmt.Sprintf("**章节范围**：第 %d 章 - 第 %d 章\n\n", plotline.StartChapter, plotline.EndChapter)
		result += fmt.Sprintf("**描述**：%s\n\n", plotline.Description)
		result += "---\n\n"
	}

	return result, nil
}
