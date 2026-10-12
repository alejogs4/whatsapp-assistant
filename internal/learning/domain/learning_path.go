package domain

import "github.com/alejogs4/whatsapp-assistant/internal/shared"

type PathStatus string

const (
	PathStatusActive    PathStatus = "ACTIVE"
	PathStatusPaused    PathStatus = "PAUSED"
	PathStatusCompleted PathStatus = "COMPLETED"
)

type LearningPath struct {
	ID             shared.PathID
	UserID         shared.UserID
	Title          string
	SourceFolderID string
	Status         PathStatus
}

func NewLearningPath(id shared.PathID, userID shared.UserID, title string, sourceFolderID string) *LearningPath {
	return &LearningPath{
		ID:             id,
		UserID:         userID,
		Title:          title,
		SourceFolderID: sourceFolderID,
		Status:         PathStatusActive,
	}
}
