package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

var _ touchpoint.TouchpointRepository = (*TouchpointRepo)(nil)

// TouchpointRepo implements touchpoint.TouchpointRepository with strict user_id scoping.
type TouchpointRepo struct {
	db *DB
}

func NewTouchpointRepo(db *DB) *TouchpointRepo {
	return &TouchpointRepo{db: db}
}

func (r *TouchpointRepo) FindByID(
	ctx context.Context,
	userID shared.UserID,
	id shared.SessionID,
) (*domain.TouchpointSession, error) {
	row := r.db.sql.QueryRowContext(ctx, `
		SELECT id, user_id, path_id, concept_key, state, touchpoint_type,
		       prompt_text, user_answer, evaluation_feedback,
		       created_at, expires_at, completed_at
		FROM touchpoint_sessions
		WHERE user_id = ? AND id = ?
	`, string(userID), string(id))
	return scanTouchpointSession(row)
}

func (r *TouchpointRepo) FindActiveSession(
	ctx context.Context,
	userID shared.UserID,
) (*domain.TouchpointSession, error) {
	row := r.db.sql.QueryRowContext(ctx, `
		SELECT id, user_id, path_id, concept_key, state, touchpoint_type,
		       prompt_text, user_answer, evaluation_feedback,
		       created_at, expires_at, completed_at
		FROM touchpoint_sessions
		WHERE user_id = ? AND state = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, string(userID), string(domain.SessionStateAwaitingAnswer))
	return scanTouchpointSession(row)
}

func (r *TouchpointRepo) FindByUserAndDate(
	ctx context.Context,
	userID shared.UserID,
	dayStart, dayEnd time.Time,
) ([]*domain.TouchpointSession, error) {
	rows, err := r.db.sql.QueryContext(ctx, `
		SELECT id, user_id, path_id, concept_key, state, touchpoint_type,
		       prompt_text, user_answer, evaluation_feedback,
		       created_at, expires_at, completed_at
		FROM touchpoint_sessions
		WHERE user_id = ? AND created_at >= ? AND created_at < ?
		ORDER BY created_at ASC
	`, string(userID), dayStart.UTC().Format(time.RFC3339Nano), dayEnd.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("sqlite: find sessions by date: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.TouchpointSession
	for rows.Next() {
		s, err := scanTouchpointSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate sessions: %w", err)
	}
	return sessions, nil
}

func (r *TouchpointRepo) Save(ctx context.Context, s *domain.TouchpointSession) error {
	var completedAt sql.NullString
	if s.CompletedAt != nil {
		completedAt = sql.NullString{
			String: s.CompletedAt.UTC().Format(time.RFC3339Nano),
			Valid:  true,
		}
	}

	_, err := r.db.sql.ExecContext(ctx, `
		INSERT INTO touchpoint_sessions (
			id, user_id, path_id, concept_key, state, touchpoint_type,
			prompt_text, user_answer, evaluation_feedback,
			created_at, expires_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			state = excluded.state,
			prompt_text = excluded.prompt_text,
			user_answer = excluded.user_answer,
			evaluation_feedback = excluded.evaluation_feedback,
			expires_at = excluded.expires_at,
			completed_at = excluded.completed_at
		WHERE touchpoint_sessions.user_id = excluded.user_id
	`,
		string(s.ID),
		string(s.UserID),
		string(s.PathID),
		string(s.ConceptKey),
		string(s.State),
		string(s.TouchpointType),
		s.PromptText,
		s.UserAnswer,
		s.EvaluationFeedback,
		s.CreatedAt.UTC().Format(time.RFC3339Nano),
		s.ExpiresAt.UTC().Format(time.RFC3339Nano),
		completedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: save touchpoint session %q: %w", s.ID, err)
	}
	return nil
}

func scanTouchpointSession(s scanner) (*domain.TouchpointSession, error) {
	var (
		id, uid, pid, concept, state, tpType string
		prompt, answer, feedback             string
		createdStr, expiresStr               string
		completedStr                         sql.NullString
	)
	if err := s.Scan(
		&id, &uid, &pid, &concept, &state, &tpType,
		&prompt, &answer, &feedback,
		&createdStr, &expiresStr, &completedStr,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrTouchpointSessionNotFound
		}
		return nil, fmt.Errorf("sqlite: scan touchpoint session: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse created_at: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse expires_at: %w", err)
	}

	sess := &domain.TouchpointSession{
		ID:                 shared.SessionID(id),
		UserID:             shared.UserID(uid),
		PathID:             shared.PathID(pid),
		ConceptKey:         shared.ConceptKey(concept),
		State:              domain.SessionState(state),
		TouchpointType:     domain.TouchpointType(tpType),
		PromptText:         prompt,
		UserAnswer:         answer,
		EvaluationFeedback: feedback,
		CreatedAt:          createdAt,
		ExpiresAt:          expiresAt,
	}
	if completedStr.Valid && completedStr.String != "" {
		parsed, err := time.Parse(time.RFC3339Nano, completedStr.String)
		if err != nil {
			return nil, fmt.Errorf("sqlite: parse completed_at: %w", err)
		}
		sess.CompletedAt = &parsed
	}
	return sess, nil
}
