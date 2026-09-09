package crawler

import (
	"regexp"
	"strings"
)

var (
	scriptRegex = regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	styleRegex  = regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	tagRegex    = regexp.MustCompile(`<[^>]+>`)
	titleRegex  = regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	metaDesc    = regexp.MustCompile(`(?i)<meta\s+name=["']description["']\s+content=["'](.*?)["']`)
	spacesRegex = regexp.MustCompile(`\s+`)
)

// ExtractBasicHTML extracts title, description, and clean text using regex without heavy dependencies.
func ExtractBasicHTML(rawHTML string) (title string, desc string, cleanText string) {
	// Extract Title
	if match := titleRegex.FindStringSubmatch(rawHTML); len(match) > 1 {
		title = strings.TrimSpace(match[1])
	}

	// Extract Description
	if match := metaDesc.FindStringSubmatch(rawHTML); len(match) > 1 {
		desc = strings.TrimSpace(match[1])
	}

	// Clean body text
	noScript := scriptRegex.ReplaceAllString(rawHTML, " ")
	noStyle := styleRegex.ReplaceAllString(noScript, " ")
	noTags := tagRegex.ReplaceAllString(noStyle, " ")
	cleanText = spacesRegex.ReplaceAllString(noTags, " ")
	cleanText = strings.TrimSpace(cleanText)

	return title, desc, cleanText
}
