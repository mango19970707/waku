package compose

import "log/slog"

// logWorkflowInput 记录正文工作流入口参数，避免日志输出大段正文造成噪音。
func logWorkflowInput(state *ContentWorkflowState) {
	slog.Info("正文工作流输入", "workflow", "content", "novel_id", state.NovelID, "volume_id", state.VolumeID, "chapter_id", state.ChapterID, "pending_chapter_ids", state.PendingChapterIDs, "user_suggestion", state.UserSuggestion)
}

// logWorkflowOutput 记录正文工作流输出状态和已保存章节。
func logWorkflowOutput(state *ContentWorkflowState) {
	slog.Info("正文工作流输出", "workflow", "content", "novel_id", state.NovelID, "volume_id", state.VolumeID, "current_node", state.CurrentNode, "saved_chapter_ids", state.SavedChapterIDs, "error_msg", state.ErrorMsg)
}

// logNodeInput 记录节点输入状态，用于定位重试和路由问题。
func logNodeInput(node string, state *ContentWorkflowState) {
	slog.Info("正文工作流节点输入", "workflow", "content", "node", node, "novel_id", state.NovelID, "volume_id", state.VolumeID, "chapter_id", state.ChapterID, "current_node", state.CurrentNode, "retry_count", state.RetryCount)
}

// logNodeOutput 记录节点输出状态，错误场景输出错误内容。
func logNodeOutput(node string, state *ContentWorkflowState, err error) {
	if err != nil {
		slog.Error("正文工作流节点输出", "workflow", "content", "node", node, "novel_id", state.NovelID, "volume_id", state.VolumeID, "chapter_id", state.ChapterID, "current_node", state.CurrentNode, "error", err.Error())
		return
	}
	slog.Info("正文工作流节点输出", "workflow", "content", "node", node, "novel_id", state.NovelID, "volume_id", state.VolumeID, "chapter_id", state.ChapterID, "current_node", state.CurrentNode)
}
