package handler

import "net/http"

type CheckoutHandler interface {
	Checkout(w http.ResponseWriter, r *http.Request)
}
