package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		apiKey  string
		opts    []Option
		wantURL string
		wantKey string
		wantTo  time.Duration
	}{
		{
			name:    "基本配置",
			baseURL: "https://api.openai.com/v1",
			apiKey:  "test-key",
			wantURL: "https://api.openai.com/v1",
			wantKey: "test-key",
			wantTo:  300 * time.Second,
		},
		{
			name:    "自定义超时",
			baseURL: "https://api.openai.com/v1",
			apiKey:  "test-key",
			opts: []Option{
				WithTimeout(10 * time.Second),
			},
			wantURL: "https://api.openai.com/v1",
			wantKey: "test-key",
			wantTo:  10 * time.Second,
		},
		{
			name:    "自定义 BaseURL",
			baseURL: "https://api.openai.com/v1",
			apiKey:  "test-key",
			opts: []Option{
				WithBaseURL("https://custom.api.com/v1"),
			},
			wantURL: "https://custom.api.com/v1",
			wantKey: "test-key",
			wantTo:  300 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.apiKey, tt.opts...)
			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %s, want %s", client.baseURL, tt.wantURL)
			}
			if client.apiKey != tt.wantKey {
				t.Errorf("apiKey = %s, want %s", client.apiKey, tt.wantKey)
			}
			if client.httpClient.Timeout != tt.wantTo {
				t.Errorf("timeout = %v, want %v", client.httpClient.Timeout, tt.wantTo)
			}
		})
	}
}

func TestChatCompletion(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != http.MethodPost {
			t.Errorf("请求方法 = %s, want POST", r.Method)
		}

		// 验证请求路径
		if r.URL.Path != "/chat/completions" {
			t.Errorf("请求路径 = %s, want /chat/completions", r.URL.Path)
		}

		// 验证请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Authorization = %s, want Bearer test-api-key", r.Header.Get("Authorization"))
		}

		// 解析请求体
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("解析请求体失败: %v", err)
		}

		// 返回模拟响应
		resp := ChatCompletionResponse{
			ID:      "chatcmpl-test123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []ChatCompletionChoice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "这是一个测试响应",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL, "test-api-key")

	// 发送请求
	req := &ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: "user", Content: "你好"},
		},
	}

	resp, err := client.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletion 失败: %v", err)
	}

	// 验证响应
	if resp.ID != "chatcmpl-test123" {
		t.Errorf("响应 ID = %s, want chatcmpl-test123", resp.ID)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("响应 Model = %s, want gpt-4o", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("响应 Choices 长度 = %d, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "这是一个测试响应" {
		t.Errorf("响应内容 = %s, want 这是一个测试响应", resp.Choices[0].Message.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestChatCompletion_Error(t *testing.T) {
	// 创建返回错误的模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: &APIError{
				Code:    401,
				Message: "Invalid API key",
				Type:    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "invalid-key")

	req := &ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: "user", Content: "test"}},
	}

	_, err := client.ChatCompletion(context.Background(), req)
	if err == nil {
		t.Fatal("期望返回错误，但得到 nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("错误类型 = %T, want *APIError", err)
	}
	if apiErr.Code != 401 {
		t.Errorf("错误码 = %d, want 401", apiErr.Code)
	}
	if apiErr.Message != "Invalid API key" {
		t.Errorf("错误消息 = %s, want Invalid API key", apiErr.Message)
	}
}

func TestChatCompletion_ContextCanceled(t *testing.T) {
	// 创建延迟响应的模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: "user", Content: "test"}},
	}

	_, err := client.ChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("期望返回超时错误，但得到 nil")
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		Code:    400,
		Type:    "invalid_request_error",
		Message: "Missing required field",
	}

	want := "openai api error: code=400, type=invalid_request_error, message=Missing required field"
	if err.Error() != want {
		t.Errorf("Error() = %s, want %s", err.Error(), want)
	}
}
