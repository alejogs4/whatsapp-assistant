package services

import (
	"context"
	"fmt"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
)

const DefaultMasteryThreshold = 0.80

// MacroCoach implements UC-4: detects when a LearningPath crosses target mastery
// and advises on prerequisite-grounded roadmap progression.
type MacroCoach struct {
	paths     learning.LearningPathRepository
	ledgers   learning.MasteryLedgerRepository
	synthesis touchpoint.SynthesisPort
	threshold float64
}

func NewMacroCoach(
	paths learning.LearningPathRepository,
	ledgers learning.MasteryLedgerRepository,
	synthesis touchpoint.SynthesisPort,
	threshold float64,
) *MacroCoach {
	if threshold <= 0 || threshold > 1.0 {
		threshold = DefaultMasteryThreshold
	}
	return &MacroCoach{
		paths:     paths,
		ledgers:   ledgers,
		synthesis: synthesis,
		threshold: threshold,
	}
}

// EvaluateProgression checks whether a specific path has reached the mastery threshold
// across its ledger concepts and returns a next-milestone proposal if ready.
func (m *MacroCoach) EvaluateProgression(
	ctx context.Context,
	userID shared.UserID,
	path *learningdomain.LearningPath,
	now time.Time,
) (*touchpoint.MilestoneRecommendation, bool, error) {
	// Query all ledger entries for the path using a far-future cutoff.
	entries, err := m.ledgers.FindDueReviews(ctx, userID, path.ID, now.AddDate(100, 0, 0))
	if err != nil {
		return nil, false, fmt.Errorf("macro_coach: load ledger entries: %w", err)
	}
	if len(entries) == 0 {
		return nil, false, nil
	}

	var total float64
	for _, e := range entries {
		total += e.StrengthScore
	}
	avg := total / float64(len(entries))
	if avg < m.threshold {
		return nil, false, nil
	}

	allPaths, err := m.paths.FindByUser(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("macro_coach: find user paths: %w", err)
	}
	var activeNames []string
	for _, p := range allPaths {
		if p.Status == learningdomain.PathStatusActive && p.ID != path.ID {
			activeNames = append(activeNames, p.Title)
		}
	}

	rec, err := m.synthesis.SynthesizeMilestone(ctx, touchpoint.MilestoneRequest{
		UserID:            userID,
		CompletedPathName: path.Title,
		ActivePathNames:   activeNames,
	})
	if err != nil {
		return nil, false, fmt.Errorf("macro_coach: synthesize milestone: %w", err)
	}
	return rec, true, nil
}

// AdviseNextSteps answers spontaneous curriculum-steering questions grounded in the user's paths.
func (m *MacroCoach) AdviseNextSteps(
	ctx context.Context,
	userID shared.UserID,
	question string,
) (*touchpoint.MilestoneRecommendation, error) {
	allPaths, err := m.paths.FindByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("macro_coach: find user paths: %w", err)
	}

	var completedName string
	var activeNames []string
	for _, p := range allPaths {
		switch p.Status {
		case learningdomain.PathStatusCompleted:
			completedName = p.Title
		case learningdomain.PathStatusActive:
			activeNames = append(activeNames, p.Title)
		}
	}

	return m.synthesis.SynthesizeMilestone(ctx, touchpoint.MilestoneRequest{
		UserID:            userID,
		CompletedPathName: completedName,
		ActivePathNames:   activeNames,
		UserQuestion:      question,
	})
}
