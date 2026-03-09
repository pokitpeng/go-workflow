package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// Listener Telegram 频道监听器
type Listener struct {
	appID       int
	appHash     string
	phone       string
	sessionPath string
	channels    map[int64]bool
	bufferSize  int
	msgChan     chan *ChannelMessage
	codeHandler func() (string, error)
}

// NewListener 创建监听器
func NewListener(appID int, appHash, phone string, opts ...Option) *Listener {
	l := &Listener{
		appID:       appID,
		appHash:     appHash,
		phone:       phone,
		sessionPath: "session.json",
		channels:    make(map[int64]bool),
		bufferSize:  100,
		codeHandler: DefaultCodeHandler(),
	}

	for _, opt := range opts {
		opt(l)
	}

	l.msgChan = make(chan *ChannelMessage, l.bufferSize)
	return l
}

// Messages 返回消息 channel
func (l *Listener) Messages() <-chan *ChannelMessage {
	return l.msgChan
}

// Run 运行监听器
func (l *Listener) Run(ctx context.Context) error {
	defer close(l.msgChan)

	dispatcher := tg.NewUpdateDispatcher()

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}

		// 获取频道ID
		var channelID int64
		if peer, ok := msg.PeerID.(*tg.PeerChannel); ok {
			channelID = peer.ChannelID
		}

		// 白名单过滤
		if len(l.channels) > 0 && !l.channels[channelID] {
			return nil
		}

		// 获取频道名称
		channelName := ""
		if ch, ok := e.Channels[channelID]; ok {
			channelName = ch.Title
		}

		// 获取发送者ID
		var senderID int64
		if from, ok := msg.FromID.(*tg.PeerUser); ok {
			senderID = from.UserID
		}

		l.msgChan <- &ChannelMessage{
			ChannelID:   channelID,
			ChannelName: channelName,
			MessageID:   msg.ID,
			Text:        msg.Message,
			Date:        msg.Date,
			SenderID:    senderID,
		}

		return nil
	})

	client := telegram.NewClient(l.appID, l.appHash, telegram.Options{
		SessionStorage: &session.FileStorage{
			Path: l.sessionPath,
		},
		UpdateHandler: dispatcher,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		flow := auth.NewFlow(
			auth.Constant(l.phone, "", auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
				return l.codeHandler()
			})),
			auth.SendCodeOptions{},
		)

		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("认证失败: %w", err)
		}

		// 阻塞等待上下文结束
		<-ctx.Done()
		return ctx.Err()
	})
}
