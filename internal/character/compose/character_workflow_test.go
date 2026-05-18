package compose

import (
	"context"
	"fmt"
	"testing"
	"waku/internal/character/repository"
	"waku/internal/model"
	plotlineRepo "waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type stubAgentExecutor struct {
	extract      string
	polish       string
	review       string
	revise       string
	extractCalls int
}

func (s *stubAgentExecutor) ExtractCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, existingCharacters string) (string, error) {
	s.extractCalls++
	return s.extract, nil
}

func (s *stubAgentExecutor) PolishCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, characters, userSuggestion string) (string, error) {
	return s.polish, nil
}

func (s *stubAgentExecutor) ReviewCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, characters string) (string, error) {
	return s.review, nil
}

func (s *stubAgentExecutor) ReviseCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, characters, reviewSuggestions string) (string, error) {
	return s.revise, nil
}

func TestCharacterWorkflow_FullGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupCharacterTestDB(t)
	novelRepo := repository.NewRepository(db)
	characterRepo := repository.NewCharacterRepository(db)

	novel := &model.Novel{
		Title:    "角色工作流完整生成",
		Author:   "测试作者A",
		Genre:    "玄幻",
		Synopsis: "简介",
		Outline:  `{"outline":"主角踏上修仙之路"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}

	executor := &stubAgentExecutor{
		extract: fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修","personality":"坚韧","background":"出身边陲小镇","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
		polish:  fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修，目光坚定","personality":"坚韧果决","background":"出身边陲小镇，立志登仙","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
		review:  fmt.Sprintf(`{"suggestions":[{"original_content":{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修，目光坚定","personality":"坚韧果决","background":"出身边陲小镇，立志登仙","plotline_ids":[1],"relationships":{}},"modification_suggestion":"补充其追查父母失踪之谜的长期目标。"}]}`, novel.ID),
		revise:  fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修，目光坚定","personality":"坚韧果决","background":"出身边陲小镇，立志登仙并追查父母失踪之谜","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
	}
	RegisterAgentExecutor(executor)

	plotlineRepository := plotlineRepo.NewPlotlineRepository(db)
	graph, err := NewCharacterWorkflow(ctx, nil, novelRepo, characterRepo, plotlineRepository)
	if err != nil {
		t.Fatalf("NewCharacterWorkflow() error = %v", err)
	}

	result, err := ExecuteCharacterWorkflow(ctx, graph, &CharacterWorkflowInput{NovelID: novel.ID})
	if err != nil {
		t.Fatalf("ExecuteCharacterWorkflow() error = %v", err)
	}

	t.Logf("Final state: CurrentNode=%s, ExtractCalls=%d, ErrorMsg=%s", result.CurrentNode, executor.extractCalls, result.ErrorMsg)

	if result.CurrentNode != "save_success" {
		t.Fatalf("expected save_success, got %s", result.CurrentNode)
	}
	if len(result.SavedCharacterIDs) != 1 {
		t.Fatalf("expected 1 saved character id, got %d", len(result.SavedCharacterIDs))
	}
	if result.SavedCharacterIDs[0] == 0 {
		t.Fatalf("expected auto-increment character id")
	}
	if executor.extractCalls != 1 {
		t.Fatalf("expected extract to run once, got %d", executor.extractCalls)
	}
}

func TestCharacterWorkflow_QuickRevisionWithNovelSuggestion(t *testing.T) {
	ctx := context.Background()
	db := setupCharacterTestDB(t)
	novelRepo := repository.NewRepository(db)
	characterRepo := repository.NewCharacterRepository(db)

	novel := &model.Novel{
		Title:    "角色工作流快速修改",
		Author:   "测试作者B",
		Genre:    "玄幻",
		Synopsis: "简介",
		Outline:  `{"outline":"主角踏上修仙之路"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}

	seed := []*model.Character{
		{NovelID: novel.ID, Name: "林霄", Role: "protagonist", Age: 18, Gender: "男", Appearance: "少年剑修", Personality: "坚韧", Background: "出身边陲小镇", PlotlineIDs: model.UintArray{1}, Relationships: model.RelationshipMap{}},
		{NovelID: novel.ID, Name: "沈夜", Role: "antagonist", Age: 28, Gender: "男", Appearance: "黑袍修士", Personality: "多疑", Background: "没落世家出身", PlotlineIDs: model.UintArray{2}, Relationships: model.RelationshipMap{}},
	}
	if err := characterRepo.CreateBatch(seed); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	polish := fmt.Sprintf(`{"characters":[{"id":%d,"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修","personality":"坚韧","background":"出身边陲小镇","plotline_ids":[1],"relationships":{"%d":"与沈夜宿敌"}},{"id":%d,"novel_id":%d,"name":"沈夜","role":"antagonist","age":28,"gender":"男","appearance":"黑袍修士，神情阴鸷，压迫感强","personality":"多疑狠厉","background":"没落世家出身，为振兴家族不择手段","plotline_ids":[2],"relationships":{"%d":"与林霄宿敌"}}]}`, seed[0].ID, novel.ID, seed[1].ID, seed[1].ID, novel.ID, seed[0].ID)
	review := fmt.Sprintf(`{"suggestions":[{"original_content":{"id":%d,"novel_id":%d,"name":"沈夜","role":"antagonist","age":28,"gender":"男","appearance":"黑袍修士，神情阴鸷，压迫感强","personality":"多疑狠厉","background":"没落世家出身，为振兴家族不择手段","plotline_ids":[2],"relationships":{"%d":"与林霄宿敌"}},"modification_suggestion":"进一步强化其掌控欲与压迫感，使其威胁更持续。"}]}`, seed[1].ID, novel.ID, seed[0].ID)
	revise := fmt.Sprintf(`{"characters":[{"id":%d,"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修","personality":"坚韧","background":"出身边陲小镇","plotline_ids":[1],"relationships":{"%d":"与沈夜宿敌"}},{"id":%d,"novel_id":%d,"name":"沈夜","role":"antagonist","age":28,"gender":"男","appearance":"黑袍修士，神情阴鸷，压迫感强","personality":"多疑狠厉、掌控欲极强","background":"没落世家出身，为振兴家族不择手段","plotline_ids":[2],"relationships":{"%d":"与林霄宿敌"}}]}`, seed[0].ID, novel.ID, seed[1].ID, seed[1].ID, novel.ID, seed[0].ID)

	executor := &stubAgentExecutor{
		extract: `{"characters":[]}`,
		polish:  polish,
		review:  review,
		revise:  revise,
	}
	RegisterAgentExecutor(executor)

	plotlineRepository := plotlineRepo.NewPlotlineRepository(db)
	graph, err := NewCharacterWorkflow(ctx, nil, novelRepo, characterRepo, plotlineRepository)
	if err != nil {
		t.Fatalf("NewCharacterWorkflow() error = %v", err)
	}

	result, err := ExecuteCharacterWorkflow(ctx, graph, &CharacterWorkflowInput{
		NovelID:        novel.ID,
		UserSuggestion: "新增一个更强势的反派压迫感",
	})
	if err != nil {
		t.Fatalf("ExecuteCharacterWorkflow() error = %v", err)
	}

	if len(result.SavedCharacterIDs) != 2 {
		t.Fatalf("expected 2 saved character ids, got %d", len(result.SavedCharacterIDs))
	}
	if executor.extractCalls != 0 {
		t.Fatalf("expected quick revision to skip extract, got %d extract calls", executor.extractCalls)
	}

	updated, err := characterRepo.GetByID(seed[1].ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updated.Personality == "多疑" {
		t.Fatalf("expected revised personality, got %s", updated.Personality)
	}
}

func setupCharacterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite error = %v", err)
	}

	if err := db.AutoMigrate(&model.Novel{}, &model.Plotline{}, &model.Character{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
}
