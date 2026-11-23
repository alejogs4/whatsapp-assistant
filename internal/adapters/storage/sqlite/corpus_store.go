package sqlite

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

var _ learning.SourceCorpusPort = (*CorpusStore)(nil)

// FolderFetcher optionally retrieves raw document chunks from an external source (e.g. Google Drive).
type FolderFetcher interface {
	FetchFolderChunks(ctx context.Context, folderID string) ([]string, error)
}

// CorpusStore implements learning.SourceCorpusPort using SQLite BLOB embeddings
// and pure-Go local cosine k-NN search strictly scoped by (user_id, path_id).
type CorpusStore struct {
	db       *DB
	embedder learning.EmbeddingPort
	fetcher  FolderFetcher
}

func NewCorpusStore(db *DB, embedder learning.EmbeddingPort, fetcher FolderFetcher) *CorpusStore {
	return &CorpusStore{
		db:       db,
		embedder: embedder,
		fetcher:  fetcher,
	}
}

func (s *CorpusStore) StoreChunks(
	ctx context.Context,
	userID shared.UserID,
	pathID shared.PathID,
	chunks []string,
) error {
	if len(chunks) == 0 {
		return nil
	}
	if s.embedder == nil {
		return fmt.Errorf("sqlite: embedder is not configured")
	}

	vectors, err := s.embedder.EmbedChunks(ctx, chunks)
	if err != nil {
		return fmt.Errorf("sqlite: embed chunks: %w", err)
	}
	if len(vectors) != len(chunks) {
		return fmt.Errorf("sqlite: embedder returned %d vectors for %d chunks", len(vectors), len(chunks))
	}

	tx, err := s.db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin chunk tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO learning_path_chunks (id, user_id, path_id, content, embedding_blob, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("sqlite: prepare chunk insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i, content := range chunks {
		chunkID := uuid.NewString()
		if _, err := stmt.ExecContext(
			ctx,
			chunkID,
			string(userID),
			string(pathID),
			content,
			encodeVector(vectors[i]),
			now,
		); err != nil {
			return fmt.Errorf("sqlite: insert chunk %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit chunks: %w", err)
	}
	return nil
}

func (s *CorpusStore) RetrieveChunks(
	ctx context.Context,
	userID shared.UserID,
	pathID shared.PathID,
	query string,
	limit int,
) ([]learning.GroundedChunk, error) {
	if limit <= 0 {
		return []learning.GroundedChunk{}, nil
	}
	if s.embedder == nil {
		return nil, fmt.Errorf("sqlite: embedder is not configured")
	}

	queryVecs, err := s.embedder.EmbedChunks(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("sqlite: embed query: %w", err)
	}
	if len(queryVecs) != 1 {
		return nil, fmt.Errorf("sqlite: expected 1 query embedding, got %d", len(queryVecs))
	}
	qVec := queryVecs[0]

	rows, err := s.db.sql.QueryContext(ctx, `
		SELECT id, content, embedding_blob
		FROM learning_path_chunks
		WHERE user_id = ? AND path_id = ?
	`, string(userID), string(pathID))
	if err != nil {
		return nil, fmt.Errorf("sqlite: query chunks: %w", err)
	}
	defer rows.Close()

	var scored []learning.GroundedChunk
	for rows.Next() {
		var id, content string
		var blob []byte
		if err := rows.Scan(&id, &content, &blob); err != nil {
			return nil, fmt.Errorf("sqlite: scan chunk: %w", err)
		}
		vec, err := decodeVector(blob)
		if err != nil {
			return nil, err
		}
		scored = append(scored, learning.GroundedChunk{
			ID:         id,
			PathID:     pathID,
			Content:    content,
			Similarity: cosineSimilarity(qVec, vec),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate chunks: %w", err)
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Similarity == scored[j].Similarity {
			return scored[i].ID < scored[j].ID
		}
		return scored[i].Similarity > scored[j].Similarity
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}
	return scored, nil
}

func (s *CorpusStore) IngestPathCorpus(
	ctx context.Context,
	userID shared.UserID,
	pathID shared.PathID,
	folderID string,
) error {
	if s.fetcher == nil {
		return fmt.Errorf("sqlite: folder fetcher is not configured for folder %q", folderID)
	}
	chunks, err := s.fetcher.FetchFolderChunks(ctx, folderID)
	if err != nil {
		return fmt.Errorf("sqlite: fetch folder %q: %w", folderID, err)
	}
	return s.StoreChunks(ctx, userID, pathID, chunks)
}
