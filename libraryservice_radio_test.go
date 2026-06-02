package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/radio"
)

//go:embed testdata/radio/stations.json
var radioStationsFixtureJSON []byte

type radioStationsFixture struct {
	RadioBrowser  radioStationFixture `json:"radiobrowser"`
	SomaFM        radioStationFixture `json:"somafm"`
	Custom        customFixture       `json:"custom"`
	FavouriteOnly radioStationFixture `json:"favouriteOnly"`
	HistoryOnly   radioStationFixture `json:"historyOnly"`
}

type radioStationFixture struct {
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

type customFixture struct {
	Name            string `json:"name"`
	StreamURL       string `json:"streamUrl"`
	Homepage        string `json:"homepage"`
	DerivedHomepage string `json:"derivedHomepage"`
	FaviconURL      string `json:"faviconUrl"`
	Tags            string `json:"tags"`
	Country         string `json:"country"`
	Codec           string `json:"codec"`
	Bitrate         int    `json:"bitrate"`
}

func loadRadioStationsFixture(t *testing.T) radioStationsFixture {
	t.Helper()
	var fx radioStationsFixture
	if err := json.Unmarshal(radioStationsFixtureJSON, &fx); err != nil {
		t.Fatalf("unmarshal radio fixtures: %v", err)
	}
	return fx
}

func fixtureToRadioStation(f radioStationFixture) radio.Station {
	return radio.Station{
		UUID:      f.UUID,
		Name:      f.Name,
		StreamURL: f.StreamURL,
		Homepage:  f.Homepage,
		Favicon:   f.Favicon,
		Country:   f.Country,
		Tags:      f.Tags,
		Bitrate:   f.Bitrate,
		Codec:     f.Codec,
		Votes:     f.Votes,
		Clicks:    f.Clicks,
	}
}

func withRadioTestHooks(t *testing.T, rb *radio.Client, soma []radio.Station) func() {
	t.Helper()
	oldClient := radioClient
	oldSoma := somafmStationsProvider
	radioClient = rb
	if soma != nil {
		somafmStationsProvider = func() ([]radio.Station, error) { return soma, nil }
	} else {
		somafmStationsProvider = func() ([]radio.Station, error) { return nil, nil }
	}
	return func() {
		radioClient = oldClient
		somafmStationsProvider = oldSoma
	}
}

func emptyRadioBrowserServer(t *testing.T) *radio.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)
	c := radio.NewClient()
	c.SetServers([]string{server.URL})
	return c
}

func radioBrowserServerForStation(t *testing.T, station radio.Station) *radio.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/stations/byuuid" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]radio.Station{station})
	}))
	t.Cleanup(server.Close)
	c := radio.NewClient()
	c.SetServers([]string{server.URL})
	return c
}

func TestGetRadioStationByUUID(t *testing.T) {
	fx := loadRadioStationsFixture(t)

	tests := []struct {
		name      string
		uuid      func(fx radioStationsFixture) string
		rb        func(t *testing.T, fx radioStationsFixture) *radio.Client
		soma      func(fx radioStationsFixture) []radio.Station
		seed      func(t *testing.T, s *LibraryService, fx radioStationsFixture)
		wantName  func(fx radioStationsFixture) string
		wantHome  func(fx radioStationsFixture) string
		wantCodec func(fx radioStationsFixture) string
		wantErr   bool
	}{
		{
			name: "radiobrowser",
			uuid: func(fx radioStationsFixture) string { return fx.RadioBrowser.UUID },
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				return radioBrowserServerForStation(t, fixtureToRadioStation(fx.RadioBrowser))
			},
			wantName:  func(fx radioStationsFixture) string { return fx.RadioBrowser.Name },
			wantHome:  func(fx radioStationsFixture) string { return fx.RadioBrowser.Homepage },
			wantCodec: func(fx radioStationsFixture) string { return fx.RadioBrowser.Codec },
		},
		{
			name: "somafm",
			uuid: func(fx radioStationsFixture) string { return fx.SomaFM.UUID },
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				return emptyRadioBrowserServer(t)
			},
			soma: func(fx radioStationsFixture) []radio.Station {
				return []radio.Station{fixtureToRadioStation(fx.SomaFM)}
			},
			wantName: func(fx radioStationsFixture) string { return fx.SomaFM.Name },
			wantHome: func(fx radioStationsFixture) string { return fx.SomaFM.Homepage },
		},
		{
			name: "custom",
			uuid: func(fx radioStationsFixture) string {
				return library.CustomRadioStationUUID(fx.Custom.StreamURL)
			},
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				return emptyRadioBrowserServer(t)
			},
			seed: func(t *testing.T, s *LibraryService, fx radioStationsFixture) {
				if _, err := s.db.AddCustomRadioStation(library.RadioCustomStation{
					StationUUID: library.CustomRadioStationUUID(fx.Custom.StreamURL),
					Name:        fx.Custom.Name,
					StreamURL:   fx.Custom.StreamURL,
					Tags:        fx.Custom.Tags,
				}); err != nil {
					t.Fatalf("AddCustomRadioStation: %v", err)
				}
			},
			wantName: func(fx radioStationsFixture) string { return fx.Custom.Name },
			wantHome: func(fx radioStationsFixture) string { return fx.Custom.DerivedHomepage },
		},
		{
			name: "favourite_only",
			uuid: func(fx radioStationsFixture) string { return fx.FavouriteOnly.UUID },
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				return emptyRadioBrowserServer(t)
			},
			seed: func(t *testing.T, s *LibraryService, fx radioStationsFixture) {
				if err := s.db.AddRadioFavourite(library.RadioFavourite{
					StationUUID: fx.FavouriteOnly.UUID,
					Name:        fx.FavouriteOnly.Name,
					StreamURL:   fx.FavouriteOnly.StreamURL,
					Homepage:    fx.FavouriteOnly.Homepage,
					Country:     fx.FavouriteOnly.Country,
					Codec:       fx.FavouriteOnly.Codec,
					Bitrate:     fx.FavouriteOnly.Bitrate,
					Tags:        fx.FavouriteOnly.Tags,
				}); err != nil {
					t.Fatalf("AddRadioFavourite: %v", err)
				}
			},
			wantName:  func(fx radioStationsFixture) string { return fx.FavouriteOnly.Name },
			wantCodec: func(fx radioStationsFixture) string { return fx.FavouriteOnly.Codec },
		},
		{
			name: "history_only",
			uuid: func(fx radioStationsFixture) string { return fx.HistoryOnly.UUID },
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				return emptyRadioBrowserServer(t)
			},
			seed: func(t *testing.T, s *LibraryService, fx radioStationsFixture) {
				if err := s.db.RecordRadioPlayback(library.RadioHistoryEntry{
					StationUUID: fx.HistoryOnly.UUID,
					Name:        fx.HistoryOnly.Name,
					StreamURL:   fx.HistoryOnly.StreamURL,
					Homepage:    fx.HistoryOnly.Homepage,
					Country:     fx.HistoryOnly.Country,
					Codec:       fx.HistoryOnly.Codec,
					Bitrate:     fx.HistoryOnly.Bitrate,
					Tags:        fx.HistoryOnly.Tags,
				}); err != nil {
					t.Fatalf("RecordRadioPlayback: %v", err)
				}
			},
			wantName: func(fx radioStationsFixture) string { return fx.HistoryOnly.Name },
		},
		{
			name: "radiobrowser_over_saved",
			uuid: func(fx radioStationsFixture) string { return fx.RadioBrowser.UUID },
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				station := fixtureToRadioStation(fx.RadioBrowser)
				station.Name = "Live RadioBrowser Name"
				return radioBrowserServerForStation(t, station)
			},
			seed: func(t *testing.T, s *LibraryService, fx radioStationsFixture) {
				if err := s.db.AddRadioFavourite(library.RadioFavourite{
					StationUUID: fx.RadioBrowser.UUID,
					Name:        "Stale Saved Name",
					StreamURL:   fx.RadioBrowser.StreamURL,
					Homepage:    fx.RadioBrowser.Homepage,
					Tags:        fx.RadioBrowser.Tags,
				}); err != nil {
					t.Fatalf("AddRadioFavourite: %v", err)
				}
			},
			wantName: func(fx radioStationsFixture) string { return "Live RadioBrowser Name" },
		},
		{
			name: "not_found",
			uuid: func(fx radioStationsFixture) string { return "missing-uuid" },
			rb: func(t *testing.T, fx radioStationsFixture) *radio.Client {
				return emptyRadioBrowserServer(t)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rb *radio.Client
			if tt.rb != nil {
				rb = tt.rb(t, fx)
			} else {
				rb = emptyRadioBrowserServer(t)
			}
			var soma []radio.Station
			if tt.soma != nil {
				soma = tt.soma(fx)
			}
			defer withRadioTestHooks(t, rb, soma)()

			s := openTestService(t)
			if tt.seed != nil {
				tt.seed(t, s, fx)
			}

			got, err := s.GetRadioStationByUUID(tt.uuid(fx))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetRadioStationByUUID: %v", err)
			}
			if tt.wantName != nil {
				if want := tt.wantName(fx); got.Name != want {
					t.Fatalf("Name = %q, want %q", got.Name, want)
				}
			}
			if tt.wantHome != nil {
				if want := tt.wantHome(fx); got.Homepage != want {
					t.Fatalf("Homepage = %q, want %q", got.Homepage, want)
				}
			}
			if tt.wantCodec != nil {
				if want := tt.wantCodec(fx); got.Codec != want {
					t.Fatalf("Codec = %q, want %q", got.Codec, want)
				}
			}
		})
	}
}
