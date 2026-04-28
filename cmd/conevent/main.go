package main

import (
	"conevent-backend/config"
	"conevent-backend/internal/api"
	"conevent-backend/internal/db"
	"conevent-backend/internal/observability"
	"conevent-backend/internal/service"
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection pool
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Create database querier
	querier := db.New(dbpool)

	// Create service layer
	eventService := service.NewEventService(querier)

	// Initialize observability
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	obsConfig := observability.Config{
		ServiceName:    cfg.Observability.ServiceName,
		ServiceVersion: cfg.Observability.ServiceVersion,
		Exporter:       cfg.Observability.TraceExporter,
		OTLPEndpoint:   cfg.Observability.OTLPEndpoint,
		PrometheusPort: cfg.Observability.PrometheusPort,
	}

	tp, err := observability.InitTracing(ctx, obsConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize tracing: %v", err)
	}

	mp, reg, err := observability.InitMetrics(ctx, obsConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize metrics: %v", err)
	}

	if err := observability.InitCustomMetrics(obsConfig.ServiceName); err != nil {
		log.Printf("Warning: Failed to initialize custom metrics: %v", err)
	}

	businessMetrics, err := observability.NewBusinessMetrics(obsConfig.ServiceName)
	if err != nil {
		log.Printf("Warning: Failed to initialize business metrics: %v", err)
	}

	defer func() {
		if tp != nil {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}
		if mp != nil {
			if err := mp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down meter provider: %v", err)
			}
		}
	}()

	// Start metrics HTTP server in a goroutine
	go func() {
		log.Printf("Starting metrics server on port %s", obsConfig.PrometheusPort)
		if err := observability.StartHTTPServer(obsConfig.PrometheusPort, reg); err != nil {
			log.Printf("Error starting metrics server: %v", err)
		} else {
			log.Printf("Metrics server stopped")
		}
	}()

	// Create handler layer
	eventHandler := api.NewEventHandler(eventService, businessMetrics)

	// Create Fiber app with timeouts from config
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	obsConfigMetrics := observability.NewMetricsMiddleware()
	app.Use(obsConfigMetrics.Handle)

	// Setup observability tracing
	observability.SetupApp(app, obsConfig.ServiceName)

	// Routes - handlers should only parse requests and call services
	app.Get("/health", api.HealthCheck)
	app.Get("/events", eventHandler.ListEvents)
	app.Post("/events", eventHandler.CreateEvent)
	app.Get("/events/:id", eventHandler.GetEvent)
	app.Put("/events/:id", eventHandler.UpdateEvent)
	app.Delete("/events/:id", eventHandler.DeleteEvent)

	// Serve OpenAPI documentation
	app.Get("/openapi.yaml", func(c *fiber.Ctx) error {
		return c.SendFile("./openapi.yaml")
	})

	// Serve Swagger UI
	app.Get("/docs/*", func(c *fiber.Ctx) error {
		return c.SendFile("./swagger.html")
	})
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("./swagger.html")
	})

	// Start server
	address := ":" + cfg.Server.Port
	log.Printf("Starting server on %s", address)
	if err := app.Listen(address); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
