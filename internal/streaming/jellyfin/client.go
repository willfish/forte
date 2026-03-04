// Package jellyfin implements the Jellyfin streaming API client.
package jellyfin

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/willfish/forte/internal/streaming"
)

// Client is a Jellyfin API client implementing streaming.Provider.
type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client

	token   string
	userID  string
	once    sync.Once
	authErr error
}

// New creates a new Jellyfin client.
func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

// NewWithHTTPClient creates a new Jellyfin client with a custom HTTP client.
func NewWithHTTPClient(baseURL, username, password string, c *http.Client) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     c,
	}
}

// deviceID returns a deterministic device identifier based on the server URL and username.
func (c *Client) deviceID() string {
	h := sha256.Sum256([]byte(c.baseURL + c.username))
	return fmt.Sprintf("%x", h[:8])
}

// authHeader returns the MediaBrowser Authorization header value.
// If authenticated, the token is included.
func (c *Client) authHeader() string {
	h := fmt.Sprintf(`MediaBrowser Client="forte", Device="Desktop", DeviceId="%s", Version="1.0.0"`, c.deviceID())
	if c.token != "" {
		h += fmt.Sprintf(`, Token="%s"`, c.token)
	}
	return h
}

// authenticate performs a one-time login to the Jellyfin server.
func (c *Client) authenticate() {
	c.once.Do(func() {
		body := fmt.Sprintf(`{"Username":%q,"Pw":%q}`, c.username, c.password)
		req, err := http.NewRequest("POST", c.baseURL+"/Users/AuthenticateByName", strings.NewReader(body))
		if err != nil {
			c.authErr = fmt.Errorf("jellyfin auth: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", c.authHeader())

		resp, err := c.http.Do(req)
		if err != nil {
			c.authErr = fmt.Errorf("jellyfin auth: %w", err)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			c.authErr = fmt.Errorf("jellyfin auth: status %d", resp.StatusCode)
			return
		}

		var authResp authResponse
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			c.authErr = fmt.Errorf("jellyfin auth: decode: %w", err)
			return
		}

		c.token = authResp.AccessToken
		c.userID = authResp.User.ID
	})
}

// request makes an authenticated GET request to the Jellyfin API.
func (c *Client) request(path string, params url.Values, target any) error {
	c.authenticate()
	if c.authErr != nil {
		return c.authErr
	}

	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("jellyfin %s: %w", path, err)
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jellyfin %s: status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("jellyfin %s: decode: %w", path, err)
	}

	return nil
}

// userPath returns a path under the authenticated user's namespace.
func (c *Client) userPath(suffix string) string {
	return fmt.Sprintf("/Users/%s%s", c.userID, suffix)
}

// Ping tests the connection to the Jellyfin server.
// Does not require authentication.
func (c *Client) Ping() error {
	resp, err := c.http.Get(c.baseURL + "/System/Ping")
	if err != nil {
		return fmt.Errorf("jellyfin ping: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jellyfin ping: status %d", resp.StatusCode)
	}
	return nil
}

// GetArtists returns all album artists.
func (c *Client) GetArtists() ([]streaming.Artist, error) {
	c.authenticate()
	if c.authErr != nil {
		return nil, c.authErr
	}

	var result itemsResponse
	if err := c.request("/Artists/AlbumArtists", url.Values{"userId": {c.userID}}, &result); err != nil {
		return nil, err
	}

	artists := make([]streaming.Artist, len(result.Items))
	for i, item := range result.Items {
		artists[i] = convertArtist(item)
	}
	return artists, nil
}

// GetAlbums returns a paginated list of albums.
func (c *Client) GetAlbums(sortBy string, offset, count int) ([]streaming.Album, error) {
	c.authenticate()
	if c.authErr != nil {
		return nil, c.authErr
	}

	var result itemsResponse
	if err := c.request(c.userPath("/Items"), url.Values{
		"IncludeItemTypes": {"MusicAlbum"},
		"Recursive":        {"true"},
		"SortBy":           {mapSortBy(sortBy)},
		"StartIndex":       {fmt.Sprint(offset)},
		"Limit":            {fmt.Sprint(count)},
	}, &result); err != nil {
		return nil, err
	}

	albums := make([]streaming.Album, len(result.Items))
	for i, item := range result.Items {
		albums[i] = convertAlbum(item)
	}
	return albums, nil
}

// GetAlbum returns an album and its tracks.
func (c *Client) GetAlbum(id string) (streaming.Album, []streaming.Track, error) {
	c.authenticate()
	if c.authErr != nil {
		return streaming.Album{}, nil, c.authErr
	}

	var albumItem itemJSON
	if err := c.request(c.userPath("/Items/"+id), nil, &albumItem); err != nil {
		return streaming.Album{}, nil, err
	}

	var tracksResult itemsResponse
	if err := c.request(c.userPath("/Items"), url.Values{
		"ParentId": {id},
		"SortBy":   {"ParentIndexNumber,IndexNumber"},
	}, &tracksResult); err != nil {
		return streaming.Album{}, nil, err
	}

	album := convertAlbum(albumItem)
	tracks := make([]streaming.Track, len(tracksResult.Items))
	for i, item := range tracksResult.Items {
		tracks[i] = convertTrack(item)
	}
	return album, tracks, nil
}

// StreamURL returns a URL for streaming a track.
// No HTTP request is made; mpv handles the URL directly.
func (c *Client) StreamURL(trackID string) string {
	return fmt.Sprintf("%s/Audio/%s/stream?static=true&api_key=%s", c.baseURL, trackID, c.token)
}

// CoverArtURL returns a URL for fetching cover art.
// No HTTP request is made; Jellyfin serves images without auth.
func (c *Client) CoverArtURL(coverArtID string) string {
	return fmt.Sprintf("%s/Items/%s/Images/Primary?maxWidth=300", c.baseURL, coverArtID)
}

// Search performs a search across artists, albums, and tracks.
func (c *Client) Search(query string) (streaming.SearchResults, error) {
	c.authenticate()
	if c.authErr != nil {
		return streaming.SearchResults{}, c.authErr
	}

	var result itemsResponse
	if err := c.request(c.userPath("/Items"), url.Values{
		"SearchTerm":       {query},
		"IncludeItemTypes": {"MusicArtist,MusicAlbum,Audio"},
		"Recursive":        {"true"},
		"Limit":            {"20"},
	}, &result); err != nil {
		return streaming.SearchResults{}, err
	}

	var results streaming.SearchResults
	for _, item := range result.Items {
		switch item.Type {
		case "MusicArtist":
			results.Artists = append(results.Artists, convertArtist(item))
		case "MusicAlbum":
			results.Albums = append(results.Albums, convertAlbum(item))
		case "Audio":
			results.Tracks = append(results.Tracks, convertTrack(item))
		}
	}
	return results, nil
}

// mapSortBy converts provider-agnostic sort names to Jellyfin SortBy values.
func mapSortBy(sortBy string) string {
	switch sortBy {
	case "alphabeticalByName":
		return "SortName"
	case "newest":
		return "DateCreated"
	case "recent":
		return "DatePlayed"
	case "frequent":
		return "PlayCount"
	case "random":
		return "Random"
	default:
		return "SortName"
	}
}

// Compile-time check that Client implements streaming.Provider.
var _ streaming.Provider = (*Client)(nil)
