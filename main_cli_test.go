package main

import "testing"

func TestHandleCLIVersion(t *testing.T) {
	handled, out := handleCLI([]string{"forte", "--version"})
	if !handled {
		t.Fatal("expected --version to be handled")
	}
	if out != "forte "+version+"\n" {
		t.Fatalf("version output = %q", out)
	}

	handled, out = handleCLI([]string{"forte", "-V"})
	if !handled || out != "forte "+version+"\n" {
		t.Fatalf("-V output = %q handled=%v", out, handled)
	}
}

func TestHandleCLIHelp(t *testing.T) {
	handled, out := handleCLI([]string{"forte", "--help"})
	if !handled {
		t.Fatal("expected --help to be handled")
	}
	if out == "" {
		t.Fatal("help output is empty")
	}
}

func TestHandleCLIPassthrough(t *testing.T) {
	handled, _ := handleCLI([]string{"forte"})
	if handled {
		t.Fatal("bare forte should start the app")
	}
}
