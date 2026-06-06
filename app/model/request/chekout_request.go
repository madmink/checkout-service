package request

type CheckoutRequest struct {
	Items []CheckoutItemRequest `json:"items"`
}

type CheckoutItemRequest struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}
