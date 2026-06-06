package service

import (
	"checkout-service/app/model/entity"
	"checkout-service/app/model/request"
	"checkout-service/app/repository"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// TestCheckoutService groups unit tests for checkoutServiceImpl.Checkout
func TestCheckoutService(t *testing.T) {
	t.Run("EmptyItems_ShouldReturnBadRequest", checkoutEmptyItemsShouldReturnBadRequest)
	t.Run("SelectProductsFailed_ShouldReturnInternalServerError", checkoutSelectProductsFailedShouldReturnInternalServerError)
	t.Run("InvalidSKU_ShouldReturnBadRequest", checkoutInvalidSKUShouldReturnBadRequest)
	t.Run("InsufficientProductInventory_ShouldReturnBadRequest", checkoutInsufficientProductInventoryShouldReturnBadRequest)
	t.Run("SelectPromotionsFailed_ShouldReturnInternalServerError", checkoutSelectPromotionsFailedShouldReturnInternalServerError)
	t.Run("MacBookWithFreeRaspberryPi_ShouldReturnCorrectTotal", checkoutMacBookWithFreeRaspberryPiShouldReturnCorrectTotal)
	t.Run("MacBookWithoutRaspberryPiInRequest_ShouldAddFreeRaspberryPi", checkoutMacBookWithoutRaspberryPiInRequestShouldAddFreeRaspberryPi)
	t.Run("GoogleHomeQty3_ShouldReturnBuy3Pay2Total", checkoutGoogleHomeQty3ShouldReturnBuy3Pay2Total)
	t.Run("AlexaQty3_ShouldReturnBulkDiscountTotal", checkoutAlexaQty3ShouldReturnBulkDiscountTotal)
	t.Run("FreeItemInventoryInsufficient_ShouldReturnBadRequest", checkoutFreeItemInventoryInsufficientShouldReturnBadRequest)
	t.Run("InsertCheckoutOrderItemsFailed_ShouldRollback", checkoutInsertCheckoutOrderItemsFailedShouldRollback)
	t.Run("UpdateInventoryFailed_ShouldRollback", checkoutUpdateInventoryFailedShouldRollback)
	t.Run("NoPromotion_ShouldReturnSubtotalAsFinalTotal", checkoutNoPromotionShouldReturnSubtotalAsFinalTotal)
}

func checkoutEmptyItemsShouldReturnBadRequest(t *testing.T) {
	// Arrange
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "checkout items are required") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"checkout items are required",
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %#v", resp.Items)
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}
}

func checkoutSelectProductsFailedShouldReturnInternalServerError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			return nil, expectedErr
		},
	}

	promotionRepo := &mockPromotionRepository{}
	checkoutRepo := &mockCheckoutRepository{}

	svc := newCheckoutServiceForTest(
		productRepo,
		promotionRepo,
		checkoutRepo,
	)

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 3,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, status)
	}

	if !strings.Contains(err.Error(), "select products by skus") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"select products by skus",
			err.Error(),
		)
	}

	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			expectedErr.Error(),
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %#v", resp.Items)
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}
}

func checkoutInvalidSKUShouldReturnBadRequest(t *testing.T) {
	// Arrange
	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			return []entity.ProductEntity{}, nil
		},
	}

	promotionRepo := &mockPromotionRepository{}
	checkoutRepo := &mockCheckoutRepository{}

	svc := newCheckoutServiceForTest(
		productRepo,
		promotionRepo,
		checkoutRepo,
	)

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "INVALID-SKU",
				Quantity: 1,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "one or more sku is invalid or inactive") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"one or more sku is invalid or inactive",
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %#v", resp.Items)
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}
}

func checkoutInsufficientProductInventoryShouldReturnBadRequest(t *testing.T) {
	// Arrange
	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			return []entity.ProductEntity{
				{
					SKU:              "120P90",
					Name:             "Google Home",
					PriceAmountMinor: 4999,
					Currency:         "USD",
					InventoryQty:     2,
					IsActive:         true,
				},
			}, nil
		},
	}

	promotionRepo := &mockPromotionRepository{}
	checkoutRepo := &mockCheckoutRepository{}

	svc := newCheckoutServiceForTest(
		productRepo,
		promotionRepo,
		checkoutRepo,
	)

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 3,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "insufficient inventory for sku 120P90") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"insufficient inventory for sku 120P90",
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %#v", resp.Items)
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}
}

func checkoutSelectPromotionsFailedShouldReturnInternalServerError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("promotion database failed")

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			return []entity.ProductEntity{
				{
					SKU:              "120P90",
					Name:             "Google Home",
					PriceAmountMinor: 4999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return nil, expectedErr
		},
	}

	checkoutRepo := &mockCheckoutRepository{}

	svc := newCheckoutServiceForTest(
		productRepo,
		promotionRepo,
		checkoutRepo,
	)

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 3,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, status)
	}

	if !strings.Contains(err.Error(), "select active promotions with rules") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"select active promotions with rules",
			err.Error(),
		)
	}

	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			expectedErr.Error(),
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %#v", resp.Items)
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}
}

func checkoutMacBookWithFreeRaspberryPiShouldReturnCorrectTotal(t *testing.T) {
	// Arrange
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	macBookProductID := uuid.New()
	raspberryPiProductID := uuid.New()
	promotionID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			return []entity.ProductEntity{
				{
					ProductID:        macBookProductID,
					SKU:              "43N23P",
					Name:             "MacBook Pro",
					PriceAmountMinor: 539999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
				{
					ProductID:        raspberryPiProductID,
					SKU:              "234234",
					Name:             "Raspberry Pi B",
					PriceAmountMinor: 3000,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{
				{
					Promotion: entity.PromotionEntity{
						PromotionID:   promotionID,
						PromotionName: "MacBook Pro Free Raspberry Pi B",
						Type:          entity.PromotionTypeFreeItem,
						IsActive:      true,
					},
					PromotionRule: entity.PromotionRuleEntity{
						PromotionRuleID: uuid.New(),
						PromotionID:     promotionID,
						RuleConfig: []byte(`{
							"trigger_sku": "43N23P",
							"target_sku": "234234",
							"free_qty_per_trigger": 1
						}`),
					},
				},
			}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 542999 {
				t.Fatalf("expected total before discount %d, got %d", 542999, checkoutOrder.TotalBeforeDiscountAmountMinor)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 3000 {
				t.Fatalf("expected total discount %d, got %d", 3000, checkoutOrder.TotalDiscountAmountMinor)
			}

			if checkoutOrder.FinalTotalAmountMinor != 539999 {
				t.Fatalf("expected final total %d, got %d", 539999, checkoutOrder.FinalTotalAmountMinor)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 2 {
				t.Fatalf("expected 2 checkout items, got %d", len(items))
			}

			return nil
		},
	}

	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "43N23P",
				Quantity: 1,
			},
			{
				SKU:      "234234",
				Quantity: 1,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	if resp.TotalBeforeDiscountAmountMinor != 542999 {
		t.Fatalf("expected total before discount %d, got %d", 542999, resp.TotalBeforeDiscountAmountMinor)
	}

	if resp.TotalDiscountAmountMinor != 3000 {
		t.Fatalf("expected total discount %d, got %d", 3000, resp.TotalDiscountAmountMinor)
	}

	if resp.FinalTotalAmountMinor != 539999 {
		t.Fatalf("expected final total %d, got %d", 539999, resp.FinalTotalAmountMinor)
	}

	if resp.Currency != "USD" {
		t.Fatalf("expected currency %q, got %q", "USD", resp.Currency)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 response items, got %d", len(resp.Items))
	}

	if len(resp.AppliedPromotions) != 1 {
		t.Fatalf("expected 1 applied promotion, got %d", len(resp.AppliedPromotions))
	}

	appliedPromotion := resp.AppliedPromotions[0]

	if appliedPromotion.PromotionID != promotionID {
		t.Fatalf("expected promotion id %s, got %s", promotionID, appliedPromotion.PromotionID)
	}

	if appliedPromotion.PromotionType != string(entity.PromotionTypeFreeItem) {
		t.Fatalf("expected promotion type %q, got %q", entity.PromotionTypeFreeItem, appliedPromotion.PromotionType)
	}

	if appliedPromotion.DiscountAmountMinor != 3000 {
		t.Fatalf("expected applied promotion discount %d, got %d", 3000, appliedPromotion.DiscountAmountMinor)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutMacBookWithoutRaspberryPiInRequestShouldAddFreeRaspberryPi(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	macBookProductID := uuid.New()
	raspberryPiProductID := uuid.New()
	promotionID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			if len(skus) != 2 {
				t.Fatalf("expected lookup sku length %d, got %d", 2, len(skus))
			}

			expectedSKUs := map[string]bool{
				"43N23P": true,
				"234234": true,
			}

			for _, sku := range skus {
				if !expectedSKUs[sku] {
					t.Fatalf("unexpected lookup sku %q", sku)
				}
			}

			return []entity.ProductEntity{
				{
					ProductID:        macBookProductID,
					SKU:              "43N23P",
					Name:             "MacBook Pro",
					PriceAmountMinor: 539999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
				{
					ProductID:        raspberryPiProductID,
					SKU:              "234234",
					Name:             "Raspberry Pi B",
					PriceAmountMinor: 3000,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			switch sku {
			case "43N23P":
				if quantity != 1 {
					t.Fatalf("expected MacBook inventory update quantity %d, got %d", 1, quantity)
				}

			case "234234":
				if quantity != 1 {
					t.Fatalf("expected Raspberry Pi inventory update quantity %d, got %d", 1, quantity)
				}

			default:
				t.Fatalf("unexpected inventory update sku: %s", sku)
			}

			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{
				{
					Promotion: entity.PromotionEntity{
						PromotionID:   promotionID,
						PromotionName: "MacBook Pro Free Raspberry Pi B",
						Type:          entity.PromotionTypeFreeItem,
						IsActive:      true,
					},
					PromotionRule: entity.PromotionRuleEntity{
						PromotionRuleID: uuid.New(),
						PromotionID:     promotionID,
						RuleConfig: []byte(`{
							"trigger_sku": "43N23P",
							"target_sku": "234234",
							"free_qty_per_trigger": 1
						}`),
					},
				},
			}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 542999 {
				t.Fatalf(
					"expected total before discount %d, got %d",
					539999,
					checkoutOrder.TotalBeforeDiscountAmountMinor,
				)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 3000 {
				t.Fatalf(
					"expected total discount %d, got %d",
					3000,
					checkoutOrder.TotalDiscountAmountMinor,
				)
			}

			// Current service behavior:
			// MacBook subtotal = 539999
			// Free Raspberry Pi line subtotal = 3000, discount = 3000, final = 0
			// Final total remains 539999.
			if checkoutOrder.FinalTotalAmountMinor != 539999 {
				t.Fatalf(
					"expected final total %d, got %d",
					539999,
					checkoutOrder.FinalTotalAmountMinor,
				)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 2 {
				t.Fatalf("expected 2 checkout items, got %d", len(items))
			}

			var foundMacBook bool
			var foundFreeRaspberryPi bool

			for _, item := range items {
				switch item.SKU {
				case "43N23P":
					foundMacBook = true

					if item.Quantity != 1 {
						t.Fatalf("expected MacBook quantity %d, got %d", 1, item.Quantity)
					}

					if item.FinalSubtotalAmountMinor != 539999 {
						t.Fatalf(
							"expected MacBook final subtotal %d, got %d",
							539999,
							item.FinalSubtotalAmountMinor,
						)
					}

				case "234234":
					foundFreeRaspberryPi = true

					if item.Quantity != 1 {
						t.Fatalf("expected Raspberry Pi quantity %d, got %d", 1, item.Quantity)
					}

					if item.SubtotalAmountMinor != 3000 {
						t.Fatalf(
							"expected Raspberry Pi subtotal %d, got %d",
							3000,
							item.SubtotalAmountMinor,
						)
					}

					if item.DiscountAmountMinor != 3000 {
						t.Fatalf(
							"expected Raspberry Pi discount %d, got %d",
							3000,
							item.DiscountAmountMinor,
						)
					}

					if item.FinalSubtotalAmountMinor != 0 {
						t.Fatalf(
							"expected Raspberry Pi final subtotal %d, got %d",
							0,
							item.FinalSubtotalAmountMinor,
						)
					}

					if item.AppliedPromotionID == nil {
						t.Fatal("expected Raspberry Pi applied promotion id, got nil")
					}
				}
			}

			if !foundMacBook {
				t.Fatal("expected MacBook item to exist")
			}

			if !foundFreeRaspberryPi {
				t.Fatal("expected free Raspberry Pi item to be added")
			}

			return nil
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "43N23P",
				Quantity: 1,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	if resp.TotalBeforeDiscountAmountMinor != 542999 {
		t.Fatalf(
			"expected total before discount %d, got %d",
			542999,
			resp.TotalBeforeDiscountAmountMinor,
		)
	}

	if resp.TotalDiscountAmountMinor != 3000 {
		t.Fatalf(
			"expected total discount %d, got %d",
			3000,
			resp.TotalDiscountAmountMinor,
		)
	}

	if resp.FinalTotalAmountMinor != 539999 {
		t.Fatalf(
			"expected final total %d, got %d",
			539999,
			resp.FinalTotalAmountMinor,
		)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 response items, got %d", len(resp.Items))
	}

	var foundMacBook bool
	var foundFreeRaspberryPi bool

	for _, item := range resp.Items {
		switch item.SKU {
		case "43N23P":
			foundMacBook = true

		case "234234":
			foundFreeRaspberryPi = true

			if item.DiscountAmountMinor != 3000 {
				t.Fatalf(
					"expected Raspberry Pi discount %d, got %d",
					3000,
					item.DiscountAmountMinor,
				)
			}

			if item.FinalSubtotalAmountMinor != 0 {
				t.Fatalf(
					"expected Raspberry Pi final subtotal %d, got %d",
					0,
					item.FinalSubtotalAmountMinor,
				)
			}
		}
	}

	if !foundMacBook {
		t.Fatal("expected MacBook item in response")
	}

	if !foundFreeRaspberryPi {
		t.Fatal("expected free Raspberry Pi item in response")
	}

	if len(resp.AppliedPromotions) != 1 {
		t.Fatalf("expected 1 applied promotion, got %d", len(resp.AppliedPromotions))
	}

	if resp.AppliedPromotions[0].PromotionID != promotionID {
		t.Fatalf(
			"expected promotion id %s, got %s",
			promotionID,
			resp.AppliedPromotions[0].PromotionID,
		)
	}

	if resp.AppliedPromotions[0].PromotionType != string(entity.PromotionTypeFreeItem) {
		t.Fatalf(
			"expected promotion type %q, got %q",
			entity.PromotionTypeFreeItem,
			resp.AppliedPromotions[0].PromotionType,
		)
	}

	if resp.AppliedPromotions[0].DiscountAmountMinor != 3000 {
		t.Fatalf(
			"expected applied promotion discount %d, got %d",
			3000,
			resp.AppliedPromotions[0].DiscountAmountMinor,
		)
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutGoogleHomeQty3ShouldReturnBuy3Pay2Total(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	googleHomeProductID := uuid.New()
	promotionID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			if len(skus) != 1 {
				t.Fatalf("expected 1 sku, got %d", len(skus))
			}

			if skus[0] != "120P90" {
				t.Fatalf("expected sku %q, got %q", "120P90", skus[0])
			}

			return []entity.ProductEntity{
				{
					ProductID:        googleHomeProductID,
					SKU:              "120P90",
					Name:             "Google Home",
					PriceAmountMinor: 4999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if sku != "120P90" {
				t.Fatalf("expected inventory update sku %q, got %q", "120P90", sku)
			}

			if quantity != 3 {
				t.Fatalf("expected inventory update quantity %d, got %d", 3, quantity)
			}

			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{
				{
					Promotion: entity.PromotionEntity{
						PromotionID:   promotionID,
						PromotionName: "Buy 3 Google Homes for the Price of 2",
						Type:          entity.PromotionTypeBuyXPayY,
						IsActive:      true,
					},
					PromotionRule: entity.PromotionRuleEntity{
						PromotionRuleID: uuid.New(),
						PromotionID:     promotionID,
						RuleConfig: []byte(`{
							"target_sku": "120P90",
							"buy_qty": 3,
							"pay_qty": 2
						}`),
					},
				},
			}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 14997 {
				t.Fatalf(
					"expected total before discount %d, got %d",
					14997,
					checkoutOrder.TotalBeforeDiscountAmountMinor,
				)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 4999 {
				t.Fatalf(
					"expected total discount %d, got %d",
					4999,
					checkoutOrder.TotalDiscountAmountMinor,
				)
			}

			if checkoutOrder.FinalTotalAmountMinor != 9998 {
				t.Fatalf(
					"expected final total %d, got %d",
					9998,
					checkoutOrder.FinalTotalAmountMinor,
				)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 checkout item, got %d", len(items))
			}

			item := items[0]

			if item.SKU != "120P90" {
				t.Fatalf("expected item sku %q, got %q", "120P90", item.SKU)
			}

			if item.Quantity != 3 {
				t.Fatalf("expected item quantity %d, got %d", 3, item.Quantity)
			}

			if item.UnitPriceAmountMinor != 4999 {
				t.Fatalf("expected unit price %d, got %d", 4999, item.UnitPriceAmountMinor)
			}

			if item.SubtotalAmountMinor != 14997 {
				t.Fatalf("expected subtotal %d, got %d", 14997, item.SubtotalAmountMinor)
			}

			if item.DiscountAmountMinor != 4999 {
				t.Fatalf("expected discount %d, got %d", 4999, item.DiscountAmountMinor)
			}

			if item.FinalSubtotalAmountMinor != 9998 {
				t.Fatalf("expected final subtotal %d, got %d", 9998, item.FinalSubtotalAmountMinor)
			}

			if item.AppliedPromotionID == nil {
				t.Fatal("expected applied promotion id, got nil")
			}

			if *item.AppliedPromotionID != promotionID {
				t.Fatalf("expected applied promotion id %s, got %s", promotionID, *item.AppliedPromotionID)
			}

			return nil
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 3,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	if resp.TotalBeforeDiscountAmountMinor != 14997 {
		t.Fatalf(
			"expected total before discount %d, got %d",
			14997,
			resp.TotalBeforeDiscountAmountMinor,
		)
	}

	if resp.TotalDiscountAmountMinor != 4999 {
		t.Fatalf(
			"expected total discount %d, got %d",
			4999,
			resp.TotalDiscountAmountMinor,
		)
	}

	if resp.FinalTotalAmountMinor != 9998 {
		t.Fatalf(
			"expected final total %d, got %d",
			9998,
			resp.FinalTotalAmountMinor,
		)
	}

	if resp.Currency != "USD" {
		t.Fatalf("expected currency %q, got %q", "USD", resp.Currency)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 response item, got %d", len(resp.Items))
	}

	item := resp.Items[0]

	if item.SKU != "120P90" {
		t.Fatalf("expected response item sku %q, got %q", "120P90", item.SKU)
	}

	if item.Quantity != 3 {
		t.Fatalf("expected response item quantity %d, got %d", 3, item.Quantity)
	}

	if item.SubtotalAmountMinor != 14997 {
		t.Fatalf("expected response subtotal %d, got %d", 14997, item.SubtotalAmountMinor)
	}

	if item.DiscountAmountMinor != 4999 {
		t.Fatalf("expected response discount %d, got %d", 4999, item.DiscountAmountMinor)
	}

	if item.FinalSubtotalAmountMinor != 9998 {
		t.Fatalf("expected response final subtotal %d, got %d", 9998, item.FinalSubtotalAmountMinor)
	}

	if item.AppliedPromotionID == nil {
		t.Fatal("expected response applied promotion id, got nil")
	}

	if *item.AppliedPromotionID != promotionID {
		t.Fatalf("expected response applied promotion id %s, got %s", promotionID, *item.AppliedPromotionID)
	}

	if len(resp.AppliedPromotions) != 1 {
		t.Fatalf("expected 1 applied promotion, got %d", len(resp.AppliedPromotions))
	}

	appliedPromotion := resp.AppliedPromotions[0]

	if appliedPromotion.PromotionID != promotionID {
		t.Fatalf("expected promotion id %s, got %s", promotionID, appliedPromotion.PromotionID)
	}

	if appliedPromotion.PromotionName != "Buy 3 Google Homes for the Price of 2" {
		t.Fatalf(
			"expected promotion name %q, got %q",
			"Buy 3 Google Homes for the Price of 2",
			appliedPromotion.PromotionName,
		)
	}

	if appliedPromotion.PromotionType != string(entity.PromotionTypeBuyXPayY) {
		t.Fatalf(
			"expected promotion type %q, got %q",
			entity.PromotionTypeBuyXPayY,
			appliedPromotion.PromotionType,
		)
	}

	if appliedPromotion.DiscountAmountMinor != 4999 {
		t.Fatalf(
			"expected applied promotion discount %d, got %d",
			4999,
			appliedPromotion.DiscountAmountMinor,
		)
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutAlexaQty3ShouldReturnBulkDiscountTotal(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	alexaProductID := uuid.New()
	promotionID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			if len(skus) != 1 {
				t.Fatalf("expected 1 sku, got %d", len(skus))
			}

			if skus[0] != "A304SD" {
				t.Fatalf("expected sku %q, got %q", "A304SD", skus[0])
			}

			return []entity.ProductEntity{
				{
					ProductID:        alexaProductID,
					SKU:              "A304SD",
					Name:             "Alexa Speaker",
					PriceAmountMinor: 10950,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if sku != "A304SD" {
				t.Fatalf("expected inventory update sku %q, got %q", "A304SD", sku)
			}

			if quantity != 3 {
				t.Fatalf("expected inventory update quantity %d, got %d", 3, quantity)
			}

			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{
				{
					Promotion: entity.PromotionEntity{
						PromotionID:   promotionID,
						PromotionName: "Alexa Speaker Bulk 10 Percent Discount",
						Type:          entity.PromotionTypeBulkPercentage,
						IsActive:      true,
					},
					PromotionRule: entity.PromotionRuleEntity{
						PromotionRuleID: uuid.New(),
						PromotionID:     promotionID,
						RuleConfig: []byte(`{
							"target_sku": "A304SD",
							"min_qty": 3,
							"discount_percentage": 10
						}`),
					},
				},
			}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 32850 {
				t.Fatalf(
					"expected total before discount %d, got %d",
					32850,
					checkoutOrder.TotalBeforeDiscountAmountMinor,
				)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 3285 {
				t.Fatalf(
					"expected total discount %d, got %d",
					3285,
					checkoutOrder.TotalDiscountAmountMinor,
				)
			}

			if checkoutOrder.FinalTotalAmountMinor != 29565 {
				t.Fatalf(
					"expected final total %d, got %d",
					29565,
					checkoutOrder.FinalTotalAmountMinor,
				)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 checkout item, got %d", len(items))
			}

			item := items[0]

			if item.SKU != "A304SD" {
				t.Fatalf("expected item sku %q, got %q", "A304SD", item.SKU)
			}

			if item.Quantity != 3 {
				t.Fatalf("expected item quantity %d, got %d", 3, item.Quantity)
			}

			if item.UnitPriceAmountMinor != 10950 {
				t.Fatalf("expected unit price %d, got %d", 10950, item.UnitPriceAmountMinor)
			}

			if item.SubtotalAmountMinor != 32850 {
				t.Fatalf("expected subtotal %d, got %d", 32850, item.SubtotalAmountMinor)
			}

			if item.DiscountAmountMinor != 3285 {
				t.Fatalf("expected discount %d, got %d", 3285, item.DiscountAmountMinor)
			}

			if item.FinalSubtotalAmountMinor != 29565 {
				t.Fatalf("expected final subtotal %d, got %d", 29565, item.FinalSubtotalAmountMinor)
			}

			if item.AppliedPromotionID == nil {
				t.Fatal("expected applied promotion id, got nil")
			}

			if *item.AppliedPromotionID != promotionID {
				t.Fatalf("expected applied promotion id %s, got %s", promotionID, *item.AppliedPromotionID)
			}

			return nil
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "A304SD",
				Quantity: 3,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	if resp.TotalBeforeDiscountAmountMinor != 32850 {
		t.Fatalf(
			"expected total before discount %d, got %d",
			32850,
			resp.TotalBeforeDiscountAmountMinor,
		)
	}

	if resp.TotalDiscountAmountMinor != 3285 {
		t.Fatalf(
			"expected total discount %d, got %d",
			3285,
			resp.TotalDiscountAmountMinor,
		)
	}

	if resp.FinalTotalAmountMinor != 29565 {
		t.Fatalf(
			"expected final total %d, got %d",
			29565,
			resp.FinalTotalAmountMinor,
		)
	}

	if resp.Currency != "USD" {
		t.Fatalf("expected currency %q, got %q", "USD", resp.Currency)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 response item, got %d", len(resp.Items))
	}

	item := resp.Items[0]

	if item.SKU != "A304SD" {
		t.Fatalf("expected response item sku %q, got %q", "A304SD", item.SKU)
	}

	if item.Quantity != 3 {
		t.Fatalf("expected response item quantity %d, got %d", 3, item.Quantity)
	}

	if item.SubtotalAmountMinor != 32850 {
		t.Fatalf("expected response subtotal %d, got %d", 32850, item.SubtotalAmountMinor)
	}

	if item.DiscountAmountMinor != 3285 {
		t.Fatalf("expected response discount %d, got %d", 3285, item.DiscountAmountMinor)
	}

	if item.FinalSubtotalAmountMinor != 29565 {
		t.Fatalf("expected response final subtotal %d, got %d", 29565, item.FinalSubtotalAmountMinor)
	}

	if item.AppliedPromotionID == nil {
		t.Fatal("expected response applied promotion id, got nil")
	}

	if *item.AppliedPromotionID != promotionID {
		t.Fatalf("expected response applied promotion id %s, got %s", promotionID, *item.AppliedPromotionID)
	}

	if len(resp.AppliedPromotions) != 1 {
		t.Fatalf("expected 1 applied promotion, got %d", len(resp.AppliedPromotions))
	}

	appliedPromotion := resp.AppliedPromotions[0]

	if appliedPromotion.PromotionID != promotionID {
		t.Fatalf("expected promotion id %s, got %s", promotionID, appliedPromotion.PromotionID)
	}

	if appliedPromotion.PromotionName != "Alexa Speaker Bulk 10 Percent Discount" {
		t.Fatalf(
			"expected promotion name %q, got %q",
			"Alexa Speaker Bulk 10 Percent Discount",
			appliedPromotion.PromotionName,
		)
	}

	if appliedPromotion.PromotionType != string(entity.PromotionTypeBulkPercentage) {
		t.Fatalf(
			"expected promotion type %q, got %q",
			entity.PromotionTypeBulkPercentage,
			appliedPromotion.PromotionType,
		)
	}

	if appliedPromotion.DiscountAmountMinor != 3285 {
		t.Fatalf(
			"expected applied promotion discount %d, got %d",
			3285,
			appliedPromotion.DiscountAmountMinor,
		)
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutFreeItemInventoryInsufficientShouldReturnBadRequest(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	macBookProductID := uuid.New()
	raspberryPiProductID := uuid.New()
	promotionID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			return []entity.ProductEntity{
				{
					ProductID:        macBookProductID,
					SKU:              "43N23P",
					Name:             "MacBook Pro",
					PriceAmountMinor: 539999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
				{
					ProductID:        raspberryPiProductID,
					SKU:              "234234",
					Name:             "Raspberry Pi B",
					PriceAmountMinor: 3000,
					Currency:         "USD",
					InventoryQty:     1,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if sku == "234234" && quantity == 2 {
				return errors.New("insufficient inventory for free item sku 234234")
			}

			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{
				{
					Promotion: entity.PromotionEntity{
						PromotionID:   promotionID,
						PromotionName: "MacBook Pro Free Raspberry Pi B",
						Type:          entity.PromotionTypeFreeItem,
						IsActive:      true,
					},
					PromotionRule: entity.PromotionRuleEntity{
						PromotionRuleID: uuid.New(),
						PromotionID:     promotionID,
						RuleConfig: []byte(`{
							"trigger_sku": "43N23P",
							"target_sku": "234234",
							"free_qty_per_trigger": 1
						}`),
					},
				},
			}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 2 {
				t.Fatalf("expected 2 checkout items, got %d", len(items))
			}

			return nil
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "43N23P",
				Quantity: 1,
			},
			{
				SKU:      "234234",
				Quantity: 1,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "update inventory for sku 234234") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"update inventory for sku 234234",
			err.Error(),
		)
	}

	if !strings.Contains(err.Error(), "insufficient inventory for free item sku 234234") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"insufficient inventory for free item sku 234234",
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %d", len(resp.Items))
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutInsertCheckoutOrderItemsFailedShouldRollback(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	expectedErr := errors.New("insert checkout order items failed")

	googleHomeProductID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			if len(skus) != 1 {
				t.Fatalf("expected lookup sku length %d, got %d", 1, len(skus))
			}

			if skus[0] != "120P90" {
				t.Fatalf("expected lookup sku %q, got %q", "120P90", skus[0])
			}

			return []entity.ProductEntity{
				{
					ProductID:        googleHomeProductID,
					SKU:              "120P90",
					Name:             "Google Home",
					PriceAmountMinor: 4999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			t.Fatal("UpdateInventoryBySKU should not be called when InsertCheckoutOrderItems fails")
			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 4999 {
				t.Fatalf(
					"expected total before discount %d, got %d",
					4999,
					checkoutOrder.TotalBeforeDiscountAmountMinor,
				)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 0 {
				t.Fatalf(
					"expected total discount %d, got %d",
					0,
					checkoutOrder.TotalDiscountAmountMinor,
				)
			}

			if checkoutOrder.FinalTotalAmountMinor != 4999 {
				t.Fatalf(
					"expected final total %d, got %d",
					4999,
					checkoutOrder.FinalTotalAmountMinor,
				)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 checkout item, got %d", len(items))
			}

			item := items[0]

			if item.SKU != "120P90" {
				t.Fatalf("expected item sku %q, got %q", "120P90", item.SKU)
			}

			if item.Quantity != 1 {
				t.Fatalf("expected item quantity %d, got %d", 1, item.Quantity)
			}

			if item.SubtotalAmountMinor != 4999 {
				t.Fatalf("expected item subtotal %d, got %d", 4999, item.SubtotalAmountMinor)
			}

			if item.FinalSubtotalAmountMinor != 4999 {
				t.Fatalf("expected item final subtotal %d, got %d", 4999, item.FinalSubtotalAmountMinor)
			}

			return expectedErr
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 1,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, status)
	}

	if !strings.Contains(err.Error(), "insert checkout order items") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"insert checkout order items",
			err.Error(),
		)
	}

	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			expectedErr.Error(),
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %d", len(resp.Items))
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutUpdateInventoryFailedShouldRollback(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	expectedErr := errors.New("insufficient inventory during update")

	googleHomeProductID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			if len(skus) != 1 {
				t.Fatalf("expected lookup sku length %d, got %d", 1, len(skus))
			}

			if skus[0] != "120P90" {
				t.Fatalf("expected lookup sku %q, got %q", "120P90", skus[0])
			}

			return []entity.ProductEntity{
				{
					ProductID:        googleHomeProductID,
					SKU:              "120P90",
					Name:             "Google Home",
					PriceAmountMinor: 4999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if sku != "120P90" {
				t.Fatalf("expected inventory update sku %q, got %q", "120P90", sku)
			}

			if quantity != 1 {
				t.Fatalf("expected inventory update quantity %d, got %d", 1, quantity)
			}

			return expectedErr
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 4999 {
				t.Fatalf(
					"expected total before discount %d, got %d",
					4999,
					checkoutOrder.TotalBeforeDiscountAmountMinor,
				)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 0 {
				t.Fatalf(
					"expected total discount %d, got %d",
					0,
					checkoutOrder.TotalDiscountAmountMinor,
				)
			}

			if checkoutOrder.FinalTotalAmountMinor != 4999 {
				t.Fatalf(
					"expected final total %d, got %d",
					4999,
					checkoutOrder.FinalTotalAmountMinor,
				)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 checkout item, got %d", len(items))
			}

			item := items[0]

			if item.SKU != "120P90" {
				t.Fatalf("expected item sku %q, got %q", "120P90", item.SKU)
			}

			if item.Quantity != 1 {
				t.Fatalf("expected item quantity %d, got %d", 1, item.Quantity)
			}

			if item.SubtotalAmountMinor != 4999 {
				t.Fatalf("expected item subtotal %d, got %d", 4999, item.SubtotalAmountMinor)
			}

			if item.FinalSubtotalAmountMinor != 4999 {
				t.Fatalf("expected item final subtotal %d, got %d", 4999, item.FinalSubtotalAmountMinor)
			}

			return nil
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 1,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "update inventory for sku 120P90") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"update inventory for sku 120P90",
			err.Error(),
		)
	}

	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			expectedErr.Error(),
			err.Error(),
		)
	}

	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response items, got %d", len(resp.Items))
	}

	if resp.FinalTotalAmountMinor != 0 {
		t.Fatalf("expected final total 0, got %d", resp.FinalTotalAmountMinor)
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

func checkoutNoPromotionShouldReturnSubtotalAsFinalTotal(t *testing.T) {
	// Arrange
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected no sqlmock error, got %v", err)
	}
	defer db.Close()

	googleHomeProductID := uuid.New()

	productRepo := &mockProductRepository{
		selectProductsBySKUsFn: func(ctx context.Context, skus []string) ([]entity.ProductEntity, error) {
			if len(skus) != 1 {
				t.Fatalf("expected lookup sku length %d, got %d", 1, len(skus))
			}

			if skus[0] != "120P90" {
				t.Fatalf("expected lookup sku %q, got %q", "120P90", skus[0])
			}

			return []entity.ProductEntity{
				{
					ProductID:        googleHomeProductID,
					SKU:              "120P90",
					Name:             "Google Home",
					PriceAmountMinor: 4999,
					Currency:         "USD",
					InventoryQty:     10,
					IsActive:         true,
				},
			}, nil
		},
		updateInventoryBySKUFn: func(ctx context.Context, exec repository.DBExecutor, sku string, quantity int) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if sku != "120P90" {
				t.Fatalf("expected inventory update sku %q, got %q", "120P90", sku)
			}

			if quantity != 2 {
				t.Fatalf("expected inventory update quantity %d, got %d", 2, quantity)
			}

			return nil
		},
	}

	promotionRepo := &mockPromotionRepository{
		selectActivePromotionsWithRulesFn: func(ctx context.Context) ([]entity.PromotionWithRuleEntity, error) {
			return []entity.PromotionWithRuleEntity{}, nil
		},
	}

	checkoutRepo := &mockCheckoutRepository{
		insertCheckoutOrderFn: func(ctx context.Context, exec repository.DBExecutor, checkoutOrder entity.CheckoutOrderEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if checkoutOrder.TotalBeforeDiscountAmountMinor != 9998 {
				t.Fatalf(
					"expected total before discount %d, got %d",
					9998,
					checkoutOrder.TotalBeforeDiscountAmountMinor,
				)
			}

			if checkoutOrder.TotalDiscountAmountMinor != 0 {
				t.Fatalf(
					"expected total discount %d, got %d",
					0,
					checkoutOrder.TotalDiscountAmountMinor,
				)
			}

			if checkoutOrder.FinalTotalAmountMinor != 9998 {
				t.Fatalf(
					"expected final total %d, got %d",
					9998,
					checkoutOrder.FinalTotalAmountMinor,
				)
			}

			return nil
		},
		insertCheckoutOrderItemsFn: func(ctx context.Context, exec repository.DBExecutor, items []entity.CheckoutOrderItemEntity) error {
			if exec == nil {
				t.Fatal("expected transaction executor, got nil")
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 checkout item, got %d", len(items))
			}

			item := items[0]

			if item.SKU != "120P90" {
				t.Fatalf("expected item sku %q, got %q", "120P90", item.SKU)
			}

			if item.Quantity != 2 {
				t.Fatalf("expected item quantity %d, got %d", 2, item.Quantity)
			}

			if item.UnitPriceAmountMinor != 4999 {
				t.Fatalf("expected unit price %d, got %d", 4999, item.UnitPriceAmountMinor)
			}

			if item.SubtotalAmountMinor != 9998 {
				t.Fatalf("expected subtotal %d, got %d", 9998, item.SubtotalAmountMinor)
			}

			if item.DiscountAmountMinor != 0 {
				t.Fatalf("expected discount %d, got %d", 0, item.DiscountAmountMinor)
			}

			if item.FinalSubtotalAmountMinor != 9998 {
				t.Fatalf("expected final subtotal %d, got %d", 9998, item.FinalSubtotalAmountMinor)
			}

			if item.AppliedPromotionID != nil {
				t.Fatalf("expected applied promotion id to be nil, got %v", item.AppliedPromotionID)
			}

			return nil
		},
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	svc := &checkoutServiceImpl{
		db:            db,
		productRepo:   productRepo,
		promotionRepo: promotionRepo,
		checkoutRepo:  checkoutRepo,
	}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 2,
			},
		},
	}

	// Act
	resp, status, err := svc.Checkout(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	if resp.TotalBeforeDiscountAmountMinor != 9998 {
		t.Fatalf(
			"expected total before discount %d, got %d",
			9998,
			resp.TotalBeforeDiscountAmountMinor,
		)
	}

	if resp.TotalDiscountAmountMinor != 0 {
		t.Fatalf(
			"expected total discount %d, got %d",
			0,
			resp.TotalDiscountAmountMinor,
		)
	}

	if resp.FinalTotalAmountMinor != 9998 {
		t.Fatalf(
			"expected final total %d, got %d",
			9998,
			resp.FinalTotalAmountMinor,
		)
	}

	if resp.Currency != "USD" {
		t.Fatalf("expected currency %q, got %q", "USD", resp.Currency)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 response item, got %d", len(resp.Items))
	}

	item := resp.Items[0]

	if item.SKU != "120P90" {
		t.Fatalf("expected response item sku %q, got %q", "120P90", item.SKU)
	}

	if item.Quantity != 2 {
		t.Fatalf("expected response item quantity %d, got %d", 2, item.Quantity)
	}

	if item.SubtotalAmountMinor != 9998 {
		t.Fatalf("expected response subtotal %d, got %d", 9998, item.SubtotalAmountMinor)
	}

	if item.DiscountAmountMinor != 0 {
		t.Fatalf("expected response discount %d, got %d", 0, item.DiscountAmountMinor)
	}

	if item.FinalSubtotalAmountMinor != 9998 {
		t.Fatalf("expected response final subtotal %d, got %d", 9998, item.FinalSubtotalAmountMinor)
	}

	if item.AppliedPromotionID != nil {
		t.Fatalf("expected response applied promotion id to be nil, got %v", item.AppliedPromotionID)
	}

	if len(resp.AppliedPromotions) != 0 {
		t.Fatalf("expected 0 applied promotions, got %d", len(resp.AppliedPromotions))
	}

	if err = sqlMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unmet sqlmock expectations: %v", err)
	}
}

// All test case for the function validateAndAggregateCheckoutRequest
func TestValidateAndAggregateCheckoutRequest(t *testing.T) {
	t.Run("EmptyItems_ShouldReturnBadRequest", validateAndAggregateCheckoutRequestEmptyItemsShouldReturnBadRequest)
	t.Run("EmptySKU_ShouldReturnBadRequest", validateAndAggregateCheckoutRequestEmptySKUShouldReturnBadRequest)
	t.Run("ZeroQuantity_ShouldReturnBadRequest", validateAndAggregateCheckoutRequestZeroQuantityShouldReturnBadRequest)
	t.Run("NegativeQuantity_ShouldReturnBadRequest", validateAndAggregateCheckoutRequestNegativeQuantityShouldReturnBadRequest)
	t.Run("DuplicateSKU_ShouldAggregateQuantity", validateAndAggregateCheckoutRequestDuplicateSKUShouldAggregateQuantity)
	t.Run("SingleValidItem_ShouldReturnOK", validateAndAggregateCheckoutRequestSingleValidItemShouldReturnOK)
	t.Run("ValidRequest_ShouldReturnSKUAndQuantityMap", validateAndAggregateCheckoutRequestValidRequestShouldReturnSKUAndQuantityMap)
}

func validateAndAggregateCheckoutRequestEmptyItemsShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "checkout items are required") {
		t.Fatalf("expected error message to contain %q, got %q", "checkout items are required", err.Error())
	}

	if requestQtyBySKU != nil {
		t.Fatalf("expected requestQtyBySKU to be nil, got %#v", requestQtyBySKU)
	}

	if skus != nil {
		t.Fatalf("expected skus to be nil, got %#v", skus)
	}
}

func validateAndAggregateCheckoutRequestEmptySKUShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "",
				Quantity: 1,
			},
		},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "sku is required") {
		t.Fatalf("expected error message to contain %q, got %q", "sku is required", err.Error())
	}

	if requestQtyBySKU != nil {
		t.Fatalf("expected requestQtyBySKU to be nil, got %#v", requestQtyBySKU)
	}

	if skus != nil {
		t.Fatalf("expected skus to be nil, got %#v", skus)
	}
}

func validateAndAggregateCheckoutRequestZeroQuantityShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 0,
			},
		},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	expectedErr := "quantity must be greater than zero for sku 120P90"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error message to contain %q, got %q", expectedErr, err.Error())
	}

	if requestQtyBySKU != nil {
		t.Fatalf("expected requestQtyBySKU to be nil, got %#v", requestQtyBySKU)
	}

	if skus != nil {
		t.Fatalf("expected skus to be nil, got %#v", skus)
	}
}

func validateAndAggregateCheckoutRequestNegativeQuantityShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: -1,
			},
		},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	expectedErr := "quantity must be greater than zero for sku 120P90"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error message to contain %q, got %q", expectedErr, err.Error())
	}

	if requestQtyBySKU != nil {
		t.Fatalf("expected requestQtyBySKU to be nil, got %#v", requestQtyBySKU)
	}

	if skus != nil {
		t.Fatalf("expected skus to be nil, got %#v", skus)
	}
}

func validateAndAggregateCheckoutRequestDuplicateSKUShouldAggregateQuantity(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 1,
			},
			{
				SKU:      "120P90",
				Quantity: 2,
			},
		},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	assertSKUs(t, skus, []string{"120P90"})
	assertQtyBySKU(t, requestQtyBySKU, map[string]int{
		"120P90": 3,
	})
}

func validateAndAggregateCheckoutRequestSingleValidItemShouldReturnOK(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 3,
			},
		},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	assertSKUs(t, skus, []string{"120P90"})
	assertQtyBySKU(t, requestQtyBySKU, map[string]int{
		"120P90": 3,
	})
}

func validateAndAggregateCheckoutRequestValidRequestShouldReturnSKUAndQuantityMap(t *testing.T) {
	svc := &checkoutServiceImpl{}

	req := request.CheckoutRequest{
		Items: []request.CheckoutItemRequest{
			{
				SKU:      "120P90",
				Quantity: 3,
			},
			{
				SKU:      "43N23P",
				Quantity: 1,
			},
			{
				SKU:      "234234",
				Quantity: 2,
			},
		},
	}

	requestQtyBySKU, skus, status, err := svc.validateAndAggregateCheckoutRequest(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	assertSKUs(t, skus, []string{"120P90", "43N23P", "234234"})
	assertQtyBySKU(t, requestQtyBySKU, map[string]int{
		"120P90": 3,
		"43N23P": 1,
		"234234": 2,
	})
}

// All test case fir the function validateProducts
func TestValidateProducts(t *testing.T) {
	t.Run("ProductCountMismatch_ShouldReturnBadRequest", validateProductsProductCountMismatchShouldReturnBadRequest)
	t.Run("MixedCurrency_ShouldReturnBadRequest", validateProductsMixedCurrencyShouldReturnBadRequest)
	t.Run("InsufficientInventory_ShouldReturnBadRequest", validateProductsInsufficientInventoryShouldReturnBadRequest)
	t.Run("ValidProducts_ShouldReturnProductMapAndCurrency", validateProductsValidProductsShouldReturnProductMapAndCurrency)

}

func validateProductsProductCountMismatchShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	products := []entity.ProductEntity{
		{
			SKU:          "120P90",
			InventoryQty: 10,
		},
	}

	skus := []string{
		"120P90",
		"43N23P",
	}

	requestQtyBySKU := map[string]int{
		"120P90": 3,
		"43N23P": 1,
	}

	productBySKU, currency, status, err := svc.validateProducts(
		products,
		skus,
		requestQtyBySKU,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "one or more sku is invalid or inactive") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"one or more sku is invalid or inactive",
			err.Error(),
		)
	}

	if productBySKU != nil {
		t.Fatalf("expected productBySKU to be nil, got %#v", productBySKU)
	}

	if currency != "" {
		t.Fatalf("expected currency to be empty, got %q", currency)
	}
}

func validateProductsMixedCurrencyShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	products := []entity.ProductEntity{
		{
			SKU:          "120P90",
			Currency:     "USD",
			InventoryQty: 10,
		},
		{
			SKU:          "43N23P",
			Currency:     "IDR",
			InventoryQty: 10,
		},
	}

	skus := []string{
		"120P90",
		"43N23P",
	}

	requestQtyBySKU := map[string]int{
		"120P90": 3,
		"43N23P": 1,
	}

	productBySKU, currency, status, err := svc.validateProducts(
		products,
		skus,
		requestQtyBySKU,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "mixed currency checkout is not allowed") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"mixed currency checkout is not allowed",
			err.Error(),
		)
	}

	if productBySKU != nil {
		t.Fatalf("expected productBySKU to be nil, got %#v", productBySKU)
	}

	if currency != "" {
		t.Fatalf("expected currency to be empty, got %q", currency)
	}
}

func validateProductsInsufficientInventoryShouldReturnBadRequest(t *testing.T) {
	svc := &checkoutServiceImpl{}

	products := []entity.ProductEntity{
		{
			SKU:          "120P90",
			Currency:     "USD",
			InventoryQty: 2,
		},
	}

	skus := []string{
		"120P90",
	}

	requestQtyBySKU := map[string]int{
		"120P90": 3,
	}

	productBySKU, currency, status, err := svc.validateProducts(
		products,
		skus,
		requestQtyBySKU,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}

	if !strings.Contains(err.Error(), "insufficient inventory for sku 120P90") {
		t.Fatalf(
			"expected error message to contain %q, got %q",
			"insufficient inventory for sku 120P90",
			err.Error(),
		)
	}

	if productBySKU != nil {
		t.Fatalf("expected productBySKU to be nil, got %#v", productBySKU)
	}

	if currency != "" {
		t.Fatalf("expected currency to be empty, got %q", currency)
	}
}

func validateProductsValidProductsShouldReturnProductMapAndCurrency(t *testing.T) {
	svc := &checkoutServiceImpl{}

	products := []entity.ProductEntity{
		{
			SKU:          "120P90",
			Name:         "Google Home",
			Currency:     "USD",
			InventoryQty: 10,
		},
		{
			SKU:          "43N23P",
			Name:         "MacBook Pro",
			Currency:     "USD",
			InventoryQty: 5,
		},
	}

	skus := []string{
		"120P90",
		"43N23P",
	}

	requestQtyBySKU := map[string]int{
		"120P90": 3,
		"43N23P": 1,
	}

	productBySKU, currency, status, err := svc.validateProducts(
		products,
		skus,
		requestQtyBySKU,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	if currency != "USD" {
		t.Fatalf("expected currency %q, got %q", "USD", currency)
	}

	if len(productBySKU) != 2 {
		t.Fatalf("expected productBySKU length %d, got %d", 2, len(productBySKU))
	}

	googleHome, exists := productBySKU["120P90"]
	if !exists {
		t.Fatal("expected productBySKU to contain sku 120P90")
	}

	if googleHome.Name != "Google Home" {
		t.Fatalf("expected product name %q, got %q", "Google Home", googleHome.Name)
	}

	macBook, exists := productBySKU["43N23P"]
	if !exists {
		t.Fatal("expected productBySKU to contain sku 43N23P")
	}

	if macBook.Name != "MacBook Pro" {
		t.Fatalf("expected product name %q, got %q", "MacBook Pro", macBook.Name)
	}
}

func assertSKUs(t *testing.T, actual []string, expected []string) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected %d skus, got %d", len(expected), len(actual))
	}

	for i, expectedSKU := range expected {
		if actual[i] != expectedSKU {
			t.Fatalf("expected sku at index %d to be %q, got %q", i, expectedSKU, actual[i])
		}
	}
}

func assertQtyBySKU(t *testing.T, actual map[string]int, expected map[string]int) {
	t.Helper()

	for sku, expectedQty := range expected {
		if actual[sku] != expectedQty {
			t.Fatalf("expected quantity for sku %s to be %d, got %d", sku, expectedQty, actual[sku])
		}
	}
}
