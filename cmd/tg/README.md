# Telegram 金融快讯分析器

监听 Telegram 指定频道的消息，通过大模型分析金融影响程度，高分消息自动推送飞书通知。

## 编译并运行

```bash
go build -o bin/tg ./cmd/tg && ./bin/tg
```

## 配置

在 `bin/.tg.env` 文件中配置以下环境变量：

```bash
# Telegram 配置
TG_APP_ID=xx
TG_APP_HASH=xx
TG_PHONE=xx
TG_SESSION_PATH=bin/session.json

# LLM 配置（OpenAI 兼容接口）
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=xx
LLM_MODEL=gpt-4o

# 飞书配置
FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/xx
FEISHU_SECRET=xx

# 触发分数阈值（默认 8）
SCORE_TRIGGER=8
```

## 工作流程

1. 监听 Telegram 指定频道的消息
2. 将消息发送给大模型进行金融影响分析
3. 大模型返回 0-10 分的影响程度评分
4. 当分数 >= 阈值时，通过飞书发送通知

## 评分标准

- 0-3 分：影响较小，常规新闻或市场噪音
- 4-6 分：有一定影响，可能影响特定板块或资产
- 7-8 分：重要新闻，可能引发市场波动
- 9-10 分：重大新闻，可能引发市场剧烈波动或趋势转折