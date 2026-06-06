package repository

import (
	"checkout-service/app/model/entity"
	"context"
)

type PromotionRepositoryInterface interface {
	SelectActivePromotionsWithRules(ctx context.Context) ([]entity.PromotionWithRuleEntity, error)
}
