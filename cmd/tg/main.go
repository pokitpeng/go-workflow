package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go-workflow/cmd/tg/biz"
	"go-workflow/pkgs/llm/openai"
	"go-workflow/pkgs/notify/feishu"
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

	// 从环境变量读取 Telegram 配置
	appID, _ := strconv.Atoi(os.Getenv("TG_APP_ID"))
	appHash := os.Getenv("TG_APP_HASH")
	phone := os.Getenv("TG_PHONE")
	sessionPath := os.Getenv("TG_SESSION_PATH")
	if sessionPath == "" {
		sessionPath = "bin/session.json"
	}

	// 从环境变量读取 LLM 配置
	llmBaseURL := os.Getenv("LLM_BASE_URL")
	llmAPIKey := os.Getenv("LLM_API_KEY")
	llmModel := os.Getenv("LLM_MODEL")
	if llmModel == "" {
		llmModel = "gpt-4o"
	}

	// 从环境变量读取飞书配置
	feishuWebhook := os.Getenv("FEISHU_WEBHOOK")
	feishuSecret := os.Getenv("FEISHU_SECRET")

	// 从环境变量读取触发分数阈值
	scoreTrigger := 5
	if v := os.Getenv("SCORE_TRIGGER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			scoreTrigger = n
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 创建 LLM 客户端
	llmClient := openai.NewClient(llmBaseURL, llmAPIKey)

	// 创建飞书机器人
	var feishuBot *feishu.FeishuBot
	if feishuSecret != "" {
		feishuBot = feishu.NewFeishuBot(feishuWebhook, feishu.WithSecret(feishuSecret))
	} else {
		feishuBot = feishu.NewFeishuBot(feishuWebhook)
	}

	// 创建分析器
	analyzer := biz.NewAnalyzer(llmClient, feishuBot,
		biz.WithModel(llmModel),
		biz.WithScoreTrigger(scoreTrigger),
	)

	// 创建 Telegram 监听器
	listener := telegram.NewListener(appID, appHash, phone,
		telegram.WithSessionPath(sessionPath),
		telegram.WithChannels(
			1693268862, // 金十数据
			// 1549184965, // 金色财经
			// 1668307169, // 币圈新闻即时快讯
			// 1387109317, // Blockbeats
		),
	)

	// 获取消息 channel
	msgChan := listener.Messages()

	// 启动分析器
	go analyzer.Run(ctx, msgChan)

	// 运行监听器
	slog.Info("启动 Telegram 监听器...",
		"model", llmModel,
		"score_trigger", scoreTrigger,
	)
	if err := listener.Run(ctx); err != nil {
		slog.Error("监听器运行失败", "error", err)
		os.Exit(1)
	}
}
