package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"
)

// Config 应用配置
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	App      AppConfig      `yaml:"app"`
	Model    ModelConfig    `yaml:"model"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `yaml:"driver"`    // sqlite, postgres
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	LogLevel string `yaml:"log_level"` // silent, error, warn, info
	DataPath string `yaml:"data_path"` // SQLite 数据文件路径
}

// AppConfig 应用配置
type AppConfig struct {
	Port int    `yaml:"port"`
	Env  string `yaml:"env"` // development, production
}

// ModelConfig AI 模型配置
type ModelConfig struct {
	Provider      string            `yaml:"provider"`       // openai, azure_openai, ollama, ark, anthropic
	APIKey        string            `yaml:"api_key"`        // API 密钥
	BaseURL       string            `yaml:"base_url"`       // 自定义 API 地址
	Model         string            `yaml:"model"`          // 模型名称
	Timeout       string            `yaml:"timeout"`        // 请求超时时间（如 "60s"）
	APIVersion    string            `yaml:"api_version"`    // Azure OpenAI API 版本
	MaxTokens     int               `yaml:"max_tokens"`     // 最大生成 token 数
	Temperature   float32           `yaml:"temperature"`    // 温度参数 (0.0-1.0)
	CustomHeaders map[string]string `yaml:"custom_headers"` // 自定义请求头
}

// Load 从 YAML 文件加载配置
//
// 参数：
//   - path: 配置文件路径
//
// 返回值：
//   - *Config: 配置对象
//   - error: 加载失败时返回错误
func Load(path string) (*Config, error) {
	// 尝试加载 .env 文件（如果存在）
	_ = godotenv.Load("../../.env")

	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量覆盖配置（优先级：环境变量 > YAML）
	loadFromEnv(&cfg)

	// 设置数据库默认值
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.LogLevel == "" {
		cfg.Database.LogLevel = "info"
	}
	if cfg.Database.DataPath == "" {
		cfg.Database.DataPath = "./data/waku.db"
	}

	// 设置应用默认值
	if cfg.App.Port == 0 {
		cfg.App.Port = 6060
	}
	if cfg.App.Env == "" {
		cfg.App.Env = "development"
	}

	// 设置模型默认值
	if cfg.Model.Provider == "" {
		cfg.Model.Provider = "openai"
	}
	if cfg.Model.Timeout == "" {
		cfg.Model.Timeout = "60s"
	}
	if cfg.Model.MaxTokens == 0 {
		cfg.Model.MaxTokens = 2000
	}
	if cfg.Model.Temperature == 0 {
		cfg.Model.Temperature = 0.7
	}

	return &cfg, nil
}

// loadFromEnv 从环境变量加载配置
//
// 参数：
//   - cfg: 配置对象指针
func loadFromEnv(cfg *Config) {
	// 数据库配置
	if v := os.Getenv("DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := os.Getenv("DB_LOG_LEVEL"); v != "" {
		cfg.Database.LogLevel = v
	}
	if v := os.Getenv("DB_DATA_PATH"); v != "" {
		cfg.Database.DataPath = v
	}

	// 应用配置
	if v := os.Getenv("APP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.App.Port = port
		}
	}
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}

	// 模型配置
	if v := os.Getenv("MODEL_PROVIDER"); v != "" {
		cfg.Model.Provider = v
	}
	if v := os.Getenv("MODEL_API_KEY"); v != "" {
		cfg.Model.APIKey = v
	}
	if v := os.Getenv("MODEL_BASE_URL"); v != "" {
		cfg.Model.BaseURL = v
	}
	if v := os.Getenv("MODEL_NAME"); v != "" {
		cfg.Model.Model = v
	}
	if v := os.Getenv("MODEL_TIMEOUT"); v != "" {
		cfg.Model.Timeout = v
	}
	if v := os.Getenv("MODEL_API_VERSION"); v != "" {
		cfg.Model.APIVersion = v
	}
	if v := os.Getenv("MODEL_MAX_TOKENS"); v != "" {
		if maxTokens, err := strconv.Atoi(v); err == nil {
			cfg.Model.MaxTokens = maxTokens
		}
	}
	if v := os.Getenv("MODEL_TEMPERATURE"); v != "" {
		if temp, err := strconv.ParseFloat(v, 32); err == nil {
			cfg.Model.Temperature = float32(temp)
		}
	}
}

// GetLogLevel 将字符串日志级别转换为 GORM 日志级别
//
// 返回值：
//   - logger.LogLevel: GORM 日志级别
func (c *DatabaseConfig) GetLogLevel() logger.LogLevel {
	switch c.LogLevel {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Info
	}
}

// GetTimeout 将字符串超时时间转换为 time.Duration
//
// 返回值：
//   - time.Duration: 超时时间
//   - error: 解析失败时返回错误
func (c *ModelConfig) GetTimeout() (time.Duration, error) {
	if c.Timeout == "" {
		return 60 * time.Second, nil
	}
	return time.ParseDuration(c.Timeout)
}
