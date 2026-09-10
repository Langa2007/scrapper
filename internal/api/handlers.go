package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"scrapper/internal/crawler"
	"scrapper/internal/service"
)

type Handler struct {
	engine   *crawler.Engine
	aiClient *service.AIClient
}

func NewHandler(engine *crawler.Engine, aiClient *service.AIClient) *Handler {
	return &Handler{
		engine:   engine,
		aiClient: aiClient,
	}
}

// HealthCheck handles GET /api/v1/health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	pyHealth, err := h.aiClient.Health(r.Context())
	pyStatus := "connected"
	if err != nil {
		pyStatus = "unreachable: " + err.Error()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "healthy",
		"service":        "go_scraper_gateway",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"python_service": pyStatus,
		"python_info":    pyHealth,
	})
}

// ScrapeURLs handles POST /api/v1/scrape
func (h *Handler) ScrapeURLs(w http.ResponseWriter, r *http.Request) {
	var req crawler.ScrapeBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload: "+err.Error())
		return
	}

	if len(req.URLs) == 0 {
		writeError(w, http.StatusBadRequest, "must provide at least one URL in 'urls'")
		return
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	ctx := r.Context()
	if req.TimeoutSec > 0 {
		timeout := time.Duration(req.TimeoutSec) * time.Second
		if timeout > 60*time.Second {
			timeout = 60 * time.Second
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	results := h.engine.ScrapeBatch(ctx, req.URLs, concurrency)

	// Enrich with Python AI cleaner if available
	for i := range results {
		if results[i].Error == "" && results[i].HTML != "" {
			cleanData, err := h.aiClient.CleanHTML(r.Context(), results[i].HTML, results[i].URL)
			if err == nil {
				if cleanText, ok := cleanData["text"].(string); ok && len(cleanText) > 0 {
					results[i].CleanText = cleanText
				}
				if title, ok := cleanData["title"].(string); ok && title != "" {
					results[i].Title = title
				}
			}
			// Do not leak heavy raw HTML in response unless requested
			results[i].HTML = ""
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":   len(results),
		"results": results,
	})
}

// CrawlSite traverses a bounded set of same-site public pages.
func (h *Handler) CrawlSite(w http.ResponseWriter, r *http.Request) {
	var req crawler.CrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload: "+err.Error())
		return
	}
	if req.StartURL == "" {
		writeError(w, http.StatusBadRequest, "start_url cannot be empty")
		return
	}
	writeJSON(w, http.StatusOK, h.engine.CrawlSite(r.Context(), req))
}

// QueryInternet handles POST /api/v1/query
// Searches the open web, crawls top results concurrently in Go, and synthesizes answers with Python AI.
func (h *Handler) QueryInternet(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req crawler.SearchAndQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload: "+err.Error())
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query cannot be empty")
		return
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	// 1. Search through the configured providers, then crawl public results.
	searchResults, err := h.aiClient.SearchInternet(r.Context(), req.Query, maxResults, "web", req.Providers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search internet: "+err.Error())
		return
	}

	if len(searchResults) == 0 {
		writeJSON(w, http.StatusOK, crawler.QueryResponse{
			Query:      req.Query,
			Answer:     "No search results found for the given query.",
			Sources:    []map[string]any{},
			Confidence: "none",
			DurationMs: time.Since(start).Milliseconds(),
			Providers:  req.Providers,
		})
		return
	}

	// 2. Extract URLs to scrape
	var urls []string
	for _, sr := range searchResults {
		if u, ok := sr["url"].(string); ok && u != "" {
			urls = append(urls, u)
		}
	}

	// 3. Scrape pages concurrently using Go's high-speed engine
	scraped := h.engine.ScrapeBatch(r.Context(), urls, 5)

	// Clean and combine documents
	var validDocs []crawler.ScrapeResult
	for i, doc := range scraped {
		if doc.Error == "" && doc.CleanText != "" {
			validDocs = append(validDocs, doc)
		} else if i < len(searchResults) {
			// Fallback to search snippet if scrape failed or was blocked
			snippet, _ := searchResults[i]["snippet"].(string)
			title, _ := searchResults[i]["title"].(string)
			validDocs = append(validDocs, crawler.ScrapeResult{
				URL:       urls[i],
				Title:     title,
				CleanText: snippet,
			})
		}
	}

	// 4. Synthesize with AI
	aiResp, err := h.aiClient.SynthesizeAnswer(r.Context(), req.Query, validDocs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ai synthesis failed: "+err.Error())
		return
	}

	answer, _ := aiResp["answer"].(string)
	confidence, _ := aiResp["confidence"].(string)
	rawSources, _ := aiResp["sources"].([]any)

	var sources []map[string]any
	for _, s := range rawSources {
		if m, ok := s.(map[string]any); ok {
			sources = append(sources, m)
		}
	}

	writeJSON(w, http.StatusOK, crawler.QueryResponse{
		Query:      req.Query,
		Answer:     answer,
		Sources:    sources,
		Confidence: confidence,
		DurationMs: time.Since(start).Milliseconds(),
		Providers:  req.Providers,
	})
}

// GetNews handles POST /api/v1/news
// Dedicated news endpoint for systems like Dira News.
func (h *Handler) GetNews(w http.ResponseWriter, r *http.Request) {
	var req crawler.NewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload: "+err.Error())
		return
	}

	if req.Topic == "" {
		writeError(w, http.StatusBadRequest, "topic cannot be empty")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}

	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = "w"
	}

	newsRes, err := h.aiClient.FetchNews(r.Context(), req.Topic, limit, timeframe)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch news: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, newsRes)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":  message,
		"status": status,
	})
}
