package repository

import (
	"checkout-service/app/model/entity"
	"checkout-service/app/repository/query"
	"checkout-service/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
)

type productRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewProductRepositoryImpl(cfg config.DatabaseConfig, db *sql.DB) ProductRepositoryInterface {
	return &productRepository{
		db:      db,
		timeout: cfg.Timeout * time.Second,
	}
}

func (r *productRepository) SelectProductBySKU(ctx context.Context, sku string) (*entity.ProductEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := r.db.PrepareContext(ctx, query.SelectProductBySKU)
	if err != nil {
		return nil, fmt.Errorf("prepare SelectProductBySKU: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close SelectProductBySKU stmt: %v", cErr)
		}
	}()

	var product entity.ProductEntity

	err = stmt.QueryRowContext(ctx, sku).Scan(
		&product.ProductID,
		&product.SKU,
		&product.Name,
		&product.PriceAmountMinor,
		&product.Currency,
		&product.InventoryQty,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("select product by sku: %w", err)
	}

	return &product, nil
}

func (r *productRepository) SelectProductsBySKUs(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := r.db.PrepareContext(ctx, query.SelectProductsBySKUs)
	if err != nil {
		return nil, fmt.Errorf("prepare SelectProductsBySKUs: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close SelectProductsBySKUs stmt: %v", cErr)
		}
	}()

	rows, err := stmt.QueryContext(ctx, pq.Array(skus))
	if err != nil {
		return nil, fmt.Errorf("query products by skus: %w", err)
	}

	defer func() {
		if cErr := rows.Close(); cErr != nil {
			log.Printf("Close SelectProductsBySKUs rows: %v", cErr)
		}
	}()

	products := make([]entity.ProductEntity, 0)

	for rows.Next() {
		var product entity.ProductEntity

		err = rows.Scan(
			&product.ProductID,
			&product.SKU,
			&product.Name,
			&product.PriceAmountMinor,
			&product.Currency,
			&product.InventoryQty,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}

		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products by skus: %w", err)
	}

	return products, nil
}

func (r *productRepository) UpdateInventoryBySKU(ctx context.Context, exec DBExecutor, sku string, quantity int) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := exec.PrepareContext(ctx, query.UpdateInventoryBySKU)
	if err != nil {
		return fmt.Errorf("prepare UpdateInventoryBySKU: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close UpdateInventoryBySKU stmt: %v", cErr)
		}
	}()

	res, err := stmt.ExecContext(
		ctx,
		quantity,   // $1
		time.Now(), // $2
		sku,        // $3
	)

	if err != nil {
		return fmt.Errorf("update inventory by sku %s: %w", sku, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check UpdateInventoryBySKU rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("UpdateInventoryBySKU: insufficient inventory or inactive product for sku %s", sku)
	}

	return nil
}
