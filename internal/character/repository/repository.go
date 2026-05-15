package repository

import "waku/internal/model"

// Repository 人物仓储接口（用于依赖注入和测试）
type Repository interface {
	CharacterRepository
}

// repository 仓储实现
type repository struct {
	CharacterRepository
}

// NewRepository 创建仓储实例
//
// 参数：
//   - characterRepo: 人物仓储实例
//
// 返回：
//   - Repository: 仓储接口实例
func NewRepository(characterRepo CharacterRepository) Repository {
	return &repository{
		CharacterRepository: characterRepo,
	}
}

// CharacterRepositoryInterface 人物仓储接口（用于 mock）
type CharacterRepositoryInterface interface {
	CreateBatch(characters []*model.Character) error
	GetByNovelID(novelID string) ([]*model.Character, error)
	GetByID(id string) (*model.Character, error)
	Update(character *model.Character) error
	Delete(id string) error
	DeleteByNovelID(novelID string) error
}
