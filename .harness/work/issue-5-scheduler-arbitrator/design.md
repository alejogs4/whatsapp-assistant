# Design: Issue #5 — 2-Touch Proactive Daemon & Attention Arbitrator

## Decisions & Tradeoffs
1. **Deterministic "The ONE Thing" Ranking (`AttentionArbitrator`)**:
   - Considers only `PathStatusActive` paths.
   - Scores candidates by overdue spaced-repetition urgency (`now.Sub(NextReviewAt)`), active misconceptions count, and lowest `StrengthScore`.
   - Ties broken deterministically by `PathID` then `ConceptKey`.
   - When `CognitiveBandwidth == LOW`, prefers a shorter/higher-familiarity concept or switches to `EVENING_STREAK_SAVER` friction.
2. **Zero-Nag Silent Backoff (`Scheduler`)**:
   - On every tick, before evaluating triggers, checks `FindActiveSession(ctx, user.ID)` and calls `session.CheckExpiration(now)`; if expired, persists `EXPIRED` silently and sends **zero** messages.
   - At `EveningReflectionHour`: inspects today's sessions via `FindByUserAndDate`. If morning session is `COMPLETED` → sends celebratory micro-reflection; if missed/expired → sends 15-second `EVENING_STREAK_SAVER` recall card. Marks 2/2 used; silent until next local calendar day.
3. **Graceful Calendar Fallback**:
   - `DefaultTelemetryAdapter` returns `BandwidthHigh` for morning and `BandwidthLow` for evening if Google Calendar is not configured or errors out.
4. **Pure Service Layer (`internal/touchpoint/services/`)**:
   - Services depend strictly on port interfaces (`UserRepository`, `LearningPathRepository`, `MasteryLedgerRepository`, `SourceCorpusPort`, `TouchpointRepository`, `LLMPort`, `MessagingPort`, `CalendarTelemetryPort`, `Clock`).

## Package & Contract Boundaries
- `internal/touchpoint/services/`:
  - `clock.go`: `Clock` interface (`Now() time.Time`) + `RealClock`.
  - `arbitrator.go`: `AttentionArbitrator.SelectTheOneThing(ctx, userID, now)`.
  - `drill_coordinator.go`: `DrillCoordinator.TriggerTouchpoint(ctx, user, tpType, now)`.
  - `evaluator.go`: `PedagogicalEvaluator.EvaluateActiveSession(ctx, user, session, answer, now)`.
  - `scheduler.go`: `Scheduler.RunTick(ctx, now)` & `Scheduler.Start(ctx)`.
- `cmd/daemon/main.go`: Composition root wiring adapters, services, and `signal.NotifyContext`.

## Verification Strategy
- **Tier 1 (`harness check`)**: `go vet ./...` and `go test -race ./...` using fake repositories, fake LLM, fake messaging, and a controllable fake `Clock`.
- **Tier 2 (BDD `.feature` specs)**: Verify 2-touch budget cap, morning/evening timezone windows, silent zero-nag expiration, idempotent ticks, and UC-2 evaluation mastery updates.
