package feishu

import (
	"os"
	"testing"
)

func getWebhookURL() string {
	return os.Getenv("FEISHU_WEBHOOK_URL")
}

func getSecret() string {
	return os.Getenv("FEISHU_SECRET")
}

func TestSendText(t *testing.T) {
	webhookURL := getWebhookURL()
	if webhookURL == "" {
		t.Skip("FEISHU_WEBHOOK_URL not set")
	}

	bot := NewFeishuBot(webhookURL)

	err := bot.SendText("这是一条测试文本消息")
	if err != nil {
		t.Fatalf("SendText failed: %v", err)
	}
}

func TestSendTextWithSign(t *testing.T) {
	webhookURL := getWebhookURL()
	secret := getSecret()
	if webhookURL == "" || secret == "" {
		t.Skip("FEISHU_WEBHOOK_URL or FEISHU_SECRET not set")
	}

	bot := NewFeishuBot(webhookURL, WithSecret(secret))

	err := bot.SendText("这是一条带签名的测试文本消息")
	if err != nil {
		t.Fatalf("SendTextWithSign failed: %v", err)
	}
}

func TestSendCard(t *testing.T) {
	webhookURL := getWebhookURL()
	if webhookURL == "" {
		t.Skip("FEISHU_WEBHOOK_URL not set")
	}

	bot := NewFeishuBot(webhookURL)

	card := NewCard().
		SetHeader("测试卡片", "blue").
		AddText("这是卡片的正文内容").
		AddText("支持多行文本").
		AddButton("点击查看", "https://example.com", "primary")

	err := bot.SendCard(card)
	if err != nil {
		t.Fatalf("SendCard failed: %v", err)
	}
}

func TestSendCardWithSign(t *testing.T) {
	webhookURL := getWebhookURL()
	secret := getSecret()
	if webhookURL == "" || secret == "" {
		t.Skip("FEISHU_WEBHOOK_URL or FEISHU_SECRET not set")
	}

	bot := NewFeishuBot(webhookURL, WithSecret(secret))

	card := NewCard().
		SetHeader("带签名的测试卡片", "red").
		AddText("这是一条带签名的卡片消息").
		AddButton("确认", "https://example.com/confirm", "primary")

	err := bot.SendCard(card)
	if err != nil {
		t.Fatalf("SendCardWithSign failed: %v", err)
	}
}
