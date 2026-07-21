# Changelog

## Unreleased

### Added

- Client / TopicResolver / typed On* / Envelope / AGV outbound (v0.1)
- outbound IdAllocator + EnvelopeBuilder (v0.2)
- FleetSession, ExtensionRegistry hook, Transport() layering (v0.3)
- Memory EventBus, L1 events, `examples/platform-wiring` (v0.4)
- testkit FakeBroker / Recording, facts.Apply (v0.5)
- Auto restore subscriptions after MQTT reconnect (`RestoreSubscriptionsOnReconnect`)
- `AGVHandle.CancelOrder` convenience instantAction

### Intentionally out of scope

See README「非目标」. P2 still optional: Order/Action builder helpers, JSONL replay,
metrics hooks, AGV-side sim helpers (F-15～F-18).
