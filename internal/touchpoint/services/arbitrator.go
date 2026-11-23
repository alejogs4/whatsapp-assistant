package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

var ErrNoActiveLearningTarget = errors.New("arbitrator: no active learning path or due concept available")

// Selection represents "The ONE Thing" chosen by the AttentionArbitrator.
type Selection struct {
	Path      *learningdomain.LearningPath
	Concept   shared.ConceptKey
	Bandwidth touchpoint.CognitiveBandwidth
}

// AttentionArbitrator eliminates decision fatigue by picking a single highest-leverage
// concept across all active LearningPaths, adapted to CognitiveBandwidth.
type AttentionArbitrator struct {
	paths     learning.LearningPathRepository
	ledgers   learning.MasteryLedgerRepository
	telemetry touchpoint.CalendarTelemetryPort
}

func NewAttentionArbitrator(
	paths learning.LearningPathRepository,
	ledgers learning.MasteryLedgerRepository,
	telemetry touchpoint.CalendarTelemetryPort,
) *AttentionArbitrator {
	return &AttentionArbitrator{
		paths:     paths,
		ledgers:   ledgers,
		telemetry: telemetry,
	}
}

func (a *AttentionArbitrator) SelectTheOneThing(
	ctx context.Context,
	userID shared.UserID,
	now time.Time,
) (*Selection, error) {
	allPaths, err := a.paths.FindByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("arbitrator: find paths: %w", err)
	}

	bandwidth := a.assessBandwidth(ctx, userID, now)

	type candidate struct {
		path    *learningdomain.LearningPath
		ledger  *learningdomain.MasteryLedger
		overdue time.Duration
	}

	var candidates []candidate
	for _, p := range allPaths {
		if p.Status != learningdomain.PathStatusActive {
			continue
		}
		due, err := a.ledgers.FindDueReviews(ctx, userID, p.ID, now)
		if err != nil {
			return nil, fmt.Errorf("arbitrator: find due reviews for path %q: %w", p.ID, err)
		}
		for _, l := range due {
			candidates = append(candidates, candidate{
				path:    p,
				ledger:  l,
				overdue: now.Sub(l.NextReviewAt),
			})
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoActiveLearningTarget
	}

	sort.Slice(candidates, func(i, j int) bool {
		ci, cj := candidates[i], candidates[j]
		if ci.overdue != cj.overdue {
			return ci.overdue > cj.overdue
		}
		if len(ci.ledger.Misconceptions) != len(cj.ledger.Misconceptions) {
			return len(ci.ledger.Misconceptions) > len(cj.ledger.Misconceptions)
		}
		if ci.ledger.StrengthScore != cj.ledger.StrengthScore {
			return ci.ledger.StrengthScore < cj.ledger.StrengthScore
		}
		if ci.path.ID != cj.path.ID {
			return ci.path.ID < cj.path.ID
		}
		return ci.ledger.ConceptKey < cj.ledger.ConceptKey
	})

	winner := candidates[0]
	return &Selection{
		Path:      winner.path,
		Concept:   winner.ledger.ConceptKey,
		Bandwidth: bandwidth,
	}, nil
}

func (a *AttentionArbitrator) assessBandwidth(
	ctx context.Context,
	userID shared.UserID,
	now time.Time,
) touchpoint.CognitiveBandwidth {
	if a.telemetry != nil {
		if bw, err := a.telemetry.GetCognitiveBandwidth(ctx, userID, now); err == nil && bw != "" {
			return bw
		}
	}
	if now.Hour() < 16 {
		return touchpoint.BandwidthHigh
	}
	return touchpoint.BandwidthLow
}
