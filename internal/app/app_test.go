package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestReadItemsKeepsShortDashComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dnsmasq.dhcphosts")
	input := "# -\nAA:BB:CC:DD:EE:FF,192.168.1.10,test-device,infinite\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := readItems(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Kind != "comment" || items[0].Raw != "# -" {
		t.Fatalf("short comment should be retained, got: %#v", items)
	}
}
