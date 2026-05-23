package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var radioArtworkClient = &http.Client{Timeout: 2 * time.Second}

var radioArtworkCache struct {
	sync.Mutex
	m map[string]string
}

func resolveRadioArtwork(favicon, homepage string) string {
	if isHTTPURL(favicon) {
		return strings.TrimSpace(favicon)
	}
	if art := somafmClient.LookupArtwork(homepage); art != "" {
		return art
	}
	if !isHTTPURL(homepage) {
		return ""
	}

	key := strings.TrimSpace(homepage)
	radioArtworkCache.Lock()
	if radioArtworkCache.m == nil {
		radioArtworkCache.m = make(map[string]string)
	}
	if cached, ok := radioArtworkCache.m[key]; ok {
		radioArtworkCache.Unlock()
		return cached
	}
	radioArtworkCache.Unlock()

	resolved := resolveArtworkFromHomepage(key)

	radioArtworkCache.Lock()
	radioArtworkCache.m[key] = resolved
	radioArtworkCache.Unlock()
	return resolved
}

func resolveArtworkFromHomepage(homepage string) string {
	req, err := http.NewRequest(http.MethodGet, homepage, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Forte/0.1 radio artwork resolver")

	resp, err := radioArtworkClient.Do(req)
	if err != nil {
		return fallbackFaviconURL(homepage)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallbackFaviconURL(homepage)
	}

	base := resp.Request.URL
	doc, err := html.Parse(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return fallbackFaviconURL(homepage)
	}
	if art := firstHomepageArtwork(doc, base); art != "" {
		return art
	}
	return fallbackFaviconURL(homepage)
}

func firstHomepageArtwork(n *html.Node, base *url.URL) string {
	var icons []string
	var walk func(*html.Node) string
	walk = func(node *html.Node) string {
		if node.Type == html.ElementNode {
			switch strings.ToLower(node.Data) {
			case "meta":
				if isImageMeta(node) {
					if v := attr(node, "content"); v != "" {
						if resolved := resolveURL(base, v); resolved != "" {
							return resolved
						}
					}
				}
			case "link":
				if isIconLink(node) {
					if v := attr(node, "href"); v != "" {
						if resolved := resolveURL(base, v); resolved != "" {
							icons = append(icons, resolved)
						}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if resolved := walk(child); resolved != "" {
				return resolved
			}
		}
		return ""
	}
	if resolved := walk(n); resolved != "" {
		return resolved
	}
	if len(icons) > 0 {
		return icons[0]
	}
	return ""
}

func isImageMeta(n *html.Node) bool {
	property := strings.ToLower(attr(n, "property"))
	name := strings.ToLower(attr(n, "name"))
	return property == "og:image" ||
		property == "og:image:url" ||
		name == "twitter:image" ||
		name == "twitter:image:src"
}

func isIconLink(n *html.Node) bool {
	rel := strings.ToLower(attr(n, "rel"))
	if rel == "" {
		return false
	}
	for _, part := range strings.Fields(rel) {
		if part == "icon" || part == "apple-touch-icon" || part == "apple-touch-icon-precomposed" {
			return true
		}
	}
	return strings.Contains(rel, " icon")
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func resolveURL(base *url.URL, value string) string {
	if value == "" || base == nil {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

func fallbackFaviconURL(homepage string) string {
	u, err := url.Parse(homepage)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	u.Path = "/favicon.ico"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func isHTTPURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
