package e2e

import (
	"bytes"
	"checkout-service/app/model/request"
	appresponse "checkout-service/app/model/response"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	checkoutAPIPath = "/checkout-api/v1/checkout"

	headerContentType = "Content-Type"
	headerRequestID   = "Log-Request-ID"
)

const (
	skuAlexa       = "A304SD"
	skuGoogleHome  = "120P90"
	skuMacBookPro  = "43N23P"
	skuNoPromotion = "234234"
)

type checkoutAPIResponse struct {
	Status     string                       `json:"status"`
	Data       appresponse.CheckoutResponse `json:"data"`
	AccessTime string                       `json:"accessTime"`
}

type checkoutE2EResult struct {
	Response        checkoutAPIResponse
	StatusCode      int
	ResponseHeaders http.Header
}

func TestE2E_Checkout_NoPromotion_ShouldReturnSubtotalAsFinalTotal(t *testing.T) {
	result := doCheckoutE2E(
		t,
		"e2e-no-promotion-001",
		[]request.CheckoutItemRequest{
			{
				SKU:      skuNoPromotion,
				Quantity: 1,
			},
		},
	)

	value := assertCheckoutSuccess(t, result)

	if value.TotalDiscountAmountMinor != 0 {
		t.Fatalf(
			"expected total_discount_amount_minor 0, got %d",
			value.TotalDiscountAmountMinor,
		)
	}

	if value.FinalTotalAmountMinor != value.TotalBeforeDiscountAmountMinor {
		t.Fatalf(
			"expected final_total_amount_minor equals total_before_discount_amount_minor, subtotal: %d, final: %d",
			value.TotalBeforeDiscountAmountMinor,
			value.FinalTotalAmountMinor,
		)
	}

	assertNoAppliedPromotion(t, value)
	assertTotalFormula(t, value)
}

func TestE2E_Checkout_MacBookWithoutRaspberryPi_ShouldAddFreeRaspberryPi(t *testing.T) {
	result := doCheckoutE2E(
		t,
		"e2e-macbook-free-rpi-001",
		[]request.CheckoutItemRequest{
			{
				SKU:      skuMacBookPro,
				Quantity: 1,
			},
		},
	)

	value := assertCheckoutSuccess(t, result)

	macBookItem := assertItemExists(t, value, skuMacBookPro)
	if macBookItem.Quantity != 1 {
		t.Fatalf("expected MacBook quantity 1, got %d", macBookItem.Quantity)
	}

	if len(value.Items) < 2 {
		t.Fatalf(
			"expected at least 2 items because free item should be added, got %d",
			len(value.Items),
		)
	}

	hasFreeItem := false
	for _, item := range value.Items {
		if item.SKU != skuMacBookPro &&
			item.DiscountAmountMinor > 0 &&
			item.FinalSubtotalAmountMinor == 0 {
			hasFreeItem = true
			break
		}
	}

	if !hasFreeItem {
		t.Fatalf("expected one free item with discount > 0 and final_subtotal_amount_minor 0")
	}

	if value.TotalDiscountAmountMinor <= 0 {
		t.Fatalf(
			"expected total_discount_amount_minor > 0, got %d",
			value.TotalDiscountAmountMinor,
		)
	}

	assertHasAppliedPromotion(t, value)
	assertTotalFormula(t, value)
}

func TestE2E_Checkout_GoogleHomeQty3_ShouldApplyBuy3Pay2Promotion(t *testing.T) {
	result := doCheckoutE2E(
		t,
		"e2e-google-home-buy3pay2-001",
		[]request.CheckoutItemRequest{
			{
				SKU:      skuGoogleHome,
				Quantity: 3,
			},
		},
	)

	value := assertCheckoutSuccess(t, result)

	item := assertItemExists(t, value, skuGoogleHome)

	if item.Quantity != 3 {
		t.Fatalf("expected Google Home quantity 3, got %d", item.Quantity)
	}

	expectedDiscount := item.UnitPriceAmountMinor
	if item.DiscountAmountMinor != expectedDiscount {
		t.Fatalf(
			"expected discount equals one unit price %d, got %d",
			expectedDiscount,
			item.DiscountAmountMinor,
		)
	}

	expectedFinalSubtotal := item.UnitPriceAmountMinor * 2
	if item.FinalSubtotalAmountMinor != expectedFinalSubtotal {
		t.Fatalf(
			"expected final subtotal equals two units %d, got %d",
			expectedFinalSubtotal,
			item.FinalSubtotalAmountMinor,
		)
	}

	assertHasAppliedPromotion(t, value)
	assertTotalFormula(t, value)
}

func TestE2E_Checkout_AlexaQty3_ShouldApplyBulkDiscount(t *testing.T) {
	result := doCheckoutE2E(
		t,
		"e2e-alexa-bulk-discount-001",
		[]request.CheckoutItemRequest{
			{
				SKU:      skuAlexa,
				Quantity: 3,
			},
		},
	)

	value := assertCheckoutSuccess(t, result)

	item := assertItemExists(t, value, skuAlexa)

	if item.Quantity != 3 {
		t.Fatalf("expected Alexa quantity 3, got %d", item.Quantity)
	}

	expectedSubtotal := item.UnitPriceAmountMinor * int64(item.Quantity)
	if item.SubtotalAmountMinor != expectedSubtotal {
		t.Fatalf(
			"expected subtotal %d, got %d",
			expectedSubtotal,
			item.SubtotalAmountMinor,
		)
	}

	expectedDiscount := item.SubtotalAmountMinor * 10 / 100
	if item.DiscountAmountMinor != expectedDiscount {
		t.Fatalf(
			"expected 10 percent discount %d, got %d",
			expectedDiscount,
			item.DiscountAmountMinor,
		)
	}

	expectedFinalSubtotal := item.SubtotalAmountMinor - expectedDiscount
	if item.FinalSubtotalAmountMinor != expectedFinalSubtotal {
		t.Fatalf(
			"expected final subtotal %d, got %d",
			expectedFinalSubtotal,
			item.FinalSubtotalAmountMinor,
		)
	}

	assertHasAppliedPromotion(t, value)
	assertTotalFormula(t, value)
}

func TestE2E_Checkout_WithRequestIDHeader_ShouldReturnSameRequestID(t *testing.T) {
	requestID := "e2e-request-id-positive-001"

	result := doCheckoutE2E(
		t,
		requestID,
		[]request.CheckoutItemRequest{
			{
				SKU:      skuAlexa,
				Quantity: 1,
			},
		},
	)

	assertCheckoutSuccess(t, result)
	assertRequestIDHeader(t, result, requestID)
}

func checkoutBaseURL() string {
	baseURL := os.Getenv("CHECKOUT_BASE_URL")
	if baseURL == "" {
		return "http://localhost:4000"
	}

	return baseURL
}

func doCheckoutE2E(
	t *testing.T,
	requestID string,
	items []request.CheckoutItemRequest,
) checkoutE2EResult {
	t.Helper()

	reqBody := request.CheckoutRequest{
		Items: items,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal checkout request: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		checkoutBaseURL()+checkoutAPIPath,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("failed to create checkout request: %v", err)
	}

	req.Header.Set(headerContentType, "application/json")

	if requestID != "" {
		req.Header.Set(headerRequestID, requestID)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to call checkout endpoint: %v", err)
	}
	defer resp.Body.Close()

	var apiResp checkoutAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode checkout response: %v", err)
	}

	return checkoutE2EResult{
		Response:        apiResp,
		StatusCode:      resp.StatusCode,
		ResponseHeaders: resp.Header,
	}
}

func assertCheckoutSuccess(t *testing.T, result checkoutE2EResult) *appresponse.CheckoutValue {
	t.Helper()

	if result.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status code 200, got %d, message: %s",
			result.StatusCode,
			result.Response.Data.Message,
		)
	}

	if !result.Response.Data.IsSuccess {
		t.Fatalf(
			"expected is_success true, got false, message: %s",
			result.Response.Data.Message,
		)
	}

	if result.Response.Data.Value == nil {
		t.Fatalf("expected value to be filled")
	}

	value := result.Response.Data.Value

	if value.CheckoutOrderID == uuid.Nil {
		t.Fatalf("expected checkout_order_id to be filled")
	}

	if value.FinalTotalAmountMinor < 0 {
		t.Fatalf(
			"expected final_total_amount_minor >= 0, got %d",
			value.FinalTotalAmountMinor,
		)
	}

	return value
}

func assertTotalFormula(t *testing.T, value *appresponse.CheckoutValue) {
	t.Helper()

	if value == nil {
		t.Fatalf("expected checkout value to be filled")
	}

	expectedFinalTotal := value.TotalBeforeDiscountAmountMinor - value.TotalDiscountAmountMinor
	if value.FinalTotalAmountMinor != expectedFinalTotal {
		t.Fatalf(
			"expected final total %d, got %d",
			expectedFinalTotal,
			value.FinalTotalAmountMinor,
		)
	}
}

func assertHasAppliedPromotion(t *testing.T, value *appresponse.CheckoutValue) {
	t.Helper()

	if value == nil {
		t.Fatalf("expected checkout value to be filled")
	}

	if len(value.AppliedPromotions) == 0 {
		t.Fatalf("expected applied_promotions to be filled")
	}
}

func assertNoAppliedPromotion(t *testing.T, value *appresponse.CheckoutValue) {
	t.Helper()

	if value == nil {
		t.Fatalf("expected checkout value to be filled")
	}

	if len(value.AppliedPromotions) != 0 {
		t.Fatalf("expected no applied promotions, got %d", len(value.AppliedPromotions))
	}
}

func findItemBySKU(
	items []appresponse.CheckoutItemValue,
	sku string,
) (appresponse.CheckoutItemValue, bool) {
	for _, item := range items {
		if item.SKU == sku {
			return item, true
		}
	}

	return appresponse.CheckoutItemValue{}, false
}

func assertItemExists(
	t *testing.T,
	value *appresponse.CheckoutValue,
	sku string,
) appresponse.CheckoutItemValue {
	t.Helper()

	if value == nil {
		t.Fatalf("expected checkout value to be filled")
	}

	item, found := findItemBySKU(value.Items, sku)
	if !found {
		t.Fatalf("expected item with sku %s to exist", sku)
	}

	return item
}

func assertRequestIDHeader(t *testing.T, result checkoutE2EResult, expectedRequestID string) {
	t.Helper()

	actualRequestID := result.ResponseHeaders.Get(headerRequestID)
	if actualRequestID != expectedRequestID {
		t.Fatalf(
			"expected response request id %s, got %s",
			expectedRequestID,
			actualRequestID,
		)
	}
}
