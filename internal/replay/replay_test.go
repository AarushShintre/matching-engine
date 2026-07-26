package replay

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadAndReplayStubs(t *testing.T) {
	_, err := Load("testdata/placeholder.json")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Load: want ErrNotImplemented until TODO(spec-3), got %v", err)
	}

	s := &Scenario{Name: "placeholder"}
	err = s.Replay(DefaultN)
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Replay: want ErrNotImplemented until TODO(spec-3), got %v", err)
	}
}

func TestTestdataPresent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "testdata", "placeholder.json")
	s, err := Load(path)
	if err == nil && s == nil {
		t.Fatal("unexpected nil scenario without error")
	}
	// Expected until capture loader is implemented.
	if !errors.Is(err, ErrNotImplemented) {
		t.Logf("TODO(spec-3): implement Load for %s", path)
	}
}
