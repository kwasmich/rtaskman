// handlers/series.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SeriesHandler struct {
	db *pgxpool.Pool
}

func NewSeriesHandler(db *pgxpool.Pool) *SeriesHandler {
	return &SeriesHandler{db: db}
}

type SeriesRequest struct {
	Name           string           `json:"name"`
	Description    *string          `json:"description,omitempty"`
	Color          *string          `json:"color,omitempty"`
	TagList        []string         `json:"tag_list,omitempty"`
	TargetInterval *string          `json:"target_interval,omitempty"`
	Meta           *json.RawMessage `json:"meta,omitempty"`
}

type SeriesResponse struct {
	ID             uuid.UUID        `json:"id"`
	RoomID         uuid.UUID        `json:"room_id"`
	CreatedAt      time.Time        `json:"created_at"`
	CreatedBy      string           `json:"created_by"`
	Name           string           `json:"name"`
	Description    *string          `json:"description,omitempty"`
	Color          *string          `json:"color,omitempty"`
	TagList        []string         `json:"tag_list,omitempty"`
	TargetInterval *string          `json:"target_interval,omitempty"`
	Meta           *json.RawMessage `json:"meta,omitempty"`
	TimeDiff       *string          `json:"time_diff,omitempty"`
}

func (h *SeriesHandler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "roomID"))
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	var req SeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	// Convert target_interval string to duration if provided
	var targetInterval *string
	if req.TargetInterval != nil {
		targetInterval = req.TargetInterval
	}

	var id uuid.UUID
	err = h.db.QueryRow(
		context.Background(),
		`INSERT INTO series (room_id, created_by, name, description, color, tag_list, target_interval, meta) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		roomID,
		createdBy,
		req.Name,
		req.Description,
		req.Color,
		req.TagList,
		targetInterval,
		req.Meta,
	).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to create series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"id": id})
}

func (h *SeriesHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "roomID"))
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(
		context.Background(),
		`SELECT id, room_id, created_at, created_by, name, description, color, tag_list, target_interval, meta, time_diffs 
		FROM active_series WHERE room_id = $1`,
		roomID,
	)
	if err != nil {
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var series []SeriesResponse
	for rows.Next() {
		var s SeriesResponse
		var timeDiff *string
		err := rows.Scan(
			&s.ID,
			&s.RoomID,
			&s.CreatedAt,
			&s.CreatedBy,
			&s.Name,
			&s.Description,
			&s.Color,
			&s.TagList,
			&s.TargetInterval,
			&s.Meta,
			&timeDiff,
		)
		if err != nil {
			http.Error(w, "Failed to scan series", http.StatusInternalServerError)
			return
		}
		s.TimeDiff = timeDiff
		series = append(series, s)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(series)
}

func (h *SeriesHandler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	seriesID, err := uuid.Parse(chi.URLParam(r, "seriesID"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	deletedBy := r.Header.Get("X-User")
	if deletedBy == "" {
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		`UPDATE series SET deleted_at = COALESCE(deleted_at, NOW()), deleted_by = COALESCE(deleted_by, $1) WHERE id = $2`,
		deletedBy,
		seriesID,
	)
	if err != nil {
		http.Error(w, "Failed to delete series", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SeriesHandler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	seriesID, err := uuid.Parse(chi.URLParam(r, "seriesID"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	var req SeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{seriesID}
	argIndex := 2

	if req.Name != "" {
		setClauses = append(setClauses, "name = $"+string(rune(argIndex)))
		args = append(args, req.Name)
		argIndex++
	}
	if req.Description != nil {
		setClauses = append(setClauses, "description = $"+string(rune(argIndex)))
		args = append(args, req.Description)
		argIndex++
	}
	if req.Color != nil {
		setClauses = append(setClauses, "color = $"+string(rune(argIndex)))
		args = append(args, req.Color)
		argIndex++
	}
	if req.TagList != nil {
		setClauses = append(setClauses, "tag_list = $"+string(rune(argIndex)))
		args = append(args, req.TagList)
		argIndex++
	}
	if req.TargetInterval != nil {
		setClauses = append(setClauses, "target_interval = $"+string(rune(argIndex)))
		args = append(args, req.TargetInterval)
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

	query := "UPDATE series SET " + strings.Join(setClauses, ", ") + " WHERE id = $" + string(rune(1))
	args = append(args, seriesID)

	_, err = h.db.Exec(context.Background(), query, args...)
	if err != nil {
		http.Error(w, "Failed to update series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
