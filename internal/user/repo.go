package user

import (
	"context"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/user"
)

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	FindByLogin(ctx context.Context, login string) (*user.User, error)
}
