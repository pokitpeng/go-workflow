package main

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config 配置结构体
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	App    AppConfig    `mapstructure:"app"`
	Log    LogConfig    `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type AppConfig struct {
	Name  string `mapstructure:"name"`
	Debug bool   `mapstructure:"debug"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	mu     sync.RWMutex
	config *Config
	viper  *viper.Viper
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configPath string) (*ConfigManager, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 首次读取配置
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	cm := &ConfigManager{
		config: &cfg,
		viper:  v,
	}

	// 设置热更新
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("检测到配置文件变化", "event", e.Name, "op", e.Op.String())

		var newCfg Config
		if err := v.Unmarshal(&newCfg); err != nil {
			slog.Error("配置解析失败，保留旧配置", "error", err)
			return
		}

		cm.mu.Lock()
		cm.config = &newCfg
		cm.mu.Unlock()

		slog.Info("配置热更新成功", "config", fmt.Sprintf("%+v", newCfg))
	})

	return cm, nil
}

// GetConfig 获取当前配置（线程安全）
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// 命令行参数
	configPath := flag.StringP("config", "c", "config.yaml", "配置文件路径")
	flag.Parse()

	// 初始化配置管理器
	cm, err := NewConfigManager(*configPath)
	if err != nil {
		slog.Error("初始化配置失败", "error", err)
		os.Exit(1)
	}

	slog.Info("配置加载成功", "config", fmt.Sprintf("%+v", cm.GetConfig()))

	// 模拟应用运行，定期打印当前配置
	slog.Info("应用启动，修改配置文件将自动热更新...")
	slog.Info("按 Ctrl+C 退出")

	// 简单的运行循环，展示配置使用
	ticker := make(chan struct{})
	go func() {
		for range ticker {
			cfg := cm.GetConfig()
			slog.Info("当前配置",
				"app.name", cfg.App.Name,
				"server.port", cfg.Server.Port,
				"debug", cfg.App.Debug,
				"log.level", cfg.Log.Level,
			)
		}
	}()

	// 阻塞主线程
	select {}
}
