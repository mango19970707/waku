package compose

import (
	"context"
	"fmt"
	"testing"
	"waku/internal/model"
	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type stubAgentExecutor struct {
	generate string
	polish   string
	review   string
	revise   string
}

func (s *stubAgentExecutor) GeneratePlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	return s.generate, nil
}

func (s *stubAgentExecutor) PolishPlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, plotlines, genre, userSuggestion string) (string, error) {
	return s.polish, nil
}

func (s *stubAgentExecutor) ReviewPlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, userSuggestion string) (string, error) {
	return s.review, nil
}

func (s *stubAgentExecutor) RevisePlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, polishedPlotlines, reviewSuggestions string) (string, error) {
	return s.revise, nil
}

func TestPlotlineWorkflow_FullGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupPlotlineTestDB(t)
	novelRepo := repository.NewRepository(db)
	plotlineRepo := repository.NewPlotlineRepository(db)

	novel := &model.Novel{
		Title:    "full-generation",
		Author:   "tester-a",
		Genre:    "xuanhuan",
		Synopsis: "synopsis",
		Outline:  `{"outline":"主角踏上修仙之路","worldview":"九州","cultivation_system":"灵根"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}

	RegisterAgentExecutor(&stubAgentExecutor{
		generate: `{"plotlines":[{"title":"修仙主线","type":"main","description":"主角一路成长","status":"active"}]}`,
		polish:   `{"plotlines":[{"title":"修仙主线","type":"main","description":"主角一路成长并结识盟友","status":"active"}]}`,
		review:   `{"reviews":[{"original_content":{"title":"修仙主线","type":"main","description":"主角一路成长并结识盟友","status":"active"},"modification_suggestion":"补强前期动机"}]}`,
		revise:   `{"plotlines":[{"title":"修仙主线","type":"main","description":"主角带着明确目标一路成长并结识盟友","status":"active"}]}`,
	})

	graph, err := NewPlotlineWorkflow(ctx, nil, novelRepo, plotlineRepo)
	if err != nil {
		t.Fatalf("NewPlotlineWorkflow() error = %v", err)
	}

	result, err := ExecutePlotlineWorkflow(ctx, graph, &PlotlineWorkflowInput{NovelID: novel.ID})
	if err != nil {
		t.Fatalf("ExecutePlotlineWorkflow() error = %v", err)
	}

	if result.CurrentNode != "save_success" {
		t.Fatalf("expected save_success, got %s", result.CurrentNode)
	}
	if len(result.SavedPlotlineIDs) != 1 {
		t.Fatalf("expected 1 saved plotline id, got %d", len(result.SavedPlotlineIDs))
	}

	saved, err := plotlineRepo.GetByNovelID(novel.ID)
	if err != nil {
		t.Fatalf("GetByNovelID() error = %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 plotline in db, got %d", len(saved))
	}
	if saved[0].Title != "修仙主线" {
		t.Fatalf("unexpected plotline title: %s", saved[0].Title)
	}
}

func TestPlotlineWorkflow_AddPlotlineFromUserSuggestion(t *testing.T) {
	ctx := context.Background()
	db := setupPlotlineTestDB(t)
	novelRepo := repository.NewRepository(db)
	plotlineRepo := repository.NewPlotlineRepository(db)

	novel := &model.Novel{
		Title:    "add-from-suggestion",
		Author:   "tester-b",
		Genre:    "xuanhuan",
		Synopsis: "synopsis",
		Outline:  `{"outline":"主角踏上修仙之路","worldview":"九州","cultivation_system":"灵根"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}

	seed := []*model.Plotline{
		{NovelID: novel.ID, Title: "修仙主线", Type: "main", Description: "主角一路成长", Status: "active"},
		{NovelID: novel.ID, Title: "宗门支线", Type: "sub", Description: "宗门纷争", Status: "active"},
	}
	if err := plotlineRepo.CreateBatch(seed); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	polishPayload := fmt.Sprintf(`{"plotlines":[
		{"id":%d,"title":"修仙主线","type":"main","description":"主角一路成长","status":"active"},
		{"id":%d,"title":"宗门支线","type":"sub","description":"宗门纷争","status":"active"},
		{"title":"双女主情感支线","type":"sub","description":"围绕彩鳞和熏儿的情感拉扯新增一条支线","status":"active"}]}`, seed[0].ID, seed[1].ID)
	reviewPayload := fmt.Sprintf(`{"reviews":[
		{"original_content":{"id":%d,"title":"修仙主线","type":"main","description":"主角一路成长","status":"active"},"modification_suggestion":"主线动机再明确一些"},
		{"original_content":{"title":"双女主情感支线","type":"sub","description":"围绕彩鳞和熏儿的情感拉扯新增一条支线","status":"active"},"modification_suggestion":"补强与主线的联动"}]}`, seed[0].ID)
	revisePayload := fmt.Sprintf(`{"plotlines":[
		{"id":%d,"title":"修仙主线","type":"main","description":"主角一路成长，并与彩鳞和熏儿形成重要关系","status":"active"},
		{"id":%d,"title":"宗门支线","type":"sub","description":"宗门纷争","status":"active"},
		{"title":"双女主情感支线","type":"sub","description":"彩鳞和熏儿分别从不同立场卷入主角成长，并与修仙主线形成直接联动","status":"active"}]}`, seed[0].ID, seed[1].ID)

	RegisterAgentExecutor(&stubAgentExecutor{
		polish: polishPayload,
		review: reviewPayload,
		revise: revisePayload,
	})

	graph, err := NewPlotlineWorkflow(ctx, nil, novelRepo, plotlineRepo)
	if err != nil {
		t.Fatalf("NewPlotlineWorkflow() error = %v", err)
	}

	result, err := ExecutePlotlineWorkflow(ctx, graph, &PlotlineWorkflowInput{
		NovelID:        novel.ID,
		UserSuggestion: "来两个女主角，彩鳞和熏儿",
	})
	if err != nil {
		t.Fatalf("ExecutePlotlineWorkflow() error = %v", err)
	}

	if len(result.SavedPlotlineIDs) != 3 {
		t.Fatalf("expected 3 saved plotline ids, got %d", len(result.SavedPlotlineIDs))
	}

	updated, err := plotlineRepo.GetByNovelID(novel.ID)
	if err != nil {
		t.Fatalf("GetByNovelID() error = %v", err)
	}
	if len(updated) != 3 {
		t.Fatalf("expected 3 plotlines in db, got %d", len(updated))
	}

	foundNew := false
	for _, item := range updated {
		if item.Title == "双女主情感支线" {
			foundNew = true
			break
		}
	}
	if !foundNew {
		t.Fatalf("expected new plotline created from user suggestion")
	}
}

func setupPlotlineTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}

	if err := db.AutoMigrate(&model.Novel{}, &model.Plotline{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
}
