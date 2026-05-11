package novel

import (
	"fmt"
	"time"
	"waku/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository Novel 数据仓库接口
type Repository interface {
	// Create 创建小说
	Create(novel *model.Novel) error

	// GetByID 根据 ID 获取小说
	GetByID(id string) (*model.Novel, error)

	// GetAll 获取所有小说（支持分页）
	GetAll(offset, limit int) ([]*model.Novel, error)

	// GetByGenre 根据类型获取小说列表
	GetByGenre(genre string, offset, limit int) ([]*model.Novel, error)

	// GetByStatus 根据状态获取小说列表
	GetByStatus(status string, offset, limit int) ([]*model.Novel, error)

	// Update 更新小说信息
	Update(novel *model.Novel) error

	// UpdateStatus 更新小说状态
	UpdateStatus(id, status string) error

	// Delete 删除小说（软删除）
	Delete(id string) error

	// HardDelete 永久删除小说
	HardDelete(id string) error

	// Count 统计小说总数
	Count() (int64, error)

	// CountByGenre 统计指定类型的小说数量
	CountByGenre(genre string) (int64, error)

	// Search 搜索小说（按标题或作者）
	Search(keyword string, offset, limit int) ([]*model.Novel, error)
}

// repository Novel 数据仓库实现
type repository struct {
	db *gorm.DB
}

// NewRepository 创建 Novel 数据仓库实例
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - Repository: Novel 数据仓库接口
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create 创建小说
//
// 参数：
//   - novel: 小说对象
//
// 返回值：
//   - error: 创建失败时返回错误
func (r *repository) Create(novel *model.Novel) error {
	// 如果没有提供 ID，自动生成 UUID
	if novel.ID == "" {
		novel.ID = uuid.New().String()
	}

	// 设置创建时间和更新时间
	now := time.Now()
	novel.CreatedAt = now
	novel.UpdatedAt = now

	// 设置默认状态
	if novel.Status == "" {
		novel.Status = "draft"
	}

	// 执行创建操作
	if err := r.db.Create(novel).Error; err != nil {
		return fmt.Errorf("创建小说失败: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取小说
//
// 参数：
//   - id: 小说 ID
//
// 返回值：
//   - *model.Novel: 小说对象
//   - error: 获取失败时返回错误
func (r *repository) GetByID(id string) (*model.Novel, error) {
	var novel model.Novel

	if err := r.db.Where("id = ?", id).First(&novel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("小说不存在: %s", id)
		}
		return nil, fmt.Errorf("获取小说失败: %w", err)
	}

	return &novel, nil
}

// GetAll 获取所有小说（支持分页）
//
// 参数：
//   - offset: 偏移量（从 0 开始）
//   - limit: 每页数量（0 表示不限制）
//
// 返回值：
//   - []*model.Novel: 小说列表
//   - error: 获取失败时返回错误
func (r *repository) GetAll(offset, limit int) ([]*model.Novel, error) {
	var novels []*model.Novel

	query := r.db.Order("created_at DESC")

	if offset > 0 {
		query = query.Offset(offset)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&novels).Error; err != nil {
		return nil, fmt.Errorf("获取小说列表失败: %w", err)
	}

	return novels, nil
}

// GetByGenre 根据类型获取小说列表
//
// 参数：
//   - genre: 小说类型
//   - offset: 偏移量
//   - limit: 每页数量
//
// 返回值：
//   - []*model.Novel: 小说列表
//   - error: 获取失败时返回错误
func (r *repository) GetByGenre(genre string, offset, limit int) ([]*model.Novel, error) {
	var novels []*model.Novel

	query := r.db.Where("genre = ?", genre).Order("created_at DESC")

	if offset > 0 {
		query = query.Offset(offset)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&novels).Error; err != nil {
		return nil, fmt.Errorf("获取小说列表失败: %w", err)
	}

	return novels, nil
}

// GetByStatus 根据状态获取小说列表
//
// 参数：
//   - status: 小说状态（draft、writing、completed）
//   - offset: 偏移量
//   - limit: 每页数量
//
// 返回值：
//   - []*model.Novel: 小说列表
//   - error: 获取失败时返回错误
func (r *repository) GetByStatus(status string, offset, limit int) ([]*model.Novel, error) {
	var novels []*model.Novel

	query := r.db.Where("status = ?", status).Order("created_at DESC")

	if offset > 0 {
		query = query.Offset(offset)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&novels).Error; err != nil {
		return nil, fmt.Errorf("获取小说列表失败: %w", err)
	}

	return novels, nil
}

// Update 更新小说信息
//
// 参数：
//   - novel: 小说对象（必须包含 ID）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *repository) Update(novel *model.Novel) error {
	if novel.ID == "" {
		return fmt.Errorf("小说 ID 不能为空")
	}

	// 更新时间
	novel.UpdatedAt = time.Now()

	// 执行更新操作（只更新非零值字段）
	result := r.db.Model(&model.Novel{}).Where("id = ?", novel.ID).Updates(novel)
	if result.Error != nil {
		return fmt.Errorf("更新小说失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("小说不存在: %s", novel.ID)
	}

	return nil
}

// UpdateStatus 更新小说状态
//
// 参数：
//   - id: 小说 ID
//   - status: 新状态（draft、writing、completed）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *repository) UpdateStatus(id, status string) error {
	result := r.db.Model(&model.Novel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("更新小说状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("小说不存在: %s", id)
	}

	return nil
}

// Delete 删除小说（软删除）
//
// 参数：
//   - id: 小说 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *repository) Delete(id string) error {
	result := r.db.Where("id = ?", id).Delete(&model.Novel{})

	if result.Error != nil {
		return fmt.Errorf("删除小说失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("小说不存在: %s", id)
	}

	return nil
}

// HardDelete 永久删除小说
//
// 参数：
//   - id: 小说 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *repository) HardDelete(id string) error {
	result := r.db.Unscoped().Where("id = ?", id).Delete(&model.Novel{})

	if result.Error != nil {
		return fmt.Errorf("永久删除小说失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("小说不存在: %s", id)
	}

	return nil
}

// Count 统计小说总数
//
// 返回值：
//   - int64: 小说总数
//   - error: 统计失败时返回错误
func (r *repository) Count() (int64, error) {
	var count int64

	if err := r.db.Model(&model.Novel{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计小说总数失败: %w", err)
	}

	return count, nil
}

// CountByGenre 统计指定类型的小说数量
//
// 参数：
//   - genre: 小说类型
//
// 返回值：
//   - int64: 小说数量
//   - error: 统计失败时返回错误
func (r *repository) CountByGenre(genre string) (int64, error) {
	var count int64

	if err := r.db.Model(&model.Novel{}).Where("genre = ?", genre).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计小说数量失败: %w", err)
	}

	return count, nil
}

// Search 搜索小说（按标题或作者）
//
// 参数：
//   - keyword: 搜索关键词
//   - offset: 偏移量
//   - limit: 每页数量
//
// 返回值：
//   - []*model.Novel: 小说列表
//   - error: 搜索失败时返回错误
func (r *repository) Search(keyword string, offset, limit int) ([]*model.Novel, error) {
	var novels []*model.Novel

	// 使用 LIKE 进行模糊搜索（标题或作者）
	searchPattern := "%" + keyword + "%"
	query := r.db.Where("title LIKE ? OR author LIKE ?", searchPattern, searchPattern).
		Order("created_at DESC")

	if offset > 0 {
		query = query.Offset(offset)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&novels).Error; err != nil {
		return nil, fmt.Errorf("搜索小说失败: %w", err)
	}

	return novels, nil
}
