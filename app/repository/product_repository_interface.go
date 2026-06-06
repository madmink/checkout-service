package repository

import (
	"checkout-service/app/model/entity"
	"context"
)

type ProductRepositoryInterface interface {
	SelectProductBySKU(ctx context.Context, sku string) (*entity.ProductEntity, error)
	UpdateInventoryBySKU(ctx context.Context, exec DBExecutor, sku string, quantity int) error
	SelectProductsBySKUs(ctx context.Context, skus []string) ([]entity.ProductEntity, error)
}
