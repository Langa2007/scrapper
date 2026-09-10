package crawler

import "testing"

func TestValidatePublicURLRejectsNonPublicTargets(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/admin",
		"http://10.0.0.2/",
		"http://169.254.169.254/latest/meta-data",
		"file:///etc/passwd",
		"https://user:secret@example.com/",
	}
	for _, rawURL := range cases {
		if _, err := validatePublicURL(rawURL); err == nil {
			t.Errorf("expected %q to be rejected", rawURL)
		}
	}
	if _, err := validatePublicURL("https://example.com/articles?q=go"); err != nil {
		t.Fatalf("expected public URL to be accepted: %v", err)
	}
}

func TestExtractLinksResolvesAndDeduplicates(t *testing.T) {
	html := `<a href="/about">About</a><a href="https://example.com/docs#top">Docs</a><a href="/about">Again</a><a href="mailto:hi@example.com">Mail</a>`
	links := ExtractLinks(html, "https://example.com/start", 10)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %#v", links)
	}
	if links[0] != "https://example.com/about" || links[1] != "https://example.com/docs" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestSameSite(t *testing.T) {
	root, _ := validatePublicURL("https://example.com/")
	same, _ := validatePublicURL("https://example.com/docs")
	subdomain, _ := validatePublicURL("https://docs.example.com/")
	if !sameSite(root, same, false) {
		t.Fatal("expected same host to be included")
	}
	if sameSite(root, subdomain, false) || !sameSite(root, subdomain, true) {
		t.Fatal("unexpected subdomain scope result")
	}
}
