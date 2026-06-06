package controller

import (
	"checkout-service/app/controller/handler"
	"checkout-service/app/controller/handler/middleware"
	"checkout-service/config"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type Controller struct {
	checkoutHandler handler.CheckoutHandler
	cfg             *config.ApplicationConfig
	env             string
}

func NewController(
	checkoutHandler handler.CheckoutHandler,
	cfg *config.ApplicationConfig,
	env string,
) *Controller {
	return &Controller{
		checkoutHandler: checkoutHandler,
		cfg:             cfg,
		env:             env,
	}
}

func (c *Controller) StartRoute(app *newrelic.Application) *http.Server {
	r := chi.NewRouter()

	r.Use(middleware.Middleware(app))
	r.Get("/health", c.checkoutHandler.HealthCheck)

	r.Route("/checkout-api/v1", func(r chi.Router) {
		r.Use(middleware.GetLogMiddleware(*c.cfg))
		r.Post("/checkout", c.checkoutHandler.Checkout)
	})

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf(":%d", c.cfg.Port.Service),
		WriteTimeout: c.cfg.Port.ServiceTimeout * time.Second,
		ReadTimeout:  c.cfg.Port.ServiceTimeout * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv
}
