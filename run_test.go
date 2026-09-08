package chicle

import (
	"errors"
	"os"
	"testing"
)

func TestRunWithoutATTYFailsClearly(t *testing.T) {
	// A picker in a pipeline with no terminal must say so, not hang or panic.
	if _, err := runOn(Config{Rows: rows("a")}, nil, errors.New("no tty")); err == nil {
		t.Fatal("expected an error when the tty could not be opened")
	}
}

func TestOpenTTYReturnsAWritableFile(t *testing.T) {
	tty, err := openTTY()
	if err != nil {
		t.Skip("no controlling terminal in this environment")
	}
	defer tty.Close()
	if _, err := tty.Stat(); err != nil {
		t.Fatalf("tty is not usable: %v", err)
	}
	if tty == os.Stdout {
		t.Fatal("Run must not draw to stdout — that is the caller's channel")
	}
}
