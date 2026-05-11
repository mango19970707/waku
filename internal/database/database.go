package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	// Host 数据库主机地址
	Host string

	// Port 数据库端口
	Port int

	// User 数据库用户名
	User string

	// Password 数据库密码
	Password string

	// DBName 数据库名称
	DBName string

	// SSLMode SSL 模式（disable、require 等）
	SSLMode string

	// LogLevel 日志级别（Silent、Error、Warn、Info）
	LogLevel logger.LogLevel
}

// Connect 连接到 PostgreSQL 数据库
//
// 参数：
//   - cfg: 数据库配置
//
// 返回值：
//   - *gorm.DB: GORM 数据库实例
//   - error: 连接失败时返回错误
func Connect(cfg *Config) (*gorm.DB, error) {
	// 构建 DSN（数据源名称）
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogLevel),
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层的 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(3600)      // 连接最大生命周期（秒）

	log.Println("数据库连接成功")
	return db, nil
}

// Close 关闭数据库连接
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - error: 关闭失败时返回错误
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭数据库连接失败: %w", err)
	}

	log.Println("数据库连接已关闭")
	return nil
}
