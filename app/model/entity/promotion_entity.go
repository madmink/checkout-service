package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PromotionType string

const (
	PromotionTypeFreeItem       PromotionType = "free_item"
	PromotionTypeBuyXPayY       PromotionType = "buy_x_pay_y"
	PromotionTypeBulkPercentage PromotionType = "bulk_percentage"
)

type PromotionWithRuleEntity struct {
	Promotion     PromotionEntity
	PromotionRule PromotionRuleEntity
}

type PromotionEntity struct {
	PromotionID   uuid.UUID
	PromotionName string
	Type          PromotionType
	Description   string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PromotionRuleEntity struct {
	PromotionRuleID uuid.UUID
	PromotionID     uuid.UUID
	RuleConfig      json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FreeItemRuleConfig struct {
	TriggerSKU        string `json:"trigger_sku"`
	TargetSKU         string `json:"target_sku"`
	FreeQtyPerTrigger int    `json:"free_qty_per_trigger"`
}

type BuyXPayYRuleConfig struct {
	TargetSKU string `json:"target_sku"`
	BuyQty    int    `json:"buy_qty"`
	PayQty    int    `json:"pay_qty"`
}

type BulkPercentageRuleConfig struct {
	TargetSKU          string `json:"target_sku"`
	MinQty             int    `json:"min_qty"`
	DiscountPercentage int    `json:"discount_percentage"`
}
