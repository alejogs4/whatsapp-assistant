package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/services"
)

type fakeAffinity struct {
	calls  int
	result *touchpoint.AffinityResult
}

func (f *fakeAffinity) ClassifyPathAffinity(_ context.Context, _ string, _ []touchpoint.PathSummary) (*touchpoint.AffinityResult, error) {
	f.calls++
	return f.result, nil
}

type fakeSynthesis struct {
	groundedCalls   int
	crossPathCalls  int
	crossPathInputs []touchpoint.PathCorpusContext
	milestoneCalls  int
	generalCalls    int
}

func (f *fakeSynthesis) AnswerGrounded(_ context.Context, _ string, pc touchpoint.PathCorpusContext) (string, error) {
	f.groundedCalls++
	return "Grounded(" + pc.PathTitle + ")", nil
}

func (f *fakeSynthesis) SynthesizeCrossPath(_ context.Context, _ string, ctxs []touchpoint.PathCorpusContext) (string, error) {
	f.crossPathCalls++
	f.crossPathInputs = ctxs
	return "CrossPathUnified", nil
}

func (f *fakeSynthesis) SynthesizeMilestone(_ context.Context, req touchpoint.MilestoneRequest) (*touchpoint.MilestoneRecommendation, error) {
	f.milestoneCalls++
	return &touchpoint.MilestoneRecommendation{
		ProposedPath: "Design Patterns",
		Reasoning:    "Natural successor to " + req.CompletedPathName,
	}, nil
}

func (f *fakeSynthesis) AnswerGeneral(_ context.Context, _ string) (string, error) {
	f.generalCalls++
	return "GeneralAnswer", nil
}

func TestCommandController_ScheduleAndSkipWithoutLedgerMutation(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)
	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	u := seedUserAndPaths(t, h, now)
	ctx := context.Background()

	cmd := services.NewCommandController(h.users, h.paths, h.sessions, h.messaging)

	handled, err := cmd.Handle(ctx, u, "/schedule morning 9")
	if err != nil || !handled {
		t.Fatalf("Handle /schedule failed: handled=%v err=%v", handled, err)
	}
	reloaded, _ := h.users.FindByID(ctx, "u-1")
	if reloaded.CircadianBaseline.MorningCatalystHour != 9 {
		t.Fatalf("expected MorningCatalystHour=9, got %d", reloaded.CircadianBaseline.MorningCatalystHour)
	}
	if h.llm.drillCalls != 0 || h.llm.evalCalls != 0 {
		t.Fatalf("expected 0 LLM calls for command")
	}

	// /skip expires active session without mutating MasteryLedger
	sess, _ := h.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeMorningCatalyst, now)
	beforeLedger, _ := h.ledgers.FindByConcept(ctx, "u-1", sess.PathID, sess.ConceptKey)

	handled, err = cmd.Handle(ctx, u, "/skip")
	if err != nil || !handled {
		t.Fatalf("Handle /skip failed: handled=%v err=%v", handled, err)
	}
	reloadedSess, _ := h.sessions.FindByID(ctx, "u-1", sess.ID)
	if reloadedSess.State != tpdomain.SessionStateExpired {
		t.Fatalf("expected EXPIRED session after /skip, got %s", reloadedSess.State)
	}
	afterLedger, _ := h.ledgers.FindByConcept(ctx, "u-1", sess.PathID, sess.ConceptKey)
	if afterLedger.StrengthScore != beforeLedger.StrengthScore {
		t.Fatalf("MasteryLedger should not change on /skip")
	}
}

func TestScatterGather_ConcurrentRetrievalAndPartialEmptyTolerance(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)
	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	seedUserAndPaths(t, h, now)
	ctx := context.Background()

	// Add an AI path with 0 chunks to verify partial empty tolerance
	pAI := learningdomain.NewLearningPath("p-ai", "u-1", "AI", "f-ai")
	_ = h.paths.Save(ctx, pAI)
	pRDD, _ := h.paths.FindByID(ctx, "u-1", "p-rdd")
	pAlgo, _ := h.paths.FindByID(ctx, "u-1", "p-algo")

	syn := &fakeSynthesis{}
	sg := services.NewCrossPathSynthesizer(h.corpus, syn, 3)

	ans, err := sg.Synthesize(ctx, "u-1", "How do graphs and RDD relate?", []*learningdomain.LearningPath{pRDD, pAlgo, pAI})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if ans != "CrossPathUnified" || syn.crossPathCalls != 1 {
		t.Fatalf("expected CrossPathUnified, got %q (calls=%d)", ans, syn.crossPathCalls)
	}
	// pRDD and pAlgo have chunks; pAI had 0 chunks and was gracefully skipped
	if len(syn.crossPathInputs) != 2 {
		t.Fatalf("expected 2 non-empty path contexts, got %d", len(syn.crossPathInputs))
	}
}

func TestMacroCoach_MilestoneThresholdDetectionAndAdvice(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)
	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	seedUserAndPaths(t, h, now)
	ctx := context.Background()

	lRDD, _ := h.ledgers.FindByConcept(ctx, "u-1", "p-rdd", "stereotypes")
	lRDD.StrengthScore = 0.85
	_ = h.ledgers.Save(ctx, lRDD)
	pRDD, _ := h.paths.FindByID(ctx, "u-1", "p-rdd")

	syn := &fakeSynthesis{}
	coach := services.NewMacroCoach(h.paths, h.ledgers, syn, 0.80)

	rec, ready, err := coach.EvaluateProgression(ctx, "u-1", pRDD, now)
	if err != nil || !ready || rec.ProposedPath != "Design Patterns" {
		t.Fatalf("EvaluateProgression failed: ready=%v rec=%+v err=%v", ready, rec, err)
	}

	advice, err := coach.AdviseNextSteps(ctx, "u-1", "What should I study after RDD?")
	if err != nil || advice.ProposedPath != "Design Patterns" {
		t.Fatalf("AdviseNextSteps failed: advice=%+v err=%v", advice, err)
	}
}

func TestInboundRouter_AllHierarchicalBranches(t *testing.T) {
	now := time.Date(2026, 11, 23, 13, 0, 0, 0, time.UTC)
	h := newTestHarness(t, stubTelemetry{bw: touchpoint.BandwidthHigh})
	u := seedUserAndPaths(t, h, now)
	ctx := context.Background()

	aff := &fakeAffinity{}
	syn := &fakeSynthesis{}
	cmd := services.NewCommandController(h.users, h.paths, h.sessions, h.messaging)
	sg := services.NewCrossPathSynthesizer(h.corpus, syn, 3)
	coach := services.NewMacroCoach(h.paths, h.ledgers, syn, 0.80)
	router := services.NewInboundRouter(
		h.users, h.paths, h.corpus, h.sessions, h.messaging,
		aff, syn, cmd, h.evaluator, sg, coach, services.RealClock{},
	)

	// 1. Active drill answer -> PedagogicalEvaluator (PathAffinity NOT called)
	_, _ = h.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeMorningCatalyst, now)
	if err := router.HandleInbound(ctx, touchpoint.InboundMessage{
		SenderPhone: u.PhoneNumber,
		Text:        "Information Holder stores state",
		ReceivedAt:  now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("HandleInbound active drill: %v", err)
	}
	if aff.calls != 0 || h.llm.evalCalls != 1 {
		t.Fatalf("expected active drill to bypass affinity: affCalls=%d evalCalls=%d", aff.calls, h.llm.evalCalls)
	}

	// 2. Single-path spontaneous query -> Grounded Q&A
	aff.result = &touchpoint.AffinityResult{MatchedPaths: []shared.PathID{"p-rdd"}, Intent: touchpoint.IntentPathBound}
	if err := router.HandleInbound(ctx, touchpoint.InboundMessage{
		SenderPhone: u.PhoneNumber,
		Text:        "Remind me what a Coordinator does",
		ReceivedAt:  now.Add(20 * time.Minute),
	}); err != nil {
		t.Fatalf("HandleInbound single-path: %v", err)
	}
	if syn.groundedCalls != 1 || h.messaging.texts[len(h.messaging.texts)-1] != "Grounded(RDD)" {
		t.Fatalf("expected Grounded(RDD), got calls=%d texts=%v", syn.groundedCalls, h.messaging.texts)
	}

	// 3. Cross-path spontaneous query -> Scatter-Gather
	aff.result = &touchpoint.AffinityResult{MatchedPaths: []shared.PathID{"p-rdd", "p-algo"}, Intent: touchpoint.IntentPathBound}
	if err := router.HandleInbound(ctx, touchpoint.InboundMessage{
		SenderPhone: u.PhoneNumber,
		Text:        "How do graphs relate to RDD?",
		ReceivedAt:  now.Add(30 * time.Minute),
	}); err != nil {
		t.Fatalf("HandleInbound cross-path: %v", err)
	}
	if syn.crossPathCalls != 1 || h.messaging.texts[len(h.messaging.texts)-1] != "CrossPathUnified" {
		t.Fatalf("expected CrossPathUnified, got calls=%d texts=%v", syn.crossPathCalls, h.messaging.texts)
	}

	// 4. Unbound curriculum query -> MacroCoach
	aff.result = &touchpoint.AffinityResult{MatchedPaths: nil, Intent: touchpoint.IntentCurriculum}
	if err := router.HandleInbound(ctx, touchpoint.InboundMessage{
		SenderPhone: u.PhoneNumber,
		Text:        "What should I learn next?",
		ReceivedAt:  now.Add(40 * time.Minute),
	}); err != nil {
		t.Fatalf("HandleInbound curriculum: %v", err)
	}
	if syn.milestoneCalls != 1 || !strings.Contains(h.messaging.texts[len(h.messaging.texts)-1], "Design Patterns") {
		t.Fatalf("expected MacroCoach proposal, got calls=%d texts=%v", syn.milestoneCalls, h.messaging.texts)
	}
}
