package service

import (
	"checkout-service/app/model/request"
	"checkout-service/app/model/response"
	"context"
)

type CheckoutServiceInterface interface {
	Checkout(ctx context.Context, req request.CheckoutRequest) (response.CheckoutValue, int, error)
}
