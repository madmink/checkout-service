package main

import (
	"checkout-service/app/controller"
	apihandler "checkout-service/app/controller/handler"
	"checkout-service/app/repository"
	"checkout-service/app/service"
	"checkout-service/config"
	"checkout-service/log"
	"context"
	"flag"
	syslog "log"
	"os"
	"os/signal"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	var env string
	flag.StringVar(&env, "env", env, "environment level")
	flag.Parse()

	cfg, err := config.ReadConfig(env)
	if err != nil {
		syslog.Fatalf("Error loading config (%v): %v\n", env, err)
	}

	// =========================
	// New Relic Init
	// =========================
	nrApp := initNewRelic(cfg)
	if nrApp != nil {
		defer nrApp.Shutdown(10 * time.Second)
	}

	// =========================
	// Logging Init
	// =========================
	logsCfg := initNewRelicLogConfig(cfg)
	log.InitLog(nrApp, logsCfg)

	// =========================
	// Monitoring Init
	// =========================
	config.InitializeProfiler(cfg)

	// =========================
	// DB Init
	// =========================
	database, err := config.InitDBConnection(cfg)
	if err != nil {
		log.Logging.Error.Fatalf("Error connecting to DB: %v\n", err)
	}

	// =========================
	// Init Repository layer along with its raw connection because of the transaction intent
	// =========================
	productRepo := repository.NewProductRepositoryImpl(cfg.Database["checkout_database"], database["checkout_database"])
	promotionRepo := repository.NewPromotionRepositoryImpl(cfg.Database["checkout_database"], database["checkout_database"])
	checkoutRepo := repository.NewCheckoutRepositoryImpl(cfg.Database["checkout_database"], database["checkout_database"])

	// =========================
	// Init Service Layer
	// =========================
	checkoutService := service.NewCheckoutServiceImpl(
		cfg,
		database["checkout_database"],
		productRepo,
		promotionRepo,
		checkoutRepo,
	)

	// =========================
	// Init Handler
	// =========================
	checkoutHandler := apihandler.NewCheckoutHandler(checkoutService)

	log.Logging.Access.Infof("Application is running, Port : %v", cfg.Port.Service)
	rest := controller.NewController(checkoutHandler, cfg, env).StartRoute(nrApp)

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Logging.Access.Infof("Shutdown Application ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rest.Shutdown(ctx); err != nil {
		log.Logging.Error.Errorf("Server Shutdown: %v", err)
	}

	log.Logging.Access.Infof("Application Stopped")
}

func initNewRelic(cfg *config.ApplicationConfig) *newrelic.Application {
	if cfg == nil {
		syslog.Println("New Relic skipped: config is nil")
		return nil
	}

	if !cfg.NewRelicCfg.IsActive {
		syslog.Println("New Relic skipped: isActive is false")
		return nil
	}

	if cfg.NewRelicCfg.LicenseKey == "" {
		syslog.Println("New Relic skipped: license_key is empty")
		return nil
	}

	if cfg.NewRelicCfg.Name == "" {
		syslog.Println("New Relic skipped: app name is empty")
		return nil
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(cfg.NewRelicCfg.Name),
		newrelic.ConfigLicense(cfg.NewRelicCfg.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigFromEnvironment(),
	)
	if err != nil {
		syslog.Printf("New Relic init failed: %v (continuing without New Relic)\n", err)
		return nil
	}

	if err := app.WaitForConnection(5 * time.Second); err != nil {
		syslog.Printf("New Relic connection failed: %v (continuing without blocking app startup)\n", err)
	}

	syslog.Println("New Relic initialized successfully")

	return app
}

func initNewRelicLogConfig(cfg *config.ApplicationConfig) *log.LogsExportConfig {
	if cfg == nil {
		return nil
	}

	if !cfg.NewRelicCfg.IsActive {
		return nil
	}

	apiKey := cfg.NewRelicCfg.IngestKey
	if apiKey == "" {
		apiKey = cfg.NewRelicCfg.LicenseKey
	}

	if apiKey == "" {
		syslog.Println("New Relic log export skipped: ingest_key and license_key are empty")
		return nil
	}

	return &log.LogsExportConfig{
		Endpoint: nrLogsEndpoint(cfg.NewRelicCfg.Region),
		APIKey:   apiKey,
		Timeout:  3 * time.Second,
	}
}

func nrLogsEndpoint(region string) string {
	switch region {
	case "EU", "eu", "Eu":
		return "https://log-api.eu.newrelic.com/log/v1"
	default:
		return "https://log-api.newrelic.com/log/v1"
	}
}
