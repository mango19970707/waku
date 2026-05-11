package utils

import (
	"github.com/cloudwego/eino/components/model"
)

var (
	ChatModel model.BaseChatModel
)

func init() {
	//cfg, err := config.Load("config/config.yaml")
	//if err != nil {
	//	slog.Error("加载配置失败: %v", err)
	//}
	//
	//ChatModel, err = NewChatModel(context.Background(), &cfg.Model)
	//if err != nil {
	//	slog.Error("创建chat model失败: %v", err)
	//}
}
