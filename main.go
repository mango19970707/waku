package main

import (
	"flag"
	"log"

	"waku/internal/config"
	"waku/internal/database"
)

func init() {

}

func main() {
	// 定义命令行参数
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	migrate := flag.Bool("migrate", false, "执行数据库迁移")
	reset := flag.Bool("reset", false, "重置数据库（危险操作）")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 数据库配置
	dbCfg := &database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		LogLevel: cfg.Database.GetLogLevel(),
	}

	// 连接数据库
	db, err := database.Connect(dbCfg)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer database.Close(db)

	// 根据命令行参数执行操作
	if *reset {
		// 重置数据库
		if err := database.ResetDatabase(db); err != nil {
			log.Fatalf("重置数据库失败: %v", err)
		}
		log.Println("数据库重置成功")
		return
	}

	if *migrate {
		// 执行迁移
		if err := database.Migrate(db); err != nil {
			log.Fatalf("数据库迁移失败: %v", err)
		}
		log.Println("数据库迁移成功")
		return
	}

	// 默认行为：启动应用
	log.Printf("Waku AI 小说生成系统启动中... (端口: %d, 环境: %s)", cfg.App.Port, cfg.App.Env)
	log.Println("使用 -migrate 执行数据库迁移")
	log.Println("使用 -reset 重置数据库")
	log.Println("使用 -config 指定配置文件路径")
}
