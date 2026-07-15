package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetAll(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetPasswordHashByID(ctx context.Context, id string) (string, error)
	Create(ctx context.Context, name, email, passwordHash string) (*User, error)
	Update(ctx context.Context, id, name, email, passwordHash string) (*User, error)
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) Repository {
	return &userRepository{db: db}
}

func (r *userRepository) GetAll(ctx context.Context) ([]User, error) {
	var users []User

	query := "SELECT id, name, email, role, created_at, updated_at FROM users ORDER BY created_at"
	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	var user User

	query := "SELECT id, name, email, role, created_at, updated_at FROM users WHERE id=$1"
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	query := "SELECT id, name, email, password_hash, role, created_at, updated_at FROM users WHERE email=$1"
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetPasswordHashByID(ctx context.Context, id string) (string, error) {
	var hash string

	query := "SELECT password_hash FROM users WHERE id=$1"
	err := r.db.GetContext(ctx, &hash, query, id)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func (r *userRepository) Create(ctx context.Context, name, email, passwordHash string) (*User, error) {
	var reg User

	query := `INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id, name, email, role, created_at, updated_at`

	err := r.db.QueryRowxContext(ctx, query, name, email, passwordHash).StructScan(&reg)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *userRepository) Update(ctx context.Context, id, name, email, passwordHash string) (*User, error) {
	var user User

	query := `UPDATE users SET name=$1, email=$2, password_hash=$3, updated_at=NOW() WHERE id=$4 RETURNING id, name, email, role, created_at, updated_at`

	err := r.db.QueryRowxContext(ctx, query, name, email, passwordHash, id).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM users WHERE id=$1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil

}
