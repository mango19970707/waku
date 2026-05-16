package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"waku/internal/model"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	volumerepository "waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type stubVolumeAgentExecutor struct {
	generate string
	polish   string
	review   string
	revise   string
}

func (s *stubVolumeAgentExecutor) GenerateVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, existingVolumes string) (string, error) {
	return s.generate, nil
}

func (s *stubVolumeAgentExecutor) PolishVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error) {
	return s.polish, nil
}

func (s *stubVolumeAgentExecutor) ReviewVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error) {
	return s.review, nil
}

func (s *stubVolumeAgentExecutor) ReviseVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error) {
	return s.revise, nil
}

func TestVolumeWorkflow_FullGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupVolumeTestDB(t)
	novelRepo := novelrepository.NewRepository(db)
	plotlineRepo := plotlinerepository.NewPlotlineRepository(db)
	volumeRepo := volumerepository.NewVolumeRepository(db)

	novel := &model.Novel{
		Title:    "volume-full-generation",
		Author:   "tester-a",
		Genre:    "玄幻",
		Synopsis: "intro",
		Outline:  `{"outline":"hero starts journey","worldview":"nine lands","cultivation_system":"spirit root"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}
	cleanupVolumeArtifacts(t, novel.Author, novel.Title)

	if err := plotlineRepo.CreateBatch([]*model.Plotline{
		{NovelID: novel.ID, Title: "main", Type: "main", Description: "main line", Status: "active"},
		{NovelID: novel.ID, Title: "side", Type: "sub", Description: "side line", Status: "active"},
	}); err != nil {
		t.Fatalf("CreateBatch plotlines error = %v", err)
	}

	RegisterAgentExecutor(&stubVolumeAgentExecutor{
		generate: `{"volumes":[{"volume_number":1,"title":"Book One","synopsis":"leave town","theme":"growth","outline":"enter world","status":"draft"},{"volume_number":2,"title":"Book Two","synopsis":"enter sect","theme":"entry","outline":"first conflict","status":"draft"}]}`,
		polish:   `{"volumes":[{"volume_number":1,"title":"Book One","synopsis":"leave town and enter the world","theme":"growth","outline":"enter world and set up the main plot","status":"draft"},{"volume_number":2,"title":"Book Two","synopsis":"enter sect and clash with it","theme":"entry","outline":"sect appears","status":"draft"}]}`,
		review:   `{"volumes":[{"volume_number":1,"title":"Book One","synopsis":"leave town and enter the world","theme":"growth","outline":"enter world and set up the main plot","status":"draft"},{"volume_number":2,"title":"Book Two","synopsis":"enter sect and clash with it","theme":"entry","outline":"sect appears","status":"draft"}]}`,
		revise:   `{"volumes":[{"volume_number":1,"title":"Book One","synopsis":"leave town and enter the world","theme":"growth","outline":"enter world and set up the main plot","status":"draft"},{"volume_number":2,"title":"Book Two","synopsis":"enter sect and clash with it","theme":"entry","outline":"sect appears","status":"draft"}]}`,
	})

	graph, err := NewVolumeWorkflow(ctx, nil, novelRepo, plotlineRepo, volumeRepo)
	if err != nil {
		t.Fatalf("NewVolumeWorkflow() error = %v", err)
	}

	result, err := ExecuteVolumeWorkflow(ctx, graph, &VolumeWorkflowInput{NovelID: novel.ID})
	if err != nil {
		t.Fatalf("ExecuteVolumeWorkflow() error = %v", err)
	}

	if result.CurrentNode != "save_success" {
		t.Fatalf("expected save_success, got %s", result.CurrentNode)
	}
	if len(result.SavedVolumeIDs) != 2 {
		t.Fatalf("expected 2 saved volume ids, got %d", len(result.SavedVolumeIDs))
	}

	saved, err := volumeRepo.GetByNovelID(novel.ID)
	if err != nil {
		t.Fatalf("GetByNovelID() error = %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 saved volumes, got %d", len(saved))
	}
}

func TestVolumeWorkflow_QuickRevision(t *testing.T) {
	ctx := context.Background()
	db := setupVolumeTestDB(t)
	novelRepo := novelrepository.NewRepository(db)
	plotlineRepo := plotlinerepository.NewPlotlineRepository(db)
	volumeRepo := volumerepository.NewVolumeRepository(db)

	novel := &model.Novel{
		Title:    "volume-quick-revision",
		Author:   "tester-b",
		Genre:    "玄幻",
		Synopsis: "intro",
		Outline:  `{"outline":"hero starts journey","worldview":"nine lands","cultivation_system":"spirit root"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}
	cleanupVolumeArtifacts(t, novel.Author, novel.Title)

	if err := plotlineRepo.CreateBatch([]*model.Plotline{
		{NovelID: novel.ID, Title: "main", Type: "main", Description: "main line", Status: "active"},
	}); err != nil {
		t.Fatalf("CreateBatch plotlines error = %v", err)
	}

	seed := []*model.Volume{
		{NovelID: novel.ID, VolumeNumber: 1, Title: "old one", Synopsis: "old synopsis", Theme: "old theme", Outline: "old outline", Status: "draft"},
		{NovelID: novel.ID, VolumeNumber: 2, Title: "old two", Synopsis: "old synopsis", Theme: "old theme", Outline: "old outline", Status: "draft"},
	}
	if err := volumeRepo.CreateBatch(seed); err != nil {
		t.Fatalf("CreateBatch volumes error = %v", err)
	}

	RegisterAgentExecutor(&stubVolumeAgentExecutor{
		polish: fmt.Sprintf(`{"volumes":[{"id":%d,"volume_number":1,"title":"Book One","synopsis":"leave town and enter the world","theme":"growth","outline":"enter world and set up the main plot","status":"draft"},{"id":%d,"volume_number":2,"title":"Book Two","synopsis":"enter sect and clash with it","theme":"entry","outline":"sect appears","status":"draft"}]}`, seed[0].ID, seed[1].ID),
		review: fmt.Sprintf(`{"volumes":[{"id":%d,"volume_number":1,"title":"Book One","synopsis":"leave town and enter the world","theme":"growth","outline":"enter world and set up the main plot","status":"draft"},{"id":%d,"volume_number":2,"title":"Book Two","synopsis":"enter sect and clash with it","theme":"entry","outline":"sect appears","status":"draft"}]}`, seed[0].ID, seed[1].ID),
		revise: fmt.Sprintf(`{"volumes":[{"id":%d,"volume_number":1,"title":"Book One","synopsis":"leave town and enter the world","theme":"growth","outline":"enter world and set up the main plot","status":"draft"},{"id":%d,"volume_number":2,"title":"Book Two Revised","synopsis":"enter sect and escalate conflict","theme":"entry","outline":"sect conflict escalates","status":"draft"}]}`, seed[0].ID, seed[1].ID),
	})

	graph, err := NewVolumeWorkflow(ctx, nil, novelRepo, plotlineRepo, volumeRepo)
	if err != nil {
		t.Fatalf("NewVolumeWorkflow() error = %v", err)
	}

	result, err := ExecuteVolumeWorkflow(ctx, graph, &VolumeWorkflowInput{
		NovelID:        novel.ID,
		VolumeID:       seed[1].ID,
		UserSuggestion: "strengthen sect conflict",
	})
	if err != nil {
		t.Fatalf("ExecuteVolumeWorkflow() error = %v", err)
	}

	if result.CurrentNode != "save_success" {
		t.Fatalf("expected save_success, got %s", result.CurrentNode)
	}
	if len(result.SavedVolumeIDs) != 2 {
		t.Fatalf("expected 2 saved volume ids, got %d", len(result.SavedVolumeIDs))
	}

	updated, err := volumeRepo.GetByID(seed[1].ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updated.Title != "Book Two Revised" {
		t.Fatalf("expected revised title, got %s", updated.Title)
	}
}

func setupVolumeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}

	if err := db.AutoMigrate(&model.Novel{}, &model.Plotline{}, &model.Volume{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
}

func cleanupVolumeArtifacts(t *testing.T, author, novelName string) {
	t.Helper()

	dir := filepath.Join(projectRootDir(), "data", "content", author, novelName, "volume")
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("cleanup artifacts error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
}
