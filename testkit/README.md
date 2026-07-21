# testkit

In-process doubles for navlink Client tests — **no real MQTT broker**.

## FakeBroker

Implements `navlink.Transport`:

| 方法 | 用途 |
|------|------|
| `Inject(ctx, topic, payload)` | 模拟车端入站（不记入 Published） |
| `Publish` / `Subscribe` | Client 正常出站 / 订阅 |
| `Published()` | 断言 master 发出的报文 |
| `Filters()` | 断言当前订阅 filter |

```go
broker := testkit.NewFakeBroker()
client, _ := navlink.New(navlink.Config{
    Interface: "uagv",
    Version:   "v2",
    Transport: broker,
})
client.OnState(...)
_ = client.Start(ctx)

_ = broker.Inject(ctx, "uagv/v2/M/S1/state", stateJSON)
pubs := broker.Published() // after PublishOrder etc.
```

## RecordingTransport

Wraps any `Transport` and records subscribe/publish for finer assertions:

```go
inner := testkit.NewFakeBroker()
rec := &testkit.Recorder{}
rt := testkit.NewRecordingTransport(inner, rec)
```

## Reconnect

`FakeBroker` implements `ReconnectAware`. Call `SimulateReconnect()` to drop
subscriptions and trigger Client's restore path (same as MQTT auto-reconnect).
