package handler

import "net/http"

type CheckoutHandler interface {
	Checkout(w http.ResponseWriter, r *http.Request)
	HealthCheck(w http.ResponseWriter, r *http.Request)
}
