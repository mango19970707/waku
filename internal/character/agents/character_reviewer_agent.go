package agents

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"waku/internal/character/prompts"
	"waku/internal/utils"
)

// CharacterReviewerInput 人物审核 Agent 的输入参数
type CharacterReviewerInput struct {
	Characters string // 待审核的人物信息（JSON 格式）
	Genre      string // 小说类型
	Outline    string // 小说大纲
}

// CharacterIssue 单个人物问题
type CharacterIssue struct {
	CharacterName string `json:"character_name"` // 人物姓名
	IssueType     string `json:"issue_type"`     // 问题类型
	Severity      string `json:"severity"`       // 严重程度（high/medium/low）
	Description   string `json:"description"`    // 问题描述
	Suggestion    string `json:"suggestion"`     // 修改建议
}

// CharacterReviewerOutput 人物审核 Agent 的输出结构
type CharacterReviewerOutput struct {
	OverallAssessment string           `json:"overall_assessment"` // 整体评价
	HasIssues         bool             `json:"has_issues"`         // 是否存在问题
	Issues            []CharacterIssue `json:"issues"`             // 问题列表
}

// NewCharacterReviewerAgent 创建人物审核 Agent
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型实例
//
// 返回：
//   - *adk.ChatModelAgent: Agent 实例
//   - error: 错误信息
func NewCharacterReviewerAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterReviewerAgent",
		Description: "审核人物设定合理性的专业 Agent",
		Instruction: prompts.CharacterReviewerPrompt,
		Model:       chatModel,
	})
}

// ReviewCharacters 审核人物信息（返回结构化 JSON）
//
// 参数：
//   - ctx: 上下文对象
//   - agent: Agent 实例
//   - input: 输入参数
//
// 返回：
//   - *CharacterReviewerOutput: 审核结果
//   - error: 错误信息
func ReviewCharacters(ctx context.Context, agent *adk.ChatModelAgent, input *CharacterReviewerInput) (*CharacterReviewerOutput, error) {
	// 构建用户查询
	userQuery := fmt.Sprintf(`请审核以下人物信息，识别不合理之处并给出修改建议。

小说类型：%s

小说大纲：
%s

人物信息：
%s

请严格按照 JSON 格式输出审核结果。`, input.Genre, input.Outline, input.Characters)

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	// 执行查询
	iter := runner.Query(ctx, userQuery)

	// 收集响应内容
	var lastMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return nil, fmt.Errorf("查询失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				// 打印流式输出的内容
				if msg.Content != "" {
					fmt.Print(msg.Content)
				}
				lastMessage = msg
			}
		}
	}

	if lastMessage == nil {
		return nil, fmt.Errorf("未收到 Agent 响应")
	}

	// 使用 MessageJSONParser 解析 JSON 输出
	parserConfig := &schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromContent,
	}
	parser := schema.NewMessageJSONParser[CharacterReviewerOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if output.OverallAssessment == "" {
		return nil, fmt.Errorf("缺少整体评价")
	}

	// 验证每个问题的必填字段
	for i, issue := range output.Issues {
		if issue.CharacterName == "" {
			return nil, fmt.Errorf("第 %d 个问题缺少人物姓名", i+1)
		}
		if issue.IssueType == "" {
			return nil, fmt.Errorf("第 %d 个问题缺少问题类型", i+1)
		}
		if issue.Severity == "" {
			return nil, fmt.Errorf("第 %d 个问题缺少严重程度", i+1)
		}
		if issue.Description == "" {
			return nil, fmt.Errorf("第 %d 个问题缺少问题描述", i+1)
		}
		if issue.Suggestion == "" {
			return nil, fmt.Errorf("第 %d 个问题缺少修改建议", i+1)
		}
	}

	return &output, nil
}
