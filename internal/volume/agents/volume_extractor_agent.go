package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/volume/prompts"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// VolumeExtractorInput 卷信息提取 Agent 的输入参数
type VolumeExtractorInput struct {
	// PartitionedText 分卷后的文本
	PartitionedText string
	// Genre 小说类型
	Genre string
}

// NewVolumeExtractorAgent 创建卷信息提取 Agent
//
// 功能：从分卷文本中提取结构化的卷信息
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 卷信息提取 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewVolumeExtractorAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ExtractVolumeInfo(ctx, agent, input)
func NewVolumeExtractorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VolumeExtractor",
		Description: "卷信息提取专家，从文本中提取结构化的卷信息",
		Instruction: prompts.VolumeExtractorPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("创建卷信息提取 Agent 失败: %w", err)
	}

	return agent, nil
}

// ExtractVolumeInfo 提取卷信息
//
// 参数：
//   - ctx: 上下文
//   - agent: 卷信息提取 Agent
//   - input: 输入参数（分卷文本、类型）
//
// 返回值：
//   - *VolumePartitionOutput: 提取的结构化卷信息
//   - error: 提取失败时返回错误
func ExtractVolumeInfo(ctx context.Context, agent *adk.ChatModelAgent, input *VolumeExtractorInput) (*VolumePartitionOutput, error) {
	if input.PartitionedText == "" {
		return nil, fmt.Errorf("分卷文本不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请从以下分卷文本中提取结构化信息：

**小说类型**：%s

**分卷文本**：
%s

请以 JSON 格式输出提取的卷信息。`, input.Genre, input.PartitionedText)

	// 创建 Runner 并执行查询（不启用流式输出，因为需要完整的 JSON）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           agent,
	})

	slog.Info("[VolumeExtractor] 开始提取卷信息...")

	// 执行查询
	result, err := runner.Invoke(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("提取卷信息失败: %w", err)
	}

	// 获取最后一条消息
	messages := result.GetMessages()
	if len(messages) == 0 {
		return nil, fmt.Errorf("未获取到响应消息")
	}

	lastMessage := messages[len(messages)-1]

	// 使用 MessageJSONParser 解析 JSON 输出
	parserConfig := &schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	}
	parser := schema.NewMessageJSONParser[VolumePartitionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	slog.Info(fmt.Sprintf("[VolumeExtractor] 提取完成，共 %d 个卷", len(output.Volumes)))

	return output, nil
}

// ExtractChapterInfo 提取章节信息（从分章文本中提取结构化信息）
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - partitionedText: 分章文本
//   - genre: 小说类型
//
// 返回值：
//   - *ChapterPartitionOutput: 提取的结构化章节信息
//   - error: 提取失败时返回错误
func ExtractChapterInfo(ctx context.Context, chatModel model.BaseChatModel, partitionedText, genre string) (*ChapterPartitionOutput, error) {
	if partitionedText == "" {
		return nil, fmt.Errorf("分章文本不能为空")
	}
	if genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}

	// 创建一个简单的提取 Agent（使用类似的提示词）
	extractPrompt := `你是一位专业的信息提取专家，擅长从文本中提取结构化信息。

请从提供的分章文本中，提取出结构化的章节信息，并以 JSON 格式输出。

输出格式：
{
  "chapters": [
    {
      "chapter_number": 1,
      "title": "章节标题",
      "synopsis": "章节简介",
      "outline": "详细的章节大纲"
    }
  ]
}

注意：
1. 必须输出有效的 JSON 格式
2. 所有字段都不能为空
3. 不要添加任何额外的文字说明`

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChapterExtractor",
		Description: "章节信息提取专家",
		Instruction: extractPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("创建章节提取 Agent 失败: %w", err)
	}

	// 构建查询消息
	query := fmt.Sprintf(`请从以下分章文本中提取结构化信息：

**小说类型**：%s

**分章文本**：
%s

请以 JSON 格式输出提取的章节信息。`, genre, partitionedText)

	// 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           agent,
	})

	slog.Info("[ChapterExtractor] 开始提取章节信息...")

	// 执行查询
	result, err := runner.Invoke(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("提取章节信息失败: %w", err)
	}

	// 获取最后一条消息
	messages := result.GetMessages()
	if len(messages) == 0 {
		return nil, fmt.Errorf("未获取到响应消息")
	}

	lastMessage := messages[len(messages)-1]

	// 使用 MessageJSONParser 解析 JSON 输出
	parserConfig := &schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	}
	parser := schema.NewMessageJSONParser[ChapterPartitionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	slog.Info(fmt.Sprintf("[ChapterExtractor] 提取完成，共 %d 个章节", len(output.Chapters)))

	return output, nil
}
