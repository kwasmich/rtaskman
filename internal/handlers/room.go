// handlers/room.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomHandler struct {
	db *pgxpool.Pool
}

func NewRoomHandler(db *pgxpool.Pool) *RoomHandler {
	return &RoomHandler{db: db}
}

type CreateRoomResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	createdBy := r.Header.Get("X-User")
	if createdBy == "" {
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	var req CreateRoomResponse
	err := h.db.QueryRow(
		context.Background(),
		`INSERT INTO room (created_by) VALUES ($1) RETURNING id`,
		createdBy,
	).Scan(&req.ID)
	if err != nil {
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		context.Background(),
		`SELECT id FROM active_room`,
	)
	if err != nil {
		http.Error(w, "Failed to fetch rooms", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rooms []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			http.Error(w, "Failed to scan room", http.StatusInternalServerError)
			return
		}
		rooms = append(rooms, id)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(rooms)
}

func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "roomID"))
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	deletedBy := r.Header.Get("X-User")
	if deletedBy == "" {
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		`UPDATE room SET deleted_at = COALESCE(deleted_at, NOW()), deleted_by = COALESCE(deleted_by, $1) WHERE id = $2`,
		deletedBy,
		roomID,
	)
	if err != nil {
		http.Error(w, "Failed to delete room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
