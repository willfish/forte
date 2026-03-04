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

func TestServersTableExists(t *testing.T) {
	db := openTestDB(t)

	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='servers'").Scan(&name)
	if err != nil {
		t.Errorf("servers table not found: %v", err)
	}
}
