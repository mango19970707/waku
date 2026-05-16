package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"waku/internal/constant"
	"waku/internal/model"
	"waku/internal/outline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type mockAgentExecutor struct {
	generateCalls int
	expandCalls   int
	polishCalls   int
	reviewCalls   int
	reviseCalls   int
	failGenerate  int
}

func (m *mockAgentExecutor) GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error) {
	m.generateCalls++
	if m.generateCalls <= m.failGenerate {
		return "", fmt.Errorf("generate failed")
	}
	return buildMockNovelJSON("星火", "少年踏入异界。", "生成大纲", true, false)
}

func (m *mockAgentExecutor) ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error) {
	m.expandCalls++
	return buildMockNovelJSON("星火", "少年踏入异界。", "扩写后的完整大纲", true, false)
}

func (m *mockAgentExecutor) PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	m.polishCalls++
	return buildMockNovelJSON("星火", "更凝练的简介。", "润色后的完整大纲", true, false)
}

func (m *mockAgentExecutor) ReviewOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	m.reviewCalls++
	return buildMockNovelJSON("星火", "更凝练的简介。", "润色后的完整大纲", true, true)
}

func (m *mockAgentExecutor) ReviseOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, suggestions, genre string) (string, error) {
	m.reviseCalls++
	return buildMockNovelJSON("星火", "最终简介。", "修改后的最终大纲", true, false)
}

func buildMockNovelJSON(title, synopsis, outlineText string, cultivation, withReview bool) (string, error) {
	outlineMap := map[string]any{
		"outline": outlineText,
	}
	if cultivation {
		outlineMap["worldview"] = "诸天万界并立"
		outlineMap["cultivation_system"] = "炼气、筑基、金丹、元婴"
	}
	if withReview {
		outlineMap["review_suggestions"] = []map[string]any{
			{
				"type":       "logic",
				"problem":    "前期冲突偏弱",
				"suggestion": "增强主角与宗门的对抗",
				"severity":   "medium",
			},
		}
	}
	outlineJSON, err := json.Marshal(outlineMap)
	if err != nil {
		return "", err
	}

	payload := map[string]string{
		"title":    title,
		"synopsis": synopsis,
		"outline":  string(outlineJSON),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        ":memory:",
	}, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Novel{}))
	return db
}

func cleanupCheckpointFiles(t *testing.T, author, novelName string) {
	t.Helper()
	require.NoError(t, os.RemoveAll(filepath.Join(projectRootDir(), "data", "content", author, novelName)))
}

func useMockExecutor(t *testing.T, executor AgentExecutor) {
	t.Helper()
	original := agentExecutor
	RegisterAgentExecutor(executor)
	t.Cleanup(func() {
		RegisterAgentExecutor(original)
	})
}

func TestOutlineWorkflowGraph_FullGeneration(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	mock := &mockAgentExecutor{}
	useMockExecutor(t, mock)

	graph, err := BuildOutlineWorkflowGraph(ctx, nil, repo)
	require.NoError(t, err)

	input := &OutlineWorkflowInput{
		NovelName:       "星火",
		Author:          "测试作者",
		Genre:           "玄幻",
		OriginalOutline: "少年穿越异界，寻找真相。",
	}
	t.Cleanup(func() {
		cleanupCheckpointFiles(t, input.Author, input.NovelName)
	})

	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	require.NoError(t, err)
	require.Equal(t, "save_success", result.CurrentNode)
	require.Equal(t, 1, mock.generateCalls)
	require.Equal(t, 1, mock.expandCalls)
	require.Equal(t, 1, mock.polishCalls)
	require.Equal(t, 1, mock.reviewCalls)
	require.Equal(t, 1, mock.reviseCalls)
	require.NotZero(t, result.NovelID)

	saved, err := repo.GetByID(result.NovelID)
	require.NoError(t, err)
	require.Equal(t, "星火", saved.Title)
	require.Equal(t, "最终简介。", saved.Synopsis)

	var outline map[string]any
	require.NoError(t, json.Unmarshal([]byte(saved.Outline), &outline))
	require.Equal(t, "修改后的最终大纲", outline["outline"])
	require.NotEmpty(t, outline["worldview"])
	require.NotEmpty(t, outline["cultivation_system"])
}

func TestOutlineWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	mock := &mockAgentExecutor{}
	useMockExecutor(t, mock)

	storedOutline, err := buildMockNovelJSON("旧标题", "旧简介", "旧大纲内容", true, false)
	require.NoError(t, err)

	var stored map[string]string
	require.NoError(t, json.Unmarshal([]byte(storedOutline), &stored))

	novel := &model.Novel{
		Title:    "旧标题",
		Author:   "测试作者",
		Genre:    "玄幻",
		Synopsis: "旧简介",
		Outline:  stored["outline"],
		Status:   constant.StatusDraft,
	}
	require.NoError(t, repo.Create(novel))

	graph, err := BuildOutlineWorkflowGraph(ctx, nil, repo)
	require.NoError(t, err)

	input := &OutlineWorkflowInput{
		NovelID:        novel.ID,
		Genre:          "玄幻",
		UserSuggestion: "增强主角的成长线。",
	}
	t.Cleanup(func() {
		cleanupCheckpointFiles(t, novel.Author, novel.Title)
	})

	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	require.NoError(t, err)
	require.Equal(t, "save_success", result.CurrentNode)
	require.Zero(t, mock.generateCalls)
	require.Zero(t, mock.expandCalls)
	require.Equal(t, 1, mock.polishCalls)
	require.Equal(t, 1, mock.reviewCalls)
	require.Equal(t, 1, mock.reviseCalls)
}

func TestOutlineWorkflowGraph_RetryGenerate(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	mock := &mockAgentExecutor{failGenerate: 1}
	useMockExecutor(t, mock)

	graph, err := BuildOutlineWorkflowGraph(ctx, nil, repo)
	require.NoError(t, err)

	input := &OutlineWorkflowInput{
		NovelName:       "重试小说",
		Author:          "测试作者",
		Genre:           "玄幻",
		OriginalOutline: "一名普通人逆袭成长。",
	}
	t.Cleanup(func() {
		cleanupCheckpointFiles(t, input.Author, input.NovelName)
	})

	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	require.NoError(t, err)
	require.Equal(t, "save_success", result.CurrentNode)
	require.Equal(t, 2, mock.generateCalls)
}

func TestOutlineWorkflowGraph_ResumeFromCheckpoint(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	mock := &mockAgentExecutor{}
	useMockExecutor(t, mock)

	author := "测试作者"
	novelName := "断点小说"
	generateJSON, err := buildMockNovelJSON("断点标题", "缓存简介", "生成阶段缓存", true, false)
	require.NoError(t, err)
	expandJSON, err := buildMockNovelJSON("断点标题", "缓存简介", "扩写阶段缓存", true, false)
	require.NoError(t, err)
	require.NoError(t, saveStageFile(author, novelName, constant.StageGenerate, generateJSON))
	require.NoError(t, saveStageFile(author, novelName, constant.StageExpand, expandJSON))
	t.Cleanup(func() {
		cleanupCheckpointFiles(t, author, novelName)
	})

	graph, err := BuildOutlineWorkflowGraph(ctx, nil, repo)
	require.NoError(t, err)

	input := &OutlineWorkflowInput{
		NovelName:       novelName,
		Author:          author,
		Genre:           "玄幻",
		OriginalOutline: "从零开始构建王国。",
	}

	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	require.NoError(t, err)
	require.Equal(t, "save_success", result.CurrentNode)
	require.Zero(t, mock.generateCalls)
	require.Zero(t, mock.expandCalls)
	require.Equal(t, 1, mock.polishCalls)
}

func TestExecuteOutlineWorkflow_InvalidInput(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	useMockExecutor(t, &mockAgentExecutor{})

	graph, err := BuildOutlineWorkflowGraph(ctx, nil, repo)
	require.NoError(t, err)

	_, err = ExecuteOutlineWorkflow(ctx, graph, &OutlineWorkflowInput{
		Author:          "",
		Genre:           "玄幻",
		OriginalOutline: "测试大纲",
	})
	require.Error(t, err)
}
