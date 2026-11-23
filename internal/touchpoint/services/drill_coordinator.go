package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/user"
	userdomain "github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

const (
	DefaultMorningExpiry = 4 * time.Hour
	DefaultEveningExpiry = 2 * time.Hour
	DefaultCorpusLimit   = 3
)

// DrillCoordinator orchestrates UC-1: grounded chunk retrieval, LLM micro-drill generation,
// TouchpointSession creation, daily budget accounting, and outbound WhatsApp delivery.
type DrillCoordinator struct {
	users      user.UserRepository
	corpus     learning.SourceCorpusPort
	sessions   touchpoint.TouchpointRepository
	llm        touchpoint.LLMPort
	messaging  touchpoint.MessagingPort
	arbitrator *AttentionArbitrator
}

func NewDrillCoordinator(
	users user.UserRepository,
	corpus learning.SourceCorpusPort,
	sessions touchpoint.TouchpointRepository,
	llm touchpoint.LLMPort,
	messaging touchpoint.MessagingPort,
	arbitrator *AttentionArbitrator,
) *DrillCoordinator {
	return &DrillCoordinator{
		users:      users,
		corpus:     corpus,
		sessions:   sessions,
		llm:        llm,
		messaging:  messaging,
		arbitrator: arbitrator,
	}
}

func (c *DrillCoordinator) TriggerTouchpoint(
	ctx context.Context,
	u *userdomain.User,
	tpType tpdomain.TouchpointType,
	now time.Time,
) (*tpdomain.TouchpointSession, error) {
	canSend, err := u.CanSendTouchpoint(now)
	if err != nil {
		return nil, err
	}
	if !canSend {
		return nil, shared.ErrTouchpointBudgetExhausted
	}

	selection, err := c.arbitrator.SelectTheOneThing(ctx, u.ID, now)
	if err != nil {
		return nil, err
	}

	chunks, err := c.corpus.RetrieveChunks(ctx, u.ID, selection.Path.ID, string(selection.Concept), DefaultCorpusLimit)
	if err != nil {
		return nil, fmt.Errorf("drill_coordinator: retrieve corpus chunks: %w", err)
	}
	corpusTexts := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		corpusTexts = append(corpusTexts, ch.Content)
	}

	effectiveType := tpType
	if selection.Bandwidth == touchpoint.BandwidthLow {
		effectiveType = tpdomain.TouchpointTypeEveningStreakSaver
	}

	drillResp, err := c.llm.GenerateMicroDrill(ctx, touchpoint.DrillRequest{
		UserID:     u.ID,
		PathID:     selection.Path.ID,
		ConceptKey: selection.Concept,
		Corpus:     corpusTexts,
		Type:       effectiveType,
	})
	if err != nil {
		return nil, fmt.Errorf("drill_coordinator: generate micro-drill: %w", err)
	}

	expiry := DefaultMorningExpiry
	if effectiveType == tpdomain.TouchpointTypeEveningStreakSaver {
		expiry = DefaultEveningExpiry
	}

	session := tpdomain.NewTouchpointSession(
		shared.SessionID(uuid.NewString()),
		u.ID,
		selection.Path.ID,
		selection.Concept,
		tpType,
		now,
	)
	if err := session.DispatchPrompt(drillResp.PromptText, now.Add(expiry)); err != nil {
		return nil, fmt.Errorf("drill_coordinator: dispatch prompt state: %w", err)
	}
	if err := u.RecordTouchpointSent(now); err != nil {
		return nil, err
	}

	if err := c.sessions.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("drill_coordinator: save session: %w", err)
	}
	if err := c.users.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("drill_coordinator: save user budget: %w", err)
	}
	if err := c.messaging.SendDrillPrompt(ctx, u.PhoneNumber, session.PromptText); err != nil {
		return nil, fmt.Errorf("drill_coordinator: send drill prompt: %w", err)
	}

	return session, nil
}
