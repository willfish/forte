package player

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGoMpvVersionIncludesPuregoCommandNullTerminatorFix(t *testing.T) {
	const modulePath = "github.com/gen2brain/go-mpv"
	const fixedVersion = "v0.4.0"

	version, ok := requireVersionFromGoMod(t, modulePath)
	if !ok {
		t.Fatalf("%s dependency not found in go.mod", modulePath)
	}
	if compareSemver(version, fixedVersion) < 0 {
		t.Fatalf("%s = %s, want >= %s for purego mpv Command argv termination fix", modulePath, version, fixedVersion)
	}
}

func requireVersionFromGoMod(t *testing.T, modulePath string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == modulePath {
			return fields[1], true
		}
	}
	return "", false
}

func compareSemver(a, b string) int {
	av := semverParts(a)
	bv := semverParts(b)
	for i := range av {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return 0
}

func semverParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	v = strings.Split(v, "-")[0]
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < len(parts) && i < len(out); i++ {
		n, _ := strconv.Atoi(parts[i])
		out[i] = n
	}
	return out
}
