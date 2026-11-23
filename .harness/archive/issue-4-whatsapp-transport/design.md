# Design: Issue #4 — WhatsApp Multi-Device Transport Adapter (`whatsmeow`)

## Decisions & Tradeoffs
1. **Driving Port (`InboundMessage` & `InboundMessageHandler`) in `internal/touchpoint/ports.go`**: Keeps `whatsmeow` completely isolated in `internal/adapters/whatsapp/` while giving Issue #6 a clean inbound contract.
2. **Strict 1:1 DM Filtering**: Drop `IsFromMe == true`, group chats (`IsGroup == true`), status broadcasts (`status@broadcast`), and empty/non-text media messages in MVP so the assistant never replies to itself or spams groups.
3. **E.164 <-> JID Normalization**: Strip non-digits (`+`, spaces, dashes) to build `<digits>@s.whatsapp.net`, and normalize inbound `Sender.User` back to `+<digits>` so `UserRepository.FindByPhoneNumber` matches reliably.
4. **SQLite Session Persistence & Terminal QR Pairing**: Use `whatsmeow/store/sqlstore` backed by a dedicated session SQLite DSN (`WHATSAPP_DB_PATH`) with `qrterminal` output on first boot.

## Package & Contract Boundaries
- `internal/touchpoint/ports.go`: Add `InboundMessage` struct and `InboundMessageHandler` interface (`HandleInbound(ctx context.Context, msg InboundMessage) error`).
- `internal/adapters/whatsapp/`:
  - `jid.go`: `PhoneToJID`, `JIDToPhone` normalization helpers.
  - `filter.go`: `ExtractInboundMessage(evt *events.Message) (touchpoint.InboundMessage, bool)`.
  - `sender.go`: `MessagingPort` implementation (`SendText`, `SendDrillPrompt`) over mockable `waSender` interface.
  - `client.go`: Concrete `whatsmeow` client lifecycle (`Start(ctx)`, QR channel rendering, event handler registration, graceful `Disconnect()`).

## Verification Strategy
- **Tier 1 (`harness check`)**: `go vet ./...` and `go test -race ./...` verifying JID normalization, message extraction/filtering, outbound drill formatting, and inbound dispatch without live WhatsApp credentials.
- **Tier 2 (BDD `.feature` specs)**: Verify all filtering guards, outbound formatting, error isolation, and lifecycle state transitions.
