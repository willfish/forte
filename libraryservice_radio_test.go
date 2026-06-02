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
	Name              string `json:"name"`
	StreamURL         string `json:"streamUrl"`
	Homepage          string `json:"homepage"`
	DerivedHomepage   string `json:"derivedHomepage"`
	FaviconURL        string `json:"faviconUrl"`
	Tags              string `json:"tags"`
	Country           string `json:"country"`
	Codec             string `json:"codec"`
	Bitrate           int    `json:"bitrate"`
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

func TestGetRadioStationByUUIDRadioBrowser(t *testing.T) {
	fx := loadRadioStationsFixture(t)
	station := fixtureToRadioStation(fx.RadioBrowser)
	rb := radioBrowserServerForStation(t, station)
	defer withRadioTestHooks(t, rb, nil)()

	s := openTestService(t)
	got, err := s.GetRadioStationByUUID(fx.RadioBrowser.UUID)
	if err != nil {
		t.Fatalf("GetRadioStationByUUID: %v", err)
	}
	if got.Name != fx.RadioBrowser.Name {
		t.Fatalf("Name = %q, want %q", got.Name, fx.RadioBrowser.Name)
	}
	if got.Homepage != fx.RadioBrowser.Homepage {
		t.Fatalf("Homepage = %q, want %q", got.Homepage, fx.RadioBrowser.Homepage)
	}
	if got.Codec != fx.RadioBrowser.Codec || got.Bitrate != fx.RadioBrowser.Bitrate {
		t.Fatalf("metadata = codec %q bitrate %d", got.Codec, got.Bitrate)
	}
}

func TestGetRadioStationByUUIDSomaFM(t *testing.T) {
	fx := loadRadioStationsFixture(t)
	soma := []radio.Station{fixtureToRadioStation(fx.SomaFM)}
	rb := emptyRadioBrowserServer(t)
	defer withRadioTestHooks(t, rb, soma)()

	s := openTestService(t)
	got, err := s.GetRadioStationByUUID(fx.SomaFM.UUID)
	if err != nil {
		t.Fatalf("GetRadioStationByUUID: %v", err)
	}
	if got.Name != fx.SomaFM.Name {
		t.Fatalf("Name = %q, want %q", got.Name, fx.SomaFM.Name)
	}
	if got.Homepage != fx.SomaFM.Homepage {
		t.Fatalf("Homepage = %q, want %q", got.Homepage, fx.SomaFM.Homepage)
	}
}

func TestGetRadioStationByUUIDCustom(t *testing.T) {
	fx := loadRadioStationsFixture(t)
	rb := emptyRadioBrowserServer(t)
	defer withRadioTestHooks(t, rb, nil)()

	s := openTestService(t)
	customUUID := library.CustomRadioStationUUID(fx.Custom.StreamURL)
	if _, err := s.db.AddCustomRadioStation(library.RadioCustomStation{
		StationUUID: customUUID,
		Name:        fx.Custom.Name,
		StreamURL:   fx.Custom.StreamURL,
		Tags:        fx.Custom.Tags,
	}); err != nil {
		t.Fatalf("AddCustomRadioStation: %v", err)
	}

	got, err := s.GetRadioStationByUUID(customUUID)
	if err != nil {
		t.Fatalf("GetRadioStationByUUID: %v", err)
	}
	if got.Name != fx.Custom.Name {
		t.Fatalf("Name = %q, want %q", got.Name, fx.Custom.Name)
	}
	if got.Homepage != fx.Custom.DerivedHomepage {
		t.Fatalf("Homepage = %q, want derived %q", got.Homepage, fx.Custom.DerivedHomepage)
	}
}

func TestGetRadioStationByUUIDFavouriteOnly(t *testing.T) {
	fx := loadRadioStationsFixture(t)
	rb := emptyRadioBrowserServer(t)
	defer withRadioTestHooks(t, rb, nil)()

	s := openTestService(t)
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

	got, err := s.GetRadioStationByUUID(fx.FavouriteOnly.UUID)
	if err != nil {
		t.Fatalf("GetRadioStationByUUID: %v", err)
	}
	if got.Country != fx.FavouriteOnly.Country || got.Codec != fx.FavouriteOnly.Codec {
		t.Fatalf("metadata mismatch: %+v", got)
	}
}

func TestGetRadioStationByUUIDHistoryOnly(t *testing.T) {
	fx := loadRadioStationsFixture(t)
	rb := emptyRadioBrowserServer(t)
	defer withRadioTestHooks(t, rb, nil)()

	s := openTestService(t)
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

	got, err := s.GetRadioStationByUUID(fx.HistoryOnly.UUID)
	if err != nil {
		t.Fatalf("GetRadioStationByUUID: %v", err)
	}
	if got.Name != fx.HistoryOnly.Name {
		t.Fatalf("Name = %q, want %q", got.Name, fx.HistoryOnly.Name)
	}
}

func TestGetRadioStationByUUIDRadioBrowserPrecedenceOverSaved(t *testing.T) {
	fx := loadRadioStationsFixture(t)
	rbStation := fixtureToRadioStation(fx.RadioBrowser)
	rbStation.Name = "Live RadioBrowser Name"
	rb := radioBrowserServerForStation(t, rbStation)
	defer withRadioTestHooks(t, rb, nil)()

	s := openTestService(t)
	if err := s.db.AddRadioFavourite(library.RadioFavourite{
		StationUUID: fx.RadioBrowser.UUID,
		Name:        "Stale Saved Name",
		StreamURL:   fx.RadioBrowser.StreamURL,
		Homepage:    fx.RadioBrowser.Homepage,
		Tags:        fx.RadioBrowser.Tags,
	}); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}

	got, err := s.GetRadioStationByUUID(fx.RadioBrowser.UUID)
	if err != nil {
		t.Fatalf("GetRadioStationByUUID: %v", err)
	}
	if got.Name != "Live RadioBrowser Name" {
		t.Fatalf("Name = %q, want live API name", got.Name)
	}
}

func TestGetRadioStationByUUIDNotFound(t *testing.T) {
	rb := emptyRadioBrowserServer(t)
	defer withRadioTestHooks(t, rb, nil)()

	s := openTestService(t)
	if _, err := s.GetRadioStationByUUID("missing-uuid"); err == nil {
		t.Fatal("expected error for missing station")
	}
}