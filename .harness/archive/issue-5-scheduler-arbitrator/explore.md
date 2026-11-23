# Explore: Issue #5 — 2-Touch Proactive Daemon & Attention Arbitrator

## Affected Topics
- `scheduler-arbitrator` (primary)
- `domain-core` (orchestrates `User`, `LearningPath`, `MasteryLedger`, `TouchpointSession`)
- `storage-vectors`, `llm-adapter`, `whatsapp-transport` (wired in `cmd/daemon`)

## Current Codebase Reality
- `User.CanSendTouchpoint(now)` and `User.RecordTouchpointSent(now)` enforce the 2-touch budget in the user's IANA timezone.
- `TouchpointSession.CheckExpiration(now)` handles silent transition `AWAITING_ANSWER` → `EXPIRED`.
- No `internal/touchpoint/services/` or `cmd/daemon/` code exists yet.
- `CalendarTelemetryPort` has no concrete adapter yet; MVP needs a graceful fallback when Google Calendar is not configured.

## Architectural Forks & Tradeoffs
1. **Scheduler Tick Mechanism (`time.Ticker` per minute vs external cron library)**:
   - *Option A (stdlib `time.Ticker` + injected `Clock`)*: Zero extra dependencies, evaluates users in their own IANA timezone (`CircadianBaseline`), trivially testable by calling `RunTick(ctx, now)` directly.
   - *Option B (`robfig/cron`)*: Adds dependency and complicates per-user dynamic timezone schedules.
   - **Recommendation**: Option A — stdlib ticker driving a pure `RunTick(ctx, now)` method with injected `Clock`.
2. **Idempotency Within Circadian Hour**:
   - A 1-minute ticker fires 60 times during the 8:00 AM hour.
   - **Recommendation**: Guard morning/evening dispatch using `TouchpointRepository.FindByUserAndDate` (skip if a `MORNING_CATALYST` or `EVENING_STREAK_SAVER` already exists for the user's local calendar day) AND `user.CanSendTouchpoint(now)`.
