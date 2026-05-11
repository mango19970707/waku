package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"waku/internal/model"
)

// Migrate 执行数据库迁移
//
// 该函数会自动创建或更新所有表结构，使其与模型定义保持一致。
// GORM 的 AutoMigrate 只会添加新字段和索引，不会删除已存在的字段。
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - error: 迁移失败时返回错误
func Migrate(db *gorm.DB) error {
	log.Println("开始执行数据库迁移...")

	// 定义需要迁移的所有模型
	models := []interface{}{
		&model.Novel{},
		&model.Volume{},
		&model.Chapter{},
		&model.Abstract{},
		&model.Character{},
		&model.Plotline{},
		&model.Memory{},
	}

	// 执行自动迁移
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}

// DropAllTables 删除所有表（危险操作，仅用于开发环境）
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - error: 删除失败时返回错误
func DropAllTables(db *gorm.DB) error {
	log.Println("警告：正在删除所有表...")

	// 定义需要删除的所有模型（按依赖关系逆序）
	models := []interface{}{
		&model.Memory{},
		&model.Plotline{},
		&model.Character{},
		&model.Abstract{},
		&model.Chapter{},
		&model.Volume{},
		&model.Novel{},
	}

	// 删除所有表
	if err := db.Migrator().DropTable(models...); err != nil {
		return fmt.Errorf("删除表失败: %w", err)
	}

	log.Println("所有表已删除")
	return nil
}

// ResetDatabase 重置数据库（删除所有表后重新创建）
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - error: 重置失败时返回错误
func ResetDatabase(db *gorm.DB) error {
	log.Println("开始重置数据库...")

	// 删除所有表
	if err := DropAllTables(db); err != nil {
		return err
	}

	// 重新创建所有表
	if err := Migrate(db); err != nil {
		return err
	}

	log.Println("数据库重置完成")
	return nil
}
