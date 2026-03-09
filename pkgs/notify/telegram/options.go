package telegram

import "fmt"

// Option 配置选项
type Option func(*Listener)

// WithSessionPath 设置会话文件路径
func WithSessionPath(path string) Option {
	return func(l *Listener) {
		l.sessionPath = path
	}
}

// WithChannels 设置频道白名单
func WithChannels(ids ...int64) Option {
	return func(l *Listener) {
		for _, id := range ids {
			l.channels[id] = true
		}
	}
}

// WithCodeHandler 设置验证码输入回调
func WithCodeHandler(handler func() (string, error)) Option {
	return func(l *Listener) {
		l.codeHandler = handler
	}
}

// WithBufferSize 设置消息 channel 缓冲区大小
func WithBufferSize(size int) Option {
	return func(l *Listener) {
		l.bufferSize = size
	}
}

// DefaultCodeHandler 默认验证码输入处理器
func DefaultCodeHandler() func() (string, error) {
	return func() (string, error) {
		fmt.Print("Enter code: ")
		var code string
		fmt.Scanln(&code)
		return code, nil
	}
}