package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	userdomain "github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

func seedUsers(t *testing.T, db *sqlite.DB, ids ...shared.UserID) {
	t.Helper()
	repo := sqlite.NewUserRepo(db)
	for i, id := range ids {
		u, err := userdomain.NewUser(id, "+1555000000"+string(rune('1'+i)), "UTC", userdomain.CircadianBaseline{
			MorningCatalystHour:   8,
			EveningReflectionHour: 20,
		})
		if err != nil {
			t.Fatalf("NewUser(%s): %v", id, err)
		}
		if err := repo.Save(context.Background(), u); err != nil {
			t.Fatalf("Save user %s: %v", id, err)
		}
	}
}

func TestLearningPathRepo_MultiTenantIsolation(t *testing.T) {
	db := openMigratedTestDB(t)
	seedUsers(t, db, "user-a", "user-b")

	repo := sqlite.NewLearningPathRepo(db)
	ctx := context.Background()

	pathA := learningdomain.NewLearningPath("path-rdd", "user-a", "RDD", "drive-rdd")
	pathB := learningdomain.NewLearningPath("path-algo", "user-b", "Algorithms", "drive-algo")
	if err := repo.Save(ctx, pathA); err != nil {
		t.Fatalf("Save pathA: %v", err)
	}
	if err := repo.Save(ctx, pathB); err != nil {
		t.Fatalf("Save pathB: %v", err)
	}

	pathsA, err := repo.FindByUser(ctx, "user-a")
	if err != nil {
		t.Fatalf("FindByUser(user-a): %v", err)
	}
	if len(pathsA) != 1 || pathsA[0].Title != "RDD" {
		t.Fatalf("expected only RDD for user-a, got %+v", pathsA)
	}

	_, err = repo.FindByID(ctx, "user-a", "path-algo")
	if !errors.Is(err, shared.ErrLearningPathNotFound) {
		t.Fatalf("expected ErrLearningPathNotFound for cross-tenant FindByID, got %v", err)
	}
}

func TestMasteryLedgerRepo_SM2StateMisconceptionsAndDueReviews(t *testing.T) {
	db := openMigratedTestDB(t)
	seedUsers(t, db, "user-a", "user-b")

	pathRepo := sqlite.NewLearningPathRepo(db)
	ledgerRepo := sqlite.NewMasteryLedgerRepo(db)
	ctx := context.Background()

	_ = pathRepo.Save(ctx, learningdomain.NewLearningPath("path-rdd", "user-a", "RDD", "folder"))

	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	past := now.Add(-48 * time.Hour)

	ledger := learningdomain.NewMasteryLedger("user-a", "path-rdd", "stereotypes", past)
	if err := ledger.RecordReview(past, 0.8); err != nil {
		t.Fatalf("RecordReview: %v", err)
	}
	ledger.AddMisconception("confuses controller with coordinator")
	ledger.AddMisconception("forgets structurer role")

	if err := ledgerRepo.Save(ctx, ledger); err != nil {
		t.Fatalf("Save ledger: %v", err)
	}

	dueA, err := ledgerRepo.FindDueReviews(ctx, "user-a", "path-rdd", now)
	if err != nil {
		t.Fatalf("FindDueReviews(user-a): %v", err)
	}
	if len(dueA) != 1 {
		t.Fatalf("expected 1 due review for user-a, got %d", len(dueA))
	}
	got := dueA[0]
	if got.StrengthScore != ledger.StrengthScore || got.EaseFactor != ledger.EaseFactor || got.IntervalDays != ledger.IntervalDays {
		t.Fatalf("SM-2 fields mismatch: got %+v, want %+v", got, ledger)
	}
	if len(got.Misconceptions) != 2 {
		t.Fatalf("expected 2 misconceptions, got %v", got.Misconceptions)
	}

	dueB, err := ledgerRepo.FindDueReviews(ctx, "user-b", "path-rdd", now)
	if err != nil {
		t.Fatalf("FindDueReviews(user-b): %v", err)
	}
	if len(dueB) != 0 {
		t.Fatalf("expected 0 due reviews for user-b, got %d", len(dueB))
	}
}
