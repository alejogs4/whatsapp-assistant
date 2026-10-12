# Design: feat(domain): implement screaming feature aggregates and hexagonal ports

Change: `issue-1-core-domain-and-ports`  
Status: Accepted  
Related Issue: https://github.com/alejogs4/whatsapp-assistant/issues/1  

---

## 1. Technical Approach

The core objective of Issue #1 is establishing the pure business domain foundation and hexagonal port contracts for the WhatsApp AI Adaptive Learning Assistant. The technical approach comprises three structural pillars:

1. **Go Module Foundation**: Initialize root module `github.com/alejogs4/whatsapp-assistant` (Go 1.25.4) with zero external module dependencies.
2. **Package-by-Feature (Screaming Architecture)**: Organize domain capabilities into feature boundaries (`internal/{shared,user,learning,touchpoint}`) rather than technical layers. Each feature package contains its pure `domain/` sub-package and its segregated `ports.go` interface contracts.
3. **Aggregate Decoupling by Typed Identity**: Eliminate object-graph coupling and cyclic dependencies by decoupling aggregates through strongly-typed identifier value types (`UserID`, `PathID`, `SessionID`, `ConceptKey`) defined in `internal/shared`. Aggregates never hold cross-package pointers or struct references.

---

## 2. Architecture Decisions

### Decision 1: Screaming Architecture (Package by Feature) vs Package by Layer
- **Choice**: Package by Feature (`internal/{shared,user,learning,touchpoint}`).
- **Alternatives Considered**: Package by Layer (`internal/models`, `internal/services`, `internal/repositories`).
- **Rationale**: High cohesion around discrete business capabilities. Screams domain intent (Robert C. Martin Screaming Architecture; Wirfs-Brock RDD). Provides autonomous bounded contexts: changes to circadian baselines or user profiles in `user/` cannot break or recompile `learning/` or `touchpoint/` domain logic.

| Dimension | Package by Layer | Package by Feature (Chosen) |
| :--- | :--- | :--- |
| **Cohesion** | Low (code scattered across layers) | High (isolated per capability) |
| **Coupling** | High (layer-wide shared models) | Low (bounded context boundaries) |
| **Go Import Cycles** | Frequent when layers cross-call | Structurally impossible |

### Decision 2: Aggregate Decoupling via Typed Identifiers vs In-Memory Pointer Graph
- **Choice**: Strongly-typed string identifiers (`shared.UserID`, `shared.PathID`, etc.).
- **Alternatives Considered**: In-memory struct pointer references (e.g. `User.LearningPaths []*LearningPath`).
- **Rationale**: Strictly obeys Eric Evans DDD Rule #3 ("Reference other aggregates by identity only"). Completely eliminates Go circular import compilation errors (`user` $\leftrightarrow$ `learning`). Directly aligns with SQLite row-level multi-tenant partitioning (`WHERE user_id = ? AND path_id = ?`).

| Dimension | Pointer Graph | Typed Identifiers (Chosen) |
| :--- | :--- | :--- |
| **Go Import Cycles** | High risk (`user` $\leftrightarrow$ `learning`) | Zero (points to `shared` kernel only) |
| **Memory Footprint** | Bloated (eager/lazy graph loading) | Minimal (single aggregate boundary) |
| **Multi-Tenancy** | Implicit | Explicit compiler-enforced partition keys |

### Decision 3: Standard Library Only for Domain Aggregates vs External Libraries
- **Choice**: Pure Go standard library (`time`, `math`, `errors`, `context`).
- **Alternatives Considered**: Third-party UUID libraries (`google/uuid`), ORMs (GORM), validation frameworks (`validator`), state machine libraries.
- **Rationale**: Fulfills Invariant 1 from `AGENTS.md`. Eliminates vendor lock-in and supply-chain vulnerabilities, guarantees instant compilation, and preserves clean hexagonal boundaries where domain entities remain 100% agnostic to drivers.

### Decision 4: Deterministic Time & Timezone Injection vs `time.Now()` Calls
- **Choice**: Explicit `now time.Time` parameter injection for domain methods mutating or validating temporal invariants, coupled with IANA timezone evaluation via `time.LoadLocation`.
- **Alternatives Considered**: Calling `time.Now()` directly inside domain methods or using global clock mocks.
- **Rationale**: Guarantees deterministic, hermetic unit testing for circadian midnight rollovers across timezones (e.g. "America/New_York", "Europe/Madrid") and session timeout expirations without flaky sleep timers or global state mutation.

---

## 3. Data Flow / Collaboration

Cross-boundary communication flows strictly from Driving Adapters through Application Services to Domain Aggregates and Outbound Ports:

```
  Driving Adapters                      Domain Layer (Hexagonal Core)                       Driven Adapters
 (Inbound Triggers)            Services                 Domain Aggregates               (Outbound Infra)
                                                   
  [Cron Daemon] ───────► [AttentionArbitrator] ───► User (Checks Budget)
  (UC-1 Catalyst)              │                   - CanSendTouchpoint(now)
                               ├──────────────────► CalendarTelemetryPort ────► [Google Calendar API]
                               │                   - GetCognitiveBandwidth
                               ├──────────────────► LearningPathRepo ──────────► [SQLite DB]
                               │                   - FindByUser
                               ├──────────────────► SourceCorpusPort ──────────► [SQLite / Vector DB]
                               │                   - RetrieveChunks
                               ├──────────────────► LLMPort ───────────────────► [Gemini LLM SDK]
                               │                   - GenerateMicroDrill
                               ├──────────────────► TouchpointSession
                               │                   - DispatchPrompt(...)
                               ├──────────────────► TouchpointRepo ────────────► [SQLite DB]
                               │                   - Save(session)
                               └──────────────────► MessagingPort ─────────────► [whatsmeow Client]
                                                   - SendDrillPrompt
                                                   
  [WhatsApp Inbound] ──► [InboundRouter] ─────────► TouchpointRepo ────────────► [SQLite DB]
  (UC-2 Learner Answer)        │                   - FindActiveSession
                               ├──────────────────► TouchpointSession
                               │                   - SubmitAnswer(now, ans)
                               ├──────────────────► LLMPort ───────────────────► [Gemini LLM SDK]
                               │                   - EvaluateAnswer
                               ├──────────────────► MasteryLedgerRepo ─────────► [SQLite DB]
                               │                   - RecordReview(now, score)
                               ├──────────────────► TouchpointSession
                               │                   - Complete(now, feedback)
                               └──────────────────► MessagingPort ─────────────► [whatsmeow Client]
                                                   - SendText(feedback)
```

---

## 4. File Structure & Changes

| File | Action | Description |
| :--- | :--- | :--- |
| `go.mod` | Create | Module initialization for `github.com/alejogs4/whatsapp-assistant` (Go 1.25.4). |
| `internal/shared/types.go` | Create | Strongly-typed domain identity primitives (`UserID`, `PathID`, `SessionID`, `ConceptKey`). |
| `internal/shared/errors.go` | Create | Domain sentinel errors for budget exhaustion, invalid transitions, expirations, and missing entities. |
| `internal/user/domain/user.go` | Create | `User` aggregate root, `CircadianBaseline`, timezone-aware 2-touch budget state machine. |
| `internal/user/domain/user_test.go` | Create | Table-driven unit tests for budget enforcement, timezone calendar rollovers, and validation. |
| `internal/user/ports.go` | Create | `UserRepository` outbound port interface. |
| `internal/learning/domain/learning_path.go` | Create | `LearningPath` aggregate root, `PathStatus` enum (`ACTIVE`, `PAUSED`, `COMPLETED`). |
| `internal/learning/domain/mastery_ledger.go` | Create | `MasteryLedger` aggregate root, SM-2-adapted spaced repetition algorithm, misconception tracking. |
| `internal/learning/domain/learning_test.go` | Create | Table-driven unit tests for review progression, interval resets, misconception deduplication, and bounds checking. |
| `internal/learning/ports.go` | Create | `LearningPathRepository`, `MasteryLedgerRepository`, `SourceCorpusPort`, and `GroundedChunk` DTO. |
| `internal/touchpoint/domain/touchpoint_session.go` | Create | `TouchpointSession` aggregate root, session lifecycle state machine, timeout silent backoff. |
| `internal/touchpoint/domain/session_test.go` | Create | Table-driven unit tests for session lifecycle transitions, timeout backoff enforcement, and invalid transitions. |
| `internal/touchpoint/ports.go` | Create | `TouchpointRepository`, `MessagingPort`, `LLMPort` (with DTOs), `CalendarTelemetryPort`. |

---

## 5. Interfaces & Contracts

### 5.1 Shared Kernel (`internal/shared`)

#### `internal/shared/types.go`
```go
package shared

type UserID string
type PathID string
type SessionID string
type ConceptKey string
```

#### `internal/shared/errors.go`
```go
package shared

import "errors"

var (
	ErrTouchpointBudgetExhausted      = errors.New("touchpoint budget exhausted for current calendar day")
	ErrInvalidSessionStateTransition = errors.New("invalid session state transition")
	ErrSessionExpired                = errors.New("touchpoint session has expired; silent backoff enforced")
	ErrUserNotFound                  = errors.New("user not found")
	ErrLearningPathNotFound          = errors.New("learning path not found")
	ErrInvalidMasteryScore           = errors.New("mastery score must be between 0.0 and 1.0")
	ErrInvalidTimezone               = errors.New("invalid or unparseable IANA timezone identifier")
)
```

---

### 5.2 User Feature (`internal/user`)

#### `internal/user/domain/user.go`
```go
package domain

import (
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type CircadianBaseline struct {
	MorningCatalystHour   int
	EveningReflectionHour int
}

type User struct {
	ID                shared.UserID
	PhoneNumber       string
	Timezone          string // IANA identifier, e.g. "America/New_York"
	CircadianBaseline CircadianBaseline
	DailyTouchesUsed  int
	LastTouchSentAt   *time.Time
}

func NewUser(id shared.UserID, phone string, tz string, baseline CircadianBaseline) (*User, error) {
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, shared.ErrInvalidTimezone
	}
	return &User{
		ID:                id,
		PhoneNumber:       phone,
		Timezone:          tz,
		CircadianBaseline: baseline,
		DailyTouchesUsed:  0,
		LastTouchSentAt:   nil,
	}, nil
}

func (u *User) isSameCalendarDay(t1, t2 time.Time, loc *time.Location) bool {
	in1 := t1.In(loc)
	in2 := t2.In(loc)
	return in1.Year() == in2.Year() && in1.YearDay() == in2.YearDay()
}

func (u *User) CanSendTouchpoint(now time.Time) (bool, error) {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return false, shared.ErrInvalidTimezone
	}

	if u.LastTouchSentAt == nil || !u.isSameCalendarDay(*u.LastTouchSentAt, now, loc) {
		return true, nil
	}

	if u.DailyTouchesUsed >= 2 {
		return false, shared.ErrTouchpointBudgetExhausted
	}

	return true, nil
}

func (u *User) RecordTouchpointSent(now time.Time) error {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return shared.ErrInvalidTimezone
	}

	if u.LastTouchSentAt != nil && !u.isSameCalendarDay(*u.LastTouchSentAt, now, loc) {
		u.DailyTouchesUsed = 0
	}

	if u.DailyTouchesUsed >= 2 {
		return shared.ErrTouchpointBudgetExhausted
	}

	u.DailyTouchesUsed++
	u.LastTouchSentAt = &now
	return nil
}

func (u *User) ResetDailyBudget(now time.Time) error {
	u.DailyTouchesUsed = 0
	return nil
}
```

#### `internal/user/ports.go`
```go
package user

import (
	"context"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id shared.UserID) (*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
}
```

---

### 5.3 Learning Feature (`internal/learning`)

#### `internal/learning/domain/learning_path.go`
```go
package domain

import "github.com/alejogs4/whatsapp-assistant/internal/shared"

type PathStatus string

const (
	PathStatusActive    PathStatus = "ACTIVE"
	PathStatusPaused    PathStatus = "PAUSED"
	PathStatusCompleted PathStatus = "COMPLETED"
)

type LearningPath struct {
	ID             shared.PathID
	UserID         shared.UserID
	Title          string
	SourceFolderID string
	Status         PathStatus
}

func NewLearningPath(id shared.PathID, userID shared.UserID, title string, sourceFolderID string) *LearningPath {
	return &LearningPath{
		ID:             id,
		UserID:         userID,
		Title:          title,
		SourceFolderID: sourceFolderID,
		Status:         PathStatusActive,
	}
}
```

#### `internal/learning/domain/mastery_ledger.go`
```go
package domain

import (
	"math"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type MasteryLedger struct {
	UserID         shared.UserID
	PathID         shared.PathID
	ConceptKey     shared.ConceptKey
	StrengthScore  float64
	IntervalDays   int
	EaseFactor     float64
	NextReviewAt   time.Time
	Misconceptions []string
}

func NewMasteryLedger(userID shared.UserID, pathID shared.PathID, concept shared.ConceptKey, now time.Time) *MasteryLedger {
	return &MasteryLedger{
		UserID:         userID,
		PathID:         pathID,
		ConceptKey:     concept,
		StrengthScore:  0.0,
		IntervalDays:   0,
		EaseFactor:     2.5,
		NextReviewAt:   now,
		Misconceptions: make([]string, 0),
	}
}

func (m *MasteryLedger) RecordReview(now time.Time, score float64) error {
	if score < 0.0 || score > 1.0 {
		return shared.ErrInvalidMasteryScore
	}

	if score >= 0.6 {
		if m.IntervalDays <= 0 {
			m.IntervalDays = 1
		} else if m.IntervalDays == 1 {
			m.IntervalDays = int(math.Round(float64(m.IntervalDays) * m.EaseFactor))
			if m.IntervalDays < 2 {
				m.IntervalDays = 2
			}
		} else {
			m.IntervalDays = int(math.Round(float64(m.IntervalDays) * m.EaseFactor))
		}
		m.EaseFactor = math.Max(1.3, m.EaseFactor+(0.1-(1.0-score)*(0.08+(1.0-score)*0.02)))
		m.StrengthScore = math.Min(1.0, m.StrengthScore+((score-m.StrengthScore)*0.3))
	} else {
		m.IntervalDays = 1
		m.EaseFactor = math.Max(1.3, m.EaseFactor-0.2)
		m.StrengthScore = math.Max(0.0, m.StrengthScore*0.5)
	}

	m.NextReviewAt = now.AddDate(0, 0, m.IntervalDays)
	return nil
}

func (m *MasteryLedger) AddMisconception(misconception string) {
	for _, existing := range m.Misconceptions {
		if existing == misconception {
			return
		}
	}
	m.Misconceptions = append(m.Misconceptions, misconception)
}

func (m *MasteryLedger) ClearMisconception(misconception string) {
	updated := make([]string, 0, len(m.Misconceptions))
	for _, existing := range m.Misconceptions {
		if existing != misconception {
			updated = append(updated, existing)
		}
	}
	m.Misconceptions = updated
}
```

#### `internal/learning/ports.go`
```go
package learning

import (
	"context"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type GroundedChunk struct {
	ID         string
	PathID     shared.PathID
	Content    string
	Similarity float32
}

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

type SourceCorpusPort interface {
	RetrieveChunks(ctx context.Context, userID shared.UserID, pathID shared.PathID, query string, limit int) ([]GroundedChunk, error)
	IngestPathCorpus(ctx context.Context, userID shared.UserID, pathID shared.PathID, folderID string) error
}
```

---

### 5.4 Touchpoint Feature (`internal/touchpoint`)

#### `internal/touchpoint/domain/touchpoint_session.go`
```go
package domain

import (
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type SessionState string

const (
	SessionStatePending        SessionState = "PENDING"
	SessionStateAwaitingAnswer SessionState = "AWAITING_ANSWER"
	SessionStateEvaluating     SessionState = "EVALUATING"
	SessionStateCompleted      SessionState = "COMPLETED"
	SessionStateExpired        SessionState = "EXPIRED"
)

type TouchpointType string

const (
	TouchpointTypeMorningCatalyst    TouchpointType = "MORNING_CATALYST"
	TouchpointTypeEveningStreakSaver TouchpointType = "EVENING_STREAK_SAVER"
)

type TouchpointSession struct {
	ID                 shared.SessionID
	UserID             shared.UserID
	PathID             shared.PathID
	ConceptKey         shared.ConceptKey
	State              SessionState
	TouchpointType     TouchpointType
	PromptText         string
	UserAnswer         string
	EvaluationFeedback string
	CreatedAt          time.Time
	ExpiresAt          time.Time
	CompletedAt        *time.Time
}

func NewTouchpointSession(
	id shared.SessionID,
	userID shared.UserID,
	pathID shared.PathID,
	concept shared.ConceptKey,
	tpType TouchpointType,
	now time.Time,
) *TouchpointSession {
	return &TouchpointSession{
		ID:             id,
		UserID:         userID,
		PathID:         pathID,
		ConceptKey:     concept,
		State:          SessionStatePending,
		TouchpointType: tpType,
		CreatedAt:      now,
	}
}

func (s *TouchpointSession) DispatchPrompt(prompt string, expiresAt time.Time) error {
	if s.State != SessionStatePending {
		return shared.ErrInvalidSessionStateTransition
	}
	s.PromptText = prompt
	s.ExpiresAt = expiresAt
	s.State = SessionStateAwaitingAnswer
	return nil
}

func (s *TouchpointSession) SubmitAnswer(now time.Time, answer string) error {
	if s.CheckExpiration(now) {
		return shared.ErrSessionExpired
	}
	if s.State != SessionStateAwaitingAnswer {
		return shared.ErrInvalidSessionStateTransition
	}
	s.UserAnswer = answer
	s.State = SessionStateEvaluating
	return nil
}

func (s *TouchpointSession) Complete(now time.Time, feedback string) error {
	if s.State != SessionStateEvaluating {
		return shared.ErrInvalidSessionStateTransition
	}
	s.EvaluationFeedback = feedback
	s.CompletedAt = &now
	s.State = SessionStateCompleted
	return nil
}

func (s *TouchpointSession) CheckExpiration(now time.Time) bool {
	if s.State == SessionStateAwaitingAnswer && now.After(s.ExpiresAt) {
		s.State = SessionStateExpired
		return true
	}
	return s.State == SessionStateExpired
}
```

#### `internal/touchpoint/ports.go`
```go
package touchpoint

import (
	"context"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

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
	UserID     shared.UserID
	PathID     shared.PathID
	ConceptKey shared.ConceptKey
	Corpus     []string
	Type       domain.TouchpointType
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

---

## 6. Testing Strategy

| Layer | What to Test | Approach |
| :--- | :--- | :--- |
| **Unit (`user/domain`)** | `CanSendTouchpoint`, `RecordTouchpointSent`, `ResetDailyBudget` | Standard table-driven tests (`tests := []struct{ name string; ... }`) with `t.Run`. Validates 2-touch quota, calendar rollover in IANA timezones ("America/New_York", "Europe/Madrid"), and rejection of invalid timezones. |
| **Unit (`learning/domain`)** | `RecordReview`, `AddMisconception`, `ClearMisconception` | Table-driven tests verifying SM-2 progression (interval increase on $\ge 0.6$, reset to 1 on failure, ease factor adjustment), misconception deduplication, score boundaries ($[0.0, 1.0]$), and multi-tenant partition isolation. |
| **Unit (`touchpoint/domain`)** | Lifecycle transitions (`DispatchPrompt`, `SubmitAnswer`, `Complete`, `CheckExpiration`) | Table-driven tests validating full happy-path drill progression, timeout expiration triggering silent backoff (`ErrSessionExpired`), and illegal state transition rejections (`ErrInvalidSessionStateTransition`). |
