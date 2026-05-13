package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	// Driver 数据库驱动类型（sqlite, postgres）
	Driver string

	// Host 数据库主机地址（PostgreSQL）
	Host string

	// Port 数据库端口（PostgreSQL）
	Port int

	// User 数据库用户名（PostgreSQL）
	User string

	// Password 数据库密码（PostgreSQL）
	Password string

	// DBName 数据库名称（PostgreSQL）
	DBName string

	// SSLMode SSL 模式（PostgreSQL: disable、require 等）
	SSLMode string

	// DataPath SQLite 数据文件路径
	DataPath string

	// LogLevel 日志级别（Silent、Error、Warn、Info）
	LogLevel logger.LogLevel
}

// Connect 连接到数据库（支持 SQLite 和 PostgreSQL）
//
// 参数：
//   - cfg: 数据库配置
//
// 返回值：
//   - *gorm.DB: GORM 数据库实例
//   - error: 连接失败时返回错误
func Connect(cfg *Config) (*gorm.DB, error) {
	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogLevel),
	}

	var db *gorm.DB
	var err error

	// 根据驱动类型选择连接方式
	switch cfg.Driver {
	case "sqlite":
		db, err = connectSQLite(cfg, gormConfig)
	case "postgres":
		db, err = connectPostgres(cfg, gormConfig)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}

	if err != nil {
		return nil, err
	}

	// 获取底层的 sql.DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)      // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)     // 最大打开连接数
	sqlDB.SetConnMaxLifetime(3600) // 连接最大生命周期（秒）

	slog.Info("数据库连接成功", "driver", cfg.Driver)
	return db, nil
}

// connectSQLite 连接到 SQLite 数据库
func connectSQLite(cfg *Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	// 确保数据目录存在
	dataDir := filepath.Dir(cfg.DataPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 连接 SQLite 数据库
	db, err := gorm.Open(sqlite.Open(cfg.DataPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接 SQLite 数据库失败: %w", err)
	}

	return db, nil
}

// connectPostgres 连接到 PostgreSQL 数据库
func connectPostgres(cfg *Config, gormConfig *gorm.Config) (*gorm.DB, error) {
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

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接 PostgreSQL 数据库失败: %w", err)
	}

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

	slog.Info("数据库连接已关闭")
	return nil
}
