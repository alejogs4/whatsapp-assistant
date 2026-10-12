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
