package compose

import (
	"context"
	"fmt"
	"testing"
	"waku/internal/character/repository"
	"waku/internal/model"

	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type stubAgentExecutor struct {
	extract string
	polish  string
	review  string
	revise  string
}

func (s *stubAgentExecutor) ExtractCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingCharacters string) (string, error) {
	return s.extract, nil
}

func (s *stubAgentExecutor) PolishCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error) {
	return s.polish, nil
}

func (s *stubAgentExecutor) ReviewCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error) {
	return s.review, nil
}

func (s *stubAgentExecutor) ReviseCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error) {
	return s.revise, nil
}

func TestCharacterWorkflow_FullGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupCharacterTestDB(t)
	novelRepo := repository.NewRepository(db)
	characterRepo := repository.NewCharacterRepository(db)

	novel := &model.Novel{
		Title:    "角色工作流-完整生成",
		Author:   "测试作者A",
		Genre:    "玄幻",
		Synopsis: "简介",
		Outline:  `{"outline":"主角踏上修仙之路"}`,
		Status:   "draft",
	}
	if err := novelRepo.Create(novel); err != nil {
		t.Fatalf("Create novel error = %v", err)
	}

	RegisterAgentExecutor(&stubAgentExecutor{
		extract: fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修","personality":"坚韧","background":"出身边陲小镇","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
		polish:  fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修，目光坚定","personality":"坚韧果决","background":"出身边陲小镇，立志登仙","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
		review:  fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修，目光坚定","personality":"坚韧果决","background":"出身边陲小镇，立志登仙","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
		revise:  fmt.Sprintf(`{"characters":[{"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修，目光坚定","personality":"坚韧果决","background":"出身边陲小镇，立志登仙并追查父母失踪之谜","plotline_ids":[1],"relationships":{}}]}`, novel.ID),
	})

	graph, err := NewCharacterWorkflow(ctx, nil, novelRepo, characterRepo)
	if err != nil {
		t.Fatalf("NewCharacterWorkflow() error = %v", err)
	}

	result, err := ExecuteCharacterWorkflow(ctx, graph, &CharacterWorkflowInput{NovelID: novel.ID})
	if err != nil {
		t.Fatalf("ExecuteCharacterWorkflow() error = %v", err)
	}

	if result.CurrentNode != "save_success" {
		t.Fatalf("expected save_success, got %s", result.CurrentNode)
	}
	if len(result.SavedCharacterIDs) != 1 {
		t.Fatalf("expected 1 saved character id, got %d", len(result.SavedCharacterIDs))
	}
	if result.SavedCharacterIDs[0] == 0 {
		t.Fatalf("expected auto-increment character id")
	}
}

func TestCharacterWorkflow_QuickRevision(t *testing.T) {
	ctx := context.Background()
	db := setupCharacterTestDB(t)
	novelRepo := repository.NewRepository(db)
	characterRepo := repository.NewCharacterRepository(db)

	novel := &model.Novel{
		Title:    "角色工作流-快速修改",
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

	polish := fmt.Sprintf(`{"characters":[{"id":%d,"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修","personality":"坚韧","background":"出身边陲小镇","plotline_ids":[1],"relationships":{}},{"id":%d,"novel_id":%d,"name":"沈夜","role":"antagonist","age":28,"gender":"男","appearance":"黑袍修士，神情阴鸷","personality":"多疑狠厉","background":"没落世家出身","plotline_ids":[2],"relationships":{}}]}`, seed[0].ID, novel.ID, seed[1].ID, novel.ID)
	revise := fmt.Sprintf(`{"characters":[{"id":%d,"novel_id":%d,"name":"林霄","role":"protagonist","age":18,"gender":"男","appearance":"少年剑修","personality":"坚韧","background":"出身边陲小镇","plotline_ids":[1],"relationships":{"%d":"与沈夜宿敌"}},{"id":%d,"novel_id":%d,"name":"沈夜","role":"antagonist","age":28,"gender":"男","appearance":"黑袍修士，神情阴鸷","personality":"多疑狠厉、掌控欲极强","background":"没落世家出身，为振兴家族不择手段","plotline_ids":[2],"relationships":{"%d":"与林霄宿敌"}}]}`, seed[0].ID, novel.ID, seed[1].ID, seed[1].ID, novel.ID, seed[0].ID)

	RegisterAgentExecutor(&stubAgentExecutor{
		polish: polish,
		review: polish,
		revise: revise,
	})

	graph, err := NewCharacterWorkflow(ctx, nil, novelRepo, characterRepo)
	if err != nil {
		t.Fatalf("NewCharacterWorkflow() error = %v", err)
	}

	result, err := ExecuteCharacterWorkflow(ctx, graph, &CharacterWorkflowInput{
		NovelID:        novel.ID,
		CharacterID:    seed[1].ID,
		UserSuggestion: "强化反派压迫感",
	})
	if err != nil {
		t.Fatalf("ExecuteCharacterWorkflow() error = %v", err)
	}

	if len(result.SavedCharacterIDs) != 2 {
		t.Fatalf("expected 2 saved character ids, got %d", len(result.SavedCharacterIDs))
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
