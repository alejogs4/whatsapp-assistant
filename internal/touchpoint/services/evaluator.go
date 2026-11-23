package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
	userdomain "github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

// PedagogicalEvaluator orchestrates UC-2: evaluates an active drill answer against grounded corpus,
// updates MasteryLedger SM-2 state, completes TouchpointSession, and sends feedback.
type PedagogicalEvaluator struct {
	corpus    learning.SourceCorpusPort
	ledgers   learning.MasteryLedgerRepository
	sessions  touchpoint.TouchpointRepository
	llm       touchpoint.LLMPort
	messaging touchpoint.MessagingPort
}

func NewPedagogicalEvaluator(
	corpus learning.SourceCorpusPort,
	ledgers learning.MasteryLedgerRepository,
	sessions touchpoint.TouchpointRepository,
	llm touchpoint.LLMPort,
	messaging touchpoint.MessagingPort,
) *PedagogicalEvaluator {
	return &PedagogicalEvaluator{
		corpus:    corpus,
		ledgers:   ledgers,
		sessions:  sessions,
		llm:       llm,
		messaging: messaging,
	}
}

func (e *PedagogicalEvaluator) EvaluateActiveSession(
	ctx context.Context,
	u *userdomain.User,
	session *tpdomain.TouchpointSession,
	answer string,
	now time.Time,
) (*touchpoint.EvaluateResponse, error) {
	if err := session.SubmitAnswer(now, answer); err != nil {
		if errors.Is(err, shared.ErrSessionExpired) {
			_ = e.sessions.Save(ctx, session)
		}
		return nil, err
	}

	chunks, err := e.corpus.RetrieveChunks(ctx, u.ID, session.PathID, string(session.ConceptKey), DefaultCorpusLimit)
	if err != nil {
		return nil, fmt.Errorf("evaluator: retrieve corpus chunks: %w", err)
	}
	corpusTexts := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		corpusTexts = append(corpusTexts, ch.Content)
	}

	evalResp, err := e.llm.EvaluateAnswer(ctx, touchpoint.EvaluateRequest{
		Prompt:     session.PromptText,
		UserAnswer: answer,
		Corpus:     corpusTexts,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluator: llm evaluate answer: %w", err)
	}

	ledger, err := e.ledgers.FindByConcept(ctx, u.ID, session.PathID, session.ConceptKey)
	if errors.Is(err, shared.ErrMasteryLedgerNotFound) {
		ledger = learningdomain.NewMasteryLedger(u.ID, session.PathID, session.ConceptKey, now)
	} else if err != nil {
		return nil, fmt.Errorf("evaluator: find mastery ledger: %w", err)
	}

	if err := ledger.RecordReview(now, evalResp.Score); err != nil {
		return nil, fmt.Errorf("evaluator: record review: %w", err)
	}
	if err := e.ledgers.Save(ctx, ledger); err != nil {
		return nil, fmt.Errorf("evaluator: save mastery ledger: %w", err)
	}

	if err := session.Complete(now, evalResp.Feedback); err != nil {
		return nil, fmt.Errorf("evaluator: complete session: %w", err)
	}
	if err := e.sessions.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("evaluator: save completed session: %w", err)
	}

	if err := e.messaging.SendText(ctx, u.PhoneNumber, evalResp.Feedback); err != nil {
		return nil, fmt.Errorf("evaluator: send feedback: %w", err)
	}

	return evalResp, nil
}
