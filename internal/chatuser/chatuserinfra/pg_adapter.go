package chatuserinfra

import (
	"context"
	"fmt"

	"github.com/Abraxas-365/opd/internal/chatuser"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/jmoiron/sqlx"
)

type PostgresStore struct {
	db *sqlx.DB
}

// NewChatUserStore creates a new PostgresStore for chat user repository
func NewChatUserStore(db *sqlx.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// GetChatUserByID retrieves a chat user by their ID
func (s *PostgresStore) GetChatUserByID(ctx context.Context, chatUserID string) (*chatuser.ChatUser, error) {
	var u chatuser.ChatUser

	query := `SELECT id, age, gender, occupation, location FROM chatUser WHERE id = $1`
	err := s.db.GetContext(ctx, &u, query, chatUserID)
	if err != nil {
		return nil, errors.ErrNotFound("Chat user not found")
	}
	return &u, nil
}

// CreateChatUser inserts a new chat user
func (s *PostgresStore) CreateChatUser(ctx context.Context, u chatuser.ChatUser) (*chatuser.ChatUser, error) {
	// First, check if user already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM chatUser WHERE id = $1)`
	err := s.db.GetContext(ctx, &exists, checkQuery, u.ID)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("Failed to check user existence: %v", err))
	}

	if exists {
		return nil, errors.ErrConflict("Chat user already exists")
	}

	// If user doesn't exist, proceed with creation
	query := `
		INSERT INTO chatUser (id, age, gender, occupation, location) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id, age, gender, occupation, location`

	err = s.db.QueryRowContext(
		ctx,
		query,
		u.ID,
		u.Age,
		u.Gender,
		u.Ocupation,
		u.Location,
	).Scan(&u.ID, &u.Age, &u.Gender, &u.Ocupation, &u.Location)

	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("Failed to create chat user: %v", err))
	}

	return &u, nil
}

func (s *PostgresStore) GetTotalCount(ctx context.Context, query string, args ...interface{}) (int, error) {
	var total int
	err := s.db.GetContext(ctx, &total, query, args...)
	if err != nil {
		return 0, errors.ErrDatabase(fmt.Sprintf("Failed to get total count: %v", err))
	}
	return total, nil
}
