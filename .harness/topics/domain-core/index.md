# Topic: `domain-core`

## Purpose
Pure Go business invariants, aggregates, and hexagonal port contracts following **Screaming Architecture (Package by Feature)**. Zero external dependencies inside `internal/{shared,user,learning,touchpoint}/domain/`.

## Authoritative Status
- **Issue #1**: Implemented & verified (28/28 tests, 98.7% coverage; archived in `ai_specs/issue-1/archive-report.md`).
- **Source of Truth**: `AGENTS.md`, `docs/product.md`, `docs/architecture.md`, `docs/domain-model.md`.

## Implemented Packages & Aggregates
- `internal/shared/` — Strongly typed IDs (`UserID`, `PathID`, `SessionID`, `ConceptKey`) and sentinel errors (`ErrInvalidTimezone`, `ErrTouchpointBudgetExhausted`, `ErrInvalidMasteryScore`, `ErrInvalidSessionStateTransition`, `ErrSessionExpired`).
- `internal/user/`
  - `domain.User`: `ID`, `PhoneNumber`, `Timezone` (IANA), `CircadianBaseline{MorningCatalystHour, EveningReflectionHour}`, `DailyTouchesUsed`, `LastTouchSentAt`. Enforces strict **2-touch daily budget** in the user's local calendar day.
  - `ports.go`: `UserRepository` (`FindByID`, `Save`).
- `internal/learning/`
  - `domain.LearningPath`: `ID`, `UserID`, `Title`, `SourceFolderID`, `Status` (`ACTIVE` | `PAUSED` | `COMPLETED`). Isolated "Bounded Context Bag".
  - `domain.MasteryLedger`: `UserID`, `PathID`, `ConceptKey`, `StrengthScore` [0..1], `IntervalDays`, `EaseFactor` (floor 1.3), `NextReviewAt`, `Misconceptions`. SM-2 spaced repetition decay (`RecordReview`, `AddMisconception`, `ClearMisconception`).
  - `ports.go`: `LearningPathRepository`, `MasteryLedgerRepository`, `SourceCorpusPort`, `GroundedChunk`.
- `internal/touchpoint/`
  - `domain.TouchpointSession`: State machine `PENDING` → `AWAITING_ANSWER` → `EVALUATING` → `COMPLETED`, plus silent expiration `AWAITING_ANSWER` → `EXPIRED` (`CheckExpiration`). Types: `MORNING_CATALYST`, `EVENING_STREAK_SAVER`.
  - `ports.go`: `TouchpointRepository`, `MessagingPort`, `LLMPort`, `CalendarTelemetryPort`, `CognitiveBandwidth` (`HIGH` | `LOW`).

## Core Invariants
1. **Zero external deps in `domain/`**: Standard library only — never import SQLite, `whatsmeow`, or LLM SDKs.
2. **Aggregate decoupling by Identity**: Cross-aggregate references use `shared.UserID` / `shared.PathID` only, preventing Go import cycles.
3. **Strict Multi-Tenancy**: Every aggregate, repository query, and vector index is scoped to `user_id`.
4. **Proactivity Budget & Zero-Nag**: Max 2 proactive touches/day; unanswered sessions expire silently with zero follow-up nudges.
5. **Bounded Context Isolation**: Retrieving context from multiple `LearningPath`s must never cross-contaminate `MasteryLedger` state.

## Implemented Port Extensions (Issues #2–#6)
- `internal/user/ports.go`: `UserRepository` (`FindByID`, `FindByPhoneNumber`, `FindAll`, `Save`).
- `internal/learning/ports.go`: `EmbeddingPort` (`EmbedChunks`), `SourceCorpusPort` (`StoreChunks`, `RetrieveChunks`, `IngestPathCorpus`), `LearningPathRepository`, `MasteryLedgerRepository`.
- `internal/touchpoint/ports.go`:
  - `TouchpointRepository` (`FindByID(ctx, userID, id)`, `FindActiveSession`, `FindByUserAndDate`, `Save`).
  - `InboundMessage` + `InboundMessageHandler` (driving port decoupling `whatsmeow` from `InboundRouter`).
  - `PathAffinityPort` (`ClassifyPathAffinity`) and `SynthesisPort` (`AnswerGrounded`, `SynthesizeCrossPath`, `SynthesizeMilestone`, `AnswerGeneral`).
- `internal/shared/errors.go`: Added `ErrMasteryLedgerNotFound` and `ErrTouchpointSessionNotFound`.
