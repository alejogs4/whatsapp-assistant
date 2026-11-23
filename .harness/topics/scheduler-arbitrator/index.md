# Topic: `scheduler-arbitrator`

## Purpose
Proactive background scheduler (`cmd/daemon` / `internal/touchpoint/services/`) and **Attention Arbitrator** enforcing the 2-touch daily budget, cognitive-load adaptation, **"The ONE Thing"** selection, and zero-nag silent backoff.

## Scope & Related Issue
- **Issue #5**: `feat(scheduler): implement 2-touch proactive daemon and attention arbitrator`
- **Key Services** (`internal/touchpoint/services/`):
  - `AttentionArbitrator`: Evaluates active `LearningPath`s + `MasteryLedger` due reviews + `CalendarTelemetryPort` bandwidth → picks **The ONE Thing** (zero decision fatigue).
  - `DrillCoordinator`: Retrieves grounded chunks + misconceptions → invokes `LLMPort.GenerateMicroDrill` → creates `TouchpointSession` (`AWAITING_ANSWER`) → dispatches via `MessagingPort`.
  - `Scheduler`: Minute-tick (or cron) loop evaluating each user in their IANA `User.Timezone` against `CircadianBaseline` (`MorningCatalystHour`, `EveningReflectionHour`).
  - `PedagogicalEvaluator` (UC-2 completion): Evaluates active drill answer → updates `MasteryLedger.RecordReview` → completes `TouchpointSession` → sends feedback.

## Architectural Invariants
1. **Strict 2-Touch Budget**: Every proactive trigger checks `user.CanSendTouchpoint(now)` and persists `user.RecordTouchpointSent(now)` atomically with session creation. Never exceed 2/day.
2. **Zero-Nag Silent Backoff**:
   - Before scheduling a new touchpoint, check any active `TouchpointSession` with `CheckExpiration(now)`.
   - If morning drill expired unanswered, mark `EXPIRED` silently — do NOT send a reminder or guilt message.
   - Evening touchpoint adapts: if morning completed → celebratory micro-reflection; if morning missed or bandwidth `LOW` → 15-second zero-friction recall card; then budget is 2/2 and system stays silent until next local calendar day.
3. **Deterministic Clock Injection**: All time-sensitive services accept a `Clock` interface (`Now() time.Time`) so morning/evening windows, timezone boundaries, DST, and expiration backoff are 100% deterministically unit-testable.
4. **Calendar Telemetry Fallback**: If `CalendarTelemetryPort` is unavailable or unconfigured in MVP, degrade gracefully to circadian baseline defaults (High in morning, Low in evening) without crashing.

## Gotchas & Constraints
- Idempotency: A scheduler tick firing twice within the same morning/evening hour must NOT send duplicate touchpoints (guard via today's existing touchpoint sessions + `DailyTouchesUsed`).
- `cmd/daemon` wires config, SQLite, Gemini, WhatsApp, Calendar stub/adapter, and Scheduler with clean shutdown.
