package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/novel/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
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
	NovelTitle string
	// Synopsis 小说简介
	Synopsis string
	// Outline 完整的大纲内容
	Outline string
	// Worldview 世界观设定
	Worldview string
	// CultivationSystem 修行体系
	CultivationSystem string
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

请提取以下信息并按照指定格式输出。`, input.Genre,
		func() string {
			if input.NovelName != "" {
				return fmt.Sprintf("\n**小说名称**：%s（如果大纲中有标题，优先使用大纲中的标题）", input.NovelName)
			}
			return ""
		}(),
		input.Outline)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlineExtractor] 开始提取大纲信息...")

	// 收集响应内容
	var result string
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
			}
		}
	}

	slog.Info("[OutlineExtractor] 大纲信息提取完成")

	// 解析提取的信息
	output, err := parseExtractedInfo(result)
	if err != nil {
		return nil, fmt.Errorf("解析提取信息失败: %w", err)
	}

	return output, nil
}

// parseExtractedInfo 解析 AI 提取的信息
func parseExtractedInfo(content string) (*OutlineExtractorOutput, error) {
	output := &OutlineExtractorOutput{}

	// 使用简单的标记解析
	// 期望格式：
	// 【小说标题】xxx
	// 【小说简介】xxx
	// 【完整大纲】xxx
	// 【世界观设定】xxx
	// 【修行体系】xxx

	output.NovelTitle = extractSection(content, "【小说标题】", "【")
	output.Synopsis = extractSection(content, "【小说简介】", "【")
	output.Outline = extractSection(content, "【完整大纲】", "【")
	output.Worldview = extractSection(content, "【世界观设定】", "【")
	output.CultivationSystem = extractSection(content, "【修行体系】", "【")

	// 验证必填字段
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

	return output, nil
}

// extractSection 从内容中提取指定标记之间的文本
func extractSection(content, startMarker, endMarker string) string {
	startIdx := findIndex(content, startMarker)
	if startIdx == -1 {
		return ""
	}

	// 跳过起始标记
	startIdx += len(startMarker)

	// 查找下一个标记
	endIdx := findIndex(content[startIdx:], endMarker)
	if endIdx == -1 {
		// 如果没有找到结束标记，提取到字符串末尾
		return trimSpace(content[startIdx:])
	}

	return trimSpace(content[startIdx : startIdx+endIdx])
}

// findIndex 查找子串的位置（不区分大小写）
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// trimSpace 去除字符串首尾空白字符
func trimSpace(s string) string {
	// 去除前导空白
	start := 0
	for start < len(s) && isSpace(s[start]) {
		start++
	}

	// 去除尾随空白
	end := len(s)
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isSpace 判断字符是否为空白字符
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
