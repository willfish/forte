package library

import (
	"net"
	"net/url"
	"strings"
)

// DeriveHomepageFromStreamURL returns a best-effort website URL from a stream URL host.
func DeriveHomepageFromStreamURL(streamURL string) string {
	u, err := url.Parse(strings.TrimSpace(streamURL))
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".local") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil {
		return ""
	}
	return "https://" + host + "/"
}