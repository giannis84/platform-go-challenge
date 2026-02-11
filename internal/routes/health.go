package routes

import (
	"net/http"

	"github.com/giannis84/platform-go-challenge/internal/database"
	"github.com/go-chi/chi/v5"
)

// RegisterHealthRoutes creates the health check endpoints.
func RegisterHealthRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
			if err := database.PingDB(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("database not ready"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ready"))
		})
	}
}
