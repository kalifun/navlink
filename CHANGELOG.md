# Changelog

## Unreleased

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

See README「非目标」。P2 still optional: Order/Action builder helpers, JSONL replay,
metrics hooks, AGV-side sim helpers (F-15～F-18).
