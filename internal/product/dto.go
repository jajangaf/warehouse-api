package product

type CreateRequest struct {
	SKU         string `json:"sku" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Price       string `json:"price" binding:"required"`
}

type UpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Price       string `json:"price" binding:"required"`
}

type ProductResponse struct {
	ID          string  `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	CreatedBy   string  `json:"created_by"`
	UpdatedBy   *string `json:"updated_by,omitempty"`
}

func ToProductResponse(p Product) ProductResponse {
	resp := ProductResponse{
		ID:          p.ID.String(),
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		CreatedBy:   p.CreatedBy.String(),
	}
	if p.UpdatedBy.Valid {
		s := p.UpdatedBy.UUID.String()
		resp.UpdatedBy = &s
	}
	return resp
}
