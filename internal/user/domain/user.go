package domain

import (
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type CircadianBaseline struct {
	MorningCatalystHour   int
	EveningReflectionHour int
}

type User struct {
	ID                shared.UserID
	PhoneNumber       string
	Timezone          string // IANA identifier, e.g. "America/New_York"
	CircadianBaseline CircadianBaseline
	DailyTouchesUsed  int
	LastTouchSentAt   *time.Time
}

func NewUser(id shared.UserID, phone string, tz string, baseline CircadianBaseline) (*User, error) {
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, shared.ErrInvalidTimezone
	}
	return &User{
		ID:                id,
		PhoneNumber:       phone,
		Timezone:          tz,
		CircadianBaseline: baseline,
		DailyTouchesUsed:  0,
		LastTouchSentAt:   nil,
	}, nil
}

func (u *User) isSameCalendarDay(t1, t2 time.Time, loc *time.Location) bool {
	in1 := t1.In(loc)
	in2 := t2.In(loc)
	return in1.Year() == in2.Year() && in1.YearDay() == in2.YearDay()
}

func (u *User) CanSendTouchpoint(now time.Time) (bool, error) {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return false, shared.ErrInvalidTimezone
	}

	if u.LastTouchSentAt == nil || !u.isSameCalendarDay(*u.LastTouchSentAt, now, loc) {
		return true, nil
	}

	if u.DailyTouchesUsed >= 2 {
		return false, shared.ErrTouchpointBudgetExhausted
	}

	return true, nil
}

func (u *User) RecordTouchpointSent(now time.Time) error {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return shared.ErrInvalidTimezone
	}

	if u.LastTouchSentAt != nil && !u.isSameCalendarDay(*u.LastTouchSentAt, now, loc) {
		u.DailyTouchesUsed = 0
	}

	if u.DailyTouchesUsed >= 2 {
		return shared.ErrTouchpointBudgetExhausted
	}

	u.DailyTouchesUsed++
	u.LastTouchSentAt = &now
	return nil
}

func (u *User) ResetDailyBudget(now time.Time) error {
	u.DailyTouchesUsed = 0
	return nil
}
