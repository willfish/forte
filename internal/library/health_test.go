package library

import "testing"

func TestHealthMonitorStartStopIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	monitor := NewHealthMonitor(db)

	monitor.Stop()
	if monitor.Running() {
		t.Fatal("monitor running after stop before start")
	}

	monitor.Start()
	monitor.Start()
	if !monitor.Running() {
		t.Fatal("monitor not running after start")
	}

	monitor.Stop()
	monitor.Stop()
	if monitor.Running() {
		t.Fatal("monitor running after stop")
	}

	monitor.Start()
	if !monitor.Running() {
		t.Fatal("monitor not running after restart")
	}
	monitor.Stop()
}
