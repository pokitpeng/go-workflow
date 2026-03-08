package feishu

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// FeishuBot 飞书机器人
type FeishuBot struct {
	webhookURL string
	secret     string
	httpClient *http.Client
}

// Option 配置选项
type Option func(*FeishuBot)

// WithSecret 设置签名密钥
func WithSecret(secret string) Option {
	return func(bot *FeishuBot) {
		bot.secret = secret
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端
func WithHTTPClient(client *http.Client) Option {
	return func(bot *FeishuBot) {
		bot.httpClient = client
	}
}

// NewFeishuBot 创建飞书机器人实例
func NewFeishuBot(webhookURL string, opts ...Option) *FeishuBot {
	bot := &FeishuBot{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	for _, opt := range opts {
		opt(bot)
	}

	return bot
}

// TextMessage 文本消息
type TextMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Card 卡片消息
type Card struct {
	Header   *CardHeader   `json:"header,omitempty"`
	Elements []CardElement `json:"elements"`
}

// CardHeader 卡片头部
type CardHeader struct {
	Title    *CardText `json:"title,omitempty"`
	Template string    `json:"template,omitempty"`
}

// CardElement 卡片元素
type CardElement struct {
	Tag     string       `json:"tag"`
	Text    *CardText    `json:"text,omitempty"`
	Actions []CardAction `json:"actions,omitempty"`
}

// CardText 卡片文本
type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// CardAction 卡片按钮
type CardAction struct {
	Tag  string    `json:"tag"`
	Text *CardText `json:"text,omitempty"`
	URL  string    `json:"url,omitempty"`
	Type string    `json:"type,omitempty"`
}

// CardMessage 卡片消息结构
type CardMessage struct {
	MsgType string `json:"msg_type"`
	Card    Card   `json:"card"`
}

// generateSign 生成签名
func (bot *FeishuBot) generateSign(timestamp int64) string {
	if bot.secret == "" {
		return ""
	}

	stringToSign := fmt.Sprintf("%d\n%s", timestamp, bot.secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write(nil)
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
}

// send 发送消息
func (bot *FeishuBot) send(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	reqURL := bot.webhookURL
	if bot.secret != "" {
		timestamp := time.Now().Unix()
		sign := bot.generateSign(timestamp)
		reqURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", bot.webhookURL, timestamp, sign)
	}

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送失败: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("飞书返回错误: code=%d, msg=%s", result.Code, result.Msg)
	}

	return nil
}

// SendText 发送文本消息
func (bot *FeishuBot) SendText(text string) error {
	msg := TextMessage{MsgType: "text"}
	msg.Content.Text = text
	return bot.send(&msg)
}

// SendCard 发送卡片消息
func (bot *FeishuBot) SendCard(card *Card) error {
	msg := CardMessage{
		MsgType: "interactive",
		Card:    *card,
	}
	return bot.send(&msg)
}

// NewCard 创建卡片
func NewCard() *Card {
	return &Card{}
}

// SetHeader 设置卡片头部
func (c *Card) SetHeader(title, template string) *Card {
	c.Header = &CardHeader{
		Title: &CardText{
			Tag:     "plain_text",
			Content: title,
		},
		Template: template,
	}
	return c
}

// AddText 添加文本元素
func (c *Card) AddText(content string) *Card {
	c.Elements = append(c.Elements, CardElement{
		Tag: "div",
		Text: &CardText{
			Tag:     "plain_text",
			Content: content,
		},
	})
	return c
}

// AddButton 添加按钮
func (c *Card) AddButton(text, linkURL, btnType string) *Card {
	c.Elements = append(c.Elements, CardElement{
		Tag: "action",
		Actions: []CardAction{
			{
				Tag:  "button",
				Text: &CardText{Tag: "plain_text", Content: text},
				URL:  linkURL,
				Type: btnType,
			},
		},
	})
	return c
}
