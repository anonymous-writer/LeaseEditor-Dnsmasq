package proxytoken

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewInitializesMissingAndEmptyFiles(t *testing.T) {
	for _, empty := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "state.json")
		if empty {
			if err := os.WriteFile(path, nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
		_, err := New(path, 30*24*time.Hour, 5*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if len(readState(t, path).Current) != 100 {
			t.Fatalf("token length = %d", len(readState(t, path).Current))
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var state State
		if err := json.Unmarshal(b, &state); err != nil {
			t.Fatal(err)
		}
		if state.Current == "" {
			t.Fatal("persisted token differs")
		}
	}
}

func TestNewRejectsNonEmptyInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path, time.Hour, time.Minute); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestRotationGracePeriod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	m, err := New(path, time.Hour, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	old := readState(t, path).Current
	now := time.Now().Add(2 * time.Hour)
	if err := m.RotateIfDue(now); err != nil {
		t.Fatal(err)
	}
	current := readState(t, path).Current
	if current == old || len(current) != 100 {
		t.Fatal("token was not rotated")
	}
	if !m.Valid(old, now.Add(4*time.Minute)) {
		t.Fatal("previous token should be valid during grace")
	}
	if m.Valid(old, now.Add(6*time.Minute)) {
		t.Fatal("previous token should be invalid after grace")
	}
}

func TestTokenIsHex(t *testing.T) {
	tok, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 100 || strings.Trim(tok, "0123456789abcdef") != "" {
		t.Fatal("unexpected token format")
	}
}

func readState(t *testing.T, path string) State {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}
