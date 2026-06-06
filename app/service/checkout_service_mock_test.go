package service

import (
	"checkout-service/app/model/entity"
	"checkout-service/app/repository"
	"context"
)

func newCheckoutServiceForTest(
	productRepo repository.ProductRepositoryInterface,
	promotionRepo repository.PromotionRepositoryInterface,
	checkoutRepo repository.CheckoutRepositoryInterface,
) *checkoutServiceImpl {
	return &checkoutServiceImpl{
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}
}

type mockProductRepository struct {
	selectProductBySKUFn   func(ctx context.Context, sku string) (*entity.ProductEntity, error)
	selectProductsBySKUsFn func(ctx context.Context, skus []string) ([]entity.ProductEntity, error)
	updateInventoryBySKUFn func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error
}

func (m *mockProductRepository) SelectProductBySKU(
	ctx context.Context,
	sku string,
) (*entity.ProductEntity, error) {
	if m.selectProductBySKUFn == nil {
		return nil, nil
	}

	return m.selectProductBySKUFn(ctx, sku)
}

func (m *mockProductRepository) SelectProductsBySKUs(
	ctx context.Context,
	skus []string,
) ([]entity.ProductEntity, error) {
	if m.selectProductsBySKUsFn == nil {
		return nil, nil
	}

	return m.selectProductsBySKUsFn(ctx, skus)
}

func (m *mockProductRepository) UpdateInventoryBySKU(
	ctx context.Context,
	exec repository.DBExecutor,
	sku string,
	quantity int,
) error {
	if m.updateInventoryBySKUFn == nil {
		return nil
	}

	return m.updateInventoryBySKUFn(ctx, exec, sku, quantity)
}

type mockPromotionRepository struct {
	selectActivePromotionsWithRulesFn func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error)
}

func (m *mockPromotionRepository) SelectActivePromotionsWithRules(
	ctx context.Context,
) ([]entity.PromotionWithRuleEntity, error) {
	if m.selectActivePromotionsWithRulesFn == nil {
		return nil, nil
	}

	return m.selectActivePromotionsWithRulesFn(ctx)
}

type mockCheckoutRepository struct {
	insertCheckoutOrderFn      func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error
	insertCheckoutOrderItemsFn func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error
}

func (m *mockCheckoutRepository) InsertCheckoutOrder(
	ctx context.Context,
	exec repository.DBExecutor,
	checkoutOrder entity.CheckoutOrderEntity,
) error {
	if m.insertCheckoutOrderFn == nil {
		return nil
	}

	return m.insertCheckoutOrderFn(ctx, exec, checkoutOrder)
}

func (m *mockCheckoutRepository) InsertCheckoutOrderItems(
	ctx context.Context,
	exec repository.DBExecutor,
	items []entity.CheckoutOrderItemEntity,
) error {
	if m.insertCheckoutOrderItemsFn == nil {
		return nil
	}

	return m.insertCheckoutOrderItemsFn(ctx, exec, items)
}
