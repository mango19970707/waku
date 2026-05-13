package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/volume/prompts"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// ChapterPartitionInput 分章 Agent 的输入参数
type ChapterPartitionInput struct {
	// VolumeOutline 卷大纲
	VolumeOutline string
	// Genre 小说类型
	Genre string
	// VolumeNumber 卷号
	VolumeNumber int
	// SuggestedChapterCount 建议的章节数（可选）
	SuggestedChapterCount int
}

// ChapterPartitionOutput 分章 Agent 的输出结构
type ChapterPartitionOutput struct {
	Chapters []ChapterInfo `json:"chapters"`
}

// ChapterInfo 单个章节的信息
type ChapterInfo struct {
	ChapterNumber int    `json:"chapter_number"`
	Title         string `json:"title"`
	Synopsis      string `json:"synopsis"`
	Outline       string `json:"outline"`
}

// NewChapterPartitionAgent 创建章节划分 Agent
//
// 功能：根据卷大纲，将其划分为若干个章节
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 分章 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewChapterPartitionAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := PartitionChapters(ctx, agent, input)
func NewChapterPartitionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChapterPartition",
		Description: "章节划分专家，将卷大纲划分为合理的章节结构",
		Instruction: prompts.ChapterPartitionPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("创建分章 Agent 失败: %w", err)
	}

	return agent, nil
}

// PartitionChapters 执行分章操作
//
// 参数：
//   - ctx: 上下文
//   - agent: 分章 Agent
//   - input: 输入参数（卷大纲、类型、卷号、建议章节数）
//
// 返回值：
//   - string: 分章结果（文本格式）
//   - error: 分章失败时返回错误
func PartitionChapters(ctx context.Context, agent *adk.ChatModelAgent, input *ChapterPartitionInput) (string, error) {
	if input.VolumeOutline == "" {
		return "", fmt.Errorf("卷大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}
	if input.VolumeNumber <= 0 {
		return "", fmt.Errorf("卷号必须大于 0")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息进行章节划分：

**小说类型**：%s

**卷号**：第 %d 卷

**卷大纲**：
%s`, input.Genre, input.VolumeNumber, input.VolumeOutline)

	// 如果提供了建议章节数，添加到查询中
	if input.SuggestedChapterCount > 0 {
		query += fmt.Sprintf(`

**建议章节数**：%d 章`, input.SuggestedChapterCount)
	}

	query += `

请将卷大纲划分为合理的章节，并以 JSON 格式输出。`

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info(fmt.Sprintf("[ChapterPartition] 开始为第 %d 卷划分章节...", input.VolumeNumber))

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("分章失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				// 打印流式输出的内容
				if msg.Content != "" {
					slog.Info(msg.Content)
					result += msg.Content
				}
			}
		}
	}

	slog.Info(fmt.Sprintf("[ChapterPartition] 第 %d 卷章节划分完成", input.VolumeNumber))

	return result, nil
}
