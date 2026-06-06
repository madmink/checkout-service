package handler

import (
	httpresponse2 "checkout-service/app/controller/handler/httpResponse"
	"checkout-service/app/model/request"
	"checkout-service/app/model/response"
	"checkout-service/app/service"
	"context"
	"encoding/json"
	"net/http"
)

type checkoutHandler struct {
	checkoutService service.CheckoutServiceInterface
}

func NewCheckoutHandler(checkoutService service.CheckoutServiceInterface) CheckoutHandler {
	return &checkoutHandler{
		checkoutService: checkoutService,
	}
}

func (c checkoutHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req request.CheckoutRequest
	var resp response.CheckoutResponse

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "headerReq", req)

		httpresponse2.ErrorHandler(ctx, err, http.StatusBadRequest, "", &resp.GeneralResponse)
		httpresponse2.Response(w, resp, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, "headerReq", req)

	data, status, err := c.checkoutService.Checkout(ctx, req)
	if err != nil {
		httpresponse2.ErrorHandler(ctx, err, status, "", &resp.GeneralResponse)
		httpresponse2.Response(w, resp, status)
		return
	}

	resp.IsSuccess = true
	resp.Value = &data

	httpresponse2.Response(w, resp, status)
}

func (c checkoutHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server is Running!"))
}
