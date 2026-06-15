package radio

import (
	"errors"
	"testing"

	"github.com/willfish/forte/internal/library"
)

type fakeStationDirectory struct {
	byUUID []Station
	err    error
}

func (f fakeStationDirectory) Search(string, int) ([]Station, error) {
	return nil, errors.New("unexpected search")
}

func (f fakeStationDirectory) SearchFiltered(string, string, string, int) ([]Station, error) {
	return nil, errors.New("unexpected filtered search")
}

func (f fakeStationDirectory) ByTag(string, int) ([]Station, error) {
	return nil, errors.New("unexpected tag search")
}

func (f fakeStationDirectory) ByCountry(string, int) ([]Station, error) {
	return nil, errors.New("unexpected country search")
}

func (f fakeStationDirectory) TopVoted(int) ([]Station, error) {
	return nil, errors.New("unexpected top voted")
}

func (f fakeStationDirectory) TopClicked(int) ([]Station, error) {
	return nil, errors.New("unexpected top clicked")
}

func (f fakeStationDirectory) ByUUID(string) ([]Station, error) {
	return f.byUUID, f.err
}

type fakeSavedStations struct {
	custom     []library.RadioCustomStation
	favourites []library.RadioFavourite
	history    []library.RadioHistoryEntry
}

func (f fakeSavedStations) GetCustomRadioStations() ([]library.RadioCustomStation, error) {
	return f.custom, nil
}

func (f fakeSavedStations) GetRadioFavourites() ([]library.RadioFavourite, error) {
	return f.favourites, nil
}

func (f fakeSavedStations) GetRadioHistory(int) ([]library.RadioHistoryEntry, error) {
	return f.history, nil
}

func TestCatalogLookupStationUsesLiveSourcesBeforeSavedFallbacks(t *testing.T) {
	catalog := NewCatalog(CatalogConfig{
		Directory: fakeStationDirectory{
			byUUID: []Station{{
				UUID:      "station-1",
				Name:      "Live name",
				StreamURL: "http://timesradio.wireless.radio/stream",
				Homepage:  " https://live.example/radio ",
				Favicon:   "https://live.example/icon.png",
				Codec:     "MP3",
			}},
		},
		SavedStations: fakeSavedStations{
			favourites: []library.RadioFavourite{{
				StationUUID: "station-1",
				Name:        "Saved name",
				StreamURL:   "https://saved.example/stream",
				Codec:       "AAC",
			}},
		},
		ResolveArtwork: func(favicon, homepage string) string {
			if favicon == "" {
				t.Fatalf("ResolveArtwork favicon = empty, homepage = %q", homepage)
			}
			return favicon
		},
	})

	got, err := catalog.LookupStation(" station-1 ")
	if err != nil {
		t.Fatalf("LookupStation: %v", err)
	}

	if got.Name != "Live name" {
		t.Fatalf("Name = %q, want live station", got.Name)
	}
	if got.StreamURL != "https://times.live.stream.broadcasting.news/stream" {
		t.Fatalf("StreamURL = %q, want normalized Times Radio URL", got.StreamURL)
	}
	if got.Homepage != "https://live.example/radio" {
		t.Fatalf("Homepage = %q, want trimmed homepage", got.Homepage)
	}
	if got.Codec != "MP3" {
		t.Fatalf("Codec = %q, want live codec", got.Codec)
	}
}

func TestCatalogLookupStationFallsBackThroughSavedStations(t *testing.T) {
	tests := []struct {
		name  string
		store fakeSavedStations
		uuid  string
		want  string
	}{
		{
			name: "custom first",
			uuid: "saved-custom",
			store: fakeSavedStations{
				custom: []library.RadioCustomStation{{
					StationUUID: "saved-custom",
					Name:        "Custom station",
					StreamURL:   "https://custom.example/stream",
				}},
			},
			want: "Custom station",
		},
		{
			name: "favourite second",
			uuid: "saved-fav",
			store: fakeSavedStations{
				favourites: []library.RadioFavourite{{
					StationUUID: "saved-fav",
					Name:        "Favourite station",
					StreamURL:   "https://fav.example/stream",
				}},
			},
			want: "Favourite station",
		},
		{
			name: "history third",
			uuid: "saved-history",
			store: fakeSavedStations{
				history: []library.RadioHistoryEntry{{
					StationUUID: "saved-history",
					Name:        "History station",
					StreamURL:   "https://history.example/stream",
				}},
			},
			want: "History station",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := NewCatalog(CatalogConfig{
				Directory:      fakeStationDirectory{},
				SavedStations:  tt.store,
				ResolveArtwork: func(favicon, homepage string) string { return favicon },
			})

			got, err := catalog.LookupStation(tt.uuid)
			if err != nil {
				t.Fatalf("LookupStation: %v", err)
			}
			if got.Name != tt.want {
				t.Fatalf("Name = %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestCatalogLookupStationUsesSomaFMForSomaUUIDs(t *testing.T) {
	catalog := NewCatalog(CatalogConfig{
		Directory: fakeStationDirectory{},
		SomaFMStations: func() ([]Station, error) {
			return []Station{{
				UUID:      "somafm-groovesalad",
				Name:      "Groove Salad",
				StreamURL: "https://somafm.example/groovesalad.mp3",
				Homepage:  "https://somafm.com/groovesalad/",
			}}, nil
		},
		ResolveArtwork: func(favicon, homepage string) string { return "resolved:" + homepage },
	})

	got, err := catalog.LookupStation("somafm-groovesalad")
	if err != nil {
		t.Fatalf("LookupStation: %v", err)
	}

	if got.Name != "Groove Salad" {
		t.Fatalf("Name = %q, want SomaFM station", got.Name)
	}
	if got.Favicon != "resolved:https://somafm.com/groovesalad/" {
		t.Fatalf("Favicon = %q, want resolved artwork", got.Favicon)
	}
}

func TestCatalogLookupStationReportsMissingStations(t *testing.T) {
	catalog := NewCatalog(CatalogConfig{
		Directory:      fakeStationDirectory{},
		SavedStations:  fakeSavedStations{},
		ResolveArtwork: func(favicon, homepage string) string { return favicon },
	})

	if _, err := catalog.LookupStation("missing"); err == nil {
		t.Fatal("expected missing station error")
	}
}
