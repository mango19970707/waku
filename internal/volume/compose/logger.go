package compose

import (
	"context"
	"log/slog"
)

const volumeWorkflowName = "volume"

func workflowLogger() *slog.Logger {
	return slog.Default().With("workflow", volumeWorkflowName)
}

func logWorkflowInput(ctx context.Context, input *VolumeWorkflowInput) {
	if input == nil {
		workflowLogger().WarnContext(ctx, "workflow.input", "input", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "workflow.input",
		"novel_id", input.NovelID,
		"volume_id", input.VolumeID,
		"user_suggestion", summarizeText(input.UserSuggestion),
	)
}

func logWorkflowOutput(ctx context.Context, state *VolumeWorkflowState, err error) {
	attrs := []any{"error", err}
	if state == nil {
		attrs = append(attrs, "state", "nil")
		workflowLogger().InfoContext(ctx, "workflow.output", attrs...)
		return
	}

	attrs = append(attrs,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"volume_id", state.VolumeID,
		"saved_volume_ids", state.SavedVolumeIDs,
		"retry_count", state.RetryCount,
		"error_msg", trimForLog(state.ErrorMsg),
	)
	workflowLogger().InfoContext(ctx, "workflow.output", attrs...)
}

func logInitNodeInput(ctx context.Context, input *VolumeWorkflowInput) {
	if input == nil {
		workflowLogger().WarnContext(ctx, "node.input", "node", "init", "input", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "node.input",
		"node", "init",
		"novel_id", input.NovelID,
		"volume_id", input.VolumeID,
		"user_suggestion", summarizeText(input.UserSuggestion),
	)
}

func logStateNodeInput(ctx context.Context, node string, state *VolumeWorkflowState) {
	if state == nil {
		workflowLogger().WarnContext(ctx, "node.input", "node", node, "state", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "node.input",
		"node", node,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"volume_id", state.VolumeID,
		"novel_name", trimForLog(state.NovelName),
		"author", trimForLog(state.Author),
		"genre", trimForLog(state.Genre),
		"outline", shortPreview(state.Outline),
		"existing_plotlines", shortPreview(state.ExistingPlotlines),
		"existing_volumes", shortPreview(state.ExistingVolumes),
		"target_volume_json", shortPreview(state.TargetVolumeJSON),
		"user_suggestion", shortPreview(state.UserSuggestion),
		"revision_mode", state.RevisionMode,
		"generated_chars", len([]rune(state.GeneratedVolumes)),
		"polished_chars", len([]rune(state.PolishedVolumes)),
		"reviewed_chars", len([]rune(state.ReviewedVolumes)),
		"revised_chars", len([]rune(state.RevisedVolumes)),
		"retry_count", state.RetryCount,
	)
}

func logNodeOutput(ctx context.Context, node string, state *VolumeWorkflowState, err error) {
	attrs := []any{"node", node, "error", err}
	if state == nil {
		attrs = append(attrs, "state", "nil")
		workflowLogger().InfoContext(ctx, "node.output", attrs...)
		return
	}

	attrs = append(attrs,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"volume_id", state.VolumeID,
		"novel_name", trimForLog(state.NovelName),
		"outline", shortPreview(state.Outline),
		"existing_plotlines", shortPreview(state.ExistingPlotlines),
		"existing_volumes", shortPreview(state.ExistingVolumes),
		"generated_chars", len([]rune(state.GeneratedVolumes)),
		"polished_chars", len([]rune(state.PolishedVolumes)),
		"reviewed_chars", len([]rune(state.ReviewedVolumes)),
		"revised_chars", len([]rune(state.RevisedVolumes)),
		"saved_volume_ids", state.SavedVolumeIDs,
		"retry_count", state.RetryCount,
		"error_msg", trimForLog(state.ErrorMsg),
	)
	workflowLogger().InfoContext(ctx, "node.output", attrs...)
}

func shortPreview(text string) string {
	const maxRunes = 48

	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}

	return string(runes[:maxRunes]) + "..."
}

func summarizeText(text string) string {
	const maxRunes = 120

	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}

	return string(runes[:maxRunes]) + "..."
}

func trimForLog(text string) string {
	return summarizeText(text)
}
