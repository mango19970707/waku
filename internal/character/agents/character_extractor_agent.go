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

// CharacterExtractorInput 人物提取 Agent 的输入参数
type CharacterExtractorInput struct {
	Outline string // 小说大纲
	Genre   string // 小说类型
}

// CharacterItem 单个人物信息
type CharacterItem struct {
	Name          string            `json:"name"`                  // 人物姓名
	Role          string            `json:"role"`                  // 角色定位（protagonist/supporting/antagonist/mentor/love_interest）
	Age           int               `json:"age,omitempty"`         // 年龄（选填）
	Gender        string            `json:"gender,omitempty"`      // 性别（选填）
	Appearance    string            `json:"appearance"`            // 外貌描述
	Personality   string            `json:"personality"`           // 性格特征
	Background    string            `json:"background"`            // 背景故事
	Relationships map[string]string `json:"relationships"`         // 角色关系映射，格式：{"角色ID": "和当前角色的关系描述"}
}

// CharacterExtractorOutput 人物提取 Agent 的输出结构
type CharacterExtractorOutput struct {
	Characters []CharacterItem `json:"characters"`
}

// NewCharacterExtractorAgent 创建人物提取 Agent
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型实例
//
// 返回：
//   - *adk.ChatModelAgent: Agent 实例
//   - error: 错误信息
func NewCharacterExtractorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterExtractorAgent",
		Description: "从小说大纲中提取人物信息的专业 Agent",
		Instruction: prompts.CharacterExtractorPrompt,
		Model:       chatModel,
	})
}

// ExtractCharacters 提取人物信息（返回结构化 JSON）
//
// 参数：
//   - ctx: 上下文对象
//   - agent: Agent 实例
//   - input: 输入参数
//
// 返回：
//   - *CharacterExtractorOutput: 提取的人物信息
//   - error: 错误信息
func ExtractCharacters(ctx context.Context, agent *adk.ChatModelAgent, input *CharacterExtractorInput) (*CharacterExtractorOutput, error) {
	// 构建用户查询
	userQuery := fmt.Sprintf(`请从以下小说大纲中提取所有重要人物的详细信息。

小说类型：%s

小说大纲：
%s

请严格按照 JSON 格式输出人物信息。`, input.Genre, input.Outline)

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
	parser := schema.NewMessageJSONParser[CharacterExtractorOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if len(output.Characters) == 0 {
		return nil, fmt.Errorf("未提取到任何人物信息")
	}

	// 验证每个人物的必填字段
	for i, char := range output.Characters {
		if char.Name == "" {
			return nil, fmt.Errorf("第 %d 个人物缺少姓名", i+1)
		}
		if char.Role == "" {
			return nil, fmt.Errorf("人物 %s 缺少角色定位", char.Name)
		}
		if char.Appearance == "" {
			return nil, fmt.Errorf("人物 %s 缺少外貌描述", char.Name)
		}
		if char.Personality == "" {
			return nil, fmt.Errorf("人物 %s 缺少性格特征", char.Name)
		}
		if char.Background == "" {
			return nil, fmt.Errorf("人物 %s 缺少背景故事", char.Name)
		}
	}

	return &output, nil
}

// ExtractCharacter 提取人物信息（返回字符串，兼容旧接口）
//
// 参数：
//   - ctx: 上下文对象
//   - agent: Agent 实例
//   - input: 输入参数
//
// 返回：
//   - string: 提取的人物信息（JSON 字符串）
//   - error: 错误信息
func ExtractCharacter(ctx context.Context, agent *adk.ChatModelAgent, input *CharacterExtractorInput) (string, error) {
	// 构建用户查询
	userQuery := fmt.Sprintf(`请从以下小说大纲中提取所有重要人物的详细信息。

小说类型：%s

小说大纲：
%s

请严格按照 JSON 格式输出人物信息。`, input.Genre, input.Outline)

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	// 执行查询
	iter := runner.Query(ctx, userQuery)

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("查询失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				// 打印流式输出的内容
				if msg.Content != "" {
					fmt.Print(msg.Content)
					result += msg.Content
				}
			}
		}
	}

	if result == "" {
		return "", fmt.Errorf("未收到 Agent 响应")
	}

	return result, nil
}
