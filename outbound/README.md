# outbound

出站 **执行端** 辅助：统一 `version` / `timestamp`，补全空的 manufacturer/serial。

## 不负责

- `headerId` / `orderUpdateId` / `actionId` 的分配与水位
- 多进程唯一、重启恢复、与调度状态机对齐

这些由 dispatcher（或其它编排端）完成后再调用：

```go
ord.HeaderId = ...
ord.OrderUpdateId = ...
res, err := client.AGV(mfr, sn).PublishOrder(ctx, ord)
if navlink.PublishAccepted(err) {
    // platform RecordSuccessfulPublish — navlink does not
    _ = res // topic / payload / header summary for reconciliation
}
```

成败语义见根目录 `publish.go`：`ClientNotStarted` / `QosNotSupported` /
`TimeoutError` / `context.Canceled` / `PublishFailed` 可区分。
仅 `err == nil` 表示 MQTT QoS 握手成功（broker 收下），不是车侧已接受 order。
