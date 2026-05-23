package library

import (
	"testing"
)

func TestAddAndGetServer(t *testing.T) {
	db := openTestDB(t)

	s := Server{ID: "srv-1", Name: "My Navidrome", Type: "subsonic", URL: "http://localhost:4533", Username: "admin", Password: "secret"}
	if err := db.AddServer(s); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	got, err := db.GetServer("srv-1")
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if got.Name != "My Navidrome" || got.URL != "http://localhost:4533" || got.Password != "secret" {
		t.Errorf("GetServer = %+v, want matching fields", got)
	}
}

func TestGetServers(t *testing.T) {
	db := openTestDB(t)

	mustExec(t, db, "INSERT INTO servers (id, name, type, url, username) VALUES ('s2', 'Zebra', 'subsonic', 'http://z', 'u')")
	mustExec(t, db, "INSERT INTO servers (id, name, type, url, username) VALUES ('s1', 'Alpha', 'subsonic', 'http://a', 'u')")

	servers, err := db.GetServers()
	if err != nil {
		t.Fatalf("GetServers: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(servers))
	}
	if servers[0].Name != "Alpha" {
		t.Errorf("servers[0].Name = %q, want Alpha (ordered by name)", servers[0].Name)
	}
}

func TestUpdateServer(t *testing.T) {
	db := openTestDB(t)

	mustExec(t, db, "INSERT INTO servers (id, name, type, url, username) VALUES ('s1', 'Old', 'subsonic', 'http://old', 'u')")

	err := db.UpdateServer(Server{ID: "s1", Name: "New", Type: "subsonic", URL: "http://new", Username: "u2", Password: "p2"})
	if err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}

	got, err := db.GetServer("s1")
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if got.Name != "New" || got.URL != "http://new" || got.Username != "u2" || got.Password != "p2" {
		t.Errorf("after update: %+v", got)
	}
}

func TestDeleteServer(t *testing.T) {
	db := openTestDB(t)

	mustExec(t, db, "INSERT INTO servers (id, name, type, url, username) VALUES ('s1', 'Test', 'subsonic', 'http://t', 'u')")

	if err := db.DeleteServer("s1"); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}

	_, err := db.GetServer("s1")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDeleteServerRemovesSyncedContent(t *testing.T) {
	db := openTestDB(t)

	mustExec(t, db, "INSERT INTO servers (id, name, type, url, username) VALUES ('s1', 'Test', 'subsonic', 'http://t', 'u')")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Artist')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, server_id, remote_id) VALUES (1, 1, 'Album', 's1', 'album-1')")
	mustExec(t, db, `INSERT INTO tracks
		(id, album_id, artist_id, title, file_path, server_id, remote_id)
		VALUES (1, 1, 1, 'Track', 'server://s1/track-1', 's1', 'track-1')`)
	mustExec(t, db, "INSERT INTO genres (id, name) VALUES (1, 'Genre')")
	mustExec(t, db, "INSERT INTO track_genres (track_id, genre_id) VALUES (1, 1)")
	mustExec(t, db, "INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (1, 'Track', 'Artist', 'Album', 'Genre')")

	if err := db.DeleteServer("s1"); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}

	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "servers", query: "SELECT COUNT(*) FROM servers WHERE id = 's1'"},
		{name: "albums", query: "SELECT COUNT(*) FROM albums WHERE server_id = 's1'"},
		{name: "tracks", query: "SELECT COUNT(*) FROM tracks WHERE server_id = 's1'"},
		{name: "track_genres", query: "SELECT COUNT(*) FROM track_genres WHERE track_id = 1"},
		{name: "fts_tracks", query: "SELECT COUNT(*) FROM fts_tracks WHERE rowid = 1"},
	} {
		var count int
		if err := db.QueryRow(tc.query).Scan(&count); err != nil {
			t.Fatalf("%s count: %v", tc.name, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", tc.name, count)
		}
	}
}

func TestServersTableExists(t *testing.T) {
	db := openTestDB(t)

	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='servers'").Scan(&name)
	if err != nil {
		t.Errorf("servers table not found: %v", err)
	}
}
