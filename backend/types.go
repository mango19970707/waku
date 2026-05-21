package backend

import "context"

type StepKey string

const (
	StepOutline    StepKey = "outline"
	StepPlotlines  StepKey = "plotlines"
	StepCharacters StepKey = "characters"
	StepVolumes    StepKey = "volumes"
	StepChapters   StepKey = "chapters"
	StepContent    StepKey = "content"
)

type WorkflowRunRequest struct {
	Step            StepKey `json:"step"`
	NovelID         uint    `json:"novel_id"`
	VolumeID        uint    `json:"volume_id"`
	ChapterID       uint    `json:"chapter_id"`
	ChapterIDs      []uint  `json:"chapter_ids"`
	CharacterID     uint    `json:"character_id"`
	PlotlineID      uint    `json:"plotline_id"`
	NovelName       string  `json:"novel_name"`
	Author          string  `json:"author"`
	Genre           string  `json:"genre"`
	OriginalOutline string  `json:"original_outline"`
	UserSuggestion  string  `json:"user_suggestion"`
}

type WorkflowRunResult struct {
	Step        StepKey        `json:"step"`
	NovelID     uint           `json:"novel_id,omitempty"`
	SavedIDs    []uint         `json:"saved_ids,omitempty"`
	CurrentNode string         `json:"current_node,omitempty"`
	Message     string         `json:"message"`
	Data        map[string]any `json:"data,omitempty"`
}

type WorkflowRunner interface {
	Run(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error)
}
