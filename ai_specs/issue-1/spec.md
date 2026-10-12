# Specification: Issue #1 — Core Domain Aggregates & Hexagonal Ports

Change: `issue-1-core-domain-and-ports`  
Status: Accepted  
Conventions: RFC 2119 (MUST, SHALL, SHOULD, MAY), Package-by-Feature Screaming Architecture, Zero External Dependencies for Domains.

---

## 1. Domain Primitive Specifications (`internal/shared`)

### Requirement: Strongly-Typed Identity Types and Sentinel Errors
The system MUST define discrete identity types to eliminate primitive obsession and prevent ID substitution bugs across feature packages. The system MUST define standard sentinel errors for domain boundary and invariant violations.

- Types:
  - `type UserID string`
  - `type PathID string`
  - `type SessionID string`
  - `type ConceptKey string`
- Sentinel Errors:
  - `ErrTouchpointBudgetExhausted`: Proactive touchpoint count exceeds daily quota.
  - `ErrInvalidSessionStateTransition`: Invalid lifecycle transition attempt on session.
  - `ErrSessionExpired`: Interaction window elapsed; silent backoff enforced.
  - `ErrUserNotFound`: Target user identity does not exist.
  - `ErrLearningPathNotFound`: Target learning path does not exist.
  - `ErrInvalidMasteryScore`: Score outside bounds [0.0, 1.0].
  - `ErrInvalidTimezone`: Unparseable IANA timezone identifier.

#### Scenario: Type-Safe Identity Isolation
- GIVEN distinct identifiers for `UserID("usr_1")`, `PathID("pth_1")`, and `SessionID("ses_1")`
- WHEN passed into typed domain methods
- THEN the compiler MUST enforce strict type boundaries preventing cross-assignment.

#### Scenario: Sentinel Error Identification
- GIVEN a domain error returned from an aggregate violation
- WHEN inspected using `errors.Is(err, shared.ErrSessionExpired)`
- THEN it MUST match the specific sentinel error.

---

## 2. Capability: `user-profile-and-budget` (`internal/user`)

### Requirement: User Aggregate Root & Daily Notification Budget
The `User` aggregate MUST encapsulate user identity, circadian baselines, and proactive touchpoint counters. The aggregate MUST enforce a strict budget invariant: at most 2 proactive touches per calendar day, evaluated within the user's localized IANA timezone. The domain package MUST NOT depend on external packages outside Go standard library.

- Entity Struct:
  - `ID`: `shared.UserID`
  - `PhoneNumber`: `string`
  - `Timezone`: `string` (IANA identifier, e.g. "America/New_York")
  - `CircadianBaseline`: `CircadianBaseline` (`MorningCatalystHour int`, `EveningReflectionHour int`)
  - `DailyTouchesUsed`: `int`
  - `LastTouchSentAt`: `*time.Time`

- Behaviors:
  - `CanSendTouchpoint(now time.Time) (bool, error)`: Returns `true` if fewer than 2 touches have occurred on the calendar day in `Timezone`.
  - `RecordTouchpointSent(now time.Time) error`: Increments `DailyTouchesUsed` and updates `LastTouchSentAt`. If budget exhausted on current calendar day, MUST return `ErrTouchpointBudgetExhausted`.
  - `ResetDailyBudget(now time.Time) error`: Manually clears `DailyTouchesUsed` to 0.

#### Scenario: Sending Touchpoint When Budget is Available
- GIVEN a user with `DailyTouchesUsed = 1` in timezone "Europe/Madrid"
- WHEN `CanSendTouchpoint(now)` is called on the same calendar day
- THEN the function MUST return `true, nil`
- AND `RecordTouchpointSent(now)` MUST increment `DailyTouchesUsed` to `2` and update `LastTouchSentAt`.

#### Scenario: Sending Touchpoint When Budget is Exhausted
- GIVEN a user with `DailyTouchesUsed = 2` and `LastTouchSentAt` on the current calendar day
- WHEN `CanSendTouchpoint(now)` is evaluated
- THEN it MUST return `false, ErrTouchpointBudgetExhausted`
- AND `RecordTouchpointSent(now)` MUST reject the mutation with `ErrTouchpointBudgetExhausted`.

#### Scenario: Automatic Budget Rollover on New Calendar Day
- GIVEN a user with `DailyTouchesUsed = 2` on 2026-10-12 in timezone "America/New_York"
- WHEN `CanSendTouchpoint(now)` is called at 00:05:00 on 2026-10-13 in "America/New_York"
- THEN `DailyTouchesUsed` MUST be treated as 0 for the new day
- AND `RecordTouchpointSent(now)` MUST succeed and set `DailyTouchesUsed = 1`.

#### Scenario: Rejection of Invalid Timezone
- GIVEN a user initialized with timezone "Mars/Phobos"
- WHEN `CanSendTouchpoint(now)` is evaluated
- THEN the method MUST return `false, ErrInvalidTimezone`.

### Outbound Port Contract: `UserRepository` (`internal/user/ports.go`)
```go
type UserRepository interface {
    FindByID(ctx context.Context, id shared.UserID) (*domain.User, error)
    Save(ctx context.Context, user *domain.User) error
}
```

---

## 3. Capability: `learning-curriculum-ledger` (`internal/learning`)

### Requirement: Learning Path and Mastery Ledger Aggregates
The `LearningPath` aggregate MUST track path title, source corpus folder, and active status (`Active`, `Paused`, `Completed`).  
The `MasteryLedger` aggregate MUST track concept mastery using spaced-repetition attributes (`StrengthScore`, `IntervalDays`, `EaseFactor`, `NextReviewAt`, `Misconceptions`).  
Every `MasteryLedger` item MUST be strictly partitioned by `(UserID, PathID)` pair. Aggregates across domains MUST reference identities strictly by typed IDs (`shared.UserID`, `shared.PathID`) to avoid memory cycles and ensure multi-tenant boundary compliance.

- Aggregates:
  - `LearningPath`: `ID shared.PathID`, `UserID shared.UserID`, `Title string`, `SourceFolderID string`, `Status PathStatus`.
  - `MasteryLedger`: `UserID shared.UserID`, `PathID shared.PathID`, `ConceptKey shared.ConceptKey`, `StrengthScore float64`, `IntervalDays int`, `EaseFactor float64`, `NextReviewAt time.Time`, `Misconceptions []string`.

- Behaviors:
  - `RecordReview(now time.Time, score float64) error`: Validates `0.0 <= score <= 1.0`. Adjusts `EaseFactor`, recalculates `IntervalDays`, updates `StrengthScore`, and computes `NextReviewAt`.
  - `AddMisconception(m string)`: Appends novel misconception string without duplicates.
  - `ClearMisconception(m string)`: Removes existing misconception once mastered.

#### Scenario: Successful Active-Recall Review Progression
- GIVEN a `MasteryLedger` entry with `IntervalDays = 2`, `EaseFactor = 2.5`, and `StrengthScore = 0.5`
- WHEN `RecordReview(now, 0.9)` is called with a passing score
- THEN `IntervalDays` MUST increase (`IntervalDays >= 5`)
- AND `NextReviewAt` MUST equal `now.AddDate(0, 0, IntervalDays)`
- AND `StrengthScore` MUST increase towards 1.0.

#### Scenario: Failed Review Resets Interval and Captures Misconception
- GIVEN a `MasteryLedger` entry with `IntervalDays = 14`
- WHEN `RecordReview(now, 0.2)` is called with a failing score
- THEN `IntervalDays` MUST reset to `1`
- AND `NextReviewAt` MUST equal `now.AddDate(0, 0, 1)`
- AND calling `AddMisconception("Confused Information Holder with Coordinator")` MUST record the entry in `Misconceptions`.

#### Scenario: Strict Isolation by User and Path
- GIVEN User A with Path 1 and User B with Path 1
- WHEN reviews are recorded for User A on concept "clean-arch"
- THEN User B's `MasteryLedger` for "clean-arch" MUST NOT be modified or accessible through User A's context.

#### Scenario: Validation of Out-of-Bounds Score
- GIVEN a review score of `1.5` or `-0.1`
- WHEN `RecordReview(now, score)` is called
- THEN it MUST return `ErrInvalidMasteryScore` and leave ledger state unmodified.

### Outbound Port Contracts (`internal/learning/ports.go`)
```go
type LearningPathRepository interface {
    FindByID(ctx context.Context, userID shared.UserID, id shared.PathID) (*domain.LearningPath, error)
    FindByUser(ctx context.Context, userID shared.UserID) ([]*domain.LearningPath, error)
    Save(ctx context.Context, path *domain.LearningPath) error
}

type MasteryLedgerRepository interface {
    FindByConcept(ctx context.Context, userID shared.UserID, pathID shared.PathID, concept shared.ConceptKey) (*domain.MasteryLedger, error)
    FindDueReviews(ctx context.Context, userID shared.UserID, pathID shared.PathID, before time.Time) ([]*domain.MasteryLedger, error)
    Save(ctx context.Context, ledger *domain.MasteryLedger) error
}

type GroundedChunk struct {
    ID         string
    PathID     shared.PathID
    Content    string
    Similarity float32
}

type SourceCorpusPort interface {
    RetrieveChunks(ctx context.Context, userID shared.UserID, pathID shared.PathID, query string, limit int) ([]GroundedChunk, error)
    IngestPathCorpus(ctx context.Context, userID shared.UserID, pathID shared.PathID, folderID string) error
}
```

---

## 4. Capability: `touchpoint-session-state` (`internal/touchpoint`)

### Requirement: Touchpoint Session State Machine & Zero-Nag Silent Backoff
The `TouchpointSession` aggregate MUST control the lifecycle of proactive drills. The aggregate SHALL transition across valid states: `Pending` -> `AwaitingAnswer` -> `Evaluating` -> `Completed`. If `now.After(ExpiresAt)` while in `AwaitingAnswer`, the session SHALL enter `Expired`. In state `Expired`, any late submission MUST be rejected with `ErrSessionExpired` to enforce the zero-nag silent backoff rule.

- Entity Struct:
  - `ID`: `shared.SessionID`
  - `UserID`: `shared.UserID`
  - `PathID`: `shared.PathID`
  - `ConceptKey`: `shared.ConceptKey`
  - `State`: `SessionState` (`Pending`, `AwaitingAnswer`, `Evaluating`, `Completed`, `Expired`)
  - `TouchpointType`: `TouchpointType` (`MorningCatalyst`, `EveningStreakSaver`)
  - `PromptText`: `string`
  - `UserAnswer`: `string`
  - `EvaluationFeedback`: `string`
  - `CreatedAt`: `time.Time`
  - `ExpiresAt`: `time.Time`
  - `CompletedAt`: `*time.Time`

- Behaviors:
  - `DispatchPrompt(prompt string, expiresAt time.Time) error`: Transitions `Pending` -> `AwaitingAnswer`.
  - `SubmitAnswer(now time.Time, answer string) error`: If `now.After(ExpiresAt)`, sets state to `Expired` and returns `ErrSessionExpired`. If in `AwaitingAnswer` and unexpired, transitions to `Evaluating` and stores `UserAnswer`. Otherwise returns `ErrInvalidSessionStateTransition`.
  - `Complete(now time.Time, feedback string) error`: Transitions `Evaluating` -> `Completed`, sets `EvaluationFeedback` and `CompletedAt`.
  - `CheckExpiration(now time.Time) bool`: Transitions to `Expired` if `now.After(ExpiresAt)` and state is `AwaitingAnswer`.

#### Scenario: Full Happy-Path Drill Progression
- GIVEN a newly created `TouchpointSession` in state `Pending`
- WHEN `DispatchPrompt("Explain Aggregate Root", now.Add(2*time.Hour))` is executed
- THEN state transitions to `AwaitingAnswer`
- WHEN `SubmitAnswer(now.Add(10*time.Minute), "An aggregate root guarantees consistency boundaries.")` is executed
- THEN state transitions to `Evaluating` with `UserAnswer` recorded
- WHEN `Complete(now.Add(11*time.Minute), "Correct and precise definition.")` is executed
- THEN state transitions to `Completed` with `EvaluationFeedback` and `CompletedAt` set.

#### Scenario: Session Expiration Enforces Silent Backoff
- GIVEN a session in state `AwaitingAnswer` with `ExpiresAt = 10:00:00`
- WHEN `SubmitAnswer(now, "Answer")` is called at `10:01:00`
- THEN the session MUST transition to `Expired`
- AND the method MUST return `ErrSessionExpired`
- AND no further answering or grading SHALL be permitted.

#### Scenario: Rejection of Invalid State Transitions
- GIVEN a session in state `Completed`
- WHEN `SubmitAnswer(now, "Another answer")` is called
- THEN the method MUST return `ErrInvalidSessionStateTransition`
- AND state MUST remain `Completed`.

### Outbound Port Contracts (`internal/touchpoint/ports.go`)
```go
type TouchpointRepository interface {
    FindByID(ctx context.Context, id shared.SessionID) (*domain.TouchpointSession, error)
    FindActiveSession(ctx context.Context, userID shared.UserID) (*domain.TouchpointSession, error)
    Save(ctx context.Context, session *domain.TouchpointSession) error
}

type MessagingPort interface {
    SendText(ctx context.Context, recipientPhoneNumber string, body string) error
    SendDrillPrompt(ctx context.Context, recipientPhoneNumber string, prompt string) error
}

type DrillRequest struct {
    UserID      shared.UserID
    PathID      shared.PathID
    ConceptKey  shared.ConceptKey
    Corpus      []string
    Type        TouchpointType
}

type DrillResponse struct {
    PromptText string
}

type EvaluateRequest struct {
    Prompt     string
    UserAnswer string
    Corpus     []string
}

type EvaluateResponse struct {
    Score    float64
    Feedback string
    Passed   bool
}

type LLMPort interface {
    GenerateMicroDrill(ctx context.Context, req DrillRequest) (*DrillResponse, error)
    EvaluateAnswer(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error)
}

type CognitiveBandwidth string

const (
    BandwidthHigh CognitiveBandwidth = "HIGH"
    BandwidthLow  CognitiveBandwidth = "LOW"
)

type CalendarTelemetryPort interface {
    GetCognitiveBandwidth(ctx context.Context, userID shared.UserID, date time.Time) (CognitiveBandwidth, error)
}
```
