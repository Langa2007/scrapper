package crawler

// ScrapeJob defines the target and parameters for scraping a single URL.
type ScrapeJob struct {
	URL        string            `json:"url"`
	TimeoutSec int               `json:"timeout_sec,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// ScrapeResult represents the scraped content and metadata from a web page.
type ScrapeResult struct {
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	Title       string            `json:"title"`
	HTML        string            `json:"html,omitempty"`
	CleanText   string            `json:"clean_text"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	DurationMs  int64             `json:"duration_ms"`
	Error       string            `json:"error,omitempty"`
}

// ScrapeBatchRequest defines a batch of URLs to scrape concurrently.
type ScrapeBatchRequest struct {
	URLs        []string `json:"urls"`
	Concurrency int      `json:"concurrency,omitempty"`
	TimeoutSec  int      `json:"timeout_sec,omitempty"`
}

// SearchAndQueryRequest represents an intelligent question-answering request.
type SearchAndQueryRequest struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

// NewsRequest represents a news retrieval request (tailored for external services like Dira News).
type NewsRequest struct {
	Topic     string `json:"topic"`
	Limit     int    `json:"limit,omitempty"`
	Timeframe string `json:"timeframe,omitempty"` // "d" for day, "w" for week, "m" for month
}

// NewsArticle represents a structured news article.
type NewsArticle struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	Summary       string `json:"summary"`
	Author        string `json:"author,omitempty"`
	PublishedDate string `json:"published_date,omitempty"`
	Source        string `json:"source"`
	Image         string `json:"image,omitempty"`
}

// NewsResponse represents the structured response for news queries.
type NewsResponse struct {
	Topic    string        `json:"topic"`
	Count    int           `json:"count"`
	Articles []NewsArticle `json:"articles"`
}

// QueryResponse represents the AI synthesized answer for a general query.
type QueryResponse struct {
	Query      string           `json:"query"`
	Answer     string           `json:"answer"`
	Sources    []map[string]any `json:"sources"`
	Confidence string           `json:"confidence"`
	DurationMs int64            `json:"duration_ms"`
}
