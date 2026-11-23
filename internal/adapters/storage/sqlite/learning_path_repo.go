package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/alejogs4/whatsapp-assistant/internal/learning"
	"github.com/alejogs4/whatsapp-assistant/internal/learning/domain"
	"github.com/alejogs4/whatsapp-assistant/internal/shared"
)

var _ learning.LearningPathRepository = (*LearningPathRepo)(nil)

// LearningPathRepo implements learning.LearningPathRepository with strict user_id scoping.
type LearningPathRepo struct {
	db *DB
}

func NewLearningPathRepo(db *DB) *LearningPathRepo {
	return &LearningPathRepo{db: db}
}

func (r *LearningPathRepo) FindByID(ctx context.Context, userID shared.UserID, id shared.PathID) (*domain.LearningPath, error) {
	row := r.db.sql.QueryRowContext(ctx, `
		SELECT id, user_id, title, source_folder_id, status
		FROM learning_paths
		WHERE user_id = ? AND id = ?
	`, string(userID), string(id))

	var pid, uid, title, folder, status string
	if err := row.Scan(&pid, &uid, &title, &folder, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrLearningPathNotFound
		}
		return nil, fmt.Errorf("sqlite: scan learning path: %w", err)
	}
	return &domain.LearningPath{
		ID:             shared.PathID(pid),
		UserID:         shared.UserID(uid),
		Title:          title,
		SourceFolderID: folder,
		Status:         domain.PathStatus(status),
	}, nil
}

func (r *LearningPathRepo) FindByUser(ctx context.Context, userID shared.UserID) ([]*domain.LearningPath, error) {
	rows, err := r.db.sql.QueryContext(ctx, `
		SELECT id, user_id, title, source_folder_id, status
		FROM learning_paths
		WHERE user_id = ?
		ORDER BY id ASC
	`, string(userID))
	if err != nil {
		return nil, fmt.Errorf("sqlite: find paths by user: %w", err)
	}
	defer rows.Close()

	var paths []*domain.LearningPath
	for rows.Next() {
		var pid, uid, title, folder, status string
		if err := rows.Scan(&pid, &uid, &title, &folder, &status); err != nil {
			return nil, fmt.Errorf("sqlite: scan learning path row: %w", err)
		}
		paths = append(paths, &domain.LearningPath{
			ID:             shared.PathID(pid),
			UserID:         shared.UserID(uid),
			Title:          title,
			SourceFolderID: folder,
			Status:         domain.PathStatus(status),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate learning paths: %w", err)
	}
	return paths, nil
}

func (r *LearningPathRepo) Save(ctx context.Context, path *domain.LearningPath) error {
	_, err := r.db.sql.ExecContext(ctx, `
		INSERT INTO learning_paths (id, user_id, title, source_folder_id, status)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			source_folder_id = excluded.source_folder_id,
			status = excluded.status
		WHERE learning_paths.user_id = excluded.user_id
	`,
		string(path.ID),
		string(path.UserID),
		path.Title,
		path.SourceFolderID,
		string(path.Status),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save learning path %q: %w", path.ID, err)
	}
	return nil
}
