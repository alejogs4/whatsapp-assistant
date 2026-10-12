package user

import (
	"context"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id shared.UserID) (*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
}
