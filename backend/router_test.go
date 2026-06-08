package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"waku/internal/model"
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

func TestRouterReturnsNovelWorkflowAndStepContent(t *testing.T) {
	db := newRouterTestDB(t)
	seedRouterNovel(t, db)
	router := NewRouter(Config{DB: db, WorkflowRunner: &fakeWorkflowRunner{}})

	workflowRequest := httptest.NewRequest(http.MethodGet, "/api/h5/novels/1/workflow", nil)
	workflowResponse := httptest.NewRecorder()
	router.ServeHTTP(workflowResponse, workflowRequest)

	if workflowResponse.Code != http.StatusOK {
		t.Fatalf("expected workflow status 200, got %d: %s", workflowResponse.Code, workflowResponse.Body.String())
	}
	if !bytes.Contains(workflowResponse.Body.Bytes(), []byte("小说大纲")) || !bytes.Contains(workflowResponse.Body.Bytes(), []byte("正文撰写")) {
		t.Fatalf("workflow steps missing: %s", workflowResponse.Body.String())
	}

	stepRequest := httptest.NewRequest(http.MethodGet, "/api/h5/novels/1/steps/chapters", nil)
	stepResponse := httptest.NewRecorder()
	router.ServeHTTP(stepResponse, stepRequest)

	if stepResponse.Code != http.StatusOK {
		t.Fatalf("expected step status 200, got %d: %s", stepResponse.Code, stepResponse.Body.String())
	}
	if !bytes.Contains(stepResponse.Body.Bytes(), []byte("chapters")) || !bytes.Contains(stepResponse.Body.Bytes(), []byte("卷一")) {
		t.Fatalf("chapter step payload missing: %s", stepResponse.Body.String())
	}
}

func TestRouterSupportsDraftArchiveDeleteAndSpeechAPIs(t *testing.T) {
	db := newRouterTestDB(t)
	seedRouterNovel(t, db)
	router := NewRouter(Config{DB: db, WorkflowRunner: &fakeWorkflowRunner{result: &WorkflowRunResult{Step: StepOutline, NovelID: 1, Message: "ok"}}})

	draftRequest := httptest.NewRequest(http.MethodPut, "/api/h5/novels/1/steps/outline/draft", bytes.NewBufferString(`{"content":{"synopsis":"新的简介"}}`))
	draftRequest.Header.Set("Content-Type", "application/json")
	draftResponse := httptest.NewRecorder()
	router.ServeHTTP(draftResponse, draftRequest)
	if draftResponse.Code != http.StatusOK {
		t.Fatalf("expected draft status 200, got %d: %s", draftResponse.Code, draftResponse.Body.String())
	}

	editRequest := httptest.NewRequest(http.MethodPost, "/api/h5/novels/1/steps/outline/edit-archived", nil)
	editResponse := httptest.NewRecorder()
	router.ServeHTTP(editResponse, editRequest)
	if editResponse.Code != http.StatusOK || !bytes.Contains(editResponse.Body.Bytes(), []byte("下游步骤可能需要重新检查")) {
		t.Fatalf("unexpected edit archived response: %d %s", editResponse.Code, editResponse.Body.String())
	}

	speechRequest := httptest.NewRequest(http.MethodPost, "/api/h5/speech/transcriptions", bytes.NewBufferString(`{"text":"口述建议"}`))
	speechRequest.Header.Set("Content-Type", "application/json")
	speechResponse := httptest.NewRecorder()
	router.ServeHTTP(speechResponse, speechRequest)
	if speechResponse.Code != http.StatusOK || !bytes.Contains(speechResponse.Body.Bytes(), []byte("口述建议")) {
		t.Fatalf("unexpected speech response: %d %s", speechResponse.Code, speechResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/h5/novels/1", nil)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func newRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.Novel{}, &model.Plotline{}, &model.Character{}, &model.Volume{}, &model.Chapter{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func seedRouterNovel(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	novel := model.Novel{ID: 1, Title: "测试小说", Author: "测试作者", Genre: "玄幻", Synopsis: "简介", Outline: `{"outline":"大纲","worldview":"世界观","cultivation_system":"修行体系","core_conflict":"核心冲突"}`, Status: "writing", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&novel).Error; err != nil {
		t.Fatalf("create novel failed: %v", err)
	}
	if err := db.Create(&model.Plotline{ID: 1, NovelID: 1, Title: "主线", Type: "main", Description: "剧情线", Status: "active"}).Error; err != nil {
		t.Fatalf("create plotline failed: %v", err)
	}
	if err := db.Create(&model.Character{ID: 1, NovelID: 1, Name: "陆玄", Role: "protagonist", Personality: "沉稳"}).Error; err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	if err := db.Create(&model.Volume{ID: 1, NovelID: 1, VolumeNumber: 1, Title: "卷一", Synopsis: "卷简介", Status: "writing"}).Error; err != nil {
		t.Fatalf("create volume failed: %v", err)
	}
	if err := db.Create(&model.Chapter{ID: 1, NovelID: 1, VolumeID: 1, ChapterNumber: 1, Title: "第一章", Synopsis: "章简介", Outline: "章大纲", Content: "正文", WordCount: 2, Status: "writing"}).Error; err != nil {
		t.Fatalf("create chapter failed: %v", err)
	}
}
