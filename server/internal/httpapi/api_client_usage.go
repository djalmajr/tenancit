package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/djalmajr/tenancit/server/internal/store/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) listAPIClientUsage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad id"})
		return
	}
	now := s.Now().UTC()
	from, err := usageDateParam(r, "from", now.AddDate(0, -1, 0))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from"})
		return
	}
	to, err := usageDateParam(r, "to", now)
	if err != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid usage window"})
		return
	}
	limit := int32(200)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 32)
		if parseErr != nil || parsed < 1 || parsed > 1000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		limit = int32(parsed)
	}
	rows, err := s.Q.ListAPIClientUsage(r.Context(), db.ListAPIClientUsageParams{
		ApiClientID: id,
		FromDay:     pgtype.Date{Time: from, Valid: true}, ToDay: pgtype.Date{Time: to, Valid: true},
		PageLimit: limit,
	})
	if err != nil {
		writeInternalError(w, r, "list API client usage", err)
		return
	}
	if rows == nil {
		rows = []db.ApiClientUsageDaily{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) listAPIClientUsageOverview(w http.ResponseWriter, r *http.Request) {
	now := s.Now().UTC()
	from, err := usageDateParam(r, "from", now.AddDate(0, -1, 0))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from"})
		return
	}
	to, err := usageDateParam(r, "to", now)
	if err != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid usage window"})
		return
	}
	rows, err := s.Q.ListAPIClientUsageOverview(r.Context(), db.ListAPIClientUsageOverviewParams{
		FromDay: pgtype.Date{Time: from, Valid: true}, ToDay: pgtype.Date{Time: to, Valid: true},
		PageLimit: 1000,
	})
	if err != nil {
		writeInternalError(w, r, "list API client usage overview", err)
		return
	}
	if rows == nil {
		rows = []db.ListAPIClientUsageOverviewRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func usageDateParam(r *http.Request, name string, fallback time.Time) (time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return dayUTC(fallback), nil
	}
	return time.Parse("2006-01-02", raw)
}

func dayUTC(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
