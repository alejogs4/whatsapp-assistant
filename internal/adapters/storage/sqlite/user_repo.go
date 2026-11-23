package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/user"
	"github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

var _ user.UserRepository = (*UserRepo)(nil)

// UserRepo implements user.UserRepository backed by SQLite.
type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(ctx context.Context, id shared.UserID) (*domain.User, error) {
	row := r.db.sql.QueryRowContext(ctx, `
		SELECT id, phone_number, timezone, morning_catalyst_hour, evening_reflection_hour,
		       daily_touches_used, last_touch_sent_at
		FROM users
		WHERE id = ?
	`, string(id))
	return scanUser(row)
}

func (r *UserRepo) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	row := r.db.sql.QueryRowContext(ctx, `
		SELECT id, phone_number, timezone, morning_catalyst_hour, evening_reflection_hour,
		       daily_touches_used, last_touch_sent_at
		FROM users
		WHERE phone_number = ?
	`, phoneNumber)
	return scanUser(row)
}

func (r *UserRepo) FindAll(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.db.sql.QueryContext(ctx, `
		SELECT id, phone_number, timezone, morning_catalyst_hour, evening_reflection_hour,
		       daily_touches_used, last_touch_sent_at
		FROM users
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: find all users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate users: %w", err)
	}
	return users, nil
}

func (r *UserRepo) Save(ctx context.Context, u *domain.User) error {
	var lastTouch sql.NullString
	if u.LastTouchSentAt != nil {
		lastTouch = sql.NullString{
			String: u.LastTouchSentAt.UTC().Format(time.RFC3339Nano),
			Valid:  true,
		}
	}

	_, err := r.db.sql.ExecContext(ctx, `
		INSERT INTO users (
			id, phone_number, timezone, morning_catalyst_hour, evening_reflection_hour,
			daily_touches_used, last_touch_sent_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			phone_number = excluded.phone_number,
			timezone = excluded.timezone,
			morning_catalyst_hour = excluded.morning_catalyst_hour,
			evening_reflection_hour = excluded.evening_reflection_hour,
			daily_touches_used = excluded.daily_touches_used,
			last_touch_sent_at = excluded.last_touch_sent_at
	`,
		string(u.ID),
		u.PhoneNumber,
		u.Timezone,
		u.CircadianBaseline.MorningCatalystHour,
		u.CircadianBaseline.EveningReflectionHour,
		u.DailyTouchesUsed,
		lastTouch,
	)
	if err != nil {
		return fmt.Errorf("sqlite: save user %q: %w", u.ID, err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (*domain.User, error) {
	var (
		id, phone, tz string
		morning, eve  int
		touches       int
		lastTouch     sql.NullString
	)
	if err := s.Scan(&id, &phone, &tz, &morning, &eve, &touches, &lastTouch); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrUserNotFound
		}
		return nil, fmt.Errorf("sqlite: scan user: %w", err)
	}

	u := &domain.User{
		ID:          shared.UserID(id),
		PhoneNumber: phone,
		Timezone:    tz,
		CircadianBaseline: domain.CircadianBaseline{
			MorningCatalystHour:   morning,
			EveningReflectionHour: eve,
		},
		DailyTouchesUsed: touches,
	}
	if lastTouch.Valid && lastTouch.String != "" {
		parsed, err := time.Parse(time.RFC3339Nano, lastTouch.String)
		if err != nil {
			return nil, fmt.Errorf("sqlite: parse last_touch_sent_at: %w", err)
		}
		u.LastTouchSentAt = &parsed
	}
	return u, nil
}
