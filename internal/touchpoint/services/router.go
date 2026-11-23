package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	"github.com/alejogs4/whatsapp-assistant/internal/user"
)

var _ touchpoint.InboundMessageHandler = (*InboundRouter)(nil)

// InboundRouter implements the 4-stage Hierarchical Classifier Pipeline:
// Stage 0: Tenant lookup by phone
// Stage 1: Operational Command escape (/pause, /resume, /skip, /stats, /schedule)
// Stage 2: Active Drill State Guard -> PedagogicalEvaluator (UC-2)
// Stage 3: Path Affinity Classifier -> Single-Path Grounded Q&A | Scatter-Gather | MacroCoach | General
type InboundRouter struct {
	users         user.UserRepository
	paths         learning.LearningPathRepository
	corpus        learning.SourceCorpusPort
	sessions      touchpoint.TouchpointRepository
	messaging     touchpoint.MessagingPort
	affinity      touchpoint.PathAffinityPort
	synthesis     touchpoint.SynthesisPort
	commands      *CommandController
	evaluator     *PedagogicalEvaluator
	scatterGather *CrossPathSynthesizer
	macroCoach    *MacroCoach
	clock         Clock
}

func NewInboundRouter(
	users user.UserRepository,
	paths learning.LearningPathRepository,
	corpus learning.SourceCorpusPort,
	sessions touchpoint.TouchpointRepository,
	messaging touchpoint.MessagingPort,
	affinity touchpoint.PathAffinityPort,
	synthesis touchpoint.SynthesisPort,
	commands *CommandController,
	evaluator *PedagogicalEvaluator,
	scatterGather *CrossPathSynthesizer,
	macroCoach *MacroCoach,
	clock Clock,
) *InboundRouter {
	if clock == nil {
		clock = RealClock{}
	}
	return &InboundRouter{
		users:         users,
		paths:         paths,
		corpus:        corpus,
		sessions:      sessions,
		messaging:     messaging,
		affinity:      affinity,
		synthesis:     synthesis,
		commands:      commands,
		evaluator:     evaluator,
		scatterGather: scatterGather,
		macroCoach:    macroCoach,
		clock:         clock,
	}
}

func (r *InboundRouter) HandleInbound(ctx context.Context, msg touchpoint.InboundMessage) error {
	now := msg.ReceivedAt
	if now.IsZero() {
		now = r.clock.Now()
	}

	// Stage 0: Strict tenant resolution by normalized phone number.
	u, err := r.users.FindByPhoneNumber(ctx, msg.SenderPhone)
	if errors.Is(err, shared.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("router: lookup user: %w", err)
	}

	// Stage 1: Operational commands always escape before drill grading or LLM calls.
	if handled, err := r.commands.Handle(ctx, u, msg.Text); handled || err != nil {
		return err
	}

	// Stage 2: Drill State Guard — active AWAITING_ANSWER session routes strictly to PedagogicalEvaluator.
	activeSession, err := r.sessions.FindActiveSession(ctx, u.ID)
	if err == nil && activeSession != nil {
		if activeSession.CheckExpiration(now) {
			_ = r.sessions.Save(ctx, activeSession)
		} else {
			_, evalErr := r.evaluator.EvaluateActiveSession(ctx, u, activeSession, msg.Text, now)
			return evalErr
		}
	} else if err != nil && !errors.Is(err, shared.ErrTouchpointSessionNotFound) {
		return fmt.Errorf("router: find active session: %w", err)
	}

	// Stage 3: Path Affinity Classifier across the user's active LearningPaths.
	allPaths, err := r.paths.FindByUser(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("router: find user paths: %w", err)
	}

	activeByID := make(map[shared.PathID]*learningdomain.LearningPath)
	summaries := make([]touchpoint.PathSummary, 0, len(allPaths))
	for _, p := range allPaths {
		if p.Status == learningdomain.PathStatusActive {
			activeByID[p.ID] = p
			summaries = append(summaries, touchpoint.PathSummary{ID: p.ID, Title: p.Title})
		}
	}

	aff, err := r.affinity.ClassifyPathAffinity(ctx, msg.Text, summaries)
	if err != nil {
		return fmt.Errorf("router: classify affinity: %w", err)
	}

	matched := make([]*learningdomain.LearningPath, 0, len(aff.MatchedPaths))
	for _, pid := range aff.MatchedPaths {
		if p, ok := activeByID[pid]; ok {
			matched = append(matched, p)
		}
	}

	switch {
	case len(matched) == 1:
		p := matched[0]
		chunks, err := r.corpus.RetrieveChunks(ctx, u.ID, p.ID, msg.Text, DefaultCorpusLimit)
		if err != nil {
			return fmt.Errorf("router: retrieve single-path chunks: %w", err)
		}
		texts := make([]string, 0, len(chunks))
		for _, ch := range chunks {
			texts = append(texts, ch.Content)
		}
		ans, err := r.synthesis.AnswerGrounded(ctx, msg.Text, touchpoint.PathCorpusContext{
			PathID:    p.ID,
			PathTitle: p.Title,
			Chunks:    texts,
		})
		if err != nil {
			return fmt.Errorf("router: grounded answer: %w", err)
		}
		return r.messaging.SendText(ctx, u.PhoneNumber, ans)

	case len(matched) >= 2:
		ans, err := r.scatterGather.Synthesize(ctx, u.ID, msg.Text, matched)
		if err != nil {
			return fmt.Errorf("router: scatter-gather: %w", err)
		}
		return r.messaging.SendText(ctx, u.PhoneNumber, ans)

	case aff.Intent == touchpoint.IntentCurriculum:
		rec, err := r.macroCoach.AdviseNextSteps(ctx, u.ID, msg.Text)
		if err != nil {
			return fmt.Errorf("router: macro coach: %w", err)
		}
		reply := fmt.Sprintf("Next recommended path: *%s*\n%s", rec.ProposedPath, rec.Reasoning)
		return r.messaging.SendText(ctx, u.PhoneNumber, reply)

	default:
		ans, err := r.synthesis.AnswerGeneral(ctx, msg.Text)
		if err != nil {
			return fmt.Errorf("router: general answer: %w", err)
		}
		return r.messaging.SendText(ctx, u.PhoneNumber, ans)
	}
}
