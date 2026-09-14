package crawler

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultUserAgent = "ScrapperBot/1.0 (+https://example.invalid/bot; contact=admin@example.invalid)"
	DefaultTimeout   = 15 * time.Second
	maxBodyBytes     = 5 * 1024 * 1024
	politeDelay      = 300 * time.Millisecond
)

type Engine struct {
	client      *http.Client
	robots      *robotsCache
	hostMu      sync.Mutex
	nextRequest map[string]time.Time
}

func NewEngine() *Engine {
	engine := &Engine{robots: newRobotsCache(), nextRequest: make(map[string]time.Time)}
	dialer := &net.Dialer{Timeout: DefaultTimeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		MaxIdleConns: 100, MaxIdleConnsPerHost: 20, IdleConnTimeout: 90 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			safeAddress, err := safeDialAddress(ctx, address)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, safeAddress)
		},
	}
	engine.client = &http.Client{
		Transport: transport, Timeout: DefaultTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			_, err := validatePublicURL(req.URL.String())
			return err
		},
	}
	return engine
}

func (e *Engine) waitForHost(ctx context.Context, host string) error {
	e.hostMu.Lock()
	now := time.Now()
	next := e.nextRequest[host]
	if next.Before(now) {
		next = now
	}
	e.nextRequest[host] = next.Add(politeDelay)
	e.hostMu.Unlock()
	if wait := time.Until(next); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func (e *Engine) ScrapeSingle(ctx context.Context, job ScrapeJob) ScrapeResult {
	start := time.Now()
	res := ScrapeResult{URL: job.URL}
	target, err := validatePublicURL(job.URL)
	if err != nil {
		res.Error = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if !e.robots.allowed(ctx, e.client, target, e.waitForHost) {
		res.Error = "blocked by robots.txt"
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if err := e.waitForHost(ctx, target.Hostname()); err != nil {
		res.Error = fmt.Sprintf("request cancelled: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
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
	res.StatusCode, res.FinalURL, res.ContentType = resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type")
	contentType := strings.ToLower(res.ContentType)
	if contentType != "" && !strings.Contains(contentType, "html") && !strings.Contains(contentType, "xml") {
		res.Error = "unsupported content type: " + res.ContentType
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		res.Error = fmt.Sprintf("read error: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	rawHTML := string(body)
	res.Title, res.Description, res.CleanText = ExtractBasicHTML(rawHTML)
	res.Links = ExtractLinks(rawHTML, res.FinalURL, 200)
	res.HTML, res.DurationMs = rawHTML, time.Since(start).Milliseconds()
	return res
}

func (e *Engine) ScrapeBatch(ctx context.Context, urls []string, concurrency int) []ScrapeResult {
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 20 {
		concurrency = 20
	}
	if len(urls) > 100 {
		urls = urls[:100]
	}
	results := make([]ScrapeResult, len(urls))
	type workItem struct {
		index int
		url   string
	}
	jobs := make(chan workItem, len(urls))
	for i, rawURL := range urls {
		jobs <- workItem{i, rawURL}
	}
	close(jobs)
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if ctx.Err() != nil {
					results[item.index] = ScrapeResult{URL: item.url, Error: "operation cancelled"}
					continue
				}
				results[item.index] = e.ScrapeSingle(ctx, ScrapeJob{URL: item.url})
			}
		}()
	}
	wg.Wait()
	return results
}

func (e *Engine) CrawlSite(ctx context.Context, req CrawlRequest) CrawlResponse {
	maxPages, maxDepth, concurrency := req.MaxPages, req.MaxDepth, req.Concurrency
	if maxPages <= 0 {
		maxPages = 25
	}
	if maxPages > 100 {
		maxPages = 100
	}
	if maxDepth < 0 {
		maxDepth = 0
	}
	if maxDepth > 3 {
		maxDepth = 3
	}
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 20 {
		concurrency = 20
	}
	start, err := validatePublicURL(req.StartURL)
	if err != nil {
		return CrawlResponse{StartURL: req.StartURL, Results: []ScrapeResult{{URL: req.StartURL, Error: err.Error()}}}
	}
	seen := map[string]struct{}{start.String(): {}}
	frontier := []string{start.String()}
	all := make([]ScrapeResult, 0, maxPages)
	for depth := 0; depth <= maxDepth && len(frontier) > 0 && len(all) < maxPages; depth++ {
		remaining := maxPages - len(all)
		if len(frontier) > remaining {
			frontier = frontier[:remaining]
		}
		pages := e.ScrapeBatch(ctx, frontier, concurrency)
		all = append(all, pages...)
		next := make([]string, 0)
		for _, page := range pages {
			if page.Error != "" {
				continue
			}
			for _, link := range page.Links {
				u, err := validatePublicURL(link)
				if err != nil || !sameSite(start, u, req.AllowSubdomains) {
					continue
				}
				canonical := u.String()
				if _, exists := seen[canonical]; exists {
					continue
				}
				seen[canonical] = struct{}{}
				next = append(next, canonical)
				if len(next) >= maxPages-len(all) {
					break
				}
			}
		}
		frontier = next
	}
	for i := range all {
		all[i].HTML = ""
	}
	return CrawlResponse{StartURL: start.String(), Count: len(all), Results: all}
}

func sameSite(root, candidate *url.URL, allowSubdomains bool) bool {
	rootHost, candidateHost := strings.ToLower(root.Hostname()), strings.ToLower(candidate.Hostname())
	return candidateHost == rootHost || (allowSubdomains && strings.HasSuffix(candidateHost, "."+rootHost))
}
