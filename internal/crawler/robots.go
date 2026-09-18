package crawler

import (
	"bufio"
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type robotsRule struct {
	path  string
	allow bool
}
type robotsEntry struct {
	rules     []robotsRule
	expiresAt time.Time
}
type robotsCache struct {
	mu      sync.Mutex
	entries map[string]robotsEntry
}

func newRobotsCache() *robotsCache { return &robotsCache{entries: make(map[string]robotsEntry)} }

func (c *robotsCache) allowed(ctx context.Context, client *http.Client, target *url.URL, waitForHost func(context.Context, string) error) bool {
	origin := target.Scheme + "://" + target.Host
	c.mu.Lock()
	entry, found := c.entries[origin]
	c.mu.Unlock()
	if !found || time.Now().After(entry.expiresAt) {
		entry = robotsEntry{rules: fetchRobots(ctx, client, origin, waitForHost), expiresAt: time.Now().Add(time.Hour)}
		c.mu.Lock()
		c.entries[origin] = entry
		c.mu.Unlock()
	}
	path := target.EscapedPath()
	if path == "" {
		path = "/"
	}
	bestLength, allowed := -1, true
	for _, rule := range entry.rules {
		if strings.HasPrefix(path, rule.path) && len(rule.path) >= bestLength {
			if len(rule.path) > bestLength || rule.allow {
				allowed = rule.allow
			}
			bestLength = len(rule.path)
		}
	}
	return allowed
}

func fetchRobots(ctx context.Context, client *http.Client, origin string, waitForHost func(context.Context, string) error) []robotsRule {
	u, err := url.Parse(origin)
	if err != nil || waitForHost(ctx, u.Hostname()) != nil {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/robots.txt", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var rules []robotsRule
	matchingAgent := false
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.ToLower(strings.TrimSpace(parts[0])), strings.TrimSpace(parts[1])
		switch key {
		case "user-agent":
			matchingAgent = value == "*" || strings.Contains(strings.ToLower(DefaultUserAgent), strings.ToLower(value))
		case "allow", "disallow":
			if matchingAgent && value != "" {
				rules = append(rules, robotsRule{path: value, allow: key == "allow"})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	return rules
}
