package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	"github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

var _ learning.MasteryLedgerRepository = (*MasteryLedgerRepo)(nil)

// MasteryLedgerRepo implements learning.MasteryLedgerRepository with strict user_id scoping.
type MasteryLedgerRepo struct {
	db *DB
}

func NewMasteryLedgerRepo(db *DB) *MasteryLedgerRepo {
	return &MasteryLedgerRepo{db: db}
}

func (r *MasteryLedgerRepo) FindByConcept(
	ctx context.Context,
	userID shared.UserID,
	pathID shared.PathID,
	concept shared.ConceptKey,
) (*domain.MasteryLedger, error) {
	row := r.db.sql.QueryRowContext(ctx, `
		SELECT user_id, path_id, concept_key, strength_score, interval_days,
		       ease_factor, next_review_at, misconceptions_json
		FROM mastery_ledger
		WHERE user_id = ? AND path_id = ? AND concept_key = ?
	`, string(userID), string(pathID), string(concept))
	return scanMasteryLedger(row)
}

func (r *MasteryLedgerRepo) FindDueReviews(
	ctx context.Context,
	userID shared.UserID,
	pathID shared.PathID,
	before time.Time,
) ([]*domain.MasteryLedger, error) {
	rows, err := r.db.sql.QueryContext(ctx, `
		SELECT user_id, path_id, concept_key, strength_score, interval_days,
		       ease_factor, next_review_at, misconceptions_json
		FROM mastery_ledger
		WHERE user_id = ? AND path_id = ? AND next_review_at <= ?
		ORDER BY next_review_at ASC, concept_key ASC
	`, string(userID), string(pathID), before.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("sqlite: find due reviews: %w", err)
	}
	defer rows.Close()

	var ledgers []*domain.MasteryLedger
	for rows.Next() {
		l, err := scanMasteryLedger(rows)
		if err != nil {
			return nil, err
		}
		ledgers = append(ledgers, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate due reviews: %w", err)
	}
	return ledgers, nil
}

func (r *MasteryLedgerRepo) Save(ctx context.Context, ledger *domain.MasteryLedger) error {
	misconceptions := ledger.Misconceptions
	if misconceptions == nil {
		misconceptions = []string{}
	}
	rawJSON, err := json.Marshal(misconceptions)
	if err != nil {
		return fmt.Errorf("sqlite: marshal misconceptions: %w", err)
	}

	_, err = r.db.sql.ExecContext(ctx, `
		INSERT INTO mastery_ledger (
			user_id, path_id, concept_key, strength_score, interval_days,
			ease_factor, next_review_at, misconceptions_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, path_id, concept_key) DO UPDATE SET
			strength_score = excluded.strength_score,
			interval_days = excluded.interval_days,
			ease_factor = excluded.ease_factor,
			next_review_at = excluded.next_review_at,
			misconceptions_json = excluded.misconceptions_json
	`,
		string(ledger.UserID),
		string(ledger.PathID),
		string(ledger.ConceptKey),
		ledger.StrengthScore,
		ledger.IntervalDays,
		ledger.EaseFactor,
		ledger.NextReviewAt.UTC().Format(time.RFC3339Nano),
		string(rawJSON),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save mastery ledger: %w", err)
	}
	return nil
}

func scanMasteryLedger(s scanner) (*domain.MasteryLedger, error) {
	var (
		uid, pid, concept string
		strength, ease    float64
		interval          int
		nextReviewStr     string
		misconceptionsStr string
	)
	if err := s.Scan(&uid, &pid, &concept, &strength, &interval, &ease, &nextReviewStr, &misconceptionsStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrMasteryLedgerNotFound
		}
		return nil, fmt.Errorf("sqlite: scan mastery ledger: %w", err)
	}

	nextReview, err := time.Parse(time.RFC3339Nano, nextReviewStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse next_review_at: %w", err)
	}

	var misconceptions []string
	if err := json.Unmarshal([]byte(misconceptionsStr), &misconceptions); err != nil {
		return nil, fmt.Errorf("sqlite: unmarshal misconceptions: %w", err)
	}
	if misconceptions == nil {
		misconceptions = []string{}
	}

	return &domain.MasteryLedger{
		UserID:         shared.UserID(uid),
		PathID:         shared.PathID(pid),
		ConceptKey:     shared.ConceptKey(concept),
		StrengthScore:  strength,
		IntervalDays:   interval,
		EaseFactor:     ease,
		NextReviewAt:   nextReview,
		Misconceptions: misconceptions,
	}, nil
}
