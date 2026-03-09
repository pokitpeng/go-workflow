package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go-workflow/pkgs/notify/telegram"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

func main() {
	// 初始化slog
	handler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.DateTime,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// 加载 .tg.env 文件
	if _, err := os.Stat("bin/.tg.env"); err == nil {
		godotenv.Load("bin/.tg.env")
	} else {
		slog.Error("加载 .tg.env 文件失败", "error", err)
		os.Exit(1)
	}

	// 从环境变量读取配置
	appID, _ := strconv.Atoi(os.Getenv("TG_APP_ID"))
	appHash := os.Getenv("TG_APP_HASH")
	phone := os.Getenv("TG_PHONE")
	sessionPath := os.Getenv("TG_SESSION_PATH")
	if sessionPath == "" {
		sessionPath = "bin/session.json"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 创建监听器
	listener := telegram.NewListener(appID, appHash, phone,
		telegram.WithSessionPath(sessionPath),
		telegram.WithChannels(
			1693268862, // 金十数据
			1549184965, // 金色财经
			1668307169, // 币圈新闻即时快讯
			1387109317, // Blockbeats
		),
	)

	// 获取消息 channel
	msgChan := listener.Messages()

	// 启动消费者
	go func() {
		for msg := range msgChan {
			slog.Info("收到频道消息",
				"channel_id", msg.ChannelID,
				"channel_name", msg.ChannelName,
				"message_id", msg.MessageID,
				"text", msg.Text,
			)
		}
		slog.Info("消息 channel 已关闭")
	}()

	// 运行监听器
	slog.Info("启动 Telegram 监听器...")
	if err := listener.Run(ctx); err != nil {
		slog.Error("监听器运行失败", "error", err)
		os.Exit(1)
	}
}
