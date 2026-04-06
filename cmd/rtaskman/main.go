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
	"github.com/jackc/pgx/v5/pgxpool"

	"kwasi-ich.de/rtaskman/internal/handlers"
)

func main() {
	// Initialize database connection
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize routers
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Register handlers
	roomHandler := handlers.NewRoomHandler(db)
	seriesHandler := handlers.NewSeriesHandler(db)
	eventHandler := handlers.NewEventHandler(db)

	r.Route("/room", func(r chi.Router) {
		r.Post("/", roomHandler.CreateRoom)
		r.Get("/", roomHandler.ListRooms)
		r.Delete("/{roomID}", roomHandler.DeleteRoom)
	})

	r.Route("/series", func(r chi.Router) {
		r.Post("/{roomID}", seriesHandler.CreateSeries)
		r.Get("/{roomID}", seriesHandler.ListSeries)
		r.Delete("/{seriesID}", seriesHandler.DeleteSeries)
		r.Patch("/{seriesID}", seriesHandler.UpdateSeries)
	})

	r.Route("/event", func(r chi.Router) {
		r.Post("/{seriesID}", eventHandler.CreateEvent)
		r.Get("/{seriesID}", eventHandler.ListEvents)
		r.Delete("/{seriesID}/{eventID}", eventHandler.DeleteEvent)
		r.Patch("/{seriesID}/{eventID}", eventHandler.UpdateEvent)
	})

	// Start server
	srv := &http.Server{
		Addr:         ":8087",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	log.Println("Server started on :8087")
	<-done
	log.Println("Server shutting down...")
	srv.Shutdown(context.Background())
}
