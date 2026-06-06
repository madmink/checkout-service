package entity

import (
	"checkout-service/app/model/constant"
	"time"

	"github.com/google/uuid"
)

type ProductEntity struct {
	ProductID        uuid.UUID
	SKU              string
	Name             string
	PriceAmountMinor int64
	Currency         constant.Currency
	InventoryQty     int
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
