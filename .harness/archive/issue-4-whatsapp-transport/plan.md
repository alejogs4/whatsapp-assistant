# Plan: Issue #4 — WhatsApp Multi-Device Transport Adapter (`whatsmeow`)

## Affected Topics
- `whatsapp-transport`
- `domain-core`

## Outcomes (WHAT)
1. **Protocol-Isolated Inbound Pipeline**: `whatsmeow` events are filtered (1:1 DMs only, ignoring self/group/broadcast/empty messages), normalized to E.164 phone + text, and dispatched to `touchpoint.InboundMessageHandler`.
2. **Outbound `MessagingPort` Adapter**: Delivers plain text and formatted micro-drill prompts via `whatsmeow` without leaking protocol types into domain packages.
3. **Persistent Multi-Device Session & QR Pairing**: First startup prints a scannable terminal QR code; subsequent startups reconnect using persisted SQLite device credentials and shut down cleanly on context cancellation.

## Non-Goals
- Meta Cloud API adapter (deferred post-MVP; swappable via `MessagingPort`).
- Voice note transcription, image/PDF attachments, or WhatsApp group chats.
- Automated live-WhatsApp E2E tests in CI.
