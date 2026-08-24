# Changelog

## Unreleased

### Changed

- MQTT transport, outbound builder, in-process bus, fleet session, and generated
  error codes live under `internal/`. Import `navlink`, `extend`, `topic`,
  `facts`, or `testkit`. Fleet config is `Config.Fleet *FleetOptions`
  (`DefaultFleetOptions()`), not `session.Options`.

## 0.8.0 — 2026-08-21

Standard InstantAction helpers (protocol convenience, not scheduling).

### Added

- `AGVHandle` helpers for official VDA5050 instant actions: `StartPause`,
  `StopPause`, `InitPosition`, `StateRequest`, `FactsheetRequest`, and
  parameter-less `StartCharging` / `StopCharging`. Callers still supply
  `headerId` / `actionId`; results go through `PublishInstantActions` and
  `ClassifyPublish`.
- `InitPositionParams` matches the spec keys only: `x`, `y`, `theta`,
  `mapId`, `lastNodeId`.

### Changed

- `CancelOrder` uses `vda5050.ActionCancelOrder` from types-go v0.6.0
  (same on-wire string).

### Intentionally out of scope

OEM `actionType` (use `PublishInstantActions`), ID allocation, when to
pause/charge, async EventBus, `logReport`, 2.1 map actions, JSONL replay,
AGV-side sim APIs.

## 0.7.0 — 2026-08-20

Production MQTT and execution visibility.

### Added

- MQTT connection surface: optional `tls.Config`, `ConnectTimeout`, last will,
  `Client.Connected()`, `OnTransportUp` / `OnTransportDown`,
  `OnSubscriptionsRestored` (reconnect subscribe success/failure).
- `Config.QoS` is `*byte` (`QoSOf`). nil defaults order/instantActions publish
  to QoS 1 and visualization subscribe to 0; a pointer to 0 is real QoS 0.
- Inbound MQTT messages are dispatched off the paho callback (bounded queue;
  visualization may drop when full). `OnHandlerError` / `OnInboundDrop`.
- `ClassifyPublish` → `Accepted` / `NotStarted` / `Uncertain`.
  `MarkPublishAttempted` for transports. FakeBroker `FailNextPublish` /
  `HangNextPublish`.
- `examples/dispatch-egress-sketch` uses three-way outcomes (timeout is uncertain).
- `examples/platform-wiring` keeps VDA `On*` off the platform event bus.

### Changed

- MQTT `Publish` no longer treats QoS 0 as “use default”, and does not hold the
  transport lock while waiting for a token.
- Unset `Config.QoS` now publishes order/instantActions at QoS 1 (was QoS 0).
- Token wait uses `WaitTimeout` so a timed-out wait does not leak a `Wait()`
  goroutine.

### Intentionally out of scope

See README「非目标」。InstantAction helpers, async EventBus, JSONL replay, and
AGV-side sim APIs are not in this release.

## 0.6.0 — 2026-07-21

Execution-endpoint baseline for dispatcher / sim wiring.

### Changed

- **Execution-end outbound**: removed IdAllocator (`HeaderIDs` / `OrderUpdateIDs` /
  `ActionIDs`), `NextActionID`, and `SyncOrderUpdateFromVehicle`. Callers must set
  `headerId` / `orderUpdateId` / `actionId` before publish. Builder only fills
  version, timestamp, and blank manufacturer/serial.
- `CancelOrder(ctx, headerID, actionID)` now requires IDs from the orchestration layer.
- `PublishOrder` / `PublishInstantActions` / `CancelOrder` return `(PublishResult, error)`.

### Added

- **Outbound light validation**: reject obviously bad packets before MQTT
  (`OutboundValidationFailed` / `IsPublishValidationFailed`); configurable via
  `Config.OutboundValidation`.
- **Inbound header policy hook**: optional `InboundPolicy` +
  `NewHeaderSequencePolicy` annotate `Accept|Stale|Duplicate` (no drop by default).
- **Example** `examples/dispatch-egress-sketch`: GetNext → Publish → Record only on
  `PublishAccepted`.
- **Execution visibility**: `PublishResult` (topic/payload/header summary) plus
  `PublishAccepted` / `IsPublishNotStarted` / `IsPublishTimeout` /
  `IsPublishCanceled` / `IsPublishQoSRejected` / `IsPublishBrokerRejected`.
  MQTT token wait honors `ctx`. Nil order / instantActions returns InvalidConfig.
- Client / TopicResolver / typed On* / Envelope / AGV outbound (v0.1)
- FleetSession, ExtensionRegistry hook, Transport() layering (v0.3)
- Memory EventBus, L1 events, `examples/platform-wiring` (v0.4)
- testkit FakeBroker / Recording, facts.Apply (v0.5)
- Auto restore subscriptions after MQTT reconnect
- `AGVHandle.CancelOrder` convenience instantAction

### Intentionally out of scope

See README「非目标」。

