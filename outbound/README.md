# outbound

出站 **执行端** 辅助：统一 `version` / `timestamp`，补全空的 manufacturer/serial。

## 不负责

- `headerId` / `orderUpdateId` / `actionId` 的分配与水位
- 多进程唯一、重启恢复、与调度状态机对齐

这些由 dispatcher（或其它编排端）完成后再调用：

```go
ord.HeaderId = ...
ord.OrderUpdateId = ...
_ = client.AGV(mfr, sn).PublishOrder(ctx, ord)
```
