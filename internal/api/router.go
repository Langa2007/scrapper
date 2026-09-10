package api

import (
	"net/http"
)

// SetupRouter registers routes and middleware.
func SetupRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("GET /api/v1/health", h.HealthCheck)
	mux.HandleFunc("POST /api/v1/scrape", h.ScrapeURLs)
	mux.HandleFunc("POST /api/v1/crawl", h.CrawlSite)
	mux.HandleFunc("POST /api/v1/query", h.QueryInternet)
	mux.HandleFunc("POST /api/v1/news", h.GetNews)

	// Wrap with CORS middleware for future JS frontend and cross-origin integration
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
