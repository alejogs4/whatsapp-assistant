package learning

import (
	"context"
	"time"

	"github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

type GroundedChunk struct {
	ID         string
	PathID     shared.PathID
	Content    string
	Similarity float32
}

type LearningPathRepository interface {
	FindByID(ctx context.Context, userID shared.UserID, id shared.PathID) (*domain.LearningPath, error)
	FindByUser(ctx context.Context, userID shared.UserID) ([]*domain.LearningPath, error)
	Save(ctx context.Context, path *domain.LearningPath) error
}

type MasteryLedgerRepository interface {
	FindByConcept(ctx context.Context, userID shared.UserID, pathID shared.PathID, concept shared.ConceptKey) (*domain.MasteryLedger, error)
	FindDueReviews(ctx context.Context, userID shared.UserID, pathID shared.PathID, before time.Time) ([]*domain.MasteryLedger, error)
	Save(ctx context.Context, ledger *domain.MasteryLedger) error
}

type SourceCorpusPort interface {
	RetrieveChunks(ctx context.Context, userID shared.UserID, pathID shared.PathID, query string, limit int) ([]GroundedChunk, error)
	IngestPathCorpus(ctx context.Context, userID shared.UserID, pathID shared.PathID, folderID string) error
}
