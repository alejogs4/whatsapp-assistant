# Topic: `whatsapp-transport`

## Purpose
Native WhatsApp Web multi-device transport adapter (`internal/adapters/whatsapp/`) built on `whatsmeow`, implementing `touchpoint.MessagingPort` for outbound delivery and dispatching inbound user messages to the core router.

## Scope & Related Issue
- **Issue #4**: `feat(whatsapp): implement whatsmeow multi-device transport adapter`
- **Implements**:
  - Driven adapter: `touchpoint.MessagingPort` (`SendText`, `SendDrillPrompt`)
  - Driving adapter: Inbound WhatsApp event listener → `InboundMessageHandler` callback/port

## Architectural Invariants
1. **Protocol Isolation**: `whatsmeow` types (`types.JID`, `events.Message`, `waE2E.Message`) never leak outside `internal/adapters/whatsapp/`. Inbound events are normalized into a domain-neutral `InboundMessage{SenderPhone, Text, ReceivedAt, MessageID}` before dispatching.
2. **Self-Message & Group Guard**: Ignore messages sent by the bot itself (`IsFromMe`), status broadcasts, and group chats (MVP operates strictly in 1:1 DMs).
3. **Phone Normalization & Multi-Tenancy**: Normalize E.164 phone numbers <-> WhatsApp JIDs (`<digits>@s.whatsapp.net`) consistently so `UserRepository.FindByPhoneNumber` resolves the exact tenant.
4. **Session Persistence & Terminal QR Pairing**: Persist `whatsmeow` device keys in SQLite (`sqlstore`) so restarts reconnect seamlessly without re-scanning QR codes; on first boot, render QR code in terminal.
5. **Graceful Lifecycle**: Respect `context.Context` cancellation for clean disconnect on `SIGINT`/`SIGTERM`, with bounded reconnect backoff.

## Gotchas & Constraints
- Never require a live WhatsApp account for Tier 1 `go test -race ./...`; `Sender` wraps `MessageSender` and `Client` wraps `LifecycleClient` for deterministic offline testing.
- **Race-safe QR shutdown**: `Client.Start(ctx)` waits on a `qrDone` channel after `Disconnect()` so the QR reader goroutine drains cleanly before `Start` returns (caught and verified by `-race`).
- `sqlstore.New` uses driver `"sqlite"` (`modernc.org/sqlite`) with `_pragma=foreign_keys(1)` appended when absent.
- `SendDrillPrompt` formats micro-drills cleanly for mobile WhatsApp readability (`*Quick Drill*` header, plain text).
- Status: **Implemented & verified** (`internal/adapters/whatsapp/`, commit `9ec762a`).
