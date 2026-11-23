package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/services"
	userdomain "github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

type stubEmbedder struct{}

func (stubEmbedder) EmbedChunks(_ context.Context, chunks []string) ([][]float32, error) {
	out := make([][]float32, len(chunks))
	for i := range chunks {
		out[i] = []float32{1, 0, 0}
	}
	return out, nil
}

type stubTelemetry struct {
	bw  touchpoint.CognitiveBandwidth
	err error
}

func (s stubTelemetry) GetCognitiveBandwidth(_ context.Context, _ shared.UserID, _ time.Time) (touchpoint.CognitiveBandwidth, error) {
	return s.bw, s.err
}

type fakeLLM struct {
	drillCalls int
	evalCalls  int
	drillText  string
	evalResp   *touchpoint.EvaluateResponse
}

func (f *fakeLLM) GenerateMicroDrill(_ context.Context, _ touchpoint.DrillRequest) (*touchpoint.DrillResponse, error) {
	f.drillCalls++
	return &touchpoint.DrillResponse{PromptText: f.drillText}, nil
}

func (f *fakeLLM) EvaluateAnswer(_ context.Context, _ touchpoint.EvaluateRequest) (*touchpoint.EvaluateResponse, error) {
	f.evalCalls++
	return f.evalResp, nil
}

type fakeMessaging struct {
	texts  []string
	drills []string
}

func (f *fakeMessaging) SendText(_ context.Context, _ string, body string) error {
	f.texts = append(f.texts, body)
	return nil
}

func (f *fakeMessaging) SendDrillPrompt(_ context.Context, _ string, prompt string) error {
	f.drills = append(f.drills, prompt)
	return nil
}

type testHarness struct {
	db          *sqlite.DB
	users       *sqlite.UserRepo
	paths       *sqlite.LearningPathRepo
	ledgers     *sqlite.MasteryLedgerRepo
	sessions    *sqlite.TouchpointRepo
	corpus      *sqlite.CorpusStore
	llm         *fakeLLM
	messaging   *fakeMessaging
	arbitrator  *services.AttentionArbitrator
	coordinator *services.DrillCoordinator
	evaluator   *services.PedagogicalEvaluator
}

func newTestHarness(t *testing.T, telemetry touchpoint.CalendarTelemetryPort) *testHarness {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	users := sqlite.NewUserRepo(db)
	paths := sqlite.NewLearningPathRepo(db)
	ledgers := sqlite.NewMasteryLedgerRepo(db)
	sessions := sqlite.NewTouchpointRepo(db)
	corpus := sqlite.NewCorpusStore(db, stubEmbedder{}, nil)
	llm := &fakeLLM{
		drillText: "Explain the Information Holder stereotype in one sentence.",
		evalResp:  &touchpoint.EvaluateResponse{Score: 0.9, Feedback: "Spot on!", Passed: true},
	}
	msg := &fakeMessaging{}
	arb := services.NewAttentionArbitrator(paths, ledgers, telemetry)
	coord := services.NewDrillCoordinator(users, corpus, sessions, llm, msg, arb)
	eval := services.NewPedagogicalEvaluator(corpus, ledgers, sessions, llm, msg)

	return &testHarness{
		db:          db,
		users:       users,
		paths:       paths,
		ledgers:     ledgers,
		sessions:    sessions,
		corpus:      corpus,
		llm:         llm,
		messaging:   msg,
		arbitrator:  arb,
		coordinator: coord,
		evaluator:   eval,
	}
}

func seedUserAndPaths(t *testing.T, h *testHarness, now time.Time) *userdomain.User {
	t.Helper()
	ctx := context.Background()
	u, err := userdomain.NewUser("u-1", "+15551234567", "America/New_York", userdomain.CircadianBaseline{
		MorningCatalystHour:   8,
		EveningReflectionHour: 20,
	})
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	_ = h.users.Save(ctx, u)

	pRDD := learningdomain.NewLearningPath("p-rdd", "u-1", "RDD", "f-rdd")
	pAlgo := learningdomain.NewLearningPath("p-algo", "u-1", "Algorithms", "f-algo")
	pComp := learningdomain.NewLearningPath("p-comp", "u-1", "Compilers", "f-comp")
	pComp.Status = learningdomain.PathStatusPaused

	_ = h.paths.Save(ctx, pRDD)
	_ = h.paths.Save(ctx, pAlgo)
	_ = h.paths.Save(ctx, pComp)

	lRDD := learningdomain.NewMasteryLedger("u-1", "p-rdd", "stereotypes", now.Add(-72*time.Hour))
	lAlgo := learningdomain.NewMasteryLedger("u-1", "p-algo", "dijkstra", now.Add(-24*time.Hour))
	lComp := learningdomain.NewMasteryLedger("u-1", "p-comp", "lexing", now.Add(-240*time.Hour))

	_ = h.ledgers.Save(ctx, lRDD)
	_ = h.ledgers.Save(ctx, lAlgo)
	_ = h.ledgers.Save(ctx, lComp)

	_ = h.corpus.StoreChunks(ctx, "u-1", "p-rdd", []string{"Information Holder knows and provides information."})
	_ = h.corpus.StoreChunks(ctx, "u-1", "p-algo", []string{"Dijkstra finds shortest paths in non-negative graphs."})
	return u
}

func TestAttentionArbitrator_PicksMostOverdueActiveAndFallsBackOnTelemetryError(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC) // 08:00 America/New_York
	h := newTestHarness(t, stubTelemetry{err: errors.New("calendar offline")})
	seedUserAndPaths(t, h, now)

	sel, err := h.arbitrator.SelectTheOneThing(context.Background(), "u-1", now)
	if err != nil {
		t.Fatalf("SelectTheOneThing: %v", err)
	}
	if sel.Path.ID != "p-rdd" || sel.Concept != "stereotypes" {
		t.Fatalf("expected p-rdd/stereotypes, got %s/%s", sel.Path.ID, sel.Concept)
	}
	if sel.Bandwidth != touchpoint.BandwidthHigh {
		t.Fatalf("expected fallback BandwidthHigh, got %s", sel.Bandwidth)
	}
}

func TestDrillCoordinator_DispatchesAndEnforcesBudget(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)
	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	u := seedUserAndPaths(t, h, now)
	ctx := context.Background()

	sess, err := h.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeMorningCatalyst, now)
	if err != nil {
		t.Fatalf("TriggerTouchpoint: %v", err)
	}
	if sess.State != tpdomain.SessionStateAwaitingAnswer {
		t.Fatalf("expected AWAITING_ANSWER, got %s", sess.State)
	}
	if u.DailyTouchesUsed != 1 || len(h.messaging.drills) != 1 {
		t.Fatalf("expected 1 touch and 1 drill sent, got touches=%d drills=%d", u.DailyTouchesUsed, len(h.messaging.drills))
	}

	// Exhaust budget
	_ = u.RecordTouchpointSent(now.Add(time.Hour))
	_, err = h.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeEveningStreakSaver, now.Add(2*time.Hour))
	if !errors.Is(err, shared.ErrTouchpointBudgetExhausted) {
		t.Fatalf("expected ErrTouchpointBudgetExhausted, got %v", err)
	}
	if h.llm.drillCalls != 1 {
		t.Fatalf("expected LLM not to be called when budget exhausted, got %d calls", h.llm.drillCalls)
	}
}

func TestPedagogicalEvaluator_CompletesSessionAndHandlesExpiration(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)
	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	u := seedUserAndPaths(t, h, now)
	ctx := context.Background()

	sess, err := h.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeMorningCatalyst, now)
	if err != nil {
		t.Fatalf("TriggerTouchpoint: %v", err)
	}

	evalResp, err := h.evaluator.EvaluateActiveSession(ctx, u, sess, "It stores and provides state.", now.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("EvaluateActiveSession: %v", err)
	}
	if !evalResp.Passed || sess.State != tpdomain.SessionStateCompleted {
		t.Fatalf("expected completed passing session, got state=%s resp=%+v", sess.State, evalResp)
	}
	ledger, _ := h.ledgers.FindByConcept(ctx, "u-1", "p-rdd", "stereotypes")
	if ledger.StrengthScore <= 0 {
		t.Fatalf("expected MasteryLedger StrengthScore > 0, got %f", ledger.StrengthScore)
	}
	if len(h.messaging.texts) != 1 || h.messaging.texts[0] != "Spot on!" {
		t.Fatalf("expected feedback sent, got %v", h.messaging.texts)
	}

	// Expired session check
	expiredSess := tpdomain.NewTouchpointSession("s-exp", "u-1", "p-rdd", "stereotypes", tpdomain.TouchpointTypeEveningStreakSaver, now)
	_ = expiredSess.DispatchPrompt("prompt", now.Add(time.Minute))
	_ = h.sessions.Save(ctx, expiredSess)

	_, err = h.evaluator.EvaluateActiveSession(ctx, u, expiredSess, "late answer", now.Add(2*time.Hour))
	if !errors.Is(err, shared.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
	reloaded, _ := h.sessions.FindByID(ctx, "u-1", "s-exp")
	if reloaded.State != tpdomain.SessionStateExpired {
		t.Fatalf("expected persisted EXPIRED state, got %s", reloaded.State)
	}
}

func TestScheduler_IdempotentMorningSilentBackoffAndEveningStreakSaver(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	morning800 := time.Date(2026, 11, 23, 8, 0, 0, 0, loc).UTC()
	morning805 := time.Date(2026, 11, 23, 8, 5, 0, 0, loc).UTC()
	afternoon1400 := time.Date(2026, 11, 23, 14, 0, 0, 0, loc).UTC()
	evening2000 := time.Date(2026, 11, 23, 20, 0, 0, 0, loc).UTC()
	evening2015 := time.Date(2026, 11, 23, 20, 15, 0, 0, loc).UTC()

	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	seedUserAndPaths(t, h, morning800)
	ctx := context.Background()

	sched := services.NewScheduler(h.users, h.sessions, h.coordinator, h.messaging, services.RealClock{}, time.Minute, nil)

	// 08:00 & 08:05 -> exactly 1 morning drill
	if err := sched.RunTick(ctx, morning800); err != nil {
		t.Fatalf("RunTick 08:00: %v", err)
	}
	if err := sched.RunTick(ctx, morning805); err != nil {
		t.Fatalf("RunTick 08:05: %v", err)
	}
	if len(h.messaging.drills) != 1 {
		t.Fatalf("expected exactly 1 morning drill, got %d", len(h.messaging.drills))
	}

	// 14:00 -> unanswered morning drill expires silently (0 messages sent)
	if err := sched.RunTick(ctx, afternoon1400); err != nil {
		t.Fatalf("RunTick 14:00: %v", err)
	}
	if len(h.messaging.drills) != 1 || len(h.messaging.texts) != 0 {
		t.Fatalf("expected silent backoff (0 new messages), got drills=%d texts=%d", len(h.messaging.drills), len(h.messaging.texts))
	}
	if _, err := h.sessions.FindActiveSession(ctx, "u-1"); !errors.Is(err, shared.ErrTouchpointSessionNotFound) {
		t.Fatalf("expected no active session after silent expiration, got %v", err)
	}

	// 20:00 -> evening streak saver fires (budget 2/2); 20:15 -> 0 messages
	if err := sched.RunTick(ctx, evening2000); err != nil {
		t.Fatalf("RunTick 20:00: %v", err)
	}
	if err := sched.RunTick(ctx, evening2015); err != nil {
		t.Fatalf("RunTick 20:15: %v", err)
	}
	if len(h.messaging.drills) != 2 {
		t.Fatalf("expected 2 total drills (morning + evening streak saver), got %d", len(h.messaging.drills))
	}
	reloadedUser, _ := h.users.FindByID(ctx, "u-1")
	if reloadedUser.DailyTouchesUsed != 2 {
		t.Fatalf("expected DailyTouchesUsed=2, got %d", reloadedUser.DailyTouchesUsed)
	}
}

var _ learning.EmbeddingPort = stubEmbedder{}
