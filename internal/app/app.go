package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Item struct {
	Kind     string `json:"kind"` // entry, comment, blank, raw
	Group    string `json:"group,omitempty"`
	Raw      string `json:"raw,omitempty"`
	MAC      string `json:"mac,omitempty"`
	IPv4     string `json:"ipv4,omitempty"`
	IPv6     string `json:"ipv6,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Lease    string `json:"lease,omitempty"`
}
type Document struct {
	Items []Item `json:"items"`
}

var macPattern = regexp.MustCompile(`(?i)^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)
var BuildVersion = "1.0.0-Beta"
var BuildReleaseDate = ""
var BuildChannel = "GoDev"
var BuildUpdatedAt = ""

type AppConfig struct {
	HostsFile            string
	Host                 string
	Port                 string
	LogLevel             string
	HTTPSecure           bool
	HTTPSecureProxyToken string
}

func Run(assets fs.FS) {
	configPath := flag.String("config", "config.yaml", "Pfad zur YAML-Konfigurationsdatei")
	flag.Parse()
	cfg, err := loadConfig(*configPath)
	if err != nil {
		LogFatalf("Konfiguration laden: %v", err)
	}
	// LOG_LEVEL environment variable takes precedence over the optional YAML key.
	if _, set := os.LookupEnv("LOG_LEVEL"); !set {
		SetLogLevel(cfg.LogLevel)
	}
	hosts := cfg.HostsFile
	page := template.Must(template.ParseFS(assets, "templates/index.html"))
	mux := http.NewServeMux()
	staticServer := http.FileServer(http.FS(assets))
	mux.Handle("/static/", http.StripPrefix("/", staticServer))
	mux.HandleFunc("/api/reservations", func(w http.ResponseWriter, r *http.Request) { handleReservations(w, r, hosts) })
	mux.HandleFunc("/api/dnsmasq/restart", handleRestartDnsmasq)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, map[string]string{"HostsFile": hosts, "Version": BuildVersion, "ReleaseDate": BuildReleaseDate, "BuildChannel": BuildChannel, "UpdatedAt": BuildUpdatedAt}); err != nil {
			LogErrorf("render page: %v", err)
		}
	})
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	handler := securityHeaders(mux)
	if cfg.HTTPSecure {
		handler = requireTrustedHTTPSProxy(handler, cfg.HTTPSecureProxyToken)
	}
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	LogInfof("LeaseEditor listening on %s; hosts_file=%s", addr, hosts)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		LogFatalf("HTTP server failed: %v", err)
	}
}

// handleRestartDnsmasq performs only the fixed dnsmasq restart operation.
// On a native installation, grant the service user this exact systemctl command
// through sudoers. In containers, restarting the host service is not supported.
func handleRestartDnsmasq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"Methode nicht erlaubt"}`, http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if os.Geteuid() == 0 {
		cmd = exec.CommandContext(ctx, "/usr/bin/systemctl", "restart", "dnsmasq")
	} else {
		cmd = exec.CommandContext(ctx, "/usr/bin/sudo", "-n", "/usr/bin/systemctl", "restart", "dnsmasq")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		LogErrorf("dnsmasq restart failed: %v (%s)", err, strings.TrimSpace(string(output)))
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Neustart fehlgeschlagen. Prüfe systemd und die sudoers-Berechtigung für LeaseEditor."})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "dnsmasq wurde neu gestartet."})
}

func loadConfig(path string) (AppConfig, error) {
	cfg := AppConfig{HostsFile: "/var/lib/dnsmasq/dnsmasq.dhcphosts", Host: "127.0.0.1", Port: "3009", LogLevel: "INFO"}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("Datei %q nicht lesbar: %w", path, err)
	}
	for lineNo, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return cfg, fmt.Errorf("%s:%d: erwartet wird key: value", path, lineNo+1)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		value = strings.Trim(value, "\"'")
		if value == "" {
			return cfg, fmt.Errorf("%s:%d: leerer Wert für %s", path, lineNo+1, key)
		}
		switch key {
		case "hosts_file":
			cfg.HostsFile = value
		case "host":
			cfg.Host = value
		case "port":
			cfg.Port = value
		case "https-secure":
			if value != "true" && value != "false" {
				return cfg, fmt.Errorf("%s:%d: https-secure muss true oder false sein", path, lineNo+1)
			}
			cfg.HTTPSecure = value == "true"
		case "https-secure-proxy-token":
			cfg.HTTPSecureProxyToken = value
		case "log_level":
			if !validLogLevel(value) {
				return cfg, fmt.Errorf("%s:%d: ungültiger log_level %q (erlaubt: DEBUG, INFO, WARN, ERROR)", path, lineNo+1, value)
			}
			cfg.LogLevel = strings.ToUpper(value)
		default:
			return cfg, fmt.Errorf("%s:%d: unbekannter Parameter %q", path, lineNo+1, key)
		}
	}
	if cfg.HTTPSecure && cfg.HTTPSecureProxyToken == "" {
		return cfg, fmt.Errorf("https-secure: true benötigt https-secure-proxy-token; setze denselben geheimen Wert als X-LeaseEditor-Proxy-Token in NGINX")
	}
	port, err := net.LookupPort("tcp", cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return cfg, fmt.Errorf("%s: ungültiger TCP-Port %q", path, cfg.Port)
	}
	return cfg, nil
}

// requireTrustedHTTPSProxy rejects direct requests when HTTPS secure mode is enabled.
// A request must originate from loopback, carry the HTTPS marker set by NGINX,
// and include the shared secret injected by the trusted NGINX proxy.
func requireTrustedHTTPSProxy(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if err != nil || ip == nil || !ip.IsLoopback() || r.Header.Get("X-Forwarded-Proto") != "https" || r.Header.Get("X-LeaseEditor-Proxy-Token") != token {
			http.Error(w, "HTTPS-Zugriff nur über den vertrauenswürdigen Reverse-Proxy erlaubt", http.StatusForbidden)
			return
		}
		// Do not pass the proxy credential further into application handlers.
		r.Header.Del("X-LeaseEditor-Proxy-Token")
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}
func handleReservations(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch r.Method {
	case http.MethodGet:
		items, err := readItems(path)
		if err != nil {
			LogErrorf("cannot read DHCP host file: %v", err)
			http.Error(w, `{"error":"Datei konnte nicht gelesen werden: `+jsonEscape(err.Error())+`"}`, http.StatusInternalServerError)
			return
		}
		LogDebugf("loaded %d DHCP host entries", len(items))
		_ = json.NewEncoder(w).Encode(Document{Items: items})
	case http.MethodPut:
		if r.ContentLength > 2<<20 {
			http.Error(w, `{"error":"Anfrage zu groß"}`, http.StatusRequestEntityTooLarge)
			return
		}
		defer r.Body.Close()
		body, err := io.ReadAll(io.LimitReader(r.Body, (2<<20)+1))
		if err != nil || len(body) > 2<<20 {
			http.Error(w, `{"error":"Ungültige Anfrage"}`, http.StatusBadRequest)
			return
		}
		var doc Document
		if err = json.Unmarshal(body, &doc); err != nil {
			http.Error(w, `{"error":"Ungültiges JSON"}`, http.StatusBadRequest)
			return
		}
		content, err := serialize(doc.Items)
		if err != nil {
			http.Error(w, `{"error":"`+jsonEscape(err.Error())+`"}`, http.StatusBadRequest)
			return
		}
		if err = saveAtomic(path, content); err != nil {
			LogErrorf("saving DHCP host file failed: %v", err)
			http.Error(w, `{"error":"Speichern fehlgeschlagen: `+jsonEscape(err.Error())+`"}`, http.StatusInternalServerError)
			return
		}
		LogInfof("saved DHCP host file (%d bytes)", len(content))
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, `{"error":"Methode nicht erlaubt"}`, http.StatusMethodNotAllowed)
	}
}
func readItems(path string) ([]Item, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	items := make([]Item, 0, len(lines))
	group := "Nicht zugeordnet"
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		// Leere Zeilen und rein dekorative Trennbalken werden nicht in die
		// Oberfläche übernommen. Beim nächsten Speichern verschwinden sie damit
		// auch aus der Datei, ohne normale Kommentare anzutasten.
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "#") {
			decorative := strings.Trim(trim, "# -=_\t")
			if decorative == "" && len(trim) >= 5 {
				continue
			}
			const marker = "# LeaseEditor-Gruppe:"
			if strings.HasPrefix(trim, marker) {
				group = strings.TrimSpace(strings.TrimPrefix(trim, marker))
				if group == "" {
					group = "Nicht zugeordnet"
				}
			}
			items = append(items, Item{Kind: "comment", Raw: line})
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 || len(parts) > 5 {
			items = append(items, Item{Kind: "raw", Raw: line})
			continue
		}
		it := Item{Kind: "entry", Group: group, MAC: strings.TrimSpace(parts[0]), IPv4: strings.TrimSpace(parts[1])}
		if len(parts) > 2 {
			it.IPv6 = strings.Trim(strings.TrimSpace(parts[2]), "[]")
		}
		if len(parts) > 3 {
			it.Hostname = strings.TrimSpace(parts[3])
		}
		if len(parts) > 4 {
			it.Lease = strings.TrimSpace(parts[4])
		}
		it.Raw = line
		items = append(items, it)
	}
	return items, nil
}
func serialize(items []Item) ([]byte, error) {
	var out bytes.Buffer
	seenMAC, seenIP := map[string]bool{}, map[string]bool{}
	lastGroup := ""
	for i, it := range items {
		line := ""
		switch it.Kind {
		case "blank":
			line = ""
		case "comment", "raw":
			if it.Kind == "comment" && strings.HasPrefix(strings.TrimSpace(it.Raw), "# LeaseEditor-Gruppe:") {
				continue
			}
			if strings.ContainsAny(it.Raw, "\r\n\x00") {
				return nil, fmt.Errorf("Zeile %d enthält ungültige Zeichen", i+1)
			}
			line = it.Raw
		case "entry":
			it.MAC = strings.TrimSpace(it.MAC)
			it.IPv4 = strings.TrimSpace(it.IPv4)
			it.IPv6 = strings.TrimSpace(it.IPv6)
			it.Hostname = strings.TrimSpace(it.Hostname)
			it.Lease = strings.TrimSpace(it.Lease)
			if !macPattern.MatchString(it.MAC) {
				return nil, fmt.Errorf("Zeile %d: ungültige MAC-Adresse", i+1)
			}
			if net.ParseIP(it.IPv4) == nil || net.ParseIP(it.IPv4).To4() == nil {
				return nil, fmt.Errorf("Zeile %d: ungültige IPv4-Adresse", i+1)
			}
			if it.IPv6 != "" && (net.ParseIP(it.IPv6) == nil || net.ParseIP(it.IPv6).To4() != nil) {
				return nil, fmt.Errorf("Zeile %d: ungültige IPv6-Adresse", i+1)
			}
			if strings.ContainsAny(it.Hostname, ",\r\n\x00") || strings.ContainsAny(it.Lease, ",\r\n\x00") {
				return nil, fmt.Errorf("Zeile %d: Name/Lease enthält unzulässige Zeichen", i+1)
			}
			mac := strings.ToLower(it.MAC)
			ip := it.IPv4
			if seenMAC[mac] {
				return nil, fmt.Errorf("Zeile %d: MAC-Adresse doppelt vorhanden", i+1)
			}
			if seenIP[ip] {
				return nil, fmt.Errorf("Zeile %d: IPv4-Adresse doppelt vorhanden", i+1)
			}
			seenMAC[mac] = true
			seenIP[ip] = true
			group := strings.TrimSpace(it.Group)
			if group == "" {
				group = "Nicht zugeordnet"
			}
			if group != lastGroup {
				if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
					out.WriteByte('\n')
				}
				out.WriteString("# LeaseEditor-Gruppe: " + strings.ReplaceAll(group, "\n", " ") + "\n")
				lastGroup = group
			}
			fields := []string{mac, ip}
			if it.IPv6 != "" || it.Hostname != "" || it.Lease != "" {
				if it.IPv6 != "" {
					fields = append(fields, "["+it.IPv6+"]")
				} else {
					fields = append(fields, "")
				}
			}
			if it.Hostname != "" || it.Lease != "" {
				fields = append(fields, it.Hostname)
			}
			if it.Lease != "" {
				fields = append(fields, it.Lease)
			}
			line = strings.Join(fields, ",")
		default:
			return nil, fmt.Errorf("Zeile %d: unbekannter Eintragstyp", i+1)
		}
		out.WriteString(line)
		if i < len(items)-1 {
			out.WriteByte('\n')
		}
	}
	if len(items) > 0 {
		out.WriteByte('\n')
	}
	return out.Bytes(), nil
}
func saveAtomic(path string, data []byte) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if info, e := os.Lstat(abs); e == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("Zieldatei ist ein Symlink; Speichern aus Sicherheitsgründen abgebrochen")
		}
		if !info.Mode().IsRegular() {
			return errors.New("Ziel ist keine reguläre Datei")
		}
		old, e := os.ReadFile(abs)
		if e != nil {
			return e
		}
		if e = os.WriteFile(abs+".bak", old, info.Mode().Perm()); e != nil {
			return fmt.Errorf("Sicherung konnte nicht geschrieben werden: %w", e)
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	tmp, err := os.CreateTemp(dir, ".leaseeditor-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	mode := os.FileMode(0644)
	if info, e := os.Stat(abs); e == nil {
		mode = info.Mode().Perm()
	}
	if err = tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, abs)
}
func jsonEscape(s string) string { b, _ := json.Marshal(s); return strings.Trim(string(b), `"`) }
