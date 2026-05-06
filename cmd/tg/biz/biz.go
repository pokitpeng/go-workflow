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

// AssetImpact 利多/利空单项资产
type AssetImpact struct {
	Symbol string `json:"symbol"` // 标的代码或名称
	Type   string `json:"type"`   // stock / crypto 等
	Reason string `json:"reason"` // 简要理由
}

// AnalysisResult 分析结果（与 systemPrompt 中 JSON schema 对齐）
type AnalysisResult struct {
	Score           int           `json:"score"`            // 影响程度分数 0-10
	Summary         string        `json:"summary"`          // 简要摘要
	Category        string        `json:"category"`         // 影响类别
	Impact          []string      `json:"impact"`           // 影响要点
	Explanation     string        `json:"explanation"`      // 影响逻辑说明
	MarketSentiment string        `json:"market_sentiment"` // bullish / bearish / neutral
	Tradable        bool          `json:"tradable"`         // 是否具备交易价值
	BullishAssets   []AssetImpact `json:"bullish_assets"`   // 看多资产
	BearishAssets   []AssetImpact `json:"bearish_assets"`   // 看空资产
	TradingLogic    string        `json:"trading_logic"`    // 资金可能流向
	Confidence      int           `json:"confidence"`       // 置信度 0-100
}

// systemPrompt 系统提示词
const systemPrompt = `你是一位顶级宏观交易员 + 事件驱动量化分析师，专门从实时新闻中挖掘“可能导致美股或加密货币短期/中期上涨或下跌”的交易机会。

你的核心目标不是“总结新闻”，而是：
1. 判断新闻是否会影响资产价格
2. 找出最可能受影响的美股/行业/加密货币
3. 判断方向（利多 / 利空）
4. 判断影响强度
5. 给出可交易的逻辑

你需要像对冲基金事件驱动交易团队一样思考。

====================
【核心原则】
====================

- 只关注“可能改变市场预期”的新闻
- 只分析真实增量信息
- 不要复读市场情绪
- 不要做空泛宏观评论
- 不要泛泛而谈“利好科技股”
- 必须尽可能具体到：
  - 美股代码
  - 行业
  - 加密货币名称
  - 受益链条

你的任务本质是：
“这条新闻之后，资金最可能买什么？卖什么？”

====================
【重点关注的事件类型】
====================

以下事件优先级最高：

【宏观经济】
- 美联储加息/降息
- CPI / 非农 / GDP / PCE
- 美债收益率剧烈变化
- 流动性政策
- 财政刺激
- 银行业风险

【AI / 科技】
- AI模型重大突破
- GPU供应链变化
- AI资本开支
- 云厂商业绩
- 半导体限制
- 数据中心需求变化

【加密货币】
- ETF批准/拒绝
- SEC监管
- 稳定币政策
- 交易所安全事件
- 链上生态爆发
- 机构资金流入
- 减半相关事件

【地缘政治】
- 战争
- 制裁
- 芯片禁运
- 贸易限制
- 能源危机

【企业】
- 超预期财报
- 指引大幅上修/下修
- 裁员
- 回购
- 并购
- 供应链变化
- 新监管影响

====================
【评分标准（严格保守）】
====================

0-2分（噪音）：
- 观点
- 分析
- 市场评论
- 技术分析
- 普通采访
- 普通产品发布
- “价格突破XX”
- KOL观点
- 没有新增信息

3-4分（轻度影响）：
- 普通财报
- 一般经济数据
- 行业动态
- 常规监管讨论
- 一般合作消息

5-6分（中度影响）：
- 超预期经济数据
- 重要监管落地
- 重要公司业绩超预期
- 行业供需变化
- 中型安全事件
- 巨头合作消息

7-8分（重大影响）：
- 美联储超预期政策
- AI产业链重大变化
- ETF批准
- 大型交易所事件
- 地缘政治升级
- 龙头公司业绩远超预期

9-10分（极重大事件）：
- 金融危机
- 战争爆发
- 系统性流动性危机
- 全球监管重大转向
- 顶级交易所暴雷
- 美联储方向性逆转

注意：
- 80%以上新闻应该在0-4分
- 9-10分极少出现
- “有人预测上涨”基本都是0-1分
- 单纯情绪新闻不给高分

====================
【分析要求】
====================

必须输出：

1. 新闻摘要
2. 市场影响方向
3. 利多/利空资产
4. 最相关的美股
5. 最相关的加密货币
6. 影响逻辑
7. 是否具有交易价值

如果新闻无法形成明确交易逻辑：
- 降低评分
- 明确说明“交易价值有限”

====================
【美股分析要求】
====================

尽量具体到股票代码：

例如：
- NVDA
- AMD
- TSLA
- COIN
- MSTR
- META
- AMZN
- GOOGL

不要只说：
- “科技股”
- “AI概念股”

而要说明：
- 谁直接受益
- 谁间接受损
- 为什么

====================
【加密货币分析要求】
====================

重点关注：
- BTC
- ETH
- SOL
- BNB
- TON
- XRP
- 稳定币
- DeFi
- AI Agent相关币

不要只说：
- “利好加密市场”

而要说明：
- 利好哪个币
- 原因是什么
- 是资金流入还是风险偏好提升

====================
【输出格式】
====================

严格输出JSON，不要输出任何额外内容：

{
  "score": <0-10整数>,
  "summary": "<一句话总结>",
  "market_sentiment": "<bullish/bearish/neutral>",
  "tradable": <true/false>,
  "category": "<宏观/AI/加密/监管/财报/地缘政治等>",
  "bullish_assets": [
    {
      "symbol": "NVDA",
      "type": "stock",
      "reason": "AI资本开支增长"
    }
  ],
  "bearish_assets": [
    {
      "symbol": "BTC",
      "type": "crypto",
      "reason": "监管风险上升"
    }
  ],
  "impact": [
    "英伟达产业链利多",
    "高估值科技股承压"
  ],
  "explanation": "<120字以内说明市场影响逻辑>",
  "trading_logic": "<简洁说明资金可能流向>",
  "confidence": <0-100整数>
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
	if result.Confidence < 0 {
		result.Confidence = 0
	} else if result.Confidence > 100 {
		result.Confidence = 100
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

	tradableLabel := "否"
	if result.Tradable {
		tradableLabel = "是"
	}
	card := feishu.NewCard().
		SetHeader(fmt.Sprintf("金融快讯 (影响分数: %d/10)", result.Score), template).
		AddText(fmt.Sprintf("来源：%s", msg.ChannelName)).
		AddText(fmt.Sprintf("摘要：%s", result.Summary)).
		AddText(fmt.Sprintf("类别：%s", result.Category)).
		AddText(fmt.Sprintf("市场情绪：%s｜可交易：%s｜置信度：%d/100",
			formatMarketSentiment(result.MarketSentiment),
			tradableLabel,
			result.Confidence,
		))

	if s := formatAssetImpacts("看多", result.BullishAssets); s != "" {
		card.AddText(s)
	}
	if s := formatAssetImpacts("看空", result.BearishAssets); s != "" {
		card.AddText(s)
	}

	// 格式化影响方向，区分利多/利空
	if len(result.Impact) > 0 {
		card.AddText(fmt.Sprintf("影响：%s", formatImpact(result.Impact)))
	}

	if result.TradingLogic != "" {
		card.AddText(fmt.Sprintf("资金流向：%s", result.TradingLogic))
	}

	card.AddText(fmt.Sprintf("分析：%s", result.Explanation)).
		AddText(fmt.Sprintf("原文：%s", truncateText(msg.Text, 500)))

	if err := a.feishuBot.SendCard(card); err != nil {
		return fmt.Errorf("发送飞书通知失败: %w", err)
	}

	return nil
}

func formatMarketSentiment(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bullish":
		return "偏多"
	case "bearish":
		return "偏空"
	case "neutral":
		return "中性"
	default:
		if s == "" {
			return "—"
		}
		return s
	}
}

func assetTypeLabel(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "stock":
		return "美股"
	case "crypto":
		return "加密"
	default:
		if t == "" {
			return "标的"
		}
		return t
	}
}

func formatAssetImpacts(title string, assets []AssetImpact) string {
	if len(assets) == 0 {
		return ""
	}
	var parts []string
	for _, a := range assets {
		parts = append(parts, fmt.Sprintf("%s(%s)：%s", a.Symbol, assetTypeLabel(a.Type), a.Reason))
	}
	return fmt.Sprintf("%s：%s", title, strings.Join(parts, "｜"))
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
		"sentiment", result.MarketSentiment,
		"tradable", result.Tradable,
		"confidence", result.Confidence,
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
