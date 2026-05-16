package agents

import (
	"context"
	"fmt"
	"strings"

	"waku/internal/utils"
	"waku/internal/volume/prompts"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// VolumePolisherInput 是分卷润色 Agent 的输入。
type VolumePolisherInput struct {
	Outline        string
	Genre          string
	Plotlines      string
	Volumes        string
	TargetVolume   string
	UserSuggestion string
}

// NewVolumePolisherAgent 创建分卷润色 Agent。
func NewVolumePolisherAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VolumePolisherAgent",
		Description: "优化分卷标题、简介、主题和结构表达的专业 Agent",
		Instruction: prompts.VolumePolisherPrompt,
		Model:       chatModel,
	})
}

// PolishVolumes 执行分卷润色。
func PolishVolumes(ctx context.Context, agent *adk.ChatModelAgent, input *VolumePolisherInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("分卷润色输入不能为空")
	}
	if strings.TrimSpace(input.Volumes) == "" {
		return "", fmt.Errorf("待润色分卷不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息润色分卷列表，并且只输出 JSON。

小说类型：
%s

小说大纲：
%s

全部剧情线：
%s

当前分卷列表：
%s

目标分卷：
%s

用户建议：
%s

要求：
1. 顶层字段只能是 volumes。
2. 保持原有卷序和已有 id。
3. 优先优化标题、简介、主题和 outline 的表达。
4. 不要输出解释文字。`, input.Genre, input.Outline, input.Plotlines, input.Volumes, input.TargetVolume, input.UserSuggestion)

	return runVolumeJSONQuery(ctx, agent, query, "分卷润色")
}
