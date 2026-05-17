package compose

import (
	"context"
	"log/slog"
)

const outlineWorkflowName = "outline_generate"

func workflowLogger() *slog.Logger {
	return slog.Default().With("workflow", outlineWorkflowName)
}

func logWorkflowInput(ctx context.Context, input *OutlineWorkflowInput) {
	if input == nil {
		workflowLogger().WarnContext(ctx, "workflow.input", "input", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "workflow.input",
		"novel_id", input.NovelID,
		"novel_name", trimForLog(input.NovelName),
		"author", trimForLog(input.Author),
		"genre", trimForLog(input.Genre),
		"original_outline", summarizeText(input.OriginalOutline),
		"user_suggestion", summarizeText(input.UserSuggestion),
	)
}

func logWorkflowOutput(ctx context.Context, state *OutlineWorkflowState, err error) {
	attrs := []any{"error", err}
	if state == nil {
		attrs = append(attrs, "state", "nil")
		workflowLogger().InfoContext(ctx, "workflow.output", attrs...)
		return
	}

	attrs = append(attrs,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"saved_novel_id", state.SavedNovelID,
		"retry_count", state.RetryCount,
		"error_msg", trimForLog(state.ErrorMsg),
	)
	workflowLogger().InfoContext(ctx, "workflow.output", attrs...)
}

func logInitNodeInput(ctx context.Context, input *OutlineWorkflowInput) {
	if input == nil {
		workflowLogger().WarnContext(ctx, "node.input", "node", "init", "input", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "node.input",
		"node", "init",
		"novel_id", input.NovelID,
		"novel_name", trimForLog(input.NovelName),
		"author", trimForLog(input.Author),
		"genre", trimForLog(input.Genre),
		"original_outline", summarizeText(input.OriginalOutline),
		"user_suggestion", summarizeText(input.UserSuggestion),
	)
}

func logStateNodeInput(ctx context.Context, node string, state *OutlineWorkflowState) {
	if state == nil {
		workflowLogger().WarnContext(ctx, "node.input", "node", node, "state", "nil")
		return
	}

	workflowLogger().InfoContext(ctx, "node.input",
		"node", node,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"novel_name", trimForLog(state.NovelName),
		"author", trimForLog(state.Author),
		"genre", trimForLog(state.Genre),
		"original_outline", shortPreview(state.OriginalOutline),
		"user_suggestion", shortPreview(state.UserSuggestion),
		"generated_chars", previewMapChars(state.GeneratedOutline),
		"expanded_chars", previewMapChars(state.ExpandedOutline),
		"polished_chars", previewMapChars(state.PolishedOutline),
		"review_chars", len([]rune(state.ReviewSuggestions)),
		"revised_chars", previewMapChars(state.RevisedOutline),
		"retry_count", state.RetryCount,
	)
}

func logNodeOutput(ctx context.Context, node string, state *OutlineWorkflowState, err error) {
	attrs := []any{"node", node, "error", err}
	if state == nil {
		attrs = append(attrs, "state", "nil")
		workflowLogger().InfoContext(ctx, "node.output", attrs...)
		return
	}

	attrs = append(attrs,
		"current_node", state.CurrentNode,
		"novel_id", state.NovelID,
		"saved_novel_id", state.SavedNovelID,
		"novel_title", trimForLog(state.NovelTitle),
		"synopsis", shortPreview(state.Synopsis),
		"outline", shortPreview(state.Outline),
		"generated_chars", previewMapChars(state.GeneratedOutline),
		"expanded_chars", previewMapChars(state.ExpandedOutline),
		"polished_chars", previewMapChars(state.PolishedOutline),
		"review_chars", len([]rune(state.ReviewSuggestions)),
		"revised_chars", previewMapChars(state.RevisedOutline),
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

func previewMapChars(value map[string]any) int {
	if len(value) == 0 {
		return 0
	}

	raw, err := marshalNovelJSONMap(value)
	if err != nil {
		return 0
	}
	return len([]rune(raw))
}
