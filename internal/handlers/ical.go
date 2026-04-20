// handlers/ical.go
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ICalHandler struct {
	db *pgxpool.Pool
}

func NewICalHandler(db *pgxpool.Pool) *ICalHandler {
	return &ICalHandler{db: db}
}

func (h *ICalHandler) GetICalFeed(w http.ResponseWriter, r *http.Request) {

	roomID := chi.URLParam(r, "roomID")
	seriesIDs := r.URL.Query()["series_id"]

	if len(seriesIDs) == 0 {
		log.Println("series_id query parameter is missing")
		http.Error(w, "series_id parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `
		SELECT id, name, description, last_date, next_target_date, next_median_date
		FROM active_series
		WHERE room_id = $1 AND id::text = ANY($2)`

	rows, err := h.db.Query(ctx, query, roomID, seriesIDs)
	if err != nil {
		log.Println("Failed to query series for calendar:", err)
		http.Error(w, "Failed to fetch series for calendar", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Generate iCalendar content
	var icalBuilder strings.Builder
	icalBuilder.WriteString("BEGIN:VCALENDAR\n")
	icalBuilder.WriteString("VERSION:2.0\n")
	icalBuilder.WriteString("PRODID:-//rtaskman//EN\n")

	duration := "PT15M"

	for rows.Next() {
		var seriesID, name string
		var description *string
		var lastDate, nextTargetDate, nextMedianDate *time.Time

		err := rows.Scan(&seriesID, &name, &description, &lastDate, &nextTargetDate, &nextMedianDate)
		if err != nil {
			log.Println("Failed to scan series data:", err)
			http.Error(w, "Failed to parse series data", http.StatusInternalServerError)
			return
		}

		log.Println("Series:", seriesID, name, description, lastDate, nextTargetDate, nextMedianDate)

		// Process the three dates (last_date, next_target_date, next_median_date)
		dates := []*time.Time{lastDate, nextTargetDate, nextMedianDate}
		for i, date := range dates {
			if date != nil && !date.IsZero() {
				icalBuilder.WriteString(fmt.Sprintf("BEGIN:VEVENT\n"))
				icalBuilder.WriteString(fmt.Sprintf("UID:%s-%d@rtaskman\n", seriesID, i))
				icalBuilder.WriteString(fmt.Sprintf("DTSTART:%s\n", date.Format("20060102T150405Z")))
				icalBuilder.WriteString(fmt.Sprintf("DURATION:%s\n", duration))
				icalBuilder.WriteString(fmt.Sprintf("SUMMARY:%s\n", escapeICalValue(name)))
				if description != nil {
					icalBuilder.WriteString(fmt.Sprintf("DESCRIPTION:%s\n", escapeICalValue(*description)))
				}
				icalBuilder.WriteString("END:VEVENT\n")
			}
		}
	}

	icalBuilder.WriteString("END:VCALENDAR\n")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"rtaskman.ics\"")
	w.Write([]byte(icalBuilder.String()))
	w.WriteHeader(http.StatusOK)
}

// Helper function to escape iCalendar values
func escapeICalValue(s string) string {
	// Replace newlines and semicolons with escaped versions
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	return s
}
