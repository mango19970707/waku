package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeWorkflowRunner struct {
	lastRequest WorkflowRunRequest
	result      *WorkflowRunResult
	err         error
}

func (r *fakeWorkflowRunner) Run(ctx context.Context, req WorkflowRunRequest) (*WorkflowRunResult, error) {
	r.lastRequest = req
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func TestRouterRunsOutlineWorkflowFromMessageEndpoint(t *testing.T) {
	runner := &fakeWorkflowRunner{result: &WorkflowRunResult{Step: StepOutline, SavedIDs: []uint{9}, CurrentNode: "saveToDB"}}
	router := NewRouter(Config{WorkflowRunner: runner})
	body := bytes.NewBufferString(`{"genre":"xuanhuan","original_outline":"young hero","user_suggestion":"stronger villain"}`)

	request := httptest.NewRequest(http.MethodPost, "/api/h5/novels/9/steps/outline/messages", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if runner.lastRequest.Step != StepOutline || runner.lastRequest.NovelID != 9 {
		t.Fatalf("unexpected request: %+v", runner.lastRequest)
	}
	if runner.lastRequest.UserSuggestion != "stronger villain" {
		t.Fatalf("user suggestion not forwarded: %+v", runner.lastRequest)
	}
}

func TestRouterUsesNestedVolumeAndChapterIDs(t *testing.T) {
	runner := &fakeWorkflowRunner{result: &WorkflowRunResult{Step: StepChapters, SavedIDs: []uint{1, 2}, CurrentNode: "saveToDB"}}
	router := NewRouter(Config{WorkflowRunner: runner})
	body := bytes.NewBufferString(`{"chapter_id":3,"user_suggestion":"add failed trade"}`)

	request := httptest.NewRequest(http.MethodPost, "/api/h5/novels/8/volumes/2/chapters/messages", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if runner.lastRequest.Step != StepChapters || runner.lastRequest.NovelID != 8 || runner.lastRequest.VolumeID != 2 || runner.lastRequest.ChapterID != 3 {
		t.Fatalf("route ids not forwarded: %+v", runner.lastRequest)
	}
}

func TestRouterRejectsUnknownStepWithChineseError(t *testing.T) {
	router := NewRouter(Config{WorkflowRunner: &fakeWorkflowRunner{}})
	request := httptest.NewRequest(http.MethodPost, "/api/h5/novels/1/steps/unknown/messages", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("不支持的创作步骤")) {
		t.Fatalf("expected chinese error, got %s", response.Body.String())
	}
}

func TestRouterMapsWorkflowErrorToBadGateway(t *testing.T) {
	router := NewRouter(Config{WorkflowRunner: &fakeWorkflowRunner{err: errors.New("model unavailable")}})
	request := httptest.NewRequest(http.MethodPost, "/api/h5/novels/1/steps/plotlines/messages", bytes.NewBufferString(`{"user_suggestion":"early foreshadowing"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", response.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["message"] != "工作流执行失败，请稍后重试" {
		t.Fatalf("unexpected message: %+v", payload)
	}
}
