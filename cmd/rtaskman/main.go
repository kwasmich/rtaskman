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

func initDB(ctx context.Context, db *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS room (
			id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
			created_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

			deleted_by TEXT,
			deleted_at TIMESTAMPTZ
		);

		CREATE TABLE IF NOT EXISTS series (
			id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
			room_id uuid REFERENCES room(id) ON DELETE CASCADE,
			created_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			name TEXT NOT NULL,
			description TEXT,
			color TEXT,
			tag_list TEXT[],
			target_interval INTERVAL,
			meta JSON,

			deleted_at TIMESTAMPTZ,
			deleted_by TEXT
		);

		CREATE TABLE IF NOT EXISTS event (
			id SERIAL PRIMARY KEY,
			series_id uuid REFERENCES series(id) ON DELETE CASCADE,
			created_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			meta JSON
		);

		CREATE OR REPLACE VIEW active_room AS
		SELECT id FROM room
		WHERE deleted_at IS NULL;

		CREATE OR REPLACE VIEW active_series AS
		SELECT
			s.id,
			s.room_id,
			s.created_at,
			s.created_by,
			s.name,
			s.description,
			s.color,
			s.tag_list,
			s.target_interval::text,
			s.meta,
			e.time_diffs,
			e.median_diff::text,
			e.last_date,
			e.last_date + s.target_interval AS next_target_date,
			e.last_date + e.median_diff AS next_median_date
		FROM series s
		LEFT JOIN LATERAL (
			SELECT
				array_agg(EXTRACT(EPOCH FROM x.time_diff)) AS time_diffs,
				MAX(x.created_at) AS last_date,
				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY x.time_diff) AS median_diff
			FROM (
				SELECT
					created_at,
					created_at - LAG(created_at) OVER (ORDER BY created_at) AS time_diff
				FROM event
				WHERE series_id = s.id
			) x
		) e ON TRUE
		WHERE s.deleted_at IS NULL;
		`

	_, err := db.Exec(ctx, query)
	return err
}

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

	err = db.Ping(ctx)
	if err != nil {
		log.Fatal("Failed to ping db:", err)
	}

	err = initDB(ctx, db)
	if err != nil {
		log.Fatal("Failed to initialize DB:", err)
	}

	// Initialize routers
	r := chi.NewRouter()
	// r.Use(middleware.RequestID)
	// r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Register handlers
	roomHandler := handlers.NewRoomHandler(db)
	seriesHandler := handlers.NewSeriesHandler(db)
	eventHandler := handlers.NewEventHandler(db)
	icalHandler := handlers.NewICalHandler(db)

	r.Route("/api/rtaskman", func(r chi.Router) {
		r.Route("/room", func(r chi.Router) {
			r.Post("/", roomHandler.CreateRoom)
			r.Get("/", roomHandler.ListRooms)
			r.Delete("/{roomID}", roomHandler.DeleteRoom)
		})

		r.Route("/room/{roomID}/series", func(r chi.Router) {
			r.Post("/", seriesHandler.CreateSeries)
			r.Get("/", seriesHandler.ListSeries)
			r.Get("/{seriesID}", seriesHandler.GetSeries)
			r.Delete("/{seriesID}", seriesHandler.DeleteSeries)
			r.Put("/{seriesID}", seriesHandler.UpdateSeries)
		})

		r.Route("/room/{roomID}/series/{seriesID}/event", func(r chi.Router) {
			r.Post("/", eventHandler.CreateEvent)
			r.Get("/", eventHandler.ListEvents)
			r.Delete("/{eventID}", eventHandler.DeleteEvent)
			r.Put("/{eventID}", eventHandler.UpdateEvent)
		})

		r.Get("/room/{roomID}/ical", icalHandler.GetICalFeed)
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
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	log.Println("Server started successfully")
	<-done
	log.Println("Server shutting down...")

	err = srv.Shutdown(ctx)
	if err != nil {
		log.Fatal("Server shutdown error:", err)
	}
}
