package compose

import (
	"context"
	"log/slog"
)

const plotlineWorkflowName = "plotline"

func workflowLogger() *slog.Logger {
	return slog.Default().With("workflow", plotlineWorkflowName)
}

func logWorkflowInput(ctx context.Context, input *PlotlineWorkflowInput) {
	if input == nil {
		workflowLogger().WarnContext(ctx, "workflow.input", "input", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "workflow.input",
		"novel_id", input.NovelID,
		"plotline_id", input.PlotlineID,
		"user_suggestion", summarizeText(input.UserSuggestion),
	)
}

func logWorkflowOutput(ctx context.Context, state *PlotlineWorkflowState, err error) {
	attrs := []any{"error", err}
	if state == nil {
		attrs = append(attrs, "state", "nil")
		workflowLogger().InfoContext(ctx, "workflow.output", attrs...)
		return
	}

	attrs = append(attrs,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"plotline_id", state.PlotlineID,
		"saved_plotline_ids", state.SavedPlotlineIDs,
		"retry_count", state.RetryCount,
		"error_msg", trimForLog(state.ErrorMsg),
	)
	workflowLogger().InfoContext(ctx, "workflow.output", attrs...)
}

func logInitNodeInput(ctx context.Context, input *PlotlineWorkflowInput) {
	if input == nil {
		workflowLogger().WarnContext(ctx, "node.input", "node", "init", "input", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "node.input",
		"node", "init",
		"novel_id", input.NovelID,
		"plotline_id", input.PlotlineID,
		"user_suggestion", summarizeText(input.UserSuggestion),
	)
}

func logStateNodeInput(ctx context.Context, node string, state *PlotlineWorkflowState) {
	if state == nil {
		workflowLogger().WarnContext(ctx, "node.input", "node", node, "state", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "node.input",
		"node", node,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"plotline_id", state.PlotlineID,
		"novel_name", trimForLog(state.NovelName),
		"author", trimForLog(state.Author),
		"genre", trimForLog(state.Genre),
		"outline", shortPreview(state.Outline),
		"user_suggestion", shortPreview(state.UserSuggestion),
		"existing_chars", len([]rune(state.ExistingPlotlines)),
		"generated_chars", len([]rune(state.GeneratedPlotlines)),
		"polished_chars", len([]rune(state.PolishedPlotlines)),
		"reviewed_chars", len([]rune(state.ReviewedPlotlines)),
		"revised_chars", len([]rune(state.RevisedPlotlines)),
		"saved_plotline_ids", state.SavedPlotlineIDs,
		"retry_count", state.RetryCount,
	)
}

func logNodeOutput(ctx context.Context, node string, state *PlotlineWorkflowState, err error) {
	attrs := []any{"node", node, "error", err}
	if state == nil {
		attrs = append(attrs, "state", "nil")
		workflowLogger().InfoContext(ctx, "node.output", attrs...)
		return
	}

	attrs = append(attrs,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"plotline_id", state.PlotlineID,
		"novel_name", trimForLog(state.NovelName),
		"outline", shortPreview(state.Outline),
		"existing_chars", len([]rune(state.ExistingPlotlines)),
		"generated_chars", len([]rune(state.GeneratedPlotlines)),
		"polished_chars", len([]rune(state.PolishedPlotlines)),
		"reviewed_chars", len([]rune(state.ReviewedPlotlines)),
		"revised_chars", len([]rune(state.RevisedPlotlines)),
		"saved_plotline_ids", state.SavedPlotlineIDs,
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
