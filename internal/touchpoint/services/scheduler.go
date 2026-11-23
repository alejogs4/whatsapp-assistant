package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/touchpoint"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/user"
	userdomain "github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

// Scheduler runs the proactive 2-touch circadian loop with silent zero-nag expiration backoff.
type Scheduler struct {
	users       user.UserRepository
	sessions    touchpoint.TouchpointRepository
	coordinator *DrillCoordinator
	messaging   touchpoint.MessagingPort
	clock       Clock
	interval    time.Duration
	logger      *slog.Logger
}

func NewScheduler(
	users user.UserRepository,
	sessions touchpoint.TouchpointRepository,
	coordinator *DrillCoordinator,
	messaging touchpoint.MessagingPort,
	clock Clock,
	interval time.Duration,
	logger *slog.Logger,
) *Scheduler {
	if clock == nil {
		clock = RealClock{}
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{
		users:       users,
		sessions:    sessions,
		coordinator: coordinator,
		messaging:   messaging,
		clock:       clock,
		interval:    interval,
		logger:      logger,
	}
}

// Start runs RunTick periodically until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.RunTick(ctx, s.clock.Now()); err != nil {
				s.logger.Error("scheduler: tick failed", "err", err)
			}
		}
	}
}

// RunTick evaluates every user once at wall-clock time now.
func (s *Scheduler) RunTick(ctx context.Context, now time.Time) error {
	allUsers, err := s.users.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("scheduler: find all users: %w", err)
	}

	for _, u := range allUsers {
		if err := s.processUser(ctx, u, now); err != nil {
			s.logger.Error("scheduler: process user failed", "user_id", u.ID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) processUser(ctx context.Context, u *userdomain.User, now time.Time) error {
	// Step 1: Silent zero-nag expiration check on any active session.
	active, err := s.sessions.FindActiveSession(ctx, u.ID)
	if err == nil && active != nil {
		if active.CheckExpiration(now) {
			if saveErr := s.sessions.Save(ctx, active); saveErr != nil {
				return fmt.Errorf("scheduler: persist expired session: %w", saveErr)
			}
		} else {
			// Active unanswered session still open; never nag or overlap.
			return nil
		}
	} else if err != nil && !errors.Is(err, shared.ErrTouchpointSessionNotFound) {
		return fmt.Errorf("scheduler: find active session: %w", err)
	}

	// Step 2: Check local timezone circadian window.
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return shared.ErrInvalidTimezone
	}
	localNow := now.In(loc)
	dayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)

	todaysSessions, err := s.sessions.FindByUserAndDate(ctx, u.ID, dayStart.UTC(), dayEnd.UTC())
	if err != nil {
		return fmt.Errorf("scheduler: find today's sessions: %w", err)
	}

	hasMorning := false
	morningCompleted := false
	hasEvening := false
	for _, sess := range todaysSessions {
		switch sess.TouchpointType {
		case tpdomain.TouchpointTypeMorningCatalyst:
			hasMorning = true
			if sess.State == tpdomain.SessionStateCompleted {
				morningCompleted = true
			}
		case tpdomain.TouchpointTypeEveningStreakSaver:
			hasEvening = true
		}
	}

	// Morning Catalyst window (idempotent within local day)
	if localNow.Hour() == u.CircadianBaseline.MorningCatalystHour && !hasMorning {
		_, err := s.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeMorningCatalyst, now)
		if errors.Is(err, shared.ErrTouchpointBudgetExhausted) || errors.Is(err, ErrNoActiveLearningTarget) {
			return nil
		}
		return err
	}

	// Evening Reflection / Streak Saver window (idempotent within local day)
	if localNow.Hour() == u.CircadianBaseline.EveningReflectionHour && !hasEvening {
		canSend, err := u.CanSendTouchpoint(now)
		if err != nil || !canSend {
			return nil
		}

		if morningCompleted {
			if err := u.RecordTouchpointSent(now); err != nil {
				return nil
			}
			if err := s.users.Save(ctx, u); err != nil {
				return err
			}
			return s.messaging.SendText(ctx, u.PhoneNumber, "Nice work completing your morning drill today! Rest up — momentum is intact.")
		}

		_, err = s.coordinator.TriggerTouchpoint(ctx, u, tpdomain.TouchpointTypeEveningStreakSaver, now)
		if errors.Is(err, shared.ErrTouchpointBudgetExhausted) || errors.Is(err, ErrNoActiveLearningTarget) {
			return nil
		}
		return err
	}

	return nil
}
