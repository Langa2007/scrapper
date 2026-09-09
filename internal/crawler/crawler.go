package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 ScraperBot/1.0"
	DefaultTimeout   = 15 * time.Second
)

// Engine manages concurrent scraping operations.
type Engine struct {
	client *http.Client
}

// NewEngine initializes a new high-performance crawler engine with connection pooling.
func NewEngine() *Engine {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Engine{
		client: &http.Client{
			Transport: transport,
			Timeout:   DefaultTimeout,
		},
	}
}

// ScrapeSingle fetches and processes a single URL.
func (e *Engine) ScrapeSingle(ctx context.Context, job ScrapeJob) ScrapeResult {
	start := time.Now()
	res := ScrapeResult{
		URL: job.URL,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", job.URL, nil)
	if err != nil {
		res.Error = fmt.Sprintf("invalid request: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	for k, v := range job.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		res.Error = fmt.Sprintf("fetch error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // Limit to 5MB
	if err != nil {
		res.Error = fmt.Sprintf("read error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	rawHTML := string(bodyBytes)
	title, desc, cleanText := ExtractBasicHTML(rawHTML)

	res.Title = title
	res.Description = desc
	res.CleanText = cleanText
	res.HTML = rawHTML
	res.DurationMs = time.Since(start).Milliseconds()

	return res
}

// ScrapeBatch scrapes multiple URLs concurrently with a worker pool.
func (e *Engine) ScrapeBatch(ctx context.Context, urls []string, concurrency int) []ScrapeResult {
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 50 {
		concurrency = 50
	}

	results := make([]ScrapeResult, len(urls))
	jobsChan := make(chan struct {
		idx int
		url string
	}, len(urls))

	for i, u := range urls {
		jobsChan <- struct {
			idx int
			url string
		}{idx: i, url: u}
	}
	close(jobsChan)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobsChan {
				select {
				case <-ctx.Done():
					results[item.idx] = ScrapeResult{
						URL:   item.url,
						Error: "operation cancelled",
					}
					return
				default:
					job := ScrapeJob{URL: item.url}
					results[item.idx] = e.ScrapeSingle(ctx, job)
				}
			}
		}()
	}

	wg.Wait()
	return results
}
