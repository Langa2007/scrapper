package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"scrapper/internal/crawler"
)

// AIClient communicates with the Python AI microservice.
type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAIClient creates a client pointing to the Python AI service (default http://127.0.0.1:5000).
func NewAIClient(baseURL string) *AIClient {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:5000"
	}
	return &AIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Health checks if Python AI service is reachable and healthy.
func (c *AIClient) Health(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// SearchInternet queries DuckDuckGo via Python AI service.
func (c *AIClient) SearchInternet(ctx context.Context, query string, maxResults int, searchType string) ([]map[string]any, error) {
	payload := map[string]any{
		"query":       query,
		"max_results": maxResults,
		"type":        searchType,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Results, nil
}

// CleanHTML uses Python's Trafilatura and BeautifulSoup to clean raw HTML into structured text.
func (c *AIClient) CleanHTML(ctx context.Context, rawHTML string, pageURL string) (map[string]any, error) {
	payload := map[string]any{
		"html": rawHTML,
		"url":  pageURL,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/clean-html", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}

// SynthesizeAnswer passes scraped documents to Python AI for synthesis.
func (c *AIClient) SynthesizeAnswer(ctx context.Context, query string, docs []crawler.ScrapeResult) (map[string]any, error) {
	var docItems []map[string]any
	for _, d := range docs {
		docItems = append(docItems, map[string]any{
			"title":   d.Title,
			"url":     d.URL,
			"text":    d.CleanText,
			"snippet": d.Description,
		})
	}

	payload := map[string]any{
		"query":     query,
		"documents": docItems,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/synthesize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}

// FetchNews asks Python AI to get and format news articles.
func (c *AIClient) FetchNews(ctx context.Context, topic string, limit int, timeframe string) (*crawler.NewsResponse, error) {
	payload := map[string]any{
		"topic":     topic,
		"limit":     limit,
		"timeframe": timeframe,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/news", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("python ai error (%d): %s", resp.StatusCode, string(b))
	}

	var res crawler.NewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
