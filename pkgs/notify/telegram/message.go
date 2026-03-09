package telegram

// ChannelMessage 频道消息
type ChannelMessage struct {
	ChannelID   int64  // 频道ID
	ChannelName string // 频道名称
	MessageID   int    // 消息ID
	Text        string // 消息内容
	Date        int    // 消息时间戳
	SenderID    int64  // 发送者ID（转发消息可能为0）
}