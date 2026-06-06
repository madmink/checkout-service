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

	"github.com/google/uuid"
)

type checkoutRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewCheckoutRepositoryImpl(cfg config.DatabaseConfig, db *sql.DB) CheckoutRepositoryInterface {
	return &checkoutRepository{
		db:      db,
		timeout: cfg.Timeout * time.Second,
	}
}

func (r *checkoutRepository) InsertCheckoutOrder(ctx context.Context, exec DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := exec.PrepareContext(ctx, query.InsertCheckoutOrder)
	if err != nil {
		return fmt.Errorf("prepare InsertCheckoutOrder: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close InsertCheckoutOrder stmt: %v", cErr)
		}
	}()

	res, err := stmt.ExecContext(
		ctx,
		checkoutOrder.CheckoutOrderID, // $1
		checkoutOrder.TotalBeforeDiscountAmountMinor, // $2
		checkoutOrder.TotalDiscountAmountMinor,       // $3
		checkoutOrder.FinalTotalAmountMinor,          // $4
		checkoutOrder.Currency,                       // $5
		checkoutOrder.CreatedAt,                      // $6
	)

	if err != nil {
		return fmt.Errorf("insert checkout order: %w", err)
	}

	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		log.Println("InsertCheckoutOrder: no rows affected")
	}

	return nil
}

func (r *checkoutRepository) InsertCheckoutOrderItems(ctx context.Context, exec DBExecutor, items []entity.CheckoutOrderItemEntity) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := exec.PrepareContext(ctx, query.InsertCheckoutOrderItems)
	if err != nil {
		return fmt.Errorf("prepare InsertCheckoutOrderItems: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close InsertCheckoutOrderItems stmt: %v", cErr)
		}
	}()

	for _, item := range items {
		res, err := stmt.ExecContext(
			ctx,
			item.CheckoutOrderItemID,      // $1
			item.CheckoutOrderID,          // $2
			item.ProductID,                // $3
			item.SKU,                      // $4
			item.Name,                     // $5
			item.Quantity,                 // $6
			item.UnitPriceAmountMinor,     // $7
			item.SubtotalAmountMinor,      // $8
			item.DiscountAmountMinor,      // $9
			item.FinalSubtotalAmountMinor, // $10
			item.AppliedPromotionID,       // $11
			item.Currency,                 // $12
			item.CreatedAt,                // $13
		)

		if err != nil {
			return fmt.Errorf("insert checkout order item with sku %s: %w", item.SKU, err)
		}

		rows, err := res.RowsAffected()
		if err == nil && rows == 0 {
			log.Printf("InsertCheckoutOrderItems: no rows affected for sku %s", item.SKU)
		}
	}

	return nil
}

func (r *checkoutRepository) SelectCheckoutOrderByID(ctx context.Context, checkoutOrderID uuid.UUID) (*entity.CheckoutOrderEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := r.db.PrepareContext(ctx, query.SelectCheckoutOrderByID)
	if err != nil {
		return nil, fmt.Errorf("prepare SelectCheckoutOrderByID: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close SelectCheckoutOrderByID stmt: %v", cErr)
		}
	}()

	var checkoutOrder entity.CheckoutOrderEntity

	err = stmt.QueryRowContext(
		ctx,
		checkoutOrderID, // $1
	).Scan(
		&checkoutOrder.CheckoutOrderID,
		&checkoutOrder.TotalBeforeDiscountAmountMinor,
		&checkoutOrder.TotalDiscountAmountMinor,
		&checkoutOrder.FinalTotalAmountMinor,
		&checkoutOrder.Currency,
		&checkoutOrder.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("select checkout order by id: %w", err)
	}

	return &checkoutOrder, nil
}

func (r *checkoutRepository) SelectCheckoutOrderItemsByCheckoutOrderID(ctx context.Context, checkoutOrderID uuid.UUID) ([]entity.CheckoutOrderItemEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := r.db.PrepareContext(ctx, query.SelectCheckoutOrderItemsByCheckoutOrderID)
	if err != nil {
		return nil, fmt.Errorf("prepare SelectCheckoutOrderItemsByCheckoutOrderID: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close SelectCheckoutOrderItemsByCheckoutOrderID stmt: %v", cErr)
		}
	}()

	rows, err := stmt.QueryContext(
		ctx,
		checkoutOrderID, // $1
	)
	if err != nil {
		return nil, fmt.Errorf("select checkout order items by checkout order id: %w", err)
	}

	defer func() {
		if cErr := rows.Close(); cErr != nil {
			log.Printf("Close SelectCheckoutOrderItemsByCheckoutOrderID rows: %v", cErr)
		}
	}()

	items := make([]entity.CheckoutOrderItemEntity, 0)

	for rows.Next() {
		var item entity.CheckoutOrderItemEntity

		err = rows.Scan(
			&item.CheckoutOrderItemID,
			&item.CheckoutOrderID,
			&item.ProductID,
			&item.SKU,
			&item.Name,
			&item.Quantity,
			&item.UnitPriceAmountMinor,
			&item.SubtotalAmountMinor,
			&item.DiscountAmountMinor,
			&item.FinalSubtotalAmountMinor,
			&item.AppliedPromotionID,
			&item.Currency,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan checkout order item: %w", err)
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkout order items: %w", err)
	}

	return items, nil
}
