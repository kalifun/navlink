# navlink

Go 语言 **VDA5050 MQTT 接入 SDK**：一个 `Client` 完成主题、强类型入站回调、强类型出站发布。

报文结构来自 [`vda5050-types-go`](https://github.com/kalifun/vda5050-types-go)，navlink 不另维护一份 schema。

## 安装

```bash
go get github.com/kalifun/navlink@v0.9.0
```

需要 Go 1.25+。

## 快速开始

```go
client, err := navlink.New(navlink.Config{
    Broker:    "tcp://localhost:1883",
    ClientID:  "master-control",
    Interface: "uagv",
    Version:   "v2",
})
if err != nil {
    log.Fatal(err)
}

client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
    fmt.Println(env.AGV.SerialNumber, st.LastNodeId)
    return nil
})

if err := client.Start(ctx); err != nil {
    log.Fatal(err)
}
defer client.Stop(ctx)
```

可运行示例：

```bash
go run ./examples/subscribe-state
go run ./examples/platform-wiring          # 协议 L1 事件 + 应用自定义事件
go run ./examples/dispatch-egress-sketch   # ClassifyPublish 三态
```

## 发布

`headerId` / `orderUpdateId` / `actionId` 由调用方填写。库只补 version、timestamp 和 topic。

```go
res, err := client.AGV(mfr, sn).PublishOrder(ctx, ord)
switch navlink.ClassifyPublish(err) {
case navlink.PublishOutcomeAccepted:
    // MQTT QoS 握手成功（broker 收下），不是「车已接受 order」
    _ = res.Topic
case navlink.PublishOutcomeNotStarted:
    // 协议 ID 可以再用
case navlink.PublishOutcomeUncertain:
    // 发送后超时 / 取消：不要复用这组 ID
}
```

发布前默认做轻量校验（`headerId == 0`、空 `orderId` 等），可用 `Config.OutboundValidation` 调整。

官方瞬时动作有 helper：`CancelOrder`、`StartPause` / `StopPause`、`InitPosition`、`StateRequest`、`FactsheetRequest`，以及无参的 `StartCharging` / `StopCharging`。厂商自定义 `actionType` 仍走 `PublishInstantActions`。

可选 `Config.InboundPolicy`（例如 `NewHeaderSequencePolicy()`）在 envelope 上标注 `Accept|Stale|Duplicate`，**默认不丢包**。

`Config.IdentityMapper` 把 `(manufacturer, serial) → robotID` 写到 `Envelope.RobotID`。厂商扩展字段走 `Config.Extensions` → `Envelope.Meta`，见 [extend/README.md](extend/README.md)。

同一套 `Client` 可对接真实 MQTT，或 `testkit` 里的 FakeBroker。

## 范围

navlink 只做 **协议执行**，不做：

- `headerId` / `orderUpdateId` / `actionId` 的分配与水位
- `Uncertain` 之后的 fencing / 换号重发
- 选车、路径规划、交通管制、何时充电等业务判定
- 默认 Redis / 跨进程 EventBus、多租户网关
- 规定消费方如何分层
- 领域事件——需要的话用 `Emit` 自己挂

VDA 收发走 `Client` / `AGV`；其它应用 MQTT 走 `Client.Transport()`。

更多见 [CHANGELOG.md](CHANGELOG.md)。

## 开发

本仓库用 **direnv + Nix**（`.envrc` → `github:kalifun/devshells#go-1_25`）：

```bash
direnv allow
make test
make check   # generr + fmt + vet + test
```

错误码由 [glitch](https://github.com/kalifun/glitch) 从 `errors/*.yaml` 生成：

```bash
make generr
```

请勿手改 `internal/gerrors/` 下的生成文件。

## License

[MIT](LICENSE)
