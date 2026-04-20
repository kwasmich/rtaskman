// handlers/room.go
package handlers

import (
	"encoding/json"
	"log"
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
		log.Println("X-User header is missing")
		http.Error(w, "X-User header is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		INSERT INTO room (created_by)
		VALUES ($1)
		RETURNING id`

	var result CreateRoomResponse
	err := h.db.QueryRow(ctx, query, createdBy).Scan(&result.ID)
	if err != nil {
		log.Println("Failed to create room:", err)
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := `SELECT id FROM active_room`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		log.Println("Failed to query rooms:", err)
		http.Error(w, "Failed to fetch rooms", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rooms []uuid.UUID

	for rows.Next() {
		var id uuid.UUID

		err := rows.Scan(&id)
		if err != nil {
			log.Println("Failed to scan room:", err)
			http.Error(w, "Failed to scan room", http.StatusInternalServerError)
			return
		}

		rooms = append(rooms, id)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if rooms == nil {
		encoder.Encode([]uuid.UUID{})
	} else {
		encoder.Encode(rooms)
	}
}

func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomIDParam := chi.URLParam(r, "roomID")

	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		log.Println("Invalid room ID:", err)
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
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
		UPDATE room
		SET deleted_at = COALESCE(deleted_at, NOW()), deleted_by = COALESCE(deleted_by, $1)
		WHERE id = $2`

	_, err = h.db.Exec(ctx, query, deletedBy, roomID)
	if err != nil {
		log.Println("Failed to delete room:", err)
		http.Error(w, "Failed to delete room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
