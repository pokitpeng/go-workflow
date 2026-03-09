# Telegram Listener Demo

演示如何使用 telegram listener 实现 Telegram 频道消息的监听功能。

## 编译

```bash
go build -o bin/tg ./cmd/tg
```

## 配置

```bash
cat << EOF > bin/.tg.env
TG_APP_ID=xx
TG_APP_HASH=xx
TG_PHONE=xx
TG_SESSION_PATH=bin/session.json
EOF
```

## 运行

```bash
./bin/tg
```