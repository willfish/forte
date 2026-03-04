// Package radio provides a client for the RadioBrowser API.
package radio

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Station represents a radio station from RadioBrowser.
type Station struct {
	UUID      string `json:"stationuuid"`
	Name      string `json:"name"`
	StreamURL string `json:"url_resolved"`
	Favicon   string `json:"favicon"`
	Country   string `json:"country"`
	Tags      string `json:"tags"`
	Bitrate   int    `json:"bitrate"`
	Codec     string `json:"codec"`
	Votes     int    `json:"votes"`
	Clicks    int    `json:"clickcount"`
}

// Client is a RadioBrowser API client with DNS-based mirror discovery.
type Client struct {
	httpClient *http.Client
	servers    []string
}

// NewClient creates a new RadioBrowser client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// discoverServers resolves RadioBrowser mirrors via DNS.
func (c *Client) discoverServers() ([]string, error) {
	if len(c.servers) > 0 {
		return c.servers, nil
	}

	addrs, err := net.LookupHost("all.api.radio-browser.info")
	if err != nil {
		return nil, fmt.Errorf("radiobrowser dns lookup: %w", err)
	}

	servers := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		names, err := net.LookupAddr(addr)
		if err != nil || len(names) == 0 {
			// Fall back to IP if reverse lookup fails.
			servers = append(servers, "https://"+addr)
			continue
		}
		host := names[0]
		// Remove trailing dot from DNS name.
		if len(host) > 0 && host[len(host)-1] == '.' {
			host = host[:len(host)-1]
		}
		servers = append(servers, "https://"+host)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("radiobrowser: no servers found")
	}

	c.servers = servers
	return servers, nil
}

// get makes a GET request to a random RadioBrowser mirror, falling back on failure.
func (c *Client) get(path string, params url.Values) ([]byte, error) {
	servers, err := c.discoverServers()
	if err != nil {
		return nil, err
	}

	// Shuffle for load distribution.
	shuffled := make([]string, len(servers))
	copy(shuffled, servers)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	var lastErr error
	for _, server := range shuffled {
		u := server + path
		if len(params) > 0 {
			u += "?" + params.Encode()
		}

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Forte/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("radiobrowser status %d: %s", resp.StatusCode, body)
			continue
		}

		return body, nil
	}

	return nil, fmt.Errorf("radiobrowser: all mirrors failed: %w", lastErr)
}

// Search searches for stations by name.
func (c *Client) Search(query string, limit int) ([]Station, error) {
	params := url.Values{
		"name":     {query},
		"limit":    {fmt.Sprintf("%d", limit)},
		"order":    {"votes"},
		"reverse":  {"true"},
		"hidebroken": {"true"},
	}
	return c.fetchStations("/json/stations/search", params)
}

// ByTag returns stations matching a tag (genre).
func (c *Client) ByTag(tag string, limit int) ([]Station, error) {
	params := url.Values{
		"tag":     {tag},
		"limit":   {fmt.Sprintf("%d", limit)},
		"order":   {"votes"},
		"reverse": {"true"},
		"hidebroken": {"true"},
	}
	return c.fetchStations("/json/stations/search", params)
}

// ByCountry returns stations matching a country.
func (c *Client) ByCountry(country string, limit int) ([]Station, error) {
	params := url.Values{
		"country":  {country},
		"limit":    {fmt.Sprintf("%d", limit)},
		"order":    {"votes"},
		"reverse":  {"true"},
		"hidebroken": {"true"},
	}
	return c.fetchStations("/json/stations/search", params)
}

// TopVoted returns the top voted stations.
func (c *Client) TopVoted(limit int) ([]Station, error) {
	params := url.Values{
		"limit":      {fmt.Sprintf("%d", limit)},
		"hidebroken": {"true"},
	}
	return c.fetchStations("/json/stations/topvote", params)
}

// TopClicked returns the most clicked stations.
func (c *Client) TopClicked(limit int) ([]Station, error) {
	params := url.Values{
		"limit":      {fmt.Sprintf("%d", limit)},
		"hidebroken": {"true"},
	}
	return c.fetchStations("/json/stations/topclick", params)
}

func (c *Client) fetchStations(path string, params url.Values) ([]Station, error) {
	body, err := c.get(path, params)
	if err != nil {
		return nil, err
	}
	return parseStations(body)
}

func parseStations(body []byte) ([]Station, error) {
	var stations []Station
	if err := json.Unmarshal(body, &stations); err != nil {
		return nil, fmt.Errorf("radiobrowser parse: %w", err)
	}
	return stations, nil
}
