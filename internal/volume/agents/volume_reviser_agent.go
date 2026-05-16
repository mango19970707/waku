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

// VolumeReviserInput 是分卷修改 Agent 的输入。
type VolumeReviserInput struct {
	Outline        string
	Genre          string
	Plotlines      string
	Volumes        string
	TargetVolume   string
	UserSuggestion string
}

// NewVolumeReviserAgent 创建分卷修改 Agent。
func NewVolumeReviserAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VolumeReviserAgent",
		Description: "根据审核结果和用户建议输出最终分卷列表的专业 Agent",
		Instruction: prompts.VolumeReviserPrompt,
		Model:       chatModel,
	})
}

// ReviseVolumes 执行分卷修改。
func ReviseVolumes(ctx context.Context, agent *adk.ChatModelAgent, input *VolumeReviserInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("分卷修改输入不能为空")
	}
	if strings.TrimSpace(input.Volumes) == "" {
		return "", fmt.Errorf("待修改分卷不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息修改分卷列表，并且只输出 JSON。

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
2. 保持已存在分卷的 id，新增分卷可以不带 id。
3. 结果必须是最终可保存版本。
4. 不要输出解释文字。`, input.Genre, input.Outline, input.Plotlines, input.Volumes, input.TargetVolume, input.UserSuggestion)

	return runVolumeJSONQuery(ctx, agent, query, "分卷修改")
}
