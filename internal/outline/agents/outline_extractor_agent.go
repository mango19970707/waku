package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// OutlineExtractorInput 大纲信息提取器的输入参数
type OutlineExtractorInput struct {
	// Outline 需要提取信息的大纲内容
	Outline string
	// Genre 小说类型
	Genre string
	// NovelName 小说名称（可选，如果提供则优先使用）
	NovelName string
}

// OutlineExtractorOutput 大纲信息提取器的输出结果
type OutlineExtractorOutput struct {
	// NovelTitle 小说标题
	NovelTitle string `json:"novel_title"`
	// Synopsis 小说简介
	Synopsis string `json:"synopsis"`
	// Outline 完整的大纲内容
	Outline string `json:"outline"`
	// Worldview 世界观设定
	Worldview string `json:"worldview"`
	// CultivationSystem 修行体系
	CultivationSystem string `json:"cultivation_system"`
}

// NewOutlineExtractorAgent 创建小说大纲信息提取 Agent
//
// 功能：从润色后的大纲中准确提取结构化信息，用于保存到数据库
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲信息提取 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlineExtractorAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ExtractOutlineInfo(ctx, agent, input)
func NewOutlineExtractorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineExtractor",
		Description: "小说大纲信息提取专家，从大纲中准确提取结构化信息",
		Instruction: prompts.OutlineExtractorPrompt,
		Model:       chatModel,
	})
}

// ExtractOutlineInfo 从大纲中提取结构化信息
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲信息提取 Agent
//   - input: 输入参数（大纲内容和小说类型）
//
// 返回值：
//   - *OutlineExtractorOutput: 提取的结构化信息
//   - error: 提取失败时返回错误
func ExtractOutlineInfo(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineExtractorInput) (*OutlineExtractorOutput, error) {
	if input.Outline == "" {
		return nil, fmt.Errorf("大纲内容不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请从以下小说大纲中提取结构化信息：

**小说类型**：%s
%s
**大纲内容**：
%s

请提取以下信息并按照 JSON 格式输出。`, input.Genre,
		func() string {
			if input.NovelName != "" {
				return fmt.Sprintf("\n**小说名称**：%s（如果大纲中有标题，优先使用大纲中的标题）", input.NovelName)
			}
			return ""
		}(),
		input.Outline)

	// 创建 JSON 解析器
	parserConfig := &schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	}
	parser := schema.NewMessageJSONParser[OutlineExtractorOutput](parserConfig)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlineExtractor] 开始提取大纲信息...")

	// 收集响应内容
	var result string
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("提取大纲信息失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				// 打印流式输出的内容
				if msg.Content != "" {
					slog.Info(msg.Content)
					result += msg.Content
				}
				lastMessage = msg
			}
		}
	}

	slog.Info("[OutlineExtractor] 大纲信息提取完成")

	// 使用 JSON 解析器解析输出
	if lastMessage == nil {
		lastMessage = &schema.Message{
			Content: result,
		}
	} else {
		// 确保 lastMessage 包含完整内容
		lastMessage.Content = result
	}

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w\n原始输出: %s", err, result)
	}

	// 验证必填字段（提供默认值）
	if output.NovelTitle == "" {
		output.NovelTitle = "未命名小说"
	}
	if output.Synopsis == "" {
		output.Synopsis = "暂无简介"
	}
	if output.Outline == "" {
		output.Outline = "暂无完整大纲"
	}
	if output.Worldview == "" {
		output.Worldview = "暂无详细世界观设定"
	}
	if output.CultivationSystem == "" {
		output.CultivationSystem = "暂无详细修行体系设定"
	}

	return &output, nil
}
