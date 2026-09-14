package crawler

type ScrapeJob struct {
	URL        string            `json:"url"`
	TimeoutSec int               `json:"timeout_sec,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type ScrapeResult struct {
	URL         string            `json:"url"`
	FinalURL    string            `json:"final_url,omitempty"`
	StatusCode  int               `json:"status_code"`
	Title       string            `json:"title"`
	HTML        string            `json:"html,omitempty"`
	CleanText   string            `json:"clean_text"`
	Description string            `json:"description,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Links       []string          `json:"links,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	DurationMs  int64             `json:"duration_ms"`
	Error       string            `json:"error,omitempty"`
}

type ScrapeBatchRequest struct {
	URLs        []string `json:"urls"`
	Concurrency int      `json:"concurrency,omitempty"`
	TimeoutSec  int      `json:"timeout_sec,omitempty"`
}

type CrawlRequest struct {
	StartURL        string `json:"start_url"`
	MaxPages        int    `json:"max_pages,omitempty"`
	MaxDepth        int    `json:"max_depth,omitempty"`
	Concurrency     int    `json:"concurrency,omitempty"`
	AllowSubdomains bool   `json:"allow_subdomains,omitempty"`
}

type CrawlResponse struct {
	StartURL string         `json:"start_url"`
	Count    int            `json:"count"`
	Results  []ScrapeResult `json:"results"`
}

type SearchAndQueryRequest struct {
	Query      string   `json:"query"`
	MaxResults int      `json:"max_results,omitempty"`
	Providers  []string `json:"providers,omitempty"`
}

type NewsRequest struct {
	Topic     string `json:"topic"`
	Limit     int    `json:"limit,omitempty"`
	Timeframe string `json:"timeframe,omitempty"`
}

type NewsArticle struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	Summary       string `json:"summary"`
	Author        string `json:"author,omitempty"`
	PublishedDate string `json:"published_date,omitempty"`
	Source        string `json:"source"`
	Image         string `json:"image,omitempty"`
}

type NewsResponse struct {
	Topic    string        `json:"topic"`
	Count    int           `json:"count"`
	Articles []NewsArticle `json:"articles"`
}

type QueryResponse struct {
	Query      string           `json:"query"`
	Answer     string           `json:"answer"`
	Sources    []map[string]any `json:"sources"`
	Confidence string           `json:"confidence"`
	DurationMs int64            `json:"duration_ms"`
	Providers  []string         `json:"providers,omitempty"`
}

type ChatRequest struct {
	Message     string `json:"message"`
	SessionID   string `json:"session_id,omitempty"`
	Mode        string `json:"mode,omitempty"`
	SiteContext string `json:"site_context,omitempty"`
}

type ChatResponse struct {
	Reply         string           `json:"reply"`
	SessionID     string           `json:"session_id"`
	Mode          string           `json:"mode"`
	Sources       []map[string]any `json:"sources"`
	HistoryLength int              `json:"history_length"`
}

type CryptoRequest struct {
	Coin string `json:"coin"`
}

type CryptoMarketRequest struct {
	Coins []string `json:"coins,omitempty"`
	Limit int      `json:"limit,omitempty"`
}
