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

// VolumeReviewerInput 是分卷审核 Agent 的输入。
type VolumeReviewerInput struct {
	Outline        string
	Genre          string
	Plotlines      string
	Volumes        string
	TargetVolume   string
	UserSuggestion string
}

// NewVolumeReviewerAgent 创建分卷审核 Agent。
func NewVolumeReviewerAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VolumeReviewerAgent",
		Description: "审查分卷逻辑连续性、节奏和设定冲突的专业 Agent",
		Instruction: prompts.VolumeReviewerPrompt,
		Model:       chatModel,
	})
}

// ReviewVolumes 执行分卷审核。
func ReviewVolumes(ctx context.Context, agent *adk.ChatModelAgent, input *VolumeReviewerInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("分卷审核输入不能为空")
	}
	if strings.TrimSpace(input.Volumes) == "" {
		return "", fmt.Errorf("待审核分卷不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息审核并修正分卷列表，并且只输出 JSON。

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
2. 输出必须仍然是可直接保存的分卷 JSON。
3. 发现逻辑问题时直接在结果中修正，而不是输出问题列表。
4. 不要输出解释文字。`, input.Genre, input.Outline, input.Plotlines, input.Volumes, input.TargetVolume, input.UserSuggestion)

	return runVolumeJSONQuery(ctx, agent, query, "分卷审核")
}
