package usersrv

import (
	"context"

	"github.com/Abraxas-365/opd/internal/user"
	"github.com/Abraxas-365/toolkit/pkg/database"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/Abraxas-365/toolkit/pkg/lucia"
)

type Service struct {
	repo user.Repository
}

func NewService(repo user.Repository) *Service {
	return &Service{
		repo,
	}
}

func (s *Service) GetUser(ctx context.Context, userID string) (*user.User, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) GetUserByProviderID(ctx context.Context, provider, providerID string) (*user.User, error) {
	u, err := s.repo.GetUserByProviderID(ctx, provider, providerID)
	if err != nil {
		return nil, err
	}

	isInWhiteList, err := s.repo.IsInWhitelist(ctx, u.Email)
	if err != nil {
		return nil, err
	} else if !isInWhiteList {
		return nil, errors.ErrUnauthorized("email is blacklisted")
	}

	return u, nil
}

func (s *Service) CreateUser(ctx context.Context, userInfo *lucia.UserInfo) (*user.User, error) {
	isInWhiteList, err := s.repo.IsInWhitelist(ctx, userInfo.Email)
	if err != nil {
		return nil, err
	} else if !isInWhiteList {
		return nil, errors.ErrUnauthorized("email is blacklisted")
	}

	u := &user.User{
		ID:               lucia.GenerateID(),
		Email:            userInfo.Email,
		IsAdmin:          false,
		Provider:         userInfo.Provider,
		ProviderID:       userInfo.ID,
		Role:             user.RoleLinkoUser,
		SubscriptionType: user.FreeTier,
	}

	u, err = s.repo.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) GetUsers(ctx context.Context, page, pageSize int) (database.PaginatedRecord[user.User], error) {
	return s.repo.GetUsers(ctx, page, pageSize)
}

func (s *Service) GetNotAdminUsers(ctx context.Context, page, pageSize int) (database.PaginatedRecord[user.User], error) {
	return s.repo.GetNotAdminUsers(ctx, page, pageSize)
}

func (s *Service) GetUsersAdminRole(ctx context.Context, page, pageSize int) (database.PaginatedRecord[user.User], error) {
	return s.repo.GetUsersAdminRole(ctx, page, pageSize)
}

func (s *Service) GetUsersByRole(ctx context.Context, role user.Role, page, pageSize int) (database.PaginatedRecord[user.User], error) {
	return s.repo.GetUsersByRole(ctx, role, page, pageSize)
}

func (s *Service) GetUsersBySubscription(ctx context.Context, subscriptionType user.SubscriptionType, page, pageSize int) (database.PaginatedRecord[user.User], error) {
	return s.repo.GetUsersBySubscription(ctx, subscriptionType, page, pageSize)
}

func (s *Service) UpdateUser(ctx context.Context, u *user.User) (*user.User, error) {
	return s.repo.UpdateUser(ctx, u)
}

func (s *Service) UpdateUserRole(ctx context.Context, userID string, role user.Role) error {
	return s.repo.UpdateUserRole(ctx, userID, role)
}

func (s *Service) UpdateUserSubscription(ctx context.Context, userID string, subscriptionType user.SubscriptionType) error {
	return s.repo.UpdateUserSubscription(ctx, userID, subscriptionType)
}

func (s *Service) PromoteUserToAdmin(ctx context.Context, userID string) error {
	return s.repo.PromoteUserToAdmin(ctx, userID)
}

func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.repo.DeleteUser(ctx, userID)
}

func (s *Service) GetWhitelist(ctx context.Context) ([]string, error) {
	return s.repo.GetWhitelist(ctx)
}

func (s *Service) AddToWhitelist(ctx context.Context, email string) error {
	return s.repo.AddToWhitelist(ctx, email)
}

func (s *Service) RemoveFromWhitelist(ctx context.Context, email string) error {
	return s.repo.RemoveFromWhitelist(ctx, email)
}
