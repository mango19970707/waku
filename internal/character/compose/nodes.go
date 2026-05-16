package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	einoModel "github.com/cloudwego/eino/components/model"

	"waku/internal/character/agents"
	"waku/internal/character/repository"
	"waku/internal/model"
)

// NovelRepository Novel 仓储接口（用于查询小说信息）
type NovelRepository interface {
	GetByID(id uint) (*model.Novel, error)
}

// ==================== 人物提取工作流节点 ====================

// initExtractNode 初始化提取工作流节点
//
// 参数：
//   - novelRepo: Novel 仓储接口
//
// 返回：
//   - func: 节点处理函数
func initExtractNode(novelRepo NovelRepository) func(context.Context, *CharacterExtractWorkflowInput) (*CharacterExtractWorkflowState, error) {
	return func(ctx context.Context, input *CharacterExtractWorkflowInput) (*CharacterExtractWorkflowState, error) {
		fmt.Println("\n========== 初始化人物提取工作流 ==========")

		// 查询小说信息
		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		// 验证大纲是否存在
		if novel.Outline == "" {
			return nil, fmt.Errorf("小说大纲为空，无法提取人物")
		}

		// 初始化工作流状态
		state := &CharacterExtractWorkflowState{
			CurrentNode:        "init",
			NovelID:            input.NovelID,
			NovelName:          novel.Title,
			Author:             novel.Author,
			Genre:              novel.Genre,
			Outline:            novel.Outline,
			ExtractedCharacters: "",
			SavedCharacterIDs:  []string{},
			ErrorMsg:           "",
			RetryCount:         0,
			MaxRetries:         3,
		}

		fmt.Printf("小说ID: %d\n", state.NovelID)
		fmt.Printf("小说名称: %s\n", state.NovelName)
		fmt.Printf("小说类型: %s\n", state.Genre)
		fmt.Printf("大纲长度: %d 字\n", len(state.Outline))

		return state, nil
	}
}

// extractNode 人物提取节点
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型
//
// 返回：
//   - func: 节点处理函数
func extractNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterExtractWorkflowState) (*CharacterExtractWorkflowState, error) {
	return func(ctx context.Context, state *CharacterExtractWorkflowState) (*CharacterExtractWorkflowState, error) {
		fmt.Println("\n========== 提取人物信息 ==========")

		// 构建文件路径
		filePath := filepath.Join("data", "content", fmt.Sprintf("%d-%s", state.NovelID, state.NovelName), "character", "提取.json")

		// 尝试从文件加载（断点续传）
		if state.ExtractedCharacters == "" {
			if existingContent, err := loadFromFile(filePath); err == nil && existingContent != "" {
				fmt.Println("检测到已有提取结果，使用断点续传...")
				state.ExtractedCharacters = existingContent
				state.CurrentNode = "extract_success"
				return state, nil
			}
		}

		// 调用 Agent 执行提取
		fmt.Println("开始提取人物信息...")
		executor := GetAgentExecutor()
		output, err := executor.ExtractCharactersJSON(ctx, chatModel, state.Outline, state.Genre)
		if err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "extract_failed"
			return state, fmt.Errorf("提取人物失败: %w", err)
		}

		// 将结果转换为 JSON 字符串
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "extract_failed"
			return state, fmt.Errorf("序列化 JSON 失败: %w", err)
		}

		state.ExtractedCharacters = string(jsonBytes)

		// 保存到文件（用于断点续传）
		if err := saveToFile(filePath, state.ExtractedCharacters); err != nil {
			fmt.Printf("警告：保存文件失败: %v\n", err)
		}

		state.CurrentNode = "extract_success"
		state.RetryCount = 0 // 重置重试计数
		fmt.Printf("提取成功，共提取 %d 个人物\n", len(output.Characters))

		return state, nil
	}
}

// saveToDBNode 保存到数据库节点
//
// 参数：
//   - characterRepo: 人物仓储
//
// 返回：
//   - func: 节点处理函数
func saveToDBNode(characterRepo repository.CharacterRepository) func(context.Context, *CharacterExtractWorkflowState) (*CharacterExtractWorkflowState, error) {
	return func(ctx context.Context, state *CharacterExtractWorkflowState) (*CharacterExtractWorkflowState, error) {
		fmt.Println("\n========== 保存人物到数据库 ==========")

		// 解析 JSON 数据
		var output agents.CharacterExtractorOutput
		if err := json.Unmarshal([]byte(state.ExtractedCharacters), &output); err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "save_failed"
			return state, fmt.Errorf("解析 JSON 失败: %w", err)
		}

		// 转换为 model.Character 对象
		var characters []*model.Character
		for _, char := range output.Characters {
			character := &model.Character{
				NovelID:       fmt.Sprintf("%d", state.NovelID),
				Name:          char.Name,
				Role:          char.Role,
				Age:           char.Age,
				Gender:        char.Gender,
				Appearance:    char.Appearance,
				Personality:   char.Personality,
				Background:    char.Background,
				Relationships: char.Relationships,
			}
			characters = append(characters, character)
		}

		// 批量插入到数据库
		if err := characterRepo.CreateBatch(characters); err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "save_failed"
			return state, fmt.Errorf("保存人物失败: %w", err)
		}

		// 收集保存后的 ID
		for _, char := range characters {
			state.SavedCharacterIDs = append(state.SavedCharacterIDs, char.ID)
		}

		state.CurrentNode = "save_success"
		state.RetryCount = 0 // 重置重试计数
		fmt.Printf("保存成功，共保存 %d 个人物\n", len(characters))

		return state, nil
	}
}

// ==================== 人物修改工作流节点 ====================

// initRevisionNode 初始化修改工作流节点
//
// 参数：
//   - novelRepo: Novel 仓储接口
//   - characterRepo: 人物仓储
//
// 返回：
//   - func: 节点处理函数
func initRevisionNode(novelRepo NovelRepository, characterRepo repository.CharacterRepository) func(context.Context, *CharacterRevisionWorkflowInput) (*CharacterRevisionWorkflowState, error) {
	return func(ctx context.Context, input *CharacterRevisionWorkflowInput) (*CharacterRevisionWorkflowState, error) {
		fmt.Println("\n========== 初始化人物修改工作流 ==========")

		// 查询小说信息
		novel, err := novelRepo.GetByID(input.NovelID)
		if err != nil {
			return nil, fmt.Errorf("查询小说失败: %w", err)
		}

		// 查询该小说的人物
		var characters []*model.Character

		if input.CharacterID != "" {
			// 如果指定了角色ID，只查询该角色
			character, err := characterRepo.GetByID(input.CharacterID)
			if err != nil {
				return nil, fmt.Errorf("查询角色失败: %w", err)
			}
			if character == nil {
				return nil, fmt.Errorf("角色不存在: %s", input.CharacterID)
			}
			characters = []*model.Character{character}
		} else {
			// 否则查询该小说的所有人物
			characters, err = characterRepo.GetByNovelID(fmt.Sprintf("%d", input.NovelID))
			if err != nil {
				return nil, fmt.Errorf("查询人物失败: %w", err)
			}
		}

		if len(characters) == 0 {
			return nil, fmt.Errorf("该小说没有人物信息，请先执行人物提取")
		}

		// 将人物转换为 JSON 格式
		var characterItems []agents.CharacterRevisionItem
		for _, char := range characters {
			item := agents.CharacterRevisionItem{
				ID:            char.ID,
				Name:          char.Name,
				Role:          char.Role,
				Age:           char.Age,
				Gender:        char.Gender,
				Appearance:    char.Appearance,
				Personality:   char.Personality,
				Background:    char.Background,
				Relationships: char.Relationships,
			}
			characterItems = append(characterItems, item)
		}

		jsonBytes, err := json.MarshalIndent(map[string]interface{}{
			"characters": characterItems,
		}, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("序列化 JSON 失败: %w", err)
		}

		// 初始化工作流状态
		state := &CharacterRevisionWorkflowState{
			CurrentNode:        "init",
			NovelID:            input.NovelID,
			CharacterID:        input.CharacterID,
			NovelName:          novel.Title,
			Author:             novel.Author,
			Genre:              novel.Genre,
			Outline:            novel.Outline,
			ExistingCharacters: string(jsonBytes),
			UserSuggestion:     input.UserSuggestion,
			RevisedCharacters:  "",
			SavedCharacterIDs:  []string{},
			ErrorMsg:           "",
			RetryCount:         0,
			MaxRetries:         3,
		}

		fmt.Printf("小说ID: %d\n", state.NovelID)
		fmt.Printf("小说名称: %s\n", state.NovelName)
		if input.CharacterID != "" {
			fmt.Printf("指定角色ID: %s\n", state.CharacterID)
		}
		fmt.Printf("现有人物数量: %d\n", len(characters))
		fmt.Printf("用户建议: %s\n", state.UserSuggestion)

		return state, nil
	}
}

// reviewNode 人物审核节点
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型
//
// 返回：
//   - func: 节点处理函数
func reviewNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterRevisionWorkflowState) (*CharacterRevisionWorkflowState, error) {
	return func(ctx context.Context, state *CharacterRevisionWorkflowState) (*CharacterRevisionWorkflowState, error) {
		fmt.Println("\n========== 审核人物信息 ==========")

		// 构建文件路径
		filePath := filepath.Join("data", "content", fmt.Sprintf("%d-%s", state.NovelID, state.NovelName), "character", "审核.json")

		// 如果指定了角色ID，添加提示信息
		if state.CharacterID != "" {
			fmt.Printf("仅审核角色: %s\n", state.CharacterID)
		}

		// 调用 Agent 执行审核
		fmt.Println("开始审核人物信息...")
		executor := GetAgentExecutor()
		output, err := executor.ReviewCharactersJSON(ctx, chatModel, state.Outline, state.Genre, state.ExistingCharacters)
		if err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "review_failed"
			return state, fmt.Errorf("审核人物失败: %w", err)
		}

		// 将结果转换为 JSON 字符串
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "review_failed"
			return state, fmt.Errorf("序列化 JSON 失败: %w", err)
		}

		state.ReviewSuggestions = string(jsonBytes)

		// 保存到文件
		if err := saveToFile(filePath, state.ReviewSuggestions); err != nil {
			fmt.Printf("警告：保存文件失败: %v\n", err)
		}

		state.CurrentNode = "review_success"
		state.RetryCount = 0 // 重置重试计数
		fmt.Printf("审核完成，共 %d 条审核意见\n", len(output.Issues))

		return state, nil
	}
}

// reviseNode 人物优化节点
//
// 参数：
//   - ctx: 上下文对象
//   - chatModel: 聊天模型
//
// 返回：
//   - func: 节点处理函数
func reviseNode(ctx context.Context, chatModel einoModel.BaseChatModel) func(context.Context, *CharacterRevisionWorkflowState) (*CharacterRevisionWorkflowState, error) {
	return func(ctx context.Context, state *CharacterRevisionWorkflowState) (*CharacterRevisionWorkflowState, error) {
		fmt.Println("\n========== 优化人物信息 ==========")

		// 构建文件路径
		filePath := filepath.Join("data", "content", fmt.Sprintf("%d-%s", state.NovelID, state.NovelName), "character", "修改.json")

		// 如果指定了角色ID，添加提示信息
		if state.CharacterID != "" {
			fmt.Printf("仅优化角色: %s\n", state.CharacterID)
		}

		// 合并用户建议和审核意见
		combinedSuggestion := state.UserSuggestion
		if state.ReviewSuggestions != "" {
			combinedSuggestion = fmt.Sprintf("用户建议：\n%s\n\n审核意见：\n%s", state.UserSuggestion, state.ReviewSuggestions)
		}

		// 调用 Agent 执行优化
		fmt.Println("开始优化人物信息...")
		executor := GetAgentExecutor()
		output, err := executor.ReviseCharactersJSON(ctx, chatModel, state.Outline, state.Genre, state.ExistingCharacters, combinedSuggestion)
		if err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "revise_failed"
			return state, fmt.Errorf("优化人物失败: %w", err)
		}

		// 将结果转换为 JSON 字符串
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "revise_failed"
			return state, fmt.Errorf("序列化 JSON 失败: %w", err)
		}

		state.RevisedCharacters = string(jsonBytes)

		// 保存到文件
		if err := saveToFile(filePath, state.RevisedCharacters); err != nil {
			fmt.Printf("警告：保存文件失败: %v\n", err)
		}

		state.CurrentNode = "revise_success"
		state.RetryCount = 0 // 重置重试计数
		fmt.Printf("优化成功，共 %d 个人物\n", len(output.Characters))

		return state, nil
	}
}

// saveOrUpdateDBNode 保存或更新到数据库节点
//
// 参数：
//   - characterRepo: 人物仓储
//
// 返回：
//   - func: 节点处理函数
func saveOrUpdateDBNode(characterRepo repository.CharacterRepository) func(context.Context, *CharacterRevisionWorkflowState) (*CharacterRevisionWorkflowState, error) {
	return func(ctx context.Context, state *CharacterRevisionWorkflowState) (*CharacterRevisionWorkflowState, error) {
		fmt.Println("\n========== 保存/更新人物到数据库 ==========")

		// 解析 JSON 数据
		var output agents.CharacterRevisionOutput
		if err := json.Unmarshal([]byte(state.RevisedCharacters), &output); err != nil {
			state.ErrorMsg = err.Error()
			state.CurrentNode = "save_failed"
			return state, fmt.Errorf("解析 JSON 失败: %w", err)
		}

		// 分类处理：有 ID 的更新，无 ID 的新增
		var toUpdate []*model.Character
		var toCreate []*model.Character

		for _, char := range output.Characters {
			character := &model.Character{
				ID:            char.ID,
				NovelID:       fmt.Sprintf("%d", state.NovelID),
				Name:          char.Name,
				Role:          char.Role,
				Age:           char.Age,
				Gender:        char.Gender,
				Appearance:    char.Appearance,
				Personality:   char.Personality,
				Background:    char.Background,
				Relationships: char.Relationships,
			}

			if char.ID != "" {
				// 有 ID，执行更新
				toUpdate = append(toUpdate, character)
			} else {
				// 无 ID，执行新增
				toCreate = append(toCreate, character)
			}
		}

		// 执行更新
		for _, char := range toUpdate {
			if err := characterRepo.Update(char); err != nil {
				state.ErrorMsg = err.Error()
				state.CurrentNode = "save_failed"
				return state, fmt.Errorf("更新人物失败: %w", err)
			}
			state.SavedCharacterIDs = append(state.SavedCharacterIDs, char.ID)
		}

		// 执行新增
		if len(toCreate) > 0 {
			if err := characterRepo.CreateBatch(toCreate); err != nil {
				state.ErrorMsg = err.Error()
				state.CurrentNode = "save_failed"
				return state, fmt.Errorf("新增人物失败: %w", err)
			}
			for _, char := range toCreate {
				state.SavedCharacterIDs = append(state.SavedCharacterIDs, char.ID)
			}
		}

		state.CurrentNode = "save_success"
		state.RetryCount = 0 // 重置重试计数
		fmt.Printf("保存/更新成功，更新 %d 个，新增 %d 个\n", len(toUpdate), len(toCreate))

		return state, nil
	}
}

// ==================== 工具函数 ====================

// saveToFile 保存内容到文件
func saveToFile(filePath, content string) error {
	// 创建目录
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// loadFromFile 从文件加载内容
func loadFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
