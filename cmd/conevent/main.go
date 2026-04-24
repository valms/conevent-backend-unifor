package main

import (
	"conevent-backend/config"
	"conevent-backend/internal/api"
	"conevent-backend/internal/db"
	"conevent-backend/internal/service"
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
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

	// Create handler layer
	eventHandler := api.NewEventHandler(eventService)

	// Create Fiber app with timeouts from config
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	})

	// Middleware
	app.Use(logger.New())

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
