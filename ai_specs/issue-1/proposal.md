# Proposal: feat(domain): implement screaming feature aggregates and hexagonal ports

Change: `issue-1-core-domain-and-ports`  
Status: Accepted  
Related Issue: https://github.com/alejogs4/whatsapp-assistant/issues/1

## Intent
Establish the core business domain foundations for the WhatsApp AI Adaptive Learning Assistant using **Package-by-Feature (Screaming Architecture)** and Hexagonal Ports. Guarantee zero external dependencies in domain packages, compile-time cyclic-import prevention via Identity (ID) decoupling, and strict `user_id` multi-tenancy from Day 1.

## Scope

### In Scope
- `internal/shared`: Typed domain IDs (`UserID`, `PathID`, `SessionID`, `ConceptKey`) and domain sentinel errors.
- `internal/user/domain`: `User` aggregate, circadian timezone profile, daily 2-touch budget state machine (`CanSendTouchpoint`, `RecordTouchpointSent`, `ResetDailyBudget`).
- `internal/user/ports.go`: Segregated `UserRepository` port.
- `internal/learning/domain`: `LearningPath` & `MasteryLedger` aggregates, concept graph references, spaced repetition state (`RecordReview`), misconception tracking.
- `internal/learning/ports.go`: `LearningPathRepository`, `MasteryLedgerRepository`, `SourceCorpusPort`.
- `internal/touchpoint/domain`: `TouchpointSession` aggregate, lifecycle state machine (`Pending` -> `AwaitingAnswer` -> `Evaluating` -> `Completed` / `Expired`), timeout and zero-nag silent backoff rules.
- `internal/touchpoint/ports.go`: `TouchpointRepository`, `MessagingPort`, `LLMPort`, `CalendarTelemetryPort`.
- Table-driven unit tests covering 100% of domain invariants and state transitions.

### Out of Scope
- Concrete adapter implementations (SQLite, whatsmeow, Gemini SDK, Google Calendar).
- Database migrations, SQL schema definitions (deferred to Issue #2).
- Network protocols, CLI wiring, or daemon runtime.

## Capabilities

### New Capabilities
- `shared-domain-primitives`: Type-safe identities (`UserID`, `PathID`, `SessionID`, `ConceptKey`) preventing primitive obsession and compile-time ID substitution bugs.
- `user-profile-and-budget`: Circadian baseline tracking and strict 2-touch proactive daily notification budget.
- `learning-curriculum-ledger`: Bounded-context curriculum tracking and spaced-repetition mastery ledger with misconception scoring.
- `touchpoint-session-state`: Adaptive proactive drill session lifecycle, answer state transitions, and zero-nag backoff policy.
- `hexagonal-contracts`: Segregated driven and driving ports decoupling domain logic from I/O infrastructure.

## Approach
- **Package-by-Feature Screaming Architecture**: Organize by capability (`internal/{shared,user,learning,touchpoint}`) rather than generic technical layers.
- **Aggregate Decoupling by Identity**: Aggregates reference each other strictly via typed IDs (`UserID`, `PathID`) from `internal/shared/`, preventing Go cyclic imports.
- **Pure Domain Invariants**: Domain packages use standard-library Go only; zero external imports or framework code.
- **Deterministic State Machines**: Explicit state transitions for `TouchpointSession` and budget counters with invariant validation.
- **Segregated Ports**: Colocate interfaces with their owning feature boundary obeying the Interface Segregation Principle (ISP).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/shared/` | New | Typed IDs (`UserID`, `PathID`, etc.) and shared value types |
| `internal/user/domain/` | New | `User` aggregate, circadian profile, 2-touch budget |
| `internal/user/ports.go` | New | `UserRepository` port contract |
| `internal/learning/domain/` | New | `LearningPath`, `MasteryLedger`, spaced-repetition state |
| `internal/learning/ports.go` | New | `LearningPathRepository`, `MasteryLedgerRepository`, `SourceCorpusPort` |
| `internal/touchpoint/domain/` | New | `TouchpointSession` aggregate and state machine |
| `internal/touchpoint/ports.go` | New | `TouchpointRepository`, `MessagingPort`, `LLMPort`, `CalendarTelemetryPort` |
| `internal/*/*_test.go` | New | Table-driven unit tests for all domain transitions |

## Rollback Plan
Revert commit; domain packages have zero infrastructure dependencies and introduce no stateful side effects.

## Success Criteria
- [ ] Feature packages compile cleanly with zero external module dependencies.
- [ ] 100% domain state transitions covered by table-driven unit tests (`t.Run`).
- [ ] Zero cyclic dependencies across packages.
- [ ] `go test ./internal/...` passes with zero failures.
