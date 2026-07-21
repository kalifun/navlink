# extend — 开放扩展钩子

navlink **不内置**任何厂商字段解析。`Registry` 只负责：

1. 让消费方注册自己的 `Extractor`
2. 入站时合并结果到 `Envelope.Meta`

OEM / 厂商语义、错误码、处置策略都留在 dispatcher、rcs-sim 或工具侧。

## 最小用法

在**你们自己的仓库**里写 Extractor，再注入 `Config.Extensions`：

```go
package vdaext

import (
	"encoding/json"

	"github.com/kalifun/navlink/extend"
)

// Meta 键由消费方定义并保持稳定；navlink 不提供预置常量。
const MetaReportedCurrentNodeID = "ReportedCurrentNodeID"

func NewRegistry() *extend.Registry {
	reg := extend.NewRegistry()
	reg.Register(extractCurrentNodeID)
	return reg
}

func extractCurrentNodeID(channel string, raw []byte) (extend.Meta, error) {
	if channel != "state" {
		return nil, nil
	}
	var probe struct {
		CurrentNodeID string `json:"currentNodeId"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if probe.CurrentNodeID == "" {
		return nil, nil
	}
	return extend.Meta{MetaReportedCurrentNodeID: probe.CurrentNodeID}, nil
}
```

接入 Client：

```go
client, err := navlink.New(navlink.Config{
	Broker:     "tcp://localhost:1883",
	ClientID:   "rcs",
	Interface:  "uagv",
	Version:    "v2",
	Extensions: vdaext.NewRegistry(),
})

client.OnState(func(ctx context.Context, env navlink.Envelope, st *state.State) error {
	if node, ok := env.Meta[vdaext.MetaReportedCurrentNodeID].(string); ok {
		_ = node // 平台自己使用
	}
	return nil
})
```

## 约定

| 项 | 建议 |
|----|------|
| Meta 键名 | 消费方定义；库不解释含义 |
| 多 Extractor | 按 `Register` 顺序执行；同名键后者覆盖 |
| 无扩展字段 | 返回 `nil, nil` |
| 解析失败 | 返回 `error`；Client 走 decode 失败钩子，不崩进程 |
| 代码位置 | 放在消费方自己的包（如 `vdaext`）；**不要**往 navlink 提厂商专用 PR |

## API

```go
type Extractor func(channel string, raw []byte) (Meta, error)

func NewRegistry() *Registry
func (r *Registry) Register(e Extractor)
func (r *Registry) Apply(channel string, raw []byte) (Meta, error)
```

`channel` 为 VDA topic 末段，如 `state`、`connection`、`visualization`。
