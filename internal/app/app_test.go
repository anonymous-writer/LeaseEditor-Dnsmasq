package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anonymous-writer/wlm-leaseeditor/internal/proxytoken"
)

func TestReadAndSerializeKeepsCommentsAndUnknownLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsmasq.dhcphosts")
	input := "# network devices\nAA:BB:CC:DD:EE:FF,192.168.1.10,[fd00::10],test-device,infinite\ncustom-directive\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := readItems(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[1].MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("unexpected parse: %#v", items)
	}
	out, err := serialize(items)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "# network devices") || !strings.Contains(string(out), "custom-directive") {
		t.Fatalf("comments/raw line not preserved: %s", out)
	}
}

func TestSerializeRejectsDuplicateIPv4(t *testing.T) {
	items := []Item{
		{Kind: "entry", MAC: "aa:bb:cc:dd:ee:01", IPv4: "192.168.1.10"},
		{Kind: "entry", MAC: "aa:bb:cc:dd:ee:02", IPv4: "192.168.1.10"},
	}
	if _, err := serialize(items); err == nil {
		t.Fatal("expected duplicate IPv4 validation error")
	}
}

func TestRequireTrustedHTTPSProxy(t *testing.T) {
	manager, err := proxytoken.New(filepath.Join(t.TempDir(), "proxy-token.json"), 30*24*time.Hour, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token := manager.Current()
	h := requireTrustedHTTPSProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }), manager)
	tests := []struct {
		name, remote, proto, supplied string
		want                          int
	}{
		{"direct local request", "127.0.0.1:5555", "", "", 403},
		{"spoofed proto without token", "127.0.0.1:5555", "https", "", 403},
		{"non-local proxy", "10.0.0.2:5555", "https", token, 403},
		{"trusted proxy", "127.0.0.1:5555", "https", token, 204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remote
			r.Header.Set("X-Forwarded-Proto", tt.proto)
			r.Header.Set("X-LeaseEditor-Proxy-Token", tt.supplied)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("got %d want %d", w.Code, tt.want)
			}
		})
	}
}
