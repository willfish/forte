package library

import (
	"path/filepath"
	"testing"
)

func TestAddAndGetLocalDirectories(t *testing.T) {
	db := openTestDB(t)
	dirA := filepath.Join(t.TempDir(), "Music A")
	dirB := filepath.Join(t.TempDir(), "Music B")

	if err := db.AddLocalDirectory(dirB); err != nil {
		t.Fatalf("AddLocalDirectory B: %v", err)
	}
	if err := db.AddLocalDirectory(dirA); err != nil {
		t.Fatalf("AddLocalDirectory A: %v", err)
	}
	if err := db.AddLocalDirectory(dirA); err != nil {
		t.Fatalf("AddLocalDirectory duplicate: %v", err)
	}

	dirs, err := db.GetLocalDirectories()
	if err != nil {
		t.Fatalf("GetLocalDirectories: %v", err)
	}
	if len(dirs) != 2 {
		t.Fatalf("got %d directories, want 2", len(dirs))
	}

	wantA, _ := filepath.Abs(dirA)
	wantB, _ := filepath.Abs(dirB)
	if dirs[0].Path != filepath.Clean(wantA) || dirs[1].Path != filepath.Clean(wantB) {
		t.Fatalf("directories = %+v, want ordered %q then %q", dirs, wantA, wantB)
	}
	if dirs[0].AddedAt == "" || dirs[1].AddedAt == "" {
		t.Fatalf("AddedAt should be populated: %+v", dirs)
	}
}

func TestRemoveLocalDirectory(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()

	if err := db.AddLocalDirectory(dir); err != nil {
		t.Fatalf("AddLocalDirectory: %v", err)
	}
	if err := db.RemoveLocalDirectory(dir); err != nil {
		t.Fatalf("RemoveLocalDirectory: %v", err)
	}

	dirs, err := db.GetLocalDirectories()
	if err != nil {
		t.Fatalf("GetLocalDirectories: %v", err)
	}
	if len(dirs) != 0 {
		t.Fatalf("got %d directories after remove, want 0", len(dirs))
	}
}

func TestAddLocalDirectoryRejectsEmptyPath(t *testing.T) {
	db := openTestDB(t)

	if err := db.AddLocalDirectory(""); err == nil {
		t.Fatal("AddLocalDirectory empty path error = nil, want error")
	}
}

func TestLocalDirectoriesTableExists(t *testing.T) {
	db := openTestDB(t)

	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='local_directories'").Scan(&name)
	if err != nil {
		t.Errorf("local_directories table not found: %v", err)
	}
}
