package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		id          shared.UserID
		phone       string
		tz          string
		baseline    domain.CircadianBaseline
		expectErr   error
		expectValid bool
	}{
		{
			name:        "valid user with Europe/Madrid",
			id:          shared.UserID("usr_1"),
			phone:       "+34600000000",
			tz:          "Europe/Madrid",
			baseline:    domain.CircadianBaseline{MorningCatalystHour: 8, EveningReflectionHour: 21},
			expectErr:   nil,
			expectValid: true,
		},
		{
			name:        "valid user with America/New_York",
			id:          shared.UserID("usr_2"),
			phone:       "+12025550123",
			tz:          "America/New_York",
			baseline:    domain.CircadianBaseline{MorningCatalystHour: 7, EveningReflectionHour: 20},
			expectErr:   nil,
			expectValid: true,
		},
		{
			name:        "invalid IANA timezone",
			id:          shared.UserID("usr_3"),
			phone:       "+12025550199",
			tz:          "Mars/Phobos",
			baseline:    domain.CircadianBaseline{MorningCatalystHour: 8, EveningReflectionHour: 20},
			expectErr:   shared.ErrInvalidTimezone,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := domain.NewUser(tt.id, tt.phone, tt.tz, tt.baseline)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
				if u != nil {
					t.Fatalf("expected nil user on error, got %+v", u)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u == nil {
				t.Fatal("expected non-nil user")
			}
			if u.ID != tt.id || u.PhoneNumber != tt.phone || u.Timezone != tt.tz {
				t.Errorf("user fields mismatch: %+v", u)
			}
			if u.DailyTouchesUsed != 0 {
				t.Errorf("expected DailyTouchesUsed to be 0, got %d", u.DailyTouchesUsed)
			}
			if u.LastTouchSentAt != nil {
				t.Errorf("expected LastTouchSentAt to be nil, got %v", u.LastTouchSentAt)
			}
		})
	}
}

func TestUser_ProactivityBudget(t *testing.T) {
	locNY, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load America/New_York: %v", err)
	}
	locMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("failed to load Europe/Madrid: %v", err)
	}

	t.Run("quota enforcement within same calendar day", func(t *testing.T) {
		now := time.Date(2026, 10, 12, 10, 0, 0, 0, locMadrid)
		u, err := domain.NewUser(
			shared.UserID("usr_madrid"),
			"+34611111111",
			"Europe/Madrid",
			domain.CircadianBaseline{MorningCatalystHour: 8, EveningReflectionHour: 21},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Initial: 0 touches
		canSend, err := u.CanSendTouchpoint(now)
		if err != nil || !canSend {
			t.Fatalf("expected canSend=true, nil; got %v, %v", canSend, err)
		}

		// First touch
		if err := u.RecordTouchpointSent(now); err != nil {
			t.Fatalf("failed to record touchpoint: %v", err)
		}
		if u.DailyTouchesUsed != 1 {
			t.Fatalf("expected DailyTouchesUsed=1, got %d", u.DailyTouchesUsed)
		}
		if u.LastTouchSentAt == nil || !u.LastTouchSentAt.Equal(now) {
			t.Fatalf("expected LastTouchSentAt=%v, got %v", now, u.LastTouchSentAt)
		}

		// Check before second touch (same day, afternoon)
		afternoon := now.Add(5 * time.Hour)
		canSend, err = u.CanSendTouchpoint(afternoon)
		if err != nil || !canSend {
			t.Fatalf("expected canSend=true, nil; got %v, %v", canSend, err)
		}

		// Second touch
		if err := u.RecordTouchpointSent(afternoon); err != nil {
			t.Fatalf("failed to record second touchpoint: %v", err)
		}
		if u.DailyTouchesUsed != 2 {
			t.Fatalf("expected DailyTouchesUsed=2, got %d", u.DailyTouchesUsed)
		}

		// Attempt third touch on same day: must be blocked
		evening := now.Add(10 * time.Hour)
		canSend, err = u.CanSendTouchpoint(evening)
		if canSend || !errors.Is(err, shared.ErrTouchpointBudgetExhausted) {
			t.Fatalf("expected canSend=false, ErrTouchpointBudgetExhausted; got %v, %v", canSend, err)
		}

		err = u.RecordTouchpointSent(evening)
		if !errors.Is(err, shared.ErrTouchpointBudgetExhausted) {
			t.Fatalf("expected ErrTouchpointBudgetExhausted on RecordTouchpointSent, got %v", err)
		}
		if u.DailyTouchesUsed != 2 {
			t.Fatalf("expected DailyTouchesUsed to remain 2, got %d", u.DailyTouchesUsed)
		}
	})

	t.Run("rollover across calendar day in America/New_York", func(t *testing.T) {
		// Day 1 late night: 23:45 in New York
		day1Night := time.Date(2026, 10, 12, 23, 45, 0, 0, locNY)
		u, err := domain.NewUser(
			shared.UserID("usr_ny"),
			"+12025550111",
			"America/New_York",
			domain.CircadianBaseline{MorningCatalystHour: 8, EveningReflectionHour: 20},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Exhaust Day 1 budget
		if err := u.RecordTouchpointSent(day1Night.Add(-2 * time.Hour)); err != nil {
			t.Fatalf("record 1 failed: %v", err)
		}
		if err := u.RecordTouchpointSent(day1Night); err != nil {
			t.Fatalf("record 2 failed: %v", err)
		}
		if u.DailyTouchesUsed != 2 {
			t.Fatalf("expected 2 touches used, got %d", u.DailyTouchesUsed)
		}

		// Day 2 morning: 00:15 in New York (30 mins later, but different calendar day)
		day2Morning := day1Night.Add(30 * time.Minute)
		canSend, err := u.CanSendTouchpoint(day2Morning)
		if err != nil || !canSend {
			t.Fatalf("expected canSend=true after midnight rollover; got %v, %v", canSend, err)
		}

		// Recording touch on Day 2 resets count to 1 for the new day
		if err := u.RecordTouchpointSent(day2Morning); err != nil {
			t.Fatalf("expected record on new day to succeed; got %v", err)
		}
		if u.DailyTouchesUsed != 1 {
			t.Fatalf("expected DailyTouchesUsed=1 after rollover, got %d", u.DailyTouchesUsed)
		}
		if !u.LastTouchSentAt.Equal(day2Morning) {
			t.Fatalf("expected LastTouchSentAt=%v, got %v", day2Morning, u.LastTouchSentAt)
		}
	})

	t.Run("manual ResetDailyBudget", func(t *testing.T) {
		now := time.Date(2026, 10, 12, 12, 0, 0, 0, locMadrid)
		u, err := domain.NewUser(
			shared.UserID("usr_reset"),
			"+34699999999",
			"Europe/Madrid",
			domain.CircadianBaseline{MorningCatalystHour: 9, EveningReflectionHour: 21},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Exhaust budget
		_ = u.RecordTouchpointSent(now)
		_ = u.RecordTouchpointSent(now)

		canSend, err := u.CanSendTouchpoint(now)
		if canSend || !errors.Is(err, shared.ErrTouchpointBudgetExhausted) {
			t.Fatalf("expected exhausted budget, got %v, %v", canSend, err)
		}

		// Reset budget
		if err := u.ResetDailyBudget(now); err != nil {
			t.Fatalf("unexpected reset error: %v", err)
		}
		if u.DailyTouchesUsed != 0 {
			t.Fatalf("expected DailyTouchesUsed=0 after reset, got %d", u.DailyTouchesUsed)
		}

		canSend, err = u.CanSendTouchpoint(now)
		if err != nil || !canSend {
			t.Fatalf("expected canSend=true after reset, got %v, %v", canSend, err)
		}
	})

	t.Run("invalid timezone during runtime check", func(t *testing.T) {
		now := time.Now()
		u := &domain.User{
			ID:               shared.UserID("usr_bad_tz"),
			PhoneNumber:      "+10000000000",
			Timezone:         "Invalid/NonExistent",
			DailyTouchesUsed: 0,
		}

		canSend, err := u.CanSendTouchpoint(now)
		if canSend || !errors.Is(err, shared.ErrInvalidTimezone) {
			t.Fatalf("expected ErrInvalidTimezone on CanSendTouchpoint, got %v, %v", canSend, err)
		}

		err = u.RecordTouchpointSent(now)
		if !errors.Is(err, shared.ErrInvalidTimezone) {
			t.Fatalf("expected ErrInvalidTimezone on RecordTouchpointSent, got %v", err)
		}
	})
}
