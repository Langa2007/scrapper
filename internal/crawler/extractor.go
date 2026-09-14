package crawler

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var (
	scriptRegex = regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	styleRegex  = regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	tagRegex    = regexp.MustCompile(`<[^>]+>`)
	titleRegex  = regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	metaDesc    = regexp.MustCompile(`(?i)<meta\s+name=["']description["']\s+content=["'](.*?)["']`)
	hrefRegex   = regexp.MustCompile(`(?is)\bhref\s*=\s*["']([^"'#][^"']*)["']`)
	spacesRegex = regexp.MustCompile(`\s+`)
)

func ExtractBasicHTML(rawHTML string) (title string, desc string, cleanText string) {
	if match := titleRegex.FindStringSubmatch(rawHTML); len(match) > 1 {
		title = strings.TrimSpace(match[1])
	}

	if match := metaDesc.FindStringSubmatch(rawHTML); len(match) > 1 {
		desc = strings.TrimSpace(match[1])
	}

	noScript := scriptRegex.ReplaceAllString(rawHTML, " ")
	noStyle := styleRegex.ReplaceAllString(noScript, " ")
	noTags := tagRegex.ReplaceAllString(noStyle, " ")
	cleanText = spacesRegex.ReplaceAllString(noTags, " ")
	cleanText = strings.TrimSpace(cleanText)

	return title, desc, cleanText
}

func ExtractLinks(rawHTML, baseURL string, maxLinks int) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	links := make([]string, 0)
	for _, match := range hrefRegex.FindAllStringSubmatch(rawHTML, -1) {
		if len(match) < 2 {
			continue
		}
		u, err := base.Parse(strings.TrimSpace(html.UnescapeString(match[1])))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		u.Fragment = ""
		canonical := u.String()
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		links = append(links, canonical)
		if len(links) >= maxLinks {
			break
		}
	}
	return links
}
