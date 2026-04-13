// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kwasi-ich.de/rtaskman/internal/handlers"
)

// func Ping(ctx context.Context, db *pgxpool.Pool) error {
// 	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
// 	defer cancel()
// 	return db.Ping(ctx)
// }

func main() {
	// Initialize database connection
	dbConnectionString := os.Getenv("DATABASE_URL")
	if dbConnectionString == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	config, err := pgxpool.ParseConfig(dbConnectionString)
	if err != nil {
		log.Fatal("Failed to parse database connection string:", err)
	}

	// config.MaxConns = 10
	// config.MinConns = 1
	// config.MaxConnLifetime = 30 * time.Minute
	// config.MaxConnIdleTime = 5 * time.Minute
	// config.HealthCheckPeriod = 1 * time.Minute

	config.BeforeConnect = func(ctx context.Context, conn *pgx.ConnConfig) error {
		log.Printf("Attempt to connect to Database %v : %v\n", conn.Host, conn.Port)
		return nil
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		log.Println("Successfully connected to Database")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// db, err := pgxpool.New(ctx, dbConnectionString)
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("Failed to create database pool:", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	// Initialize routers
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Register handlers
	roomHandler := handlers.NewRoomHandler(db)
	seriesHandler := handlers.NewSeriesHandler(db)
	eventHandler := handlers.NewEventHandler(db)

	r.Route("/api/rtaskman", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			r.Post("/", roomHandler.CreateRoom)
			r.Get("/", roomHandler.ListRooms)
			r.Delete("/{roomID}", roomHandler.DeleteRoom)
		})

		r.Route("/{roomID}/series", func(r chi.Router) {
			r.Post("/", seriesHandler.CreateSeries)
			r.Get("/", seriesHandler.ListSeries)
			// r.Get("/{seriesID}", seriesHandler.GetSeries)
			r.Delete("/{seriesID}", seriesHandler.DeleteSeries)
			r.Put("/{seriesID}", seriesHandler.UpdateSeries)
		})

		r.Route("/{roomID}/series/{seriesID}/event", func(r chi.Router) {
			r.Post("/", eventHandler.CreateEvent)
			r.Get("/", eventHandler.ListEvents)
			r.Delete("/{eventID}", eventHandler.DeleteEvent)
			r.Put("/{eventID}", eventHandler.UpdateEvent)
		})

		// r.Get("/room/{roomID}/ical", seriesHandler.GetICalFeed)
	})

	// Start server
	srv := &http.Server{
		Addr:    ":8087",
		Handler: r,
		// ReadTimeout:  5 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Server starting on %s", srv.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	log.Println("Server started successfully")
	<-done
	log.Println("Server shutting down...")

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
}
