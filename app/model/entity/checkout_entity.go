package entity

import (
	"checkout-service/app/model/constant"
	"time"

	"github.com/google/uuid"
)

type CheckoutOrderEntity struct {
	CheckoutOrderID                uuid.UUID
	TotalBeforeDiscountAmountMinor int64
	TotalDiscountAmountMinor       int64
	FinalTotalAmountMinor          int64
	Currency                       constant.Currency
	CreatedAt                      time.Time
}

type CheckoutOrderItemEntity struct {
	CheckoutOrderItemID      uuid.UUID
	CheckoutOrderID          uuid.UUID
	ProductID                uuid.UUID
	SKU                      string
	Name                     string
	Quantity                 int
	UnitPriceAmountMinor     int64
	SubtotalAmountMinor      int64
	DiscountAmountMinor      int64
	FinalSubtotalAmountMinor int64
	AppliedPromotionID       *uuid.UUID
	Currency                 constant.Currency
	CreatedAt                time.Time
}
