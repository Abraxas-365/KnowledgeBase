package user

import (
	"context"

	"github.com/Abraxas-365/toolkit/pkg/database"
)

type Repository interface {
	GetUserByProviderID(ctx context.Context, provider, providerID string) (*User, error)
	CreateUser(ctx context.Context, u *User) (*User, error)
	GetUsers(ctx context.Context, page, pageSize int) (database.PaginatedRecord[User], error)
	GetNotAdminUsers(ctx context.Context, page, pageSize int) (database.PaginatedRecord[User], error)
	GetUsersAdminRole(ctx context.Context, page, pageSize int) (database.PaginatedRecord[User], error)
	GetUsersByRole(ctx context.Context, role Role, page, pageSize int) (database.PaginatedRecord[User], error)
	GetUsersBySubscription(ctx context.Context, subscriptionType SubscriptionType, page, pageSize int) (database.PaginatedRecord[User], error)
	UpdateUser(ctx context.Context, user *User) (*User, error)
	UpdateUserRole(ctx context.Context, userID string, role Role) error
	UpdateUserSubscription(ctx context.Context, userID string, subscriptionType SubscriptionType) error
	DeleteUser(ctx context.Context, userID string) error
	GetWhitelist(ctx context.Context) ([]string, error)
	AddToWhitelist(ctx context.Context, email string) error
	RemoveFromWhitelist(ctx context.Context, email string) error
	IsInWhitelist(ctx context.Context, email string) (bool, error)
	PromoteUserToAdmin(ctx context.Context, userID string) error
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetTotalCount(ctx context.Context, query string, args ...interface{}) (int, error)
}

