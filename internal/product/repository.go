package product

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	Create(ctx context.Context, sku, name, description, price, createdBy string) (*Product, error)
	Update(ctx context.Context, id, name, description, price string) (*Product, error)
	Delete(ctx context.Context, id string) error
}

type productRepository struct {
	db *sqlx.DB
}

func NewProductRepository(db *sqlx.DB) Repository {
	return &productRepository{db: db}
}

func (r *productRepository) GetAll(ctx context.Context) ([]Product, error) {
	var products []Product

	query := "SELECT id, sku, name, description, price, created_by, created_at, updated_by, updated_at FROM products ORDER BY created_at"
	err := r.db.SelectContext(ctx, &products, query)
	if err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*Product, error) {
	var product Product

	query := "SELECT id, sku, name, description, price, created_by, created_at, updated_by, updated_at FROM products WHERE id = $1"
	err := r.db.GetContext(ctx, &product, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &product, nil
}

func (r *productRepository) Create(ctx context.Context, sku, name, description, price, createdBy string) (*Product, error) {
	var req Product

	query := `INSERT INTO products(sku, name, description, price, created_by) VALUES ($1, $2, $3, $4, $5) RETURNING id, sku, name, description, price, created_by, created_at, updated_by, updated_at`
	err := r.db.QueryRowxContext(ctx, query, sku, name, description, price, createdBy).StructScan(&req)
	if err != nil {
		return nil, err
	}

	return &req, nil
}

func (r *productRepository) Update(ctx context.Context, id, name, description, price string) (*Product, error) {
	var req Product

	query := `UPDATE products SET name=$1, description=$2, price=$3, updated_at=NOW() WHERE id=$4 RETURNING id, sku, name, description, price, created_by, created_at, updated_by, updated_at`
	err := r.db.QueryRowxContext(ctx, query, name, description, price, id).StructScan(&req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &req, nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`
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
