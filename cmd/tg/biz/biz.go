package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"go-workflow/pkgs/llm/openai"
	"go-workflow/pkgs/notify/feishu"
	"go-workflow/pkgs/notify/telegram"
)

// Analyzer 金融消息分析器
type Analyzer struct {
	llmClient    *openai.Client
	feishuBot    *feishu.FeishuBot
	model        string
	scoreTrigger int
}

// AnalyzerOption 分析器配置选项
type AnalyzerOption func(*Analyzer)

// WithModel 设置模型
func WithModel(model string) AnalyzerOption {
	return func(a *Analyzer) {
		a.model = model
	}
}

// WithScoreTrigger 设置触发分数阈值
func WithScoreTrigger(score int) AnalyzerOption {
	return func(a *Analyzer) {
		a.scoreTrigger = score
	}
}

// NewAnalyzer 创建分析器
func NewAnalyzer(llmClient *openai.Client, feishuBot *feishu.FeishuBot, opts ...AnalyzerOption) *Analyzer {
	a := &Analyzer{
		llmClient:    llmClient,
		feishuBot:    feishuBot,
		model:        "gpt-4o",
		scoreTrigger: 8,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Score       int      `json:"score"`       // 影响程度分数 0-10
	Summary     string   `json:"summary"`     // 简要摘要
	Category    string   `json:"category"`    // 影响类别（如：货币政策、地缘政治、行业动态等）
	Impact      []string `json:"impact"`      // 影响方向（如：利多、利空、中性）
	Explanation string   `json:"explanation"` // 详细解释
}

// systemPrompt 系统提示词
const systemPrompt = `你是一位资深金融分析师，专门筛选对股市和加密货币市场有重大影响的新闻。

【核心原则】
- 你的目标是过滤噪音，只保留真正重要的新闻
- 评分必须保守，80%以上的新闻应该在0-4分
- 只有直接影响市场定价的新闻才能获得高分
- 观点、预测、分析类文章通常不配高分

【评分标准（严格执行）】

0-2分（噪音）：大多数新闻属于此类
- 行业会议、论坛发言、专家观点
- 企业常规公告（财报预告、人事变动、产品发布）
- 市场分析、研报解读、技术分析
- 政策解读、市场评论、观点文章
- 价格波动报道（如"比特币突破XX美元"）

3-4分（轻度影响）：
- 央行官员讲话、政策吹风
- 二线经济数据（PMI、零售销售等）
- 行业监管草案征求意见
- 重要企业财报（非超预期）

5-6分（中度影响）：
- 央行利率决议（符合预期）
- 一线经济数据超预期（CPI、非农、GDP）
- 主要国家财政政策调整
- 行业重大监管政策落地

7-8分（重要新闻）：
- 美联储/欧央行意外加息或降息
- 重大地缘政治事件（战争爆发、制裁升级）
- 主要经济体政策重大转向
- 加密货币监管重大变化（ETF批准、禁令）

9-10分（重大新闻，极少见）：
- 金融危机级别的黑天鹅事件
- 主要央行政策方向性逆转
- 战争爆发、政权更迭等重大地缘事件
- 加密货币交易所倒闭、重大安全事件

【高分反例】（这些不配高分）
- "某分析师预测比特币将涨至XX" → 0-1分
- "某机构发布行业报告" → 0-1分
- "某官员发表讲话" → 1-2分
- "某公司发布新产品" → 1-2分
- "市场回顾/周报" → 0分

请严格按照以下 JSON 格式输出分析结果，不要输出其他内容：
{
  "score": <0-10的整数，保守评分>,
  "summary": "<一句话摘要>",
  "impact": [
    "XXX利多",
    "XXX利空"
  ],
  "category": "<影响类别>",
  "explanation": "<120字以内解释影响逻辑>"
}`

// Analyze 分析消息
func (a *Analyzer) Analyze(ctx context.Context, msg *telegram.ChannelMessage) (*AnalysisResult, error) {
	userPrompt := fmt.Sprintf("请分析以下新闻：\n\n来源：%s\n内容：\n%s", msg.ChannelName, msg.Text)

	resp, err := a.llmClient.ChatCompletion(ctx, &openai.ChatCompletionRequest{
		Model: a.model,
		Messages: []openai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("调用 LLM 失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空响应")
	}

	content := resp.Choices[0].Message.Content

	// 解析 JSON 结果
	result, err := parseAnalysisResult(content)
	if err != nil {
		return nil, fmt.Errorf("解析分析结果失败: %w", err)
	}

	return result, nil
}

// parseAnalysisResult 解析分析结果
func parseAnalysisResult(content string) (*AnalysisResult, error) {
	// 提取 JSON 部分（处理可能存在的 markdown 代码块或思考内容）
	jsonStr := extractJSON(content)

	var result AnalysisResult
	if err := json.NewDecoder(strings.NewReader(jsonStr)).Decode(&result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, content: %s", err, jsonStr)
	}

	// 校验分数范围
	if result.Score < 0 {
		result.Score = 0
	} else if result.Score > 10 {
		result.Score = 10
	}

	return &result, nil
}

// ShouldNotify 判断是否需要通知
func (a *Analyzer) ShouldNotify(result *AnalysisResult) bool {
	return result.Score >= a.scoreTrigger
}

// Notify 发送飞书通知
func (a *Analyzer) Notify(ctx context.Context, msg *telegram.ChannelMessage, result *AnalysisResult) error {
	// 根据分数选择卡片颜色
	template := "blue" // 默认蓝色
	switch {
	case result.Score >= 9:
		template = "red" // 红色 - 重大
	case result.Score >= 7:
		template = "orange" // 橙色 - 重要
	case result.Score >= 5:
		template = "yellow" // 黄色 - 一般
	}

	card := feishu.NewCard().
		SetHeader(fmt.Sprintf("金融快讯 (影响分数: %d/10)", result.Score), template).
		AddText(fmt.Sprintf("来源：%s", msg.ChannelName)).
		AddText(fmt.Sprintf("摘要：%s", result.Summary)).
		AddText(fmt.Sprintf("类别：%s", result.Category))

	// 格式化影响方向，区分利多/利空
	if len(result.Impact) > 0 {
		card.AddText(fmt.Sprintf("影响：%s", formatImpact(result.Impact)))
	}

	card.AddText(fmt.Sprintf("分析：%s", result.Explanation)).
		AddText(fmt.Sprintf("原文：%s", truncateText(msg.Text, 500)))

	if err := a.feishuBot.SendCard(card); err != nil {
		return fmt.Errorf("发送飞书通知失败: %w", err)
	}

	return nil
}

// formatImpact 格式化影响方向，添加图标区分利多/利空
func formatImpact(impacts []string) string {
	var formatted []string
	for _, impact := range impacts {
		switch {
		case strings.Contains(impact, "利多"):
			formatted = append(formatted, "📈 "+impact)
		case strings.Contains(impact, "利空"):
			formatted = append(formatted, "📉 "+impact)
		default:
			formatted = append(formatted, "➖ "+impact)
		}
	}
	return strings.Join(formatted, " | ")
}

// extractJSON 从内容中提取 JSON 字符串
// 处理多种情况：markdown 代码块、思考内容后的 JSON、纯 JSON
func extractJSON(content string) string {
	// 1. 尝试提取 ```json 代码块
	if idx := strings.Index(content, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// 2. 尝试提取 ``` 代码块（非 json 标记）
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + 3
		// 跳过可能的语言标记行
		if newlineIdx := strings.Index(content[start:], "\n"); newlineIdx != -1 && newlineIdx < 20 {
			start += newlineIdx + 1
		}
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// 3. 尝试查找 JSON 对象（从 { 开始到最后的 } 结束）
	if start := strings.Index(content, "{"); start != -1 {
		// 找到最后一个 }
		if end := strings.LastIndex(content, "}"); end != -1 && end > start {
			return strings.TrimSpace(content[start : end+1])
		}
	}

	// 4. 原样返回
	return strings.TrimSpace(content)
}

// truncateText 截断文本
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// Process 处理单条消息
func (a *Analyzer) Process(ctx context.Context, msg *telegram.ChannelMessage) {
	slog.Info("开始分析消息",
		"channel", msg.ChannelName,
		"message_id", msg.MessageID,
		"text_preview", truncateText(msg.Text, 50),
	)

	result, err := a.Analyze(ctx, msg)
	if err != nil {
		slog.Error("分析消息失败", "error", err, "channel", msg.ChannelName)
		return
	}

	slog.Info("分析完成",
		"channel", msg.ChannelName,
		"score", result.Score,
		"summary", result.Summary,
		"impact", strings.Join(result.Impact, ", "),
	)

	if a.ShouldNotify(result) {
		slog.Info("触发通知阈值，发送飞书通知", "score", result.Score)
		if err := a.Notify(ctx, msg, result); err != nil {
			slog.Error("发送通知失败", "error", err)
			return
		}
		slog.Info("飞书通知发送成功")
	}
}

// Run 运行消息处理器
func (a *Analyzer) Run(ctx context.Context, msgChan <-chan *telegram.ChannelMessage) {
	slog.Info("分析器启动", "model", a.model, "score_trigger", a.scoreTrigger)

	for {
		select {
		case <-ctx.Done():
			slog.Info("分析器停止")
			return
		case msg, ok := <-msgChan:
			if !ok {
				slog.Info("消息通道已关闭，分析器停止")
				return
			}
			a.Process(ctx, msg)
		}
	}
}
