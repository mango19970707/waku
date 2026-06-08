package backend

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appconfig "waku/internal/config"
	"waku/internal/database"
	"waku/internal/utils"
)

type Server struct {
	DB     *gorm.DB
	Router *gin.Engine
}

func NewServer(ctx context.Context, configPath string) (*Server, error) {
	// 服务启动阶段统一加载配置，便于后续前端、脚本和测试复用同一个后端入口。
	cfg, err := appconfig.Load(configPath)
	if err != nil {
		return nil, err
	}
	db, err := database.Connect(&database.Config{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		DataPath: cfg.Database.DataPath,
		LogLevel: cfg.Database.GetLogLevel(),
	})
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(db); err != nil {
		return nil, err
	}
	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("初始化 AI 模型失败: %w", err)
	}
	router := NewRouter(Config{DB: db, WorkflowRunner: NewDefaultWorkflowRunner(db, chatModel)})
	return &Server{DB: db, Router: router}, nil
}
