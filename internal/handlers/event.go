// handlers/event.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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
	SeriesID  uuid.UUID        `json:"series_id"`
	CreatedAt time.Time        `json:"created_at"`
	CreatedBy string           `json:"created_by"`
	Meta      *json.RawMessage `json:"meta,omitempty"`
	TimeDiff  *string          `json:"time_diff,omitempty"`
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	seriesID, err := uuid.Parse(chi.URLParam(r, "seriesID"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var event EventResponse
	err = h.db.QueryRow(
		context.Background(),
		`INSERT INTO event (series_id, created_by, created_at, meta) VALUES ($1, $2, $3, $4) RETURNING id, series_id, created_at, created_by, meta`,
		seriesID,
		createdBy,
		req.CreatedAt,
		req.Meta,
	).Scan(&event.ID, &event.SeriesID, &event.CreatedAt, &event.CreatedBy, &event.Meta)
	if err != nil {
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	seriesID, err := uuid.Parse(chi.URLParam(r, "seriesID"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(
		context.Background(),
		`SELECT id, created_at, created_by, meta, created_at - LAG(created_at) OVER (ORDER BY created_at) AS time_diff 
		FROM event WHERE series_id = $1`,
		seriesID,
	)
	if err != nil {
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []EventResponse
	for rows.Next() {
		var e EventResponse
		var timeDiff *string
		err := rows.Scan(&e.ID, &e.CreatedAt, &e.CreatedBy, &e.Meta, &timeDiff)
		if err != nil {
			http.Error(w, "Failed to scan event", http.StatusInternalServerError)
			return
		}
		e.TimeDiff = timeDiff
		events = append(events, e)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(events)
}

func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	seriesID, err := uuid.Parse(chi.URLParam(r, "seriesID"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	eventID, err := strconv.Atoi(chi.URLParam(r, "eventID"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		`DELETE FROM event WHERE series_id = $1 AND id = $2`,
		seriesID,
		eventID,
	)
	if err != nil {
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	seriesID, err := uuid.Parse(chi.URLParam(r, "seriesID"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	eventID, err := strconv.Atoi(chi.URLParam(r, "eventID"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	args := []interface{}{seriesID, eventID}
	setClauses := []string{}
	argIndex := 3

	if req.CreatedAt != nil {
		setClauses = append(setClauses, "created_at = $"+string(rune(argIndex)))
		args = append(args, req.CreatedAt)
		argIndex++
	}
	if req.Meta != nil {
		setClauses = append(setClauses, "meta = $"+string(rune(argIndex)))
		args = append(args, req.Meta)
		argIndex++
	}

	if len(setClauses) == 0 {
		http.Error(w, "At least one field must be updated", http.StatusBadRequest)
		return
	}

	query := "UPDATE event SET " + strings.Join(setClauses, ", ") + " WHERE series_id = $" + string(rune(1)) + " AND id = $" + string(rune(2))
	args = append(args, seriesID, eventID)

	_, err = h.db.Exec(context.Background(), query, args...)
	if err != nil {
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
