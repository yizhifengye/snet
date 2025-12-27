# 🛠️ 工具包含：

## 🚀 快速开始测试：

```bash
# 1. 启动 SNET 服务器
cd examples/server && go run main.go

# 2. 启动网络代理（另一个终端）
cd examples/network_test
go build -o proxy proxy.go
# 使用 proxy.go 模拟弱网环境
./proxy --listen :8081 --target localhost:8080 --disconnect 3s

# 3. 修改客户端连接到代理端口 8081
# 在 client/main.go 中改为：
return net.Dial("tcp", "localhost:8081")

# 4. 启动客户端测试
cd examples/client && go run main.go
```

## 🧪 测试场景：

1. **高延迟测试** - 200-500ms 延迟
2. **丢包测试** - 10% 丢包率
3. **不稳定连接** - 每30秒断开重连
4. **极端条件** - 延迟+丢包组合
5. **自定义参数** - 按需配置

## 🔍 观察重点：

- SNET 是否能自动重连
- 重连过程中数据是否丢失
- 性能在弱网下的表现
- 超时处理是否正确

这套工具比传统的 `tc` 或 `netem` 更简单易用，而且是专门为测试 TCP 连接重连设计的。你可以根据需要调整各种参数来模拟真实的弱网环境。
