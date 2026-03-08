# Config Hot Reload Demo

演示如何使用 viper + fsnotify 实现 YAML 配置文件的热更新功能。

## 编译

```bash
go build -o bin/demo ./cmd/demo
```

## 运行

```bash
# 使用默认配置文件 (config.yaml)
./bin/demo

# 使用 -c 指定配置文件
./bin/demo -c ./cmd/demo/config.yaml

# 使用 --config 指定配置文件
./bin/demo --config ./cmd/demo/config.yaml

# 查看帮助
./bin/demo --help
```