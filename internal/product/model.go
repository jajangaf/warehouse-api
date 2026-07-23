package product

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID     `db:"id"`
	SKU         string        `db:"sku"`
	Name        string        `db:"name"`
	Description string        `db:"description"`
	Price       string        `db:"price"`
	CreatedBy   uuid.UUID     `db:"created_by"`
	UpdatedBy   uuid.NullUUID `db:"updated_by"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}
