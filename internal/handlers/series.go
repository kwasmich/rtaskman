// handlers/series.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
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

type SeriesCreateRequest struct {
	Name           string           `json:"name"`
	Description    *string          `json:"description,omitempty"`
	Color          *string          `json:"color,omitempty"`
	TagList        []string         `json:"tag_list,omitempty"`
	TargetInterval *string          `json:"target_interval,omitempty"`
	Meta           *json.RawMessage `json:"meta,omitempty"`
}

type SeriesCreateResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *SeriesHandler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		log.Println("X-User header is missing")
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	var req SeriesCreateRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Failed to decode request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		log.Println("Name is required")
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		INSERT INTO series (room_id, created_by, name, description, color, tag_list, target_interval, meta) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var result SeriesCreateResponse

	err = h.db.QueryRow(
		ctx,
		query,
		roomID,
		createdBy,
		req.Name,
		req.Description,
		req.Color,
		req.TagList,
		req.TargetInterval,
		req.Meta,
	).Scan(&result.ID)
	if err != nil {
		log.Println("Failed to create series:", err)
		http.Error(w, "Failed to create series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

type SeriesResponse struct {
	ID             uuid.UUID        `json:"id"`
	CreatedAt      time.Time        `json:"created_at"`
	CreatedBy      string           `json:"created_by"`
	Name           string           `json:"name"`
	Description    *string          `json:"description,omitempty"`
	Color          *string          `json:"color,omitempty"`
	TagList        []string         `json:"tag_list,omitempty"`
	TargetInterval *string          `json:"target_interval,omitempty"`
	Meta           *json.RawMessage `json:"meta,omitempty"`
	TimeDiffs      []*float64       `json:"time_diffs,omitempty"`
	MedianDiff     *string          `json:"median_diff,omitempty"`
	LastDate       *time.Time       `json:"last_date,omitempty"`
	NextTargetDate *time.Time       `json:"next_target_date,omitempty"`
	NextMedianDate *time.Time       `json:"next_median_date,omitempty"`
}

func (h *SeriesHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		SELECT id, created_at, created_by, name, description, color, tag_list, target_interval, meta, time_diffs, median_diff, last_date, next_target_date, next_median_date
		FROM active_series
		WHERE room_id = $1`

	rows, err := h.db.Query(ctx, query, roomID)
	if err != nil {
		log.Println("Failed to query series:", err)
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var series []SeriesResponse
	for rows.Next() {
		var s SeriesResponse
		err := rows.Scan(
			&s.ID,
			&s.CreatedAt,
			&s.CreatedBy,
			&s.Name,
			&s.Description,
			&s.Color,
			&s.TagList,
			&s.TargetInterval,
			&s.Meta,
			&s.TimeDiffs,
			&s.MedianDiff,
			&s.LastDate,
			&s.NextTargetDate,
			&s.NextMedianDate,
		)
		if err != nil {
			log.Println("Error scanning series:", err)
			http.Error(w, "Failed to scan series", http.StatusInternalServerError)
			return
		}

		series = append(series, s)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if series == nil {
		encoder.Encode([]SeriesResponse{})
	} else {
		encoder.Encode(series)
	}
}

func (h *SeriesHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	roomID, err := uuid.Parse(roomIDParam)
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
		SELECT id, created_at, created_by, name, description, color, tag_list, target_interval, meta, time_diffs, median_diff, last_date, next_target_date, next_median_date
		FROM active_series
		WHERE room_id = $1 AND id = $2`

	var result SeriesResponse

	err = h.db.QueryRow(ctx, query, roomID, seriesID).Scan(&result.ID, &result.CreatedAt, &result.CreatedBy, &result.Name, &result.Description, &result.Color, &result.TagList, &result.TargetInterval, &result.Meta, &result.TimeDiffs, &result.MedianDiff, &result.LastDate, &result.NextTargetDate, &result.NextMedianDate)
	if err != nil {
		log.Println("Failed to query series:", err)
		http.Error(w, "Failed to fetch series", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
}

func (h *SeriesHandler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	roomID, err := uuid.Parse(roomIDParam)
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

	deletedBy := r.Header.Get("X-User")
	if deletedBy == "" {
		log.Println("X-User header is missing")
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		UPDATE series
		SET deleted_at = COALESCE(deleted_at, NOW()), deleted_by = COALESCE(deleted_by, $1)
		WHERE room_id = $2 AND id = $3`

	_, err = h.db.Exec(ctx, query, deletedBy, roomID, seriesID)
	if err != nil {
		log.Println("Failed to delete series:", err)
		http.Error(w, "Failed to delete series", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/****************************************/
/* TODO: Implement UpdateSeries handler */
/****************************************/

func (h *SeriesHandler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	roomID, err := uuid.Parse(roomIDParam)
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

	var req SeriesCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Failed to decode request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		log.Println("Name is required")
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		UPDATE series
		SET name = $1, description = $2, color = $3, tag_list = $4, target_interval = $5, meta = $6
		WHERE room_id = $7 AND id = $8`

	_, err = h.db.Exec(
		ctx,
		query,
		req.Name,
		req.Description,
		req.Color,
		req.TagList,
		req.TargetInterval,
		req.Meta,
		roomID,
		seriesID,
	)
	if err != nil {
		log.Println("Failed to update series:", err)
		http.Error(w, "Failed to update series", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
