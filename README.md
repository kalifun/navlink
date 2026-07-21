# navlink

Go 语言下的 **VDA5050 协议接入 SDK**：同一套 API 完成 MQTT 连接、主题解析、强类型收发。

## 环境

本仓库使用 **direnv + Nix flake**（`.envrc` → `github:kalifun/devshells#go-1_25`）。

```bash
direnv allow
```

错误码由 **glitch** 从 `errors/*.yaml` 生成：

```bash
make generr
```

## Quick Start

```go
client, err := navlink.New(navlink.Config{
    Broker:    "tcp://localhost:1883",
    ClientID:  "my-rcs",
    Interface: "uagv",
    Version:   "v2",
})
client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
    fmt.Println(env.AGV.SerialNumber, st.LastNodeId)
    return nil
})
_ = client.Start(ctx)
defer client.Stop(ctx)
```

可运行示例：

```bash
go run ./examples/subscribe-state
go run ./examples/platform-wiring   # 集中注册 L1 + 平台自定义事件
```

## 布局

```text
navlink/                 # Client 公共 API
topic/                   # TopicResolver（唯一主题真相源）
outbound/                # 出站执行辅助（version/timestamp；ID 由调度填写）
session/                 # FleetSession（connection → per-AGV 订阅）
extend/                  # ExtensionRegistry（开放钩子；见 extend/README.md）
bus/                     # Memory EventBus
facts/                   # 连续 State 的协议 Fact 投影（无调度语义）
testkit/                 # FakeBroker / Recording（见 testkit/README.md）
mqtt/                    # 字节级 MQTT transport（不懂 VDA）
gerrors/                 # glitch 生成，勿手改
errors/*.yaml            # 错误码源
examples/subscribe-state
examples/platform-wiring
```

## 开发

```bash
make test
make check   # generr + fmt + vet + test
```

## 非目标

navlink 是协议 **执行端**，**不做**也不逐渐滑向：

- `headerId` / `orderUpdateId` / `actionId` 分配与水位（属调度编排）
- 选车、任务分解、RHCR、交管、充电、completion / grant 等业务判定
- 跨进程中台、默认 Redis EventBus、多租户协议网关
- 大而全 Processor / 插件微内核
- 用库代码强制消费方架构
- 领域事件（如 `EpisodeOpened`）——平台用 `Emit` 自行挂载

VDA 收发走 `Client` / `AGV`；非 VDA 的 application MQTT 用 `Client.Transport()`。

详见 [CHANGELOG.md](CHANGELOG.md) 与 [outbound/README.md](outbound/README.md)。
