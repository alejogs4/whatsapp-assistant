package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

func TestUserRepo_SaveAndReloadByIDAndPhone(t *testing.T) {
	db := openMigratedTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	u, err := domain.NewUser("u-1", "+15551234567", "America/New_York", domain.CircadianBaseline{
		MorningCatalystHour:   8,
		EveningReflectionHour: 20,
	})
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}

	sentAt := time.Date(2026, 11, 23, 13, 0, 0, 123456000, time.UTC)
	if err := u.RecordTouchpointSent(sentAt); err != nil {
		t.Fatalf("RecordTouchpointSent: %v", err)
	}

	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save: %v", err)
	}

	byID, err := repo.FindByID(ctx, "u-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if byID.PhoneNumber != "+15551234567" || byID.Timezone != "America/New_York" || byID.DailyTouchesUsed != 1 {
		t.Fatalf("unexpected user from FindByID: %+v", byID)
	}
	if byID.LastTouchSentAt == nil || !byID.LastTouchSentAt.Equal(sentAt) {
		t.Fatalf("expected LastTouchSentAt=%v, got %v", sentAt, byID.LastTouchSentAt)
	}

	byPhone, err := repo.FindByPhoneNumber(ctx, "+15551234567")
	if err != nil {
		t.Fatalf("FindByPhoneNumber: %v", err)
	}
	if byPhone.ID != "u-1" {
		t.Fatalf("expected ID u-1, got %s", byPhone.ID)
	}

	_, err = repo.FindByID(ctx, "missing")
	if !errors.Is(err, shared.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepo_UpsertAndFindAll(t *testing.T) {
	db := openMigratedTestDB(t)
	repo := sqlite.NewUserRepo(db)
	ctx := context.Background()

	u, _ := domain.NewUser("u-1", "+15550001111", "UTC", domain.CircadianBaseline{
		MorningCatalystHour:   8,
		EveningReflectionHour: 20,
	})
	t1 := time.Date(2026, 11, 23, 8, 0, 0, 0, time.UTC)
	_ = u.RecordTouchpointSent(t1)
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	t2 := time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	if err := u.RecordTouchpointSent(t2); err != nil {
		t.Fatalf("second RecordTouchpointSent: %v", err)
	}
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	reloaded, err := repo.FindByID(ctx, "u-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if reloaded.DailyTouchesUsed != 2 {
		t.Fatalf("expected DailyTouchesUsed=2, got %d", reloaded.DailyTouchesUsed)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 user, got %d", len(all))
	}
}
