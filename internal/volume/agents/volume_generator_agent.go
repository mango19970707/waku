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

// VolumeGeneratorInput 是分卷生成 Agent 的输入。
type VolumeGeneratorInput struct {
	Outline         string
	Genre           string
	Plotlines       string
	ExistingVolumes string
}

// NewVolumeGeneratorAgent 创建分卷生成 Agent。
func NewVolumeGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VolumeGeneratorAgent",
		Description: "基于小说大纲和全部剧情线生成初始分卷列表的专业 Agent",
		Instruction: prompts.VolumeGeneratorPrompt,
		Model:       chatModel,
	})
}

// GenerateVolumes 执行分卷生成。
func GenerateVolumes(ctx context.Context, agent *adk.ChatModelAgent, input *VolumeGeneratorInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("分卷生成输入不能为空")
	}
	if strings.TrimSpace(input.Outline) == "" {
		return "", fmt.Errorf("小说大纲不能为空")
	}
	if strings.TrimSpace(input.Genre) == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息生成初始分卷列表，并且只输出 JSON。

小说类型：
%s

小说大纲：
%s

全部剧情线：
%s

已有分卷上下文：
%s

要求：
1. 顶层字段只能是 volumes。
2. 每个分卷必须包含 volume_number、title、outline。
3. 生成结果要体现完整的阶段推进和卷间连续性。
4. 不要输出解释文字。`, input.Genre, input.Outline, input.Plotlines, input.ExistingVolumes)

	return runVolumeJSONQuery(ctx, agent, query, "分卷生成")
}
