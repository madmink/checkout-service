package service

import (
	"checkout-service/app/model/constant"
	"checkout-service/app/model/entity"
	"checkout-service/app/model/request"
	"checkout-service/app/model/response"
	"checkout-service/app/repository"
	"checkout-service/config"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type checkoutServiceImpl struct {
	config        *config.ApplicationConfig
	db            *sql.DB
	productRepo   repository.ProductRepositoryInterface
	promotionRepo repository.PromotionRepositoryInterface
	checkoutRepo  repository.CheckoutRepositoryInterface
}

func NewCheckoutServiceImpl(
	config *config.ApplicationConfig,
	db *sql.DB,
	productRepo repository.ProductRepositoryInterface,
	promotionRepo repository.PromotionRepositoryInterface,
	checkoutRepo repository.CheckoutRepositoryInterface,
) CheckoutServiceInterface {
	return &checkoutServiceImpl{
		config:        config,
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}
}

type promotionApplyContext struct {
	CheckoutOrderID uuid.UUID
	Now             time.Time

	RequestQtyBySKU map[string]int
	ProductBySKU    map[string]entity.ProductEntity
	ItemBySKU       map[string]*entity.CheckoutOrderItemEntity

	CheckoutItems     *[]entity.CheckoutOrderItemEntity
	InventoryUpdates  map[string]int
	AppliedPromotions *[]response.AppliedPromotionValue
}

func (s *checkoutServiceImpl) Checkout(ctx context.Context, req request.CheckoutRequest) (response.CheckoutValue, int, error) {
	var resp response.CheckoutValue

	requestQtyBySKU, skus, status, err := s.validateAndAggregateCheckoutRequest(req)
	if err != nil {
		return resp, status, err
	}

	promotionsWithRules, err := s.promotionRepo.SelectActivePromotionsWithRules(ctx)
	if err != nil {
		return resp, http.StatusInternalServerError, fmt.Errorf("select active promotions with rules: %w", err)
	}

	lookupSKUs, status, err := s.appendFreeItemTargetSKUs(
		skus,
		requestQtyBySKU,
		promotionsWithRules,
	)
	if err != nil {
		return resp, status, err
	}

	products, err := s.productRepo.SelectProductsBySKUs(ctx, lookupSKUs)
	if err != nil {
		return resp, http.StatusInternalServerError, fmt.Errorf("select products by skus: %w", err)
	}

	productBySKU, currency, status, err := s.validateProducts(
		products,
		lookupSKUs,
		requestQtyBySKU,
	)
	if err != nil {
		return resp, status, err
	}

	checkoutOrderID := uuid.New()
	now := time.Now()

	checkoutItems, itemBySKU, _ := s.buildInitialCheckoutItems(
		checkoutOrderID,
		now,
		skus,
		requestQtyBySKU,
		productBySKU,
	)

	appliedPromotions := make([]response.AppliedPromotionValue, 0)
	inventoryUpdates := s.buildInitialInventoryUpdates(requestQtyBySKU)

	promoCtx := &promotionApplyContext{
		CheckoutOrderID: checkoutOrderID,
		Now:             now,

		RequestQtyBySKU: requestQtyBySKU,
		ProductBySKU:    productBySKU,
		ItemBySKU:       itemBySKU,

		CheckoutItems:     &checkoutItems,
		InventoryUpdates:  inventoryUpdates,
		AppliedPromotions: &appliedPromotions,
	}

	status, err = s.applyPromotions(
		promotionsWithRules,
		promoCtx,
	)
	if err != nil {
		return resp, status, err
	}

	totalBeforeDiscount, totalDiscount, finalTotal := s.calculateCheckoutTotals(checkoutItems)

	checkoutOrder := entity.CheckoutOrderEntity{
		CheckoutOrderID:                checkoutOrderID,
		TotalBeforeDiscountAmountMinor: totalBeforeDiscount,
		TotalDiscountAmountMinor:       totalDiscount,
		FinalTotalAmountMinor:          finalTotal,
		Currency:                       currency,
		CreatedAt:                      now,
	}

	status, err = s.persistCheckout(
		ctx,
		checkoutOrder,
		checkoutItems,
		inventoryUpdates,
	)
	if err != nil {
		return resp, status, err
	}

	resp = s.buildCheckoutResponse(
		checkoutOrderID,
		checkoutItems,
		appliedPromotions,
		totalBeforeDiscount,
		totalDiscount,
		finalTotal,
		currency,
	)

	return resp, http.StatusOK, nil
}

func (s *checkoutServiceImpl) validateAndAggregateCheckoutRequest(req request.CheckoutRequest) (map[string]int, []string, int, error) {
	if len(req.Items) == 0 {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("checkout items are required")
	}

	requestQtyBySKU := make(map[string]int)
	skus := make([]string, 0, len(req.Items))

	for _, item := range req.Items {
		if item.SKU == "" {
			return nil, nil, http.StatusBadRequest, fmt.Errorf("sku is required")
		}

		if item.Quantity <= 0 {
			return nil, nil, http.StatusBadRequest, fmt.Errorf("quantity must be greater than zero for sku %s", item.SKU)
		}

		if _, exists := requestQtyBySKU[item.SKU]; !exists {
			skus = append(skus, item.SKU)
		}

		requestQtyBySKU[item.SKU] += item.Quantity
	}

	return requestQtyBySKU, skus, http.StatusOK, nil
}

func (s *checkoutServiceImpl) validateProducts(products []entity.ProductEntity, skus []string, requestQtyBySKU map[string]int) (map[string]entity.ProductEntity, constant.Currency, int, error) {
	if len(products) != len(skus) {
		return nil, "", http.StatusBadRequest, fmt.Errorf("one or more sku is invalid or inactive")
	}

	productBySKU := make(map[string]entity.ProductEntity)

	for _, product := range products {
		productBySKU[product.SKU] = product
	}

	var currency constant.Currency

	for i, sku := range skus {
		product := productBySKU[sku]

		if i == 0 {
			currency = product.Currency
		}

		if product.Currency != currency {
			return nil, "", http.StatusBadRequest, fmt.Errorf("mixed currency checkout is not allowed")
		}

		requestedQty := requestQtyBySKU[sku]
		if requestedQty > product.InventoryQty {
			return nil, "", http.StatusBadRequest, fmt.Errorf("insufficient inventory for sku %s", sku)
		}
	}

	return productBySKU, currency, http.StatusOK, nil
}

func (s *checkoutServiceImpl) buildInitialCheckoutItems(checkoutOrderID uuid.UUID, now time.Time, skus []string, requestQtyBySKU map[string]int, productBySKU map[string]entity.ProductEntity) (
	[]entity.CheckoutOrderItemEntity, map[string]*entity.CheckoutOrderItemEntity, int64) {
	checkoutItems := make([]entity.CheckoutOrderItemEntity, 0, len(skus))
	itemBySKU := make(map[string]*entity.CheckoutOrderItemEntity)

	var totalBeforeDiscount int64

	for _, sku := range skus {
		product := productBySKU[sku]
		qty := requestQtyBySKU[sku]

		subtotal := product.PriceAmountMinor * int64(qty)

		item := entity.CheckoutOrderItemEntity{
			CheckoutOrderItemID:      uuid.New(),
			CheckoutOrderID:          checkoutOrderID,
			ProductID:                product.ProductID,
			SKU:                      product.SKU,
			Name:                     product.Name,
			Quantity:                 qty,
			UnitPriceAmountMinor:     product.PriceAmountMinor,
			SubtotalAmountMinor:      subtotal,
			DiscountAmountMinor:      0,
			FinalSubtotalAmountMinor: subtotal,
			AppliedPromotionID:       nil,
			Currency:                 product.Currency,
			CreatedAt:                now,
		}

		checkoutItems = append(checkoutItems, item)
		itemBySKU[sku] = &checkoutItems[len(checkoutItems)-1]

		totalBeforeDiscount += subtotal
	}

	return checkoutItems, itemBySKU, totalBeforeDiscount
}

func (s *checkoutServiceImpl) buildInitialInventoryUpdates(
	requestQtyBySKU map[string]int,
) map[string]int {
	inventoryUpdates := make(map[string]int)

	for sku, qty := range requestQtyBySKU {
		inventoryUpdates[sku] += qty
	}

	return inventoryUpdates
}

func (s *checkoutServiceImpl) applyPromotions(promotionsWithRules []entity.PromotionWithRuleEntity, promoCtx *promotionApplyContext) (int, error) {
	for _, promotionWithRule := range promotionsWithRules {
		promotion := promotionWithRule.Promotion
		rule := promotionWithRule.PromotionRule

		var status int
		var err error

		switch promotion.Type {
		case entity.PromotionTypeFreeItem:
			status, err = s.applyFreeItemPromotion(
				promotion,
				rule,
				promoCtx,
			)

		case entity.PromotionTypeBuyXPayY:
			status, err = s.applyBuyXPayYPromotion(
				promotion,
				rule,
				promoCtx,
			)

		case entity.PromotionTypeBulkPercentage:
			status, err = s.applyBulkPercentagePromotion(
				promotion,
				rule,
				promoCtx,
			)

		default:
			continue
		}

		if err != nil {
			return status, err
		}
	}

	return http.StatusOK, nil
}

func (s *checkoutServiceImpl) applyFreeItemPromotion(
	promotion entity.PromotionEntity,
	rule entity.PromotionRuleEntity,
	promoCtx *promotionApplyContext,
) (int, error) {
	var cfg entity.FreeItemRuleConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("parse free item rule config: %w", err)
	}

	triggerQty := promoCtx.RequestQtyBySKU[cfg.TriggerSKU]
	if triggerQty <= 0 {
		return http.StatusOK, nil
	}

	freeQty := triggerQty * cfg.FreeQtyPerTrigger
	if freeQty <= 0 {
		return http.StatusOK, nil
	}

	freeProduct, exists := promoCtx.ProductBySKU[cfg.TargetSKU]
	if !exists {
		return http.StatusBadRequest, fmt.Errorf("free item sku %s is not available", cfg.TargetSKU)
	}

	if freeQty > freeProduct.InventoryQty {
		return http.StatusBadRequest, fmt.Errorf("insufficient inventory for free item sku %s", cfg.TargetSKU)
	}

	discountAmount := freeProduct.PriceAmountMinor * int64(freeQty)

	freeItem, exists := promoCtx.ItemBySKU[cfg.TargetSKU]
	if !exists {
		item := entity.CheckoutOrderItemEntity{
			CheckoutOrderItemID:      uuid.New(),
			CheckoutOrderID:          promoCtx.CheckoutOrderID,
			ProductID:                freeProduct.ProductID,
			SKU:                      freeProduct.SKU,
			Name:                     freeProduct.Name,
			Quantity:                 freeQty,
			UnitPriceAmountMinor:     freeProduct.PriceAmountMinor,
			SubtotalAmountMinor:      discountAmount,
			DiscountAmountMinor:      discountAmount,
			FinalSubtotalAmountMinor: 0,
			AppliedPromotionID:       &promotion.PromotionID,
			Currency:                 freeProduct.Currency,
			CreatedAt:                promoCtx.Now,
		}

		*promoCtx.CheckoutItems = append(*promoCtx.CheckoutItems, item)
		promoCtx.ItemBySKU[cfg.TargetSKU] = &(*promoCtx.CheckoutItems)[len(*promoCtx.CheckoutItems)-1]
	} else {
		freeItem.DiscountAmountMinor += discountAmount
		freeItem.FinalSubtotalAmountMinor -= discountAmount
		freeItem.AppliedPromotionID = &promotion.PromotionID

		if freeItem.FinalSubtotalAmountMinor < 0 {
			freeItem.FinalSubtotalAmountMinor = 0
		}
	}

	promoCtx.InventoryUpdates[cfg.TargetSKU] += freeQty

	*promoCtx.AppliedPromotions = appendAppliedPromotion(
		*promoCtx.AppliedPromotions,
		promotion,
		discountAmount,
	)

	return http.StatusOK, nil
}

func (s *checkoutServiceImpl) applyBuyXPayYPromotion(
	promotion entity.PromotionEntity,
	rule entity.PromotionRuleEntity,
	promoCtx *promotionApplyContext,
) (int, error) {
	var cfg entity.BuyXPayYRuleConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("parse buy x pay y rule config: %w", err)
	}

	item, exists := promoCtx.ItemBySKU[cfg.TargetSKU]
	if !exists {
		return http.StatusOK, nil
	}

	if cfg.BuyQty <= 0 || cfg.PayQty <= 0 || cfg.PayQty >= cfg.BuyQty {
		return http.StatusOK, nil
	}

	freeGroupCount := item.Quantity / cfg.BuyQty
	freeQty := freeGroupCount * (cfg.BuyQty - cfg.PayQty)

	if freeQty <= 0 {
		return http.StatusOK, nil
	}

	discountAmount := item.UnitPriceAmountMinor * int64(freeQty)

	item.DiscountAmountMinor += discountAmount
	item.FinalSubtotalAmountMinor -= discountAmount
	item.AppliedPromotionID = &promotion.PromotionID

	*promoCtx.AppliedPromotions = appendAppliedPromotion(
		*promoCtx.AppliedPromotions,
		promotion,
		discountAmount,
	)

	return http.StatusOK, nil
}

func (s *checkoutServiceImpl) applyBulkPercentagePromotion(
	promotion entity.PromotionEntity,
	rule entity.PromotionRuleEntity,
	promoCtx *promotionApplyContext,
) (int, error) {
	var cfg entity.BulkPercentageRuleConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("parse bulk percentage rule config: %w", err)
	}

	item, exists := promoCtx.ItemBySKU[cfg.TargetSKU]
	if !exists {
		return http.StatusOK, nil
	}

	if item.Quantity < cfg.MinQty {
		return http.StatusOK, nil
	}

	if cfg.DiscountPercentage <= 0 {
		return http.StatusOK, nil
	}

	discountAmount := item.SubtotalAmountMinor * int64(cfg.DiscountPercentage) / 100

	item.DiscountAmountMinor += discountAmount
	item.FinalSubtotalAmountMinor -= discountAmount
	item.AppliedPromotionID = &promotion.PromotionID

	*promoCtx.AppliedPromotions = appendAppliedPromotion(
		*promoCtx.AppliedPromotions,
		promotion,
		discountAmount,
	)

	return http.StatusOK, nil
}

func appendAppliedPromotion(
	items []response.AppliedPromotionValue,
	promotion entity.PromotionEntity,
	discountAmount int64,
) []response.AppliedPromotionValue {
	return append(items, response.AppliedPromotionValue{
		PromotionID:         promotion.PromotionID,
		PromotionName:       promotion.PromotionName,
		PromotionType:       string(promotion.Type),
		DiscountAmountMinor: discountAmount,
	})
}

func (s *checkoutServiceImpl) calculateCheckoutTotals(
	checkoutItems []entity.CheckoutOrderItemEntity,
) (int64, int64, int64) {
	var totalBeforeDiscount int64
	var totalDiscount int64
	var finalTotal int64

	for _, item := range checkoutItems {
		totalBeforeDiscount += item.SubtotalAmountMinor
		totalDiscount += item.DiscountAmountMinor
		finalTotal += item.FinalSubtotalAmountMinor
	}

	return totalBeforeDiscount, totalDiscount, finalTotal
}

func (s *checkoutServiceImpl) persistCheckout(ctx context.Context, checkoutOrder entity.CheckoutOrderEntity, checkoutItems []entity.CheckoutOrderItemEntity, inventoryUpdates map[string]int) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("begin checkout transaction: %w", err)
	}

	committed := false

	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("rollback checkout transaction: %v", rbErr)
			}
		}
	}()

	if err = s.checkoutRepo.InsertCheckoutOrder(ctx, tx, checkoutOrder); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("insert checkout order: %w", err)
	}

	if err = s.checkoutRepo.InsertCheckoutOrderItems(ctx, tx, checkoutItems); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("insert checkout order items: %w", err)
	}

	for sku, quantity := range inventoryUpdates {
		if err = s.productRepo.UpdateInventoryBySKU(ctx, tx, sku, quantity); err != nil {
			return http.StatusBadRequest, fmt.Errorf("update inventory for sku %s: %w", sku, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("commit checkout transaction: %w", err)
	}

	committed = true

	return http.StatusOK, nil
}

func (s *checkoutServiceImpl) buildCheckoutResponse(
	checkoutOrderID uuid.UUID,
	checkoutItems []entity.CheckoutOrderItemEntity,
	appliedPromotions []response.AppliedPromotionValue,
	totalBeforeDiscount int64,
	totalDiscount int64,
	finalTotal int64,
	currency constant.Currency,
) response.CheckoutValue {
	resp := response.CheckoutValue{
		CheckoutOrderID: checkoutOrderID,
		Items:           make([]response.CheckoutItemValue, 0, len(checkoutItems)),

		AppliedPromotions: appliedPromotions,

		TotalBeforeDiscountAmountMinor: totalBeforeDiscount,
		TotalDiscountAmountMinor:       totalDiscount,
		FinalTotalAmountMinor:          finalTotal,
		Currency:                       string(currency),
	}

	for _, item := range checkoutItems {
		resp.Items = append(resp.Items, response.CheckoutItemValue{
			CheckoutOrderItemID:      item.CheckoutOrderItemID,
			ProductID:                item.ProductID,
			SKU:                      item.SKU,
			Name:                     item.Name,
			Quantity:                 item.Quantity,
			UnitPriceAmountMinor:     item.UnitPriceAmountMinor,
			SubtotalAmountMinor:      item.SubtotalAmountMinor,
			DiscountAmountMinor:      item.DiscountAmountMinor,
			FinalSubtotalAmountMinor: item.FinalSubtotalAmountMinor,
			AppliedPromotionID:       item.AppliedPromotionID,
			Currency:                 string(item.Currency),
		})
	}

	return resp
}

func (s *checkoutServiceImpl) appendFreeItemTargetSKUs(
	skus []string,
	requestQtyBySKU map[string]int,
	promotionsWithRules []entity.PromotionWithRuleEntity,
) ([]string, int, error) {
	existsBySKU := make(map[string]bool)

	for _, sku := range skus {
		existsBySKU[sku] = true
	}

	for _, promotionWithRule := range promotionsWithRules {
		promotion := promotionWithRule.Promotion
		rule := promotionWithRule.PromotionRule

		if promotion.Type != entity.PromotionTypeFreeItem {
			continue
		}

		var cfg entity.FreeItemRuleConfig
		if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("parse free item rule config: %w", err)
		}

		triggerQty := requestQtyBySKU[cfg.TriggerSKU]
		if triggerQty <= 0 {
			continue
		}

		if cfg.TargetSKU == "" {
			continue
		}

		if !existsBySKU[cfg.TargetSKU] {
			skus = append(skus, cfg.TargetSKU)
			existsBySKU[cfg.TargetSKU] = true
		}
	}

	return skus, http.StatusOK, nil
}
