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

// CharacterRevisionInput 人物优化 Agent 的输入参数
type CharacterRevisionInput struct {
	Outline            string // 小说大纲
	Genre              string // 小说类型
	ExistingCharacters string // 现有人物信息（JSON 格式）
	UserSuggestion     string // 用户建议
}

// CharacterRevisionItem 单个人物信息（包含 ID）
type CharacterRevisionItem struct {
	ID            string            `json:"id"`                    // 人物ID（修改时保留，新增时为空）
	Name          string            `json:"name"`                  // 人物姓名
	Role          string            `json:"role"`                  // 角色定位
	Age           int               `json:"age,omitempty"`         // 年龄（选填）
	Gender        string            `json:"gender,omitempty"`      // 性别（选填）
	Appearance    string            `json:"appearance"`            // 外貌描述
	Personality   string            `json:"personality"`           // 性格特征
	Background    string            `json:"background"`            // 背景故事
	Relationships map[string]string `json:"relationships"`         // 角色关系映射，格式：{"角色ID": "和当前角色的关系描述"}
}

// CharacterRevisionOutput 人物优化 Agent 的输出结构
type CharacterRevisionOutput struct {
	Characters []CharacterRevisionItem `json:"characters"`
}

// NewCharacterRevisionAgent 创建人物优化 Agent
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型实例
//
// 返回：
//   - *adk.ChatModelAgent: Agent 实例
//   - error: 错误信息
func NewCharacterRevisionAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterRevisionAgent",
		Description: "根据用户建议优化人物信息的专业 Agent",
		Instruction: prompts.CharacterRevisionPrompt,
		Model:       chatModel,
	})
}

// ReviseCharacters 优化人物信息（返回结构化 JSON）
//
// 参数：
//   - ctx: 上下文对象
//   - agent: Agent 实例
//   - input: 输入参数
//
// 返回：
//   - *CharacterRevisionOutput: 优化后的人物信息
//   - error: 错误信息
func ReviseCharacters(ctx context.Context, agent *adk.ChatModelAgent, input *CharacterRevisionInput) (*CharacterRevisionOutput, error) {
	// 构建用户查询
	userQuery := fmt.Sprintf(`请根据用户建议优化以下人物信息。

小说类型：%s

小说大纲：
%s

现有人物信息：
%s

用户建议：
%s

请严格按照 JSON 格式输出优化后的人物信息。注意：
1. 修改现有人物时，保留原有的 ID
2. 新增人物时，ID 设置为空字符串 ""
3. 删除人物时，直接在输出中省略该人物`, input.Genre, input.Outline, input.ExistingCharacters, input.UserSuggestion)

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
	parser := schema.NewMessageJSONParser[CharacterRevisionOutput](parserConfig)

	output, err := parser.Parse(ctx, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 输出失败: %w", err)
	}

	// 验证输出
	if len(output.Characters) == 0 {
		return nil, fmt.Errorf("优化后的人物列表为空")
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
