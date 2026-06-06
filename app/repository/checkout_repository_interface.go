package repository

import (
	"checkout-service/app/model/entity"
	"context"
)

type CheckoutRepositoryInterface interface {
	InsertCheckoutOrder(ctx context.Context, exec DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error
	InsertCheckoutOrderItems(ctx context.Context, exec DBExecutor, items []entity.CheckoutOrderItemEntity) error
}
