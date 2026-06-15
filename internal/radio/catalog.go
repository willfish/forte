package radio

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/willfish/forte/internal/library"
)

// StationView is the JSON-friendly station projection exposed to the frontend.
type StationView struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	StreamURL string `json:"streamUrl"`
	Homepage  string `json:"homepage"`
	Favicon   string `json:"favicon"`
	Country   string `json:"country"`
	Tags      string `json:"tags"`
	Bitrate   int    `json:"bitrate"`
	Codec     string `json:"codec"`
	Votes     int    `json:"votes"`
	Clicks    int    `json:"clicks"`
}

// StationDirectory is the live station lookup interface backed by RadioBrowser.
type StationDirectory interface {
	Search(query string, limit int) ([]Station, error)
	SearchFiltered(country, codec, tag string, limit int) ([]Station, error)
	ByTag(tag string, limit int) ([]Station, error)
	ByCountry(country string, limit int) ([]Station, error)
	TopVoted(limit int) ([]Station, error)
	TopClicked(limit int) ([]Station, error)
	ByUUID(stationUUID string) ([]Station, error)
}

// SavedStationStore is the persisted station subset needed by Catalog.
type SavedStationStore interface {
	GetCustomRadioStations() ([]library.RadioCustomStation, error)
	GetRadioFavourites() ([]library.RadioFavourite, error)
	GetRadioHistory(limit int) ([]library.RadioHistoryEntry, error)
}

// CatalogConfig wires a radio catalog to live and persisted station sources.
type CatalogConfig struct {
	Directory      StationDirectory
	SomaFMStations func() ([]Station, error)
	SavedStations  SavedStationStore
	ResolveArtwork func(favicon, homepage string) string
}

// Catalog coordinates live radio lookup, saved station fallback, and station projection.
type Catalog struct {
	directory      StationDirectory
	somaFMStations func() ([]Station, error)
	savedStations  SavedStationStore
	resolveArtwork func(favicon, homepage string) string
}

// NewCatalog creates a radio catalog from configured station sources.
func NewCatalog(config CatalogConfig) *Catalog {
	resolveArtwork := config.ResolveArtwork
	if resolveArtwork == nil {
		resolveArtwork = func(favicon, _ string) string { return favicon }
	}
	return &Catalog{
		directory:      config.Directory,
		somaFMStations: config.SomaFMStations,
		savedStations:  config.SavedStations,
		resolveArtwork: resolveArtwork,
	}
}

// LookupStation returns full station metadata by UUID. Live metadata wins over saved snapshots.
func (c *Catalog) LookupStation(stationUUID string) (StationView, error) {
	stationUUID = strings.TrimSpace(stationUUID)
	if stationUUID == "" {
		return StationView{}, fmt.Errorf("station uuid is required")
	}

	if c.directory != nil {
		if stations, err := c.directory.ByUUID(stationUUID); err == nil && len(stations) > 0 {
			return c.ProjectStations(stations)[0], nil
		}
	}

	if strings.HasPrefix(stationUUID, "somafm-") && c.somaFMStations != nil {
		stations, err := c.somaFMStations()
		if err != nil {
			return StationView{}, err
		}
		for _, station := range stations {
			if station.UUID == stationUUID {
				return c.ProjectStations([]Station{station})[0], nil
			}
		}
	}

	if c.savedStations != nil {
		if station, ok, err := c.lookupSavedStation(stationUUID); err != nil {
			return StationView{}, err
		} else if ok {
			return station, nil
		}
	}

	return StationView{}, fmt.Errorf("station not found")
}

// Search returns projected stations matching a name query.
func (c *Catalog) Search(query string, limit int) ([]StationView, error) {
	stations, err := c.directory.Search(query, limit)
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// SearchFiltered returns projected stations matching optional filters.
func (c *Catalog) SearchFiltered(country, codec, tag string, limit int) ([]StationView, error) {
	stations, err := c.directory.SearchFiltered(country, codec, tag, limit)
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// ByTag returns projected stations for a tag.
func (c *Catalog) ByTag(tag string, limit int) ([]StationView, error) {
	stations, err := c.directory.ByTag(tag, limit)
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// ByCountry returns projected stations for a country.
func (c *Catalog) ByCountry(country string, limit int) ([]StationView, error) {
	stations, err := c.directory.ByCountry(country, limit)
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// TopVoted returns projected top-voted stations.
func (c *Catalog) TopVoted(limit int) ([]StationView, error) {
	stations, err := c.directory.TopVoted(limit)
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// TopClicked returns projected top-clicked stations.
func (c *Catalog) TopClicked(limit int) ([]StationView, error) {
	stations, err := c.directory.TopClicked(limit)
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// SomaFM returns projected SomaFM stations.
func (c *Catalog) SomaFM() ([]StationView, error) {
	stations, err := c.somaFMStations()
	if err != nil {
		return nil, err
	}
	return c.ProjectStations(stations), nil
}

// ProjectStations normalizes live station metadata for frontend use.
func (c *Catalog) ProjectStations(stations []Station) []StationView {
	result := make([]StationView, len(stations))
	favicons := make([]string, len(stations))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, station := range stations {
		i, station := i, station
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			favicons[i] = c.resolveArtwork(station.Favicon, station.Homepage)
		}()
	}
	wg.Wait()
	for i, station := range stations {
		result[i] = StationView{
			UUID:      station.UUID,
			Name:      station.Name,
			StreamURL: NormalizeStreamURL(station.StreamURL),
			Homepage:  strings.TrimSpace(station.Homepage),
			Favicon:   favicons[i],
			Country:   station.Country,
			Tags:      station.Tags,
			Bitrate:   station.Bitrate,
			Codec:     station.Codec,
			Votes:     station.Votes,
			Clicks:    station.Clicks,
		}
	}
	return result
}

func (c *Catalog) lookupSavedStation(stationUUID string) (StationView, bool, error) {
	custom, err := c.savedStations.GetCustomRadioStations()
	if err != nil {
		return StationView{}, false, err
	}
	for _, station := range custom {
		if station.StationUUID == stationUUID {
			return c.projectSavedStation(stationUUID, station.Name, station.StreamURL, station.FaviconURL, station.Homepage, station.Tags, station.Country, station.Bitrate, station.Codec), true, nil
		}
	}

	favourites, err := c.savedStations.GetRadioFavourites()
	if err != nil {
		return StationView{}, false, err
	}
	for _, favourite := range favourites {
		if favourite.StationUUID == stationUUID {
			return c.projectSavedStation(favourite.StationUUID, favourite.Name, favourite.StreamURL, favourite.FaviconURL, favourite.Homepage, favourite.Tags, favourite.Country, favourite.Bitrate, favourite.Codec), true, nil
		}
	}

	history, err := c.savedStations.GetRadioHistory(200)
	if err != nil {
		return StationView{}, false, err
	}
	for _, entry := range history {
		if entry.StationUUID == stationUUID {
			return c.projectSavedStation(entry.StationUUID, entry.Name, entry.StreamURL, entry.FaviconURL, entry.Homepage, entry.Tags, entry.Country, entry.Bitrate, entry.Codec), true, nil
		}
	}

	return StationView{}, false, nil
}

func (c *Catalog) projectSavedStation(stationUUID, name, streamURL, faviconURL, homepage, tags, country string, bitrate int, codec string) StationView {
	favicon := faviconURL
	if favicon == "" && homepage != "" {
		favicon = c.resolveArtwork("", homepage)
	}
	return StationView{
		UUID:      stationUUID,
		Name:      name,
		StreamURL: NormalizeStreamURL(streamURL),
		Homepage:  strings.TrimSpace(homepage),
		Favicon:   favicon,
		Country:   country,
		Tags:      tags,
		Bitrate:   bitrate,
		Codec:     codec,
	}
}

// NormalizeStreamURL applies known station stream URL migrations.
func NormalizeStreamURL(streamURL string) string {
	u, err := url.Parse(streamURL)
	if err != nil {
		return streamURL
	}
	switch strings.ToLower(u.Hostname()) {
	case "timesradio.wireless.radio":
		return "https://times.live.stream.broadcasting.news/stream"
	default:
		return streamURL
	}
}
