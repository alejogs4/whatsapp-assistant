package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	tpdomain "github.com/alejogs4/whatsapp-assistant/internal/touchpoint/domain"
)

func TestTouchpointRepo_ActiveSessionAndMultiTenantScoping(t *testing.T) {
	db := openMigratedTestDB(t)
	seedUsers(t, db, "user-a", "user-b")

	pathRepo := sqlite.NewLearningPathRepo(db)
	tpRepo := sqlite.NewTouchpointRepo(db)
	ctx := context.Background()

	_ = pathRepo.Save(ctx, learningdomain.NewLearningPath("path-a", "user-a", "RDD", "f1"))
	_ = pathRepo.Save(ctx, learningdomain.NewLearningPath("path-b", "user-b", "AI", "f2"))

	todayStart := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	yesterday := todayStart.Add(-12 * time.Hour)
	morning := todayStart.Add(8 * time.Hour)

	completedYesterday := tpdomain.NewTouchpointSession("s-old", "user-a", "path-a", "c1", tpdomain.TouchpointTypeMorningCatalyst, yesterday)
	_ = completedYesterday.DispatchPrompt("old prompt", yesterday.Add(4*time.Hour))
	_ = completedYesterday.SubmitAnswer(yesterday.Add(1*time.Hour), "ans")
	_ = completedYesterday.Complete(yesterday.Add(1*time.Hour), "good")
	if err := tpRepo.Save(ctx, completedYesterday); err != nil {
		t.Fatalf("Save old session: %v", err)
	}

	activeA := tpdomain.NewTouchpointSession("s-active-a", "user-a", "path-a", "c2", tpdomain.TouchpointTypeMorningCatalyst, morning)
	_ = activeA.DispatchPrompt("active prompt A", morning.Add(4*time.Hour))
	if err := tpRepo.Save(ctx, activeA); err != nil {
		t.Fatalf("Save activeA: %v", err)
	}

	activeB := tpdomain.NewTouchpointSession("s-active-b", "user-b", "path-b", "c3", tpdomain.TouchpointTypeMorningCatalyst, morning)
	_ = activeB.DispatchPrompt("active prompt B", morning.Add(4*time.Hour))
	if err := tpRepo.Save(ctx, activeB); err != nil {
		t.Fatalf("Save activeB: %v", err)
	}

	gotActive, err := tpRepo.FindActiveSession(ctx, "user-a")
	if err != nil {
		t.Fatalf("FindActiveSession(user-a): %v", err)
	}
	if gotActive.ID != "s-active-a" {
		t.Fatalf("expected s-active-a, got %s", gotActive.ID)
	}

	_, err = tpRepo.FindByID(ctx, "user-b", "s-active-a")
	if !errors.Is(err, shared.ErrTouchpointSessionNotFound) {
		t.Fatalf("expected ErrTouchpointSessionNotFound for cross-tenant FindByID, got %v", err)
	}

	todays, err := tpRepo.FindByUserAndDate(ctx, "user-a", todayStart, todayStart.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("FindByUserAndDate: %v", err)
	}
	if len(todays) != 1 || todays[0].ID != "s-active-a" {
		t.Fatalf("expected only today's session s-active-a, got %+v", todays)
	}
}
