package main

import (
	"bytes"
	"os"
	"testing"
)

func TestWailsLinuxTrayMenuOpenDoesNotInvokeClickHandlerPatch(t *testing.T) {
	patch, err := os.ReadFile("patches/wails-status-notifier-click-menu-split.patch")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(patch, []byte("-\t\tif s.parent.clickHandler != nil {")) ||
		!bytes.Contains(patch, []byte("-\t\t\ts.parent.clickHandler()")) {
		t.Fatal("patch must remove the left-click handler from the DBus menu opened event")
	}

	if !bytes.Contains(patch, []byte("case \"opened\":")) ||
		!bytes.Contains(patch, []byte("s.parent.onMenuOpen")) {
		t.Fatal("patch must preserve the DBus menu opened callback path")
	}
}
