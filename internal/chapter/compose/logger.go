package compose

import (
	"log/slog"
)

// logWorkflowInput 记录工作流输入
func logWorkflowInput(state *ChapterWorkflowState) {
	slog.Info("章节工作流输入",
		"workflow", "chapter",
		"novel_id", state.NovelID,
		"volume_id", state.VolumeID,
		"chapter_id", state.ChapterID,
		"user_suggestion", state.UserSuggestion,
		"revision_mode", state.RevisionMode,
	)
}

// logWorkflowOutput 记录工作流输出
func logWorkflowOutput(state *ChapterWorkflowState) {
	slog.Info("章节工作流输出",
		"workflow", "chapter",
		"novel_id", state.NovelID,
		"volume_id", state.VolumeID,
		"current_node", state.CurrentNode,
		"saved_chapter_ids", state.SavedChapterIDs,
		"error_msg", state.ErrorMsg,
	)
}

// logNodeInput 记录节点输入
func logNodeInput(node string, state *ChapterWorkflowState) {
	slog.Info("章节工作流节点输入",
		"workflow", "chapter",
		"node", node,
		"novel_id", state.NovelID,
		"volume_id", state.VolumeID,
		"chapter_id", state.ChapterID,
		"current_node", state.CurrentNode,
		"retry_count", state.RetryCount,
	)
}

// logNodeOutput 记录节点输出
func logNodeOutput(node string, state *ChapterWorkflowState, err error) {
	if err != nil {
		slog.Error("章节工作流节点输出",
			"workflow", "chapter",
			"node", node,
			"novel_id", state.NovelID,
			"volume_id", state.VolumeID,
			"current_node", state.CurrentNode,
			"error", err.Error(),
		)
	} else {
		slog.Info("章节工作流节点输出",
			"workflow", "chapter",
			"node", node,
			"novel_id", state.NovelID,
			"volume_id", state.VolumeID,
			"current_node", state.CurrentNode,
		)
	}
}
