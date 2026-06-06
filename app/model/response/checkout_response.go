package response

import "github.com/google/uuid"

type CheckoutResponse struct {
	GeneralResponse
	IsSuccess bool           `json:"is_success"`
	Value     *CheckoutValue `json:"value,omitempty"`
}

type CheckoutValue struct {
	CheckoutOrderID                uuid.UUID               `json:"checkout_order_id,omitempty"`
	Items                          []CheckoutItemValue     `json:"items,omitempty"`
	AppliedPromotions              []AppliedPromotionValue `json:"applied_promotions,omitempty"`
	TotalBeforeDiscountAmountMinor int64                   `json:"total_before_discount_amount_minor,omitempty"`
	TotalDiscountAmountMinor       int64                   `json:"total_discount_amount_minor,omitempty"`
	FinalTotalAmountMinor          int64                   `json:"final_total_amount_minor,omitempty"`
	Currency                       string                  `json:"currency,omitempty"`
}

type CheckoutItemValue struct {
	CheckoutOrderItemID      uuid.UUID  `json:"checkout_order_item_id,omitempty"`
	ProductID                uuid.UUID  `json:"product_id,omitempty"`
	SKU                      string     `json:"sku,omitempty"`
	Name                     string     `json:"name,omitempty"`
	Quantity                 int        `json:"quantity,omitempty"`
	UnitPriceAmountMinor     int64      `json:"unit_price_amount_minor,omitempty"`
	SubtotalAmountMinor      int64      `json:"subtotal_amount_minor,omitempty"`
	DiscountAmountMinor      int64      `json:"discount_amount_minor,omitempty"`
	FinalSubtotalAmountMinor int64      `json:"final_subtotal_amount_minor,omitempty"`
	AppliedPromotionID       *uuid.UUID `json:"applied_promotion_id,omitempty"`
	Currency                 string     `json:"currency,omitempty"`
}

type AppliedPromotionValue struct {
	PromotionID         uuid.UUID `json:"promotion_id,omitempty"`
	PromotionName       string    `json:"promotion_name,omitempty"`
	PromotionType       string    `json:"promotion_type,omitempty"`
	DiscountAmountMinor int64     `json:"discount_amount_minor,omitempty"`
}
