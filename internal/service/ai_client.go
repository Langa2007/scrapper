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

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

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

func (c *AIClient) SearchInternet(ctx context.Context, query string, maxResults int, searchType string, providers []string) ([]map[string]any, error) {
	payload := map[string]any{
		"query":       query,
		"max_results": maxResults,
		"type":        searchType,
		"providers":   providers,
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

func (c *AIClient) Chat(ctx context.Context, req crawler.ChatRequest) (*crawler.ChatResponse, error) {
	if req.SessionID == "" {
		req.SessionID = "default"
	}
	if req.Mode == "" {
		req.Mode = "general"
	}

	body, _ := json.Marshal(map[string]any{
		"message":      req.Message,
		"session_id":   req.SessionID,
		"mode":         req.Mode,
		"site_context": req.SiteContext,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat error (%d): %s", resp.StatusCode, string(b))
	}

	var res crawler.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *AIClient) GetCrypto(ctx context.Context, coin string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{"coin": coin})
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/crypto", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("coin '%s' not found", coin)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("crypto error (%d): %s", resp.StatusCode, string(b))
	}

	var res map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *AIClient) GetTrendingCrypto(ctx context.Context) (map[string]any, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/ai/crypto/trending", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(httpReq)
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

func (c *AIClient) GetCryptoMarket(ctx context.Context, coins []string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 10
	}
	body, _ := json.Marshal(map[string]any{"coins": coins, "limit": limit})
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/ai/crypto/market", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
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
