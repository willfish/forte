// Package subsonic implements the Subsonic/OpenSubsonic streaming API client.
package subsonic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/willfish/forte/internal/streaming"
)

const (
	apiVersion = "1.16.1"
	clientName = "forte"
)

// Client is a Subsonic API client implementing streaming.Provider.
type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

// New creates a new Subsonic client.
func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

// NewWithHTTPClient creates a new Subsonic client with a custom HTTP client.
func NewWithHTTPClient(baseURL, username, password string, c *http.Client) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     c,
	}
}

// authParams returns URL query parameters for authentication.
func (c *Client) authParams() (url.Values, error) {
	salt, err := generateSalt(8)
	if err != nil {
		return nil, err
	}
	return url.Values{
		"u": {c.username},
		"t": {generateToken(c.password, salt)},
		"s": {salt},
		"v": {apiVersion},
		"c": {clientName},
		"f": {"json"},
	}, nil
}

// request makes an authenticated request to the Subsonic API.
func (c *Client) request(method string, params url.Values) (*responseBody, error) {
	auth, err := c.authParams()
	if err != nil {
		return nil, err
	}

	if params == nil {
		params = url.Values{}
	}
	for k, v := range auth {
		params[k] = v
	}

	reqURL := fmt.Sprintf("%s/rest/%s.view?%s", c.baseURL, method, params.Encode())
	resp, err := c.http.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("subsonic %s: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var envelope subsonicResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("subsonic %s: decode: %w", method, err)
	}

	body := &envelope.Response
	if body.Status != "ok" && body.Error != nil {
		return nil, body.Error
	}

	return body, nil
}

// Ping tests the connection to the Subsonic server.
func (c *Client) Ping() error {
	_, err := c.request("ping", nil)
	return err
}

// GetArtists returns all artists, flattening the indexed structure.
func (c *Client) GetArtists() ([]streaming.Artist, error) {
	body, err := c.request("getArtists", nil)
	if err != nil {
		return nil, err
	}

	if body.Artists == nil {
		return nil, nil
	}

	var artists []streaming.Artist
	for _, idx := range body.Artists.Index {
		for _, a := range idx.Artists {
			artists = append(artists, convertArtist(a))
		}
	}
	return artists, nil
}

// GetAlbums returns a paginated list of albums.
func (c *Client) GetAlbums(sortBy string, offset, count int) ([]streaming.Album, error) {
	body, err := c.request("getAlbumList2", url.Values{
		"type":   {sortBy},
		"offset": {fmt.Sprint(offset)},
		"size":   {fmt.Sprint(count)},
	})
	if err != nil {
		return nil, err
	}

	if body.AlbumList == nil {
		return nil, nil
	}

	albums := make([]streaming.Album, len(body.AlbumList.Albums))
	for i, a := range body.AlbumList.Albums {
		albums[i] = convertAlbum(a)
	}
	return albums, nil
}

// GetAlbum returns an album and its tracks.
func (c *Client) GetAlbum(id string) (streaming.Album, []streaming.Track, error) {
	body, err := c.request("getAlbum", url.Values{"id": {id}})
	if err != nil {
		return streaming.Album{}, nil, err
	}

	if body.Album == nil {
		return streaming.Album{}, nil, fmt.Errorf("subsonic getAlbum: no album in response")
	}

	album := convertAlbum(body.Album.albumJSON)
	tracks := make([]streaming.Track, len(body.Album.Songs))
	for i, s := range body.Album.Songs {
		tracks[i] = convertTrack(s)
	}
	return album, tracks, nil
}

// StreamURL returns an authenticated URL for streaming a track.
// No HTTP request is made; mpv handles the URL directly.
func (c *Client) StreamURL(trackID string) string {
	auth, err := c.authParams()
	if err != nil {
		return ""
	}
	auth.Set("id", trackID)
	return fmt.Sprintf("%s/rest/stream.view?%s", c.baseURL, auth.Encode())
}

// CoverArtURL returns an authenticated URL for fetching cover art.
// No HTTP request is made.
func (c *Client) CoverArtURL(coverArtID string) string {
	auth, err := c.authParams()
	if err != nil {
		return ""
	}
	auth.Set("id", coverArtID)
	return fmt.Sprintf("%s/rest/getCoverArt.view?%s", c.baseURL, auth.Encode())
}

// Search performs a search across artists, albums, and tracks.
func (c *Client) Search(query string) (streaming.SearchResults, error) {
	body, err := c.request("search3", url.Values{"query": {query}})
	if err != nil {
		return streaming.SearchResults{}, err
	}

	var results streaming.SearchResults
	if body.Search == nil {
		return results, nil
	}

	for _, a := range body.Search.Artists {
		results.Artists = append(results.Artists, convertArtist(a))
	}
	for _, a := range body.Search.Albums {
		results.Albums = append(results.Albums, convertAlbum(a))
	}
	for _, s := range body.Search.Songs {
		results.Tracks = append(results.Tracks, convertTrack(s))
	}
	return results, nil
}

// Compile-time check that Client implements streaming.Provider.
var _ streaming.Provider = (*Client)(nil)
