// handlers/event.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventHandler struct {
	db *pgxpool.Pool
}

func NewEventHandler(db *pgxpool.Pool) *EventHandler {
	return &EventHandler{db: db}
}

type EventRequest struct {
	CreatedAt *time.Time       `json:"created_at,omitempty"`
	Meta      *json.RawMessage `json:"meta,omitempty"`
}

type EventResponse struct {
	ID        int              `json:"id"`
	SeriesID  *uuid.UUID       `json:"series_id,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	CreatedBy string           `json:"created_by"`
	Meta      *json.RawMessage `json:"meta,omitempty"`
	TimeDiff  *string          `json:"time_diff,omitempty"`
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	_, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	seriesIDParam := chi.URLParam(r, "seriesID")

	seriesID, err := uuid.Parse(seriesIDParam)
	if err != nil {
		log.Println("Invalid series ID:", err)
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		log.Println("X-User header is missing")
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	var req EventRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Failed to decode request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		INSERT INTO event (series_id, created_by, created_at, meta)
		VALUES ($1, $2, COALESCE($3, NOW()), $4)
		RETURNING id, series_id, created_by, created_at, meta`

	var event EventResponse
	err = h.db.QueryRow(
		ctx,
		query,
		seriesID,
		createdBy,
		req.CreatedAt,
		req.Meta,
	).Scan(&event.ID, &event.SeriesID, &event.CreatedBy, &event.CreatedAt, &event.Meta)
	if err != nil {
		log.Println("Failed to insert event:", err)
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	_, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	seriesIDParam := chi.URLParam(r, "seriesID")

	seriesID, err := uuid.Parse(seriesIDParam)
	if err != nil {
		log.Println("Invalid series ID:", err)
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		SELECT id, created_at, created_by, meta, (created_at - LAG(created_at) OVER (ORDER BY created_at))::text AS time_diff 
		FROM event
		WHERE series_id = $1`

	rows, err := h.db.Query(ctx, query, seriesID)
	if err != nil {
		log.Println("Failed to fetch events:", err)
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []EventResponse
	for rows.Next() {
		var e EventResponse
		err := rows.Scan(&e.ID, &e.CreatedAt, &e.CreatedBy, &e.Meta, &e.TimeDiff)
		if err != nil {
			log.Println("Failed to scan event:", err)
			http.Error(w, "Failed to scan event", http.StatusInternalServerError)
			return
		}

		events = append(events, e)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if events == nil {
		encoder.Encode([]EventResponse{})
	} else {
		encoder.Encode(events)
	}
}

func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	_, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	seriesIDParam := chi.URLParam(r, "seriesID")

	seriesID, err := uuid.Parse(seriesIDParam)
	if err != nil {
		log.Println("Invalid series ID:", err)
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	eventIDParam := chi.URLParam(r, "eventID")

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		log.Println("Invalid event ID:", err)
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		DELETE FROM event
		WHERE series_id = $1 AND id = $2`

	_, err = h.db.Exec(ctx, query, seriesID, eventID)
	if err != nil {
		log.Println("Failed to delete event:", err)
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	_, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	seriesIDParam := chi.URLParam(r, "seriesID")

	seriesID, err := uuid.Parse(seriesIDParam)
	if err != nil {
		log.Println("Invalid series ID:", err)
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	eventIDParam := chi.URLParam(r, "eventID")

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		log.Println("Invalid event ID:", err)
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Failed to decode request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		UPDATE event
		SET created_at = COALESCE($1, NOW()), meta = $2
		WHERE series_id = $3 AND id = $4`

	_, err = h.db.Exec(ctx, query, req.CreatedAt, req.Meta, seriesID, eventID)
	if err != nil {
		log.Println("Failed to update event:", err)
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
