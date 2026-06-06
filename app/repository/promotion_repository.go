package repository

import (
	"checkout-service/app/model/entity"
	"checkout-service/app/repository/query"
	"checkout-service/config"
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

type promotionRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewPromotionRepositoryImpl(cfg config.DatabaseConfig, db *sql.DB) PromotionRepositoryInterface {
	return &promotionRepository{
		db:      db,
		timeout: cfg.Timeout * time.Second,
	}
}

func (r *promotionRepository) SelectActivePromotionsWithRules(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	stmt, err := r.db.PrepareContext(ctx, query.SelectActivePromotionsWithRules)
	if err != nil {
		return nil, fmt.Errorf("prepare SelectActivePromotionsWithRules: %w", err)
	}

	defer func() {
		if cErr := stmt.Close(); cErr != nil {
			log.Printf("Close SelectActivePromotionsWithRules stmt: %v", cErr)
		}
	}()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("query active promotions with rules: %w", err)
	}

	defer func() {
		if cErr := rows.Close(); cErr != nil {
			log.Printf("Close SelectActivePromotionsWithRules rows: %v", cErr)
		}
	}()

	promotionsWithRules := make([]entity.PromotionWithRuleEntity, 0)

	for rows.Next() {
		var promotionWithRule entity.PromotionWithRuleEntity

		err = rows.Scan(
			&promotionWithRule.Promotion.PromotionID,
			&promotionWithRule.Promotion.PromotionName,
			&promotionWithRule.Promotion.Type,
			&promotionWithRule.Promotion.Description,
			&promotionWithRule.Promotion.IsActive,
			&promotionWithRule.Promotion.CreatedAt,
			&promotionWithRule.Promotion.UpdatedAt,

			&promotionWithRule.PromotionRule.PromotionRuleID,
			&promotionWithRule.PromotionRule.PromotionID,
			&promotionWithRule.PromotionRule.RuleConfig,
			&promotionWithRule.PromotionRule.CreatedAt,
			&promotionWithRule.PromotionRule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan active promotion with rule: %w", err)
		}

		promotionsWithRules = append(promotionsWithRules, promotionWithRule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active promotions with rules: %w", err)
	}

	return promotionsWithRules, nil
}
