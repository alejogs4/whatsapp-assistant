# Explore: Issue #4 — WhatsApp Multi-Device Transport Adapter (`whatsmeow`)

## Affected Topics
- `whatsapp-transport` (primary)
- `domain-core` (implements `touchpoint.MessagingPort`)

## Current Codebase Reality
- `internal/touchpoint/ports.go` defines `MessagingPort` (`SendText`, `SendDrillPrompt`).
- No `internal/adapters/whatsapp/` package or `whatsmeow` dependency exists yet.
- No domain-neutral inbound message contract exists yet for decoupling `whatsmeow` events from the inbound router (Issue #6).

## Architectural Forks & Tradeoffs
1. **Inbound Handler Contract**:
   - Define `InboundMessage{MessageID, SenderPhone, Text, ReceivedAt}` and `InboundHandler func(ctx context.Context, msg InboundMessage) error` in `internal/touchpoint/ports.go` (or `internal/adapters/whatsapp/`).
   - **Recommendation**: Place `InboundMessage` and `InboundMessageHandler` in `internal/touchpoint/ports.go` so Issue #4 (WhatsApp driving adapter) and Issue #6 (Router service) depend only on the port, never on each other.
2. **Offline Testability of `whatsmeow`**:
   - Live WhatsApp pairing requires a phone and network.
   - **Recommendation**: Wrap `*whatsmeow.Client` behind a minimal internal `waSender` interface for `SendText`/`SendDrillPrompt` and test event filtering/normalization with synthetic `events.Message` structs.
