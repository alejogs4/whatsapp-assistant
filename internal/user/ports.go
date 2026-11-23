package user

import (
	"context"

	"github.com/alejogs4/whatsapp-assistant/internal/shared"
	"github.com/alejogs4/whatsapp-assistant/internal/user/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id shared.UserID) (*domain.User, error)
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error)
	FindAll(ctx context.Context) ([]*domain.User, error)
	Save(ctx context.Context, user *domain.User) error
}
