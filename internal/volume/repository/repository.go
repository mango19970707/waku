package repository

import "waku/internal/model"

// VolumeRepository 定义分卷数据仓储接口。
type VolumeRepository interface {
	CreateBatch(volumes []*model.Volume) error
	GetByID(id uint) (*model.Volume, error)
	GetByNovelID(novelID uint) ([]*model.Volume, error)
	Update(volume *model.Volume) error
	Delete(id uint) error
	DeleteByNovelID(novelID uint) error
}
