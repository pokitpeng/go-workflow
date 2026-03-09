# Telegram 频道监听包设计

## 概述

将 Telegram 频道监听功能封装为独立包，提供简洁的 Go channel 方式消费消息。

## 需求

- 仅监听频道消息
- 支持频道白名单过滤
- 使用 Go channel 异步消费消息
- 手机号 + 验证码认证方式

## 包结构

```
pkgs/notify/telegram/
├── telegram.go      # 核心监听器
├── options.go       # 配置选项
├── message.go       # 消息结构定义
└── telegram_test.go # 测试
```

## 核心组件

### Listener

```go
type Listener struct {
    appID       int
    appHash     string
    phone       string
    sessionPath string
    channels    map[int64]bool
    msgChan     chan *ChannelMessage
    client      *telegram.Client
    codeHandler func() (string, error)
}
```

### ChannelMessage

```go
type ChannelMessage struct {
    ChannelID   int64
    ChannelName string
    MessageID   int
    Text        string
    Date        int
    SenderID    int64
}
```

## API 设计

### 创建监听器

```go
listener := telegram.NewListener(appID, appHash, phone,
    telegram.WithSessionPath("session.json"),
    telegram.WithChannels(123456789, -1001234567890),
    telegram.WithCodeHandler(func() (string, error) {
        fmt.Print("Enter code: ")
        var code string
        fmt.Scanln(&code)
        return code, nil
    }),
)
```

### 消费消息

```go
msgChan := listener.Messages()

go listener.Run(ctx)

for msg := range msgChan {
    fmt.Printf("[%s] %s\n", msg.ChannelName, msg.Text)
}
```

## 配置选项

- `WithSessionPath(path string)` - 设置会话文件路径
- `WithChannels(ids ...int64)` - 设置频道白名单
- `WithCodeHandler(handler func() (string, error))` - 设置验证码输入回调
- `WithBufferSize(size int)` - 设置消息 channel 缓冲区大小

## 认证流程

1. 首次运行调用 `codeHandler` 获取验证码
2. 使用 `session.FileStorage` 持久化会话
3. 后续运行自动恢复会话

## 错误处理

- 认证失败返回错误
- 网络断开自动重连（gotd 内置）
- channel 关闭时通知消费者