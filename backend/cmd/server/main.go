package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"waku/backend"
)

func main() {
	configPath := os.Getenv("WAKU_CONFIG")
	if configPath == "" {
		configPath = "config/config.yaml"
	}
	server, err := backend.NewServer(context.Background(), configPath)
	if err != nil {
		log.Fatalf("启动后端失败: %v", err)
	}
	if err := server.Router.Run(fmt.Sprintf(":%d", server.Port)); err != nil {
		log.Fatalf("Gin 服务异常退出: %v", err)
	}
}
