package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/volume/prompts"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// VolumePartitionInput 分卷 Agent 的输入参数
type VolumePartitionInput struct {
	// NovelOutline 小说大纲
	NovelOutline string
	// Genre 小说类型
	Genre string
	// SuggestedVolumeCount 建议的卷数（可选）
	SuggestedVolumeCount int
}

// VolumePartitionOutput 分卷 Agent 的输出结构
type VolumePartitionOutput struct {
	Volumes []VolumeInfo `json:"volumes"`
}

// VolumeInfo 单个卷的信息
type VolumeInfo struct {
	VolumeNumber int    `json:"volume_number"`
	Title        string `json:"title"`
	Synopsis     string `json:"synopsis"`
	Theme        string `json:"theme"`
	Outline      string `json:"outline"`
}

// NewVolumePartitionAgent 创建小说分卷 Agent
//
// 功能：根据小说大纲，将其划分为若干个卷
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 分卷 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewVolumePartitionAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := PartitionVolumes(ctx, agent, input)
func NewVolumePartitionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VolumePartition",
		Description: "小说分卷专家，将完整大纲划分为合理的卷结构",
		Instruction: prompts.VolumePartitionPrompt,
		Model:       chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("创建分卷 Agent 失败: %w", err)
	}

	return agent, nil
}

// PartitionVolumes 执行分卷操作
//
// 参数：
//   - ctx: 上下文
//   - agent: 分卷 Agent
//   - input: 输入参数（小说大纲、类型、建议卷数）
//
// 返回值：
//   - string: 分卷结果（文本格式）
//   - error: 分卷失败时返回错误
func PartitionVolumes(ctx context.Context, agent *adk.ChatModelAgent, input *VolumePartitionInput) (string, error) {
	if input.NovelOutline == "" {
		return "", fmt.Errorf("小说大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息进行分卷规划：

**小说类型**：%s

**小说大纲**：
%s`, input.Genre, input.NovelOutline)

	// 如果提供了建议卷数，添加到查询中
	if input.SuggestedVolumeCount > 0 {
		query += fmt.Sprintf(`

**建议卷数**：%d 卷`, input.SuggestedVolumeCount)
	}

	query += `

请将大纲划分为合理的卷，并以 JSON 格式输出。`

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[VolumePartition] 开始分卷...")

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("分卷失败: %w", event.Err)
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

	slog.Info("[VolumePartition] 分卷完成")

	return result, nil
}
