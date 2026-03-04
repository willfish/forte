// Command demo seeds the Forte database with realistic fixture data
// for screenshots and demonstrations. Safe to run multiple times.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/willfish/forte/internal/library"
)

func main() {
	dbPath := defaultDBPath()
	if p := os.Getenv("FORTE_DB"); p != "" {
		dbPath = p
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create dir: %v", err)
	}

	db, err := library.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Check if fixtures already seeded by looking for the sentinel artist.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM artists WHERE name = 'Radiohead'").Scan(&count); err != nil {
		log.Fatalf("check seed: %v", err)
	}
	if count > 0 {
		fmt.Println("Fixtures already seeded. Delete the database to re-seed.")
		return
	}

	seed(db)
	fmt.Printf("Seeded demo data at %s\n", dbPath)
}

func defaultDBPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("config dir: %v", err)
	}
	return filepath.Join(dir, "forte", "library.db")
}

// artwork generates a coloured JPEG placeholder.
func artwork(r, g, b uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	c := color.RGBA{R: r, G: g, B: b, A: 255}
	for y := range 300 {
		for x := range 300 {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func exec(db *library.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		log.Fatalf("exec %q: %v", query, err)
	}
}

func seed(db *library.DB) {
	type album struct {
		id       int
		artist   string
		title    string
		year     int
		tracks   []string
		durations []int // ms per track
		r, g, b  uint8 // artwork colour
	}

	albums := []album{
		{1, "Radiohead", "OK Computer", 1997, []string{
			"Airbag", "Paranoid Android", "Subterranean Homesick Alien", "Exit Music (For a Film)",
			"Let Down", "Karma Police", "Fitter Happier", "Electioneering",
			"Climbing Up the Walls", "No Surprises", "Lucky", "The Tourist",
		}, nil, 30, 60, 90},
		{2, "Radiohead", "Kid A", 2000, []string{
			"Everything in Its Right Place", "Kid A", "The National Anthem",
			"How to Disappear Completely", "Treefingers", "Optimistic",
			"In Limbo", "Idioteque", "Morning Bell", "Motion Picture Soundtrack",
		}, nil, 200, 200, 200},
		{3, "Radiohead", "In Rainbows", 2007, []string{
			"15 Step", "Bodysnatchers", "Nude", "Weird Fishes/Arpeggi",
			"All I Need", "Faust Arp", "Reckoner", "House of Cards",
			"Jigsaw Falling into Place", "Videotape",
		}, nil, 200, 50, 30},
		{4, "Bjork", "Homogenic", 1997, []string{
			"Hunter", "Joga", "Unravel", "Bachelorette", "All Neon Like",
			"5 Years", "Immature", "Alarm Call", "Pluto", "All Is Full of Love",
		}, nil, 180, 140, 100},
		{5, "Bjork", "Vespertine", 2001, []string{
			"Hidden Place", "Cocoon", "It's Not Up to You", "Undo",
			"Pagan Poetry", "Frosti", "Aurora", "An Echo, a Stain",
			"Sun in My Mouth", "Heirloom", "Harm of Will", "Unison",
		}, nil, 240, 240, 250},
		{6, "Portishead", "Dummy", 1994, []string{
			"Mysterons", "Sour Times", "Strangers", "It Could Be Sweet",
			"Wandering Star", "It's a Fire", "Numb", "Roads",
			"Pedestal", "Biscuit", "Glory Box",
		}, nil, 40, 40, 50},
		{7, "Portishead", "Third", 2008, []string{
			"Silence", "Hunter", "Nylon Smile", "The Rip",
			"Plastic", "We Carry On", "Deep Water", "Machine Gun",
			"Small", "Magic Doors", "Threads",
		}, nil, 20, 20, 20},
		{8, "Massive Attack", "Mezzanine", 1998, []string{
			"Angel", "Risingson", "Teardrop", "Inertia Creeps",
			"Exchange", "Dissolved Girl", "Man Next Door",
			"Black Milk", "Mezzanine", "Group Four", "(Exchange)",
		}, nil, 10, 10, 15},
		{9, "Boards of Canada", "Music Has the Right to Children", 1998, []string{
			"Wildlife Analysis", "An Eagle in Your Mind", "The Color of the Fire",
			"Telephasic Workshop", "Triangles & Rhombuses", "Sixtyten",
			"Turquoise Hexagon Sun", "Kaini Industries", "Bocuma",
			"Roygbiv", "Rue the Whirl", "Aquarius", "Olson",
			"Pete Standing Alone", "Smokes Quantity", "Open the Light",
			"One Very Important Thought",
		}, nil, 80, 140, 90},
		{10, "Aphex Twin", "Selected Ambient Works 85-92", 1992, []string{
			"Xtal", "Tha", "Pulsewidth", "Ageispolis", "i",
			"Green Calx", "Heliosphan", "We Are the Music Makers",
			"Schottkey 7th Path", "Ptolemy", "Hedphelym",
			"Delphium", "Actium",
		}, nil, 120, 80, 160},
		{11, "Aphex Twin", "Richard D. James Album", 1996, []string{
			"4", "Cornish Acid", "Peek 824545201",
			"Fingerbib", "Corn Mouth", "To Cure a Weakling Child",
			"Goon Gumpas", "Yellow Calx", "Girl/Boy Song", "Logon Rock Witch",
		}, nil, 200, 170, 50},
		{12, "Sigur Ros", "Agaetis Byrjun", 1999, []string{
			"Intro", "Svefn-g-englar", "Staralfur", "Flugufrelsarinn",
			"Ny Batteri", "Hjartad Hamast", "Vidrar Vel Til Loftarasa",
			"Olsen Olsen", "Agaetis Byrjun", "Avalon",
		}, nil, 200, 210, 220},
		{13, "Mogwai", "Young Team", 1997, []string{
			"Yes! I Am a Long Way from Home", "Like Herod",
			"Katrien", "Radar Maker", "Tracy", "Summer",
			"With Portfolio", "R U Still in 2 It?", "A Cheery Wave from Stranded Youngsters",
			"Mogwai Fear Satan",
		}, nil, 100, 100, 100},
		{14, "Nick Drake", "Pink Moon", 1972, []string{
			"Pink Moon", "Place to Be", "Road", "Which Will",
			"Horn", "Things Behind the Sun", "Know", "Parasite",
			"Ride", "Harvest Breed", "From the Morning",
		}, nil, 220, 150, 170},
		{15, "Talk Talk", "Spirit of Eden", 1988, []string{
			"The Rainbow", "Eden", "Desire", "Inheritance", "I Believe in You", "Wealth",
		}, nil, 60, 100, 60},
		{16, "Cocteau Twins", "Heaven or Las Vegas", 1990, []string{
			"Cherry-Coloured Funk", "Pitch the Baby", "Iceblink Luck",
			"Fifty-Fifty Clown", "Heaven or Las Vegas", "I Wear Your Ring",
			"Fotzepolitic", "Wolf in the Breast", "Road, River and Rail", "Frou-Frou Foxes in Midsummer Fires",
		}, nil, 100, 50, 150},
		{17, "My Bloody Valentine", "Loveless", 1991, []string{
			"Only Shallow", "Loomer", "Touched", "To Here Knows When",
			"When You Sleep", "I Only Said", "Come in Alone",
			"Sometimes", "Blown a Wish", "What You Want", "Soon",
		}, nil, 200, 60, 100},
		{18, "Brian Eno", "Music for Airports", 1978, []string{
			"1/1", "2/1", "1/2", "2/2",
		}, nil, 200, 220, 240},
		{19, "Burial", "Untrue", 2007, []string{
			"Archangel", "Near Dark", "Ghost Hardware", "Endorphin",
			"Etched Headplate", "In McDonalds", "Untrue", "Shell of Light",
			"Dog Shelter", "Homeless", "UK", "Raver",
		}, nil, 30, 30, 40},
		{20, "The Cure", "Disintegration", 1989, []string{
			"Plainsong", "Pictures of You", "Closedown", "Lovesong",
			"Last Dance", "Lullaby", "Fascination Street", "Prayers for Rain",
			"The Same Deep Water as You", "Disintegration", "Homesick", "Untitled",
		}, nil, 50, 50, 80},
		{21, "Slowdive", "Souvlaki", 1993, []string{
			"Alison", "Machine Gun", "40 Days", "Sing",
			"Here She Comes", "Souvlaki Space Station", "When the Sun Hits",
			"Altogether", "Melon Yellow", "Dagger",
		}, nil, 180, 200, 220},
		{22, "Can", "Future Days", 1973, []string{
			"Future Days", "Spray", "Moonshake", "Bel Air",
		}, nil, 150, 180, 130},
		{23, "Stereolab", "Dots and Loops", 1997, []string{
			"Brakhage", "Miss Modular", "The Flower Called Nowhere",
			"Diagonals", "Prisoner of Mars", "Rainbo Conversation",
			"Refractions in the Plastic Pulse", "Parsec", "Ticker-Tape of the Unconscious",
		}, nil, 230, 200, 50},
	}

	// Collect unique artists.
	artistMap := map[string]int{}
	artistID := 0
	for _, a := range albums {
		if _, ok := artistMap[a.artist]; !ok {
			artistID++
			artistMap[a.artist] = artistID
		}
	}

	// Insert artists.
	for name, id := range artistMap {
		exec(db, "INSERT INTO artists (id, name, sort_name) VALUES (?, ?, ?)", id, name, name)
	}

	// Insert albums and tracks.
	trackID := 0
	for _, a := range albums {
		art := artwork(a.r, a.g, a.b)
		aID := artistMap[a.artist]
		exec(db, `INSERT INTO albums (id, artist_id, title, year, track_count, artwork_blob)
			VALUES (?, ?, ?, ?, ?, ?)`, a.id, aID, a.title, a.year, len(a.tracks), art)

		for i, title := range a.tracks {
			trackID++
			dur := 180000 + rand.IntN(180000) // 3-6 min
			exec(db, `INSERT INTO tracks (id, album_id, artist_id, title, track_number, disc_number, duration_ms, file_path, format, bitrate)
				VALUES (?, ?, ?, ?, ?, 1, ?, ?, 'flac', 1411)`,
				trackID, a.id, aID, title, i+1, dur,
				fmt.Sprintf("/demo/%s/%s/%02d - %s.flac", a.artist, a.title, i+1, title))

			// FTS entry.
			exec(db, `INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (?, ?, ?, ?, '')`,
				trackID, title, a.artist, a.title)
		}
	}

	// Genres.
	genres := []string{"Electronic", "Rock", "Post-Rock", "Ambient", "Trip-Hop", "Shoegaze", "Folk", "Krautrock", "Experimental"}
	for i, g := range genres {
		exec(db, "INSERT INTO genres (id, name) VALUES (?, ?)", i+1, g)
	}

	// Map some track-genre associations (first 50 tracks get genres).
	genreMap := map[string][]int{
		"Radiohead":           {1, 2},
		"Bjork":               {1, 7},
		"Portishead":          {5},
		"Massive Attack":      {5, 1},
		"Boards of Canada":    {1, 4},
		"Aphex Twin":          {1, 8},
		"Sigur Ros":           {3},
		"Mogwai":              {3, 2},
		"Nick Drake":          {7},
		"Talk Talk":           {3, 9},
		"Cocteau Twins":       {6, 9},
		"My Bloody Valentine": {6, 2},
		"Brian Eno":           {4},
		"Burial":              {1},
		"The Cure":            {2, 6},
		"Slowdive":            {6},
		"Can":                 {8, 9},
		"Stereolab":           {1, 8},
	}
	tid := 0
	for _, a := range albums {
		gids := genreMap[a.artist]
		for range a.tracks {
			tid++
			for _, gid := range gids {
				exec(db, "INSERT OR IGNORE INTO track_genres (track_id, genre_id) VALUES (?, ?)", tid, gid)
			}
		}
	}

	// Playlists.
	exec(db, "INSERT INTO playlists (id, name) VALUES (1, 'Late Night')")
	exec(db, "INSERT INTO playlists (id, name) VALUES (2, 'Favourites')")
	exec(db, "INSERT INTO playlists (id, name) VALUES (3, 'Shoegaze')")

	// Playlist tracks (pick some track IDs).
	lateNight := []int{48, 49, 56, 57, 66, 67, 3, 4} // ambient/mellow tracks
	for i, t := range lateNight {
		exec(db, "INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES (1, ?, ?)", t, i)
	}
	favs := []int{1, 2, 6, 16, 33, 42, 58, 72, 85, 100}
	for i, t := range favs {
		if t <= trackID {
			exec(db, "INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES (2, ?, ?)", t, i)
		}
	}
	shoegaze := []int{85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95}
	for i, t := range shoegaze {
		if t <= trackID {
			exec(db, "INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES (3, ?, ?)", t, i)
		}
	}

	// Play history - generate 200 plays spread over the last 90 days.
	now := time.Now()
	for i := range 200 {
		t := 1 + rand.IntN(trackID)
		hoursAgo := rand.IntN(90 * 24)
		playedAt := now.Add(-time.Duration(hoursAgo) * time.Hour)
		dur := 180000 + rand.IntN(120000)
		exec(db, `INSERT INTO play_history (id, track_id, played_at, duration_played_ms) VALUES (?, ?, ?, ?)`,
			i+1, t, playedAt.UTC().Format("2006-01-02 15:04:05"), dur)
	}

	// Artist metadata cache entries.
	meta := []struct {
		artist string
		bio    string
		area   string
		tags   string
	}{
		{"Radiohead", "English rock band formed in Abingdon, Oxfordshire in 1985. Known for their experimental approach to rock music.", "Oxfordshire, England", "alternative rock, art rock, electronic"},
		{"Bjork", "Icelandic singer, songwriter, and record producer. Known for her eclectic musical style and visual artistry.", "Reykjavik, Iceland", "art pop, electronic, experimental"},
		{"Portishead", "English band formed in 1991 in Bristol. Pioneers of the trip-hop genre.", "Bristol, England", "trip-hop, electronic, downtempo"},
		{"Massive Attack", "English trip-hop group formed in 1988 in Bristol. One of the most influential groups of the 1990s.", "Bristol, England", "trip-hop, electronic, dub"},
		{"Boards of Canada", "Scottish electronic music duo formed in 1986. Known for their nostalgic, pastoral sound.", "Edinburgh, Scotland", "ambient, electronic, IDM"},
		{"Aphex Twin", "Electronic musician Richard David James, born in Limerick, Ireland. Widely regarded as one of the most inventive electronic artists.", "Cornwall, England", "IDM, ambient, electronic"},
		{"Sigur Ros", "Icelandic post-rock band formed in 1994 in Reykjavik. Known for their ethereal sound.", "Reykjavik, Iceland", "post-rock, ambient, experimental"},
		{"Mogwai", "Scottish post-rock band formed in 1995 in Glasgow. Known for their contrasts of quiet and loud.", "Glasgow, Scotland", "post-rock, shoegaze, experimental"},
		{"Nick Drake", "English singer-songwriter and musician (1948-1974). His music was largely unrecognised during his lifetime.", "Tanworth-in-Arden, England", "folk, singer-songwriter, baroque pop"},
		{"The Cure", "English rock band formed in 1976 in Crawley, West Sussex. Known for their gothic rock and new wave sound.", "Crawley, England", "gothic rock, new wave, post-punk"},
		{"Slowdive", "English shoegaze band formed in 1989 in Reading. Pioneers of the shoegaze genre.", "Reading, England", "shoegaze, dream pop, noise pop"},
		{"Brian Eno", "English musician, composer, and record producer. Pioneer of ambient music.", "Woodbridge, England", "ambient, art rock, electronic"},
		{"Burial", "English electronic musician William Emmanuel Bevan. Known for his atmospheric and emotional electronic music.", "London, England", "dubstep, ambient, UK garage"},
	}
	for _, m := range meta {
		id, ok := artistMap[m.artist]
		if !ok {
			continue
		}
		exec(db, `INSERT INTO artist_metadata (artist_id, bio, mb_area, mb_tags, mb_type) VALUES (?, ?, ?, ?, 'Group')`,
			id, m.bio, m.area, m.tags)
	}

	fmt.Printf("  %d artists, %d albums, %d tracks, 3 playlists, 200 plays\n",
		len(artistMap), len(albums), trackID)
}
