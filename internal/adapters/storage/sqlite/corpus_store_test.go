package sqlite_test

import (
	"context"
	"math"
	"testing"

	"github.com/alejogs4/whatsapp-assistant/internal/adapters/storage/sqlite"
	learningdomain "github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
)

type fakeEmbedder struct {
	vectors map[string][]float32
}

func (f *fakeEmbedder) EmbedChunks(_ context.Context, chunks []string) ([][]float32, error) {
	out := make([][]float32, len(chunks))
	for i, c := range chunks {
		if v, ok := f.vectors[c]; ok {
			out[i] = v
		} else {
			out[i] = []float32{0, 0, 0}
		}
	}
	return out, nil
}

func TestCorpusStore_CosineRankingAndMultiTenantIsolation(t *testing.T) {
	db := openMigratedTestDB(t)
	seedUsers(t, db, "user-a", "user-b")

	pathRepo := sqlite.NewLearningPathRepo(db)
	ctx := context.Background()
	_ = pathRepo.Save(ctx, learningdomain.NewLearningPath("path-rdd", "user-a", "RDD", "f1"))
	_ = pathRepo.Save(ctx, learningdomain.NewLearningPath("path-algo", "user-a", "Algorithms", "f2"))
	_ = pathRepo.Save(ctx, learningdomain.NewLearningPath("path-rdd-b", "user-b", "RDD", "f3"))

	embedder := &fakeEmbedder{
		vectors: map[string][]float32{
			"chunk-1":     {1, 0, 0},
			"chunk-2":     {0, 1, 0},
			"chunk-3":     {0.6, 0.8, 0},
			"algo-chunk":  {0, 1, 0},
			"userb-chunk": {0, 1, 0},
			"query-near2": {0, 1, 0},
			"zero-chunk":  {0, 0, 0},
		},
	}

	store := sqlite.NewCorpusStore(db, embedder, nil)
	if err := store.StoreChunks(ctx, "user-a", "path-rdd", []string{"chunk-1", "chunk-2", "chunk-3"}); err != nil {
		t.Fatalf("StoreChunks user-a rdd: %v", err)
	}
	if err := store.StoreChunks(ctx, "user-a", "path-algo", []string{"algo-chunk"}); err != nil {
		t.Fatalf("StoreChunks user-a algo: %v", err)
	}
	if err := store.StoreChunks(ctx, "user-b", "path-rdd-b", []string{"userb-chunk"}); err != nil {
		t.Fatalf("StoreChunks user-b rdd: %v", err)
	}

	got, err := store.RetrieveChunks(ctx, "user-a", "path-rdd", "query-near2", 2)
	if err != nil {
		t.Fatalf("RetrieveChunks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	if got[0].Content != "chunk-2" || got[1].Content != "chunk-3" {
		t.Fatalf("unexpected ranking: got [%s (%.2f), %s (%.2f)]", got[0].Content, got[0].Similarity, got[1].Content, got[1].Similarity)
	}
	for _, c := range got {
		if c.PathID != "path-rdd" {
			t.Fatalf("cross-path contamination: %+v", c)
		}
		if c.Similarity < 0 || c.Similarity > 1.0001 {
			t.Fatalf("similarity out of [0,1]: %f", c.Similarity)
		}
	}

	// Zero-norm vector safety
	if err := store.StoreChunks(ctx, "user-a", "path-rdd", []string{"zero-chunk"}); err != nil {
		t.Fatalf("StoreChunks zero-chunk: %v", err)
	}
	all, err := store.RetrieveChunks(ctx, "user-a", "path-rdd", "query-near2", 10)
	if err != nil {
		t.Fatalf("RetrieveChunks all: %v", err)
	}
	for _, c := range all {
		if math.IsNaN(float64(c.Similarity)) {
			t.Fatalf("got NaN similarity for chunk %q", c.Content)
		}
		if c.Content == "zero-chunk" && c.Similarity != 0 {
			t.Fatalf("expected 0 similarity for zero-norm vector, got %f", c.Similarity)
		}
	}
}
