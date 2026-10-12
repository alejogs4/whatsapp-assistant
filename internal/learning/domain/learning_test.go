package domain_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

func TestLearningPath(t *testing.T) {
	pathID := shared.PathID("path_clean_arch")
	userID := shared.UserID("usr_123")
	title := "Clean Architecture & DDD"
	folderID := "gdrive_folder_abc"

	p := domain.NewLearningPath(pathID, userID, title, folderID)
	if p == nil {
		t.Fatal("expected non-nil LearningPath")
	}

	if p.ID != pathID || p.UserID != userID || p.Title != title || p.SourceFolderID != folderID {
		t.Errorf("field mismatch on NewLearningPath: %+v", p)
	}

	if p.Status != domain.PathStatusActive {
		t.Errorf("expected initial status %v, got %v", domain.PathStatusActive, p.Status)
	}

	// Verify status mutations
	p.Status = domain.PathStatusPaused
	if p.Status != domain.PathStatusPaused {
		t.Errorf("expected status %v, got %v", domain.PathStatusPaused, p.Status)
	}

	p.Status = domain.PathStatusCompleted
	if p.Status != domain.PathStatusCompleted {
		t.Errorf("expected status %v, got %v", domain.PathStatusCompleted, p.Status)
	}
}

func TestMasteryLedger_RecordReview(t *testing.T) {
	now := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	userID := shared.UserID("usr_456")
	pathID := shared.PathID("path_hexagonal")
	concept := shared.ConceptKey("aggregate_root")

	t.Run("score bounds validation", func(t *testing.T) {
		tests := []struct {
			name      string
			score     float64
			expectErr error
		}{
			{name: "negative score", score: -0.01, expectErr: shared.ErrInvalidMasteryScore},
			{name: "negative large score", score: -1.0, expectErr: shared.ErrInvalidMasteryScore},
			{name: "greater than 1.0", score: 1.01, expectErr: shared.ErrInvalidMasteryScore},
			{name: "much greater than 1.0", score: 5.0, expectErr: shared.ErrInvalidMasteryScore},
			{name: "valid boundary 0.0", score: 0.0, expectErr: nil},
			{name: "valid boundary 1.0", score: 1.0, expectErr: nil},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ledger := domain.NewMasteryLedger(userID, pathID, concept, now)
				err := ledger.RecordReview(now, tt.score)
				if tt.expectErr != nil {
					if !errors.Is(err, tt.expectErr) {
						t.Fatalf("expected error %v, got %v", tt.expectErr, err)
					}
					// State must remain intact
					if ledger.IntervalDays != 0 || ledger.EaseFactor != 2.5 || ledger.StrengthScore != 0.0 {
						t.Errorf("state corrupted on invalid score: %+v", ledger)
					}
				} else {
					if err != nil {
						t.Fatalf("unexpected error for valid score: %v", err)
					}
				}
			})
		}
	})

	t.Run("SM-2 active-recall interval progression on passing scores", func(t *testing.T) {
		ledger := domain.NewMasteryLedger(userID, pathID, concept, now)

		// 1st review: passing score (0.9)
		// IntervalDays was 0, should become 1
		day1 := now
		if err := ledger.RecordReview(day1, 0.9); err != nil {
			t.Fatalf("1st review error: %v", err)
		}
		if ledger.IntervalDays != 1 {
			t.Fatalf("expected IntervalDays=1 on 1st review, got %d", ledger.IntervalDays)
		}
		expectedNext := day1.AddDate(0, 0, 1)
		if !ledger.NextReviewAt.Equal(expectedNext) {
			t.Fatalf("expected NextReviewAt=%v, got %v", expectedNext, ledger.NextReviewAt)
		}
		if ledger.StrengthScore <= 0.0 {
			t.Fatalf("expected StrengthScore > 0, got %f", ledger.StrengthScore)
		}

		// 2nd review: passing score (0.95)
		// IntervalDays was 1, should become >= 2
		day2 := day1.AddDate(0, 0, 1)
		if err := ledger.RecordReview(day2, 0.95); err != nil {
			t.Fatalf("2nd review error: %v", err)
		}
		if ledger.IntervalDays < 2 {
			t.Fatalf("expected IntervalDays >= 2 on 2nd review, got %d", ledger.IntervalDays)
		}

		// 3rd review: passing score (1.0)
		// IntervalDays should scale by EaseFactor
		prevInterval := ledger.IntervalDays
		day3 := day2.AddDate(0, 0, prevInterval)
		if err := ledger.RecordReview(day3, 1.0); err != nil {
			t.Fatalf("3rd review error: %v", err)
		}
		if ledger.IntervalDays <= prevInterval {
			t.Fatalf("expected IntervalDays > %d, got %d", prevInterval, ledger.IntervalDays)
		}
		if ledger.StrengthScore > 1.0 {
			t.Fatalf("StrengthScore must not exceed 1.0, got %f", ledger.StrengthScore)
		}
	})

	t.Run("failing review resets interval and decreases ease factor", func(t *testing.T) {
		ledger := domain.NewMasteryLedger(userID, pathID, concept, now)
		ledger.IntervalDays = 14
		ledger.EaseFactor = 2.4
		ledger.StrengthScore = 0.8

		// Failing review with score 0.3
		failTime := now.AddDate(0, 0, 14)
		if err := ledger.RecordReview(failTime, 0.3); err != nil {
			t.Fatalf("record review failed: %v", err)
		}

		// Invariant: interval resets to 1
		if ledger.IntervalDays != 1 {
			t.Fatalf("expected interval reset to 1, got %d", ledger.IntervalDays)
		}
		expectedNext := failTime.AddDate(0, 0, 1)
		if !ledger.NextReviewAt.Equal(expectedNext) {
			t.Fatalf("expected NextReviewAt=%v, got %v", expectedNext, ledger.NextReviewAt)
		}

		// Ease factor decreased (2.4 - 0.2 = 2.2)
		if math.Abs(ledger.EaseFactor-2.2) > 1e-9 {
			t.Fatalf("expected EaseFactor approx 2.2, got %f", ledger.EaseFactor)
		}

		// Strength score halved (0.8 * 0.5 = 0.4)
		if ledger.StrengthScore != 0.4 {
			t.Fatalf("expected StrengthScore=0.4, got %f", ledger.StrengthScore)
		}
	})

	t.Run("ease factor lower bound at 1.3", func(t *testing.T) {
		ledger := domain.NewMasteryLedger(userID, pathID, concept, now)
		ledger.EaseFactor = 1.35

		// Repeated failures
		_ = ledger.RecordReview(now, 0.1)
		if ledger.EaseFactor < 1.3 {
			t.Fatalf("EaseFactor fell below 1.3: %f", ledger.EaseFactor)
		}
		_ = ledger.RecordReview(now, 0.1)
		if ledger.EaseFactor < 1.3 {
			t.Fatalf("EaseFactor fell below 1.3: %f", ledger.EaseFactor)
		}
		if ledger.EaseFactor != 1.3 {
			t.Fatalf("expected EaseFactor pinned at 1.3, got %f", ledger.EaseFactor)
		}
	})
}

func TestMasteryLedger_Misconceptions(t *testing.T) {
	now := time.Now()
	ledger := domain.NewMasteryLedger(
		shared.UserID("usr_1"),
		shared.PathID("path_1"),
		shared.ConceptKey("rdd_stereotypes"),
		now,
	)

	// Initially empty
	if len(ledger.Misconceptions) != 0 {
		t.Fatalf("expected empty misconceptions, got %v", ledger.Misconceptions)
	}

	// Add unique misconception
	m1 := "Confused Information Holder with Service"
	ledger.AddMisconception(m1)
	if len(ledger.Misconceptions) != 1 || ledger.Misconceptions[0] != m1 {
		t.Fatalf("expected [%s], got %v", m1, ledger.Misconceptions)
	}

	// Add duplicate misconception -> should be deduplicated
	ledger.AddMisconception(m1)
	if len(ledger.Misconceptions) != 1 {
		t.Fatalf("duplicate misconception was added: %v", ledger.Misconceptions)
	}

	// Add second misconception
	m2 := "Assumed Entities are anemic"
	ledger.AddMisconception(m2)
	if len(ledger.Misconceptions) != 2 {
		t.Fatalf("expected 2 misconceptions, got %v", ledger.Misconceptions)
	}

	// Clear first misconception
	ledger.ClearMisconception(m1)
	if len(ledger.Misconceptions) != 1 || ledger.Misconceptions[0] != m2 {
		t.Fatalf("expected only [%s], got %v", m2, ledger.Misconceptions)
	}

	// Clear non-existent misconception (no-op)
	ledger.ClearMisconception("Non-existent misconception")
	if len(ledger.Misconceptions) != 1 || ledger.Misconceptions[0] != m2 {
		t.Fatalf("clearing non-existent affected list: %v", ledger.Misconceptions)
	}

	// Clear remaining misconception
	ledger.ClearMisconception(m2)
	if len(ledger.Misconceptions) != 0 {
		t.Fatalf("expected empty list after clearing all, got %v", ledger.Misconceptions)
	}
}
