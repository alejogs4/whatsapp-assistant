# Tasks: feat(domain): implement screaming feature aggregates and hexagonal ports

Change: `issue-1-core-domain-and-ports`  
Status: Accepted  
Related Issue: https://github.com/alejogs4/whatsapp-assistant/issues/1  

## Phase 1: Project & Shared Kernel Foundation
- [ ] 1.1 Initialize Go module `github.com/alejogs4/whatsapp-assistant` via `go mod init`.
- [ ] 1.2 Implement `internal/shared/types.go` declaring strongly-typed identity types (`UserID`, `PathID`, `SessionID`, `ConceptKey`).
- [ ] 1.3 Implement `internal/shared/errors.go` defining domain sentinel errors (`ErrTouchpointBudgetExhausted`, `ErrInvalidSessionStateTransition`, `ErrSessionExpired`, `ErrUserNotFound`, `ErrLearningPathNotFound`, `ErrInvalidMasteryScore`, `ErrInvalidTimezone`).

## Phase 2: User Feature & Proactivity Budget (TDD)
- [ ] 2.1 [RED] Write failing table-driven unit tests in `internal/user/domain/user_test.go` covering budget quota (2 touches max), IANA timezone rollover ("America/New_York", "Europe/Madrid"), and invalid timezones.
- [ ] 2.2 [GREEN] Implement `User` aggregate root and budget methods (`CanSendTouchpoint`, `RecordTouchpointSent`, `ResetDailyBudget`) in `internal/user/domain/user.go` to make tests pass.
- [ ] 2.3 [PORT] Define `UserRepository` outbound port interface in `internal/user/ports.go`.

## Phase 3: Learning Curriculum & Mastery Ledger (TDD)
- [ ] 3.1 [RED] Write failing table-driven unit tests in `internal/learning/domain/learning_test.go` covering `RecordReview` SM-2 interval progression/reset, ease factor adjustments, score bounds validation ([0.0, 1.0]), and misconception deduplication.
- [ ] 3.2 [GREEN] Implement `LearningPath` aggregate and `PathStatus` enum in `internal/learning/domain/learning_path.go`.
- [ ] 3.3 [GREEN] Implement `MasteryLedger` aggregate and review algorithms in `internal/learning/domain/mastery_ledger.go` to make tests pass.
- [ ] 3.4 [PORT] Define `LearningPathRepository`, `MasteryLedgerRepository`, `SourceCorpusPort`, and `GroundedChunk` DTO in `internal/learning/ports.go`.

## Phase 4: Touchpoint Proactive Drill Session & State Machine (TDD)
- [ ] 4.1 [RED] Write failing table-driven unit tests in `internal/touchpoint/domain/session_test.go` covering lifecycle transitions (`Pending` -> `AwaitingAnswer` -> `Evaluating` -> `Completed`), silent backoff expiration timeout rejection, and invalid transition guards.
- [ ] 4.2 [GREEN] Implement `TouchpointSession` aggregate and state machine in `internal/touchpoint/domain/touchpoint_session.go` to make tests pass.
- [ ] 4.3 [PORT] Define `TouchpointRepository`, `MessagingPort`, `LLMPort` (with DTOs), and `CalendarTelemetryPort` in `internal/touchpoint/ports.go`.

## Phase 5: Full Suite Verification & Zero-Dependency Validation
- [ ] 5.1 Run `go test -v ./...` verifying 100% test pass rate across all domain packages.
- [ ] 5.2 Verify zero external module dependencies in `go.mod` (Go standard library only).
- [ ] 5.3 Verify zero cyclic imports across packages via `go vet ./...`.
