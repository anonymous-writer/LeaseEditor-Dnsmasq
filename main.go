package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Item struct {
	Kind     string `json:"kind"` // entry, comment, blank, raw
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

//go:embed static/*
var staticFiles embed.FS

var macPattern = regexp.MustCompile(`(?i)^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)
var page = template.Must(template.New("index").Parse(indexHTML))

type AppConfig struct {
	HostsFile string
	Host      string
	Port      string
}

func main() {
	configPath := flag.String("config", "config.yaml", "Pfad zur YAML-Konfigurationsdatei")
	flag.Parse()
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Konfiguration laden: %v", err)
	}
	hosts := cfg.HostsFile
	mux := http.NewServeMux()
	staticServer := http.FileServer(http.FS(staticFiles))
	mux.Handle("/static/", http.StripPrefix("/", staticServer))
	mux.HandleFunc("/api/reservations", func(w http.ResponseWriter, r *http.Request) { handleReservations(w, r, hosts) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, map[string]string{"HostsFile": hosts}); err != nil {
			log.Printf("render page: %v", err)
		}
	})
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	server := &http.Server{Addr: addr, Handler: securityHeaders(mux), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("WLM LeaseEditor listening on %s; file=%s", addr, hosts)
	log.Fatal(server.ListenAndServe())
}
func loadConfig(path string) (AppConfig, error) {
	cfg := AppConfig{HostsFile: "/var/lib/dnsmasq/dnsmasq.dhcphosts", Host: "127.0.0.1", Port: "3009"}
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
		default:
			return cfg, fmt.Errorf("%s:%d: unbekannter Parameter %q", path, lineNo+1, key)
		}
	}
	port, err := net.LookupPort("tcp", cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return cfg, fmt.Errorf("%s: ungültiger TCP-Port %q", path, cfg.Port)
	}
	return cfg, nil
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
			http.Error(w, `{"error":"Datei konnte nicht gelesen werden: `+jsonEscape(err.Error())+`"}`, http.StatusInternalServerError)
			return
		}
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
			http.Error(w, `{"error":"Speichern fehlgeschlagen: `+jsonEscape(err.Error())+`"}`, http.StatusInternalServerError)
			return
		}
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
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			items = append(items, Item{Kind: "blank", Raw: line})
			continue
		}
		if strings.HasPrefix(trim, "#") {
			items = append(items, Item{Kind: "comment", Raw: line})
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 || len(parts) > 5 {
			items = append(items, Item{Kind: "raw", Raw: line})
			continue
		}
		it := Item{Kind: "entry", MAC: strings.TrimSpace(parts[0]), IPv4: strings.TrimSpace(parts[1])}
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
	for i, it := range items {
		line := ""
		switch it.Kind {
		case "blank":
			line = ""
		case "comment", "raw":
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

const indexHTML = `<!doctype html>
<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>LeaseEditor – Dnsmasq Network Dashboard</title><style>
:root{color-scheme:dark;--bg:#0b1220;--panel:#142033;--panel2:#1b2a40;--line:#2b3b51;--text:#e7eef8;--muted:#9aabc1;--blue:#2d6df6;--cyan:#59b8f5;--red:#f05c70}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.45 system-ui,-apple-system,Segoe UI,sans-serif}header{display:flex;align-items:center;gap:24px;padding:18px 30px;background:#101a2b;border-bottom:1px solid var(--line);min-height:112px}header .brand{flex:1;min-width:0}header .repo-status{margin-left:auto;flex-shrink:0;display:inline-flex;align-items:center;gap:4px;padding:9px 13px;border:1px solid var(--line);border-radius:9px;background:var(--panel);color:var(--muted);font-size:12px;text-decoration:none;white-space:nowrap}header .repo-status strong{color:#579cf5;font-weight:600}header .repo-status:hover{border-color:#579cf5;color:var(--text)}@media(max-width:700px){header{gap:14px;padding:14px 16px;flex-wrap:wrap}header img{width:48px;height:60px}h1{font-size:32px}header .repo-status{margin-left:62px}header .brand{flex:1}}header img{width:68px;height:84px;object-fit:contain}h1{font-size:42px;line-height:1.1;margin:0;font-weight:750;letter-spacing:-1px}h1 .brand-accent{color:#579cf5}header p{margin:5px 0 0;color:var(--text);font-size:16px;font-weight:600}header .tagline{margin-top:5px;color:var(--muted);font-size:12px;font-weight:400}main{max-width:1500px;margin:24px auto;padding:0 20px 78px}.toolbar{display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-bottom:16px}.toolbar input{flex:1;min-width:220px}.btn{border:1px solid #3a4d67;border-radius:8px;background:var(--panel2);color:var(--text);padding:10px 14px;cursor:pointer;font-weight:600}.btn.primary{background:var(--blue);border-color:var(--blue)}.btn.danger{border-color:#793744;color:#ffb5bf}.btn:disabled{opacity:.5;cursor:wait}input,select{width:100%;background:#0d1726;color:var(--text);border:1px solid #35465d;border-radius:7px;padding:9px 10px;font:inherit}input:focus,select:focus{outline:2px solid #3679ff;outline-offset:1px}.meta{color:var(--muted);margin:8px 0 14px}.table-wrap{overflow:auto;border:1px solid var(--line);border-radius:10px;background:var(--panel)}table{width:100%;border-collapse:collapse;min-width:980px}th{text-align:left;color:#a9bad0;background:#1b2a40;font-size:11px;letter-spacing:.06em;text-transform:uppercase;position:sticky;top:0}th,td{padding:10px 11px;border-bottom:1px solid var(--line)}tr:last-child td{border-bottom:0}.section td{background:#19283d;color:#81c7ff;font-weight:700}.section input{font-weight:700;color:#81c7ff}.actions{white-space:nowrap}.actions button{padding:6px 9px;margin-right:4px}.empty{padding:32px;text-align:center;color:var(--muted)}.status{padding:10px 13px;border-radius:8px;margin:0 0 14px;display:none}.status.ok{display:block;background:#12372f;color:#a7f3d0}.status.err{display:block;background:#431c28;color:#ffc2ca}.warning{padding:12px 14px;background:#322a16;border:1px solid #665526;border-radius:8px;color:#f8dfa0;margin-bottom:16px}.modal-backdrop{display:none;position:fixed;inset:0;background:#0009;align-items:center;justify-content:center;padding:20px}.modal-backdrop.open{display:flex}.modal{width:min(620px,100%);background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:22px;box-shadow:0 20px 80px #0008}.modal h2{margin:0 0 18px}.fields{display:grid;grid-template-columns:1fr 1fr;gap:13px}.field label{display:block;font-size:12px;color:var(--muted);margin-bottom:5px}.field.full{grid-column:1/-1}.modal-actions{display:flex;justify-content:flex-end;gap:8px;margin-top:22px}.foot{color:var(--muted);font-size:12px;margin-top:18px}.versionbar{position:fixed;right:14px;bottom:14px;padding:9px 13px;border:1px solid var(--line);border-radius:9px;background:#142033;color:var(--muted);font-size:12px;text-decoration:none;box-shadow:0 4px 18px #0003}.versionbar strong{color:#579cf5;font-weight:600}.versionbar:hover{border-color:#579cf5;color:var(--text)}@media(max-width:650px){header{padding:14px 16px;gap:14px}header img{width:48px;height:62px}h1{font-size:30px}header p{font-size:14px}header .tagline{font-size:11px}.versionbar{right:8px;bottom:8px}main{padding:0 12px 70px;margin:16px auto}.fields{grid-template-columns:1fr}.field.full{grid-column:auto}.toolbar .btn{flex:1}}
</style></head><body><header><img src="/static/wlm-logo.png" alt="LeaseEditor Logo"><div class="brand"><h1>Lease<span class="brand-accent">Editor</span></h1><p>Dnsmasq Network Dashboard</p><div class="tagline">DHCP-Leases · Geräteerkennung · Hersteller · Netzwerkstatus</div></div></header><main><div class="warning"><strong>Direktes Speichern:</strong> Änderungen schreiben die Datei auf dem Server. Vor jedem Speichern wird eine Sicherung als <code>.bak</code> erstellt. Verwende das Tool nur in einem vertrauenswürdigen Netzwerk.</div><div id="status" class="status"></div><div class="toolbar"><input id="search" placeholder="Suche nach MAC, IP, Hostname oder Kommentar…" aria-label="Suchen"><button class="btn" id="reload">Neu laden</button><button class="btn" id="add">+ Reservierung</button><button class="btn primary" id="save">Änderungen speichern</button></div><div class="meta"><span id="count">Lade Datei …</span> · Datei: <code>{{.HostsFile}}</code></div><div class="table-wrap"><table><thead><tr><th>MAC-Adresse</th><th>IPv4-Adresse</th><th>IPv6-Adresse</th><th>Hostname</th><th>Lease</th><th>Aktionen</th></tr></thead><tbody id="rows"></tbody></table><div id="empty" class="empty" hidden>Keine passenden Einträge gefunden.</div></div><div class="foot">LeaseEditor · Änderungen werden erst mit „Änderungen speichern“ auf die Datei geschrieben. Kommentare und unbekannte Zeilen werden beim Speichern beibehalten.</div></main><a class="versionbar" href="https://github.com/anonymous-writer/LeaseEditor-Dnsmasq" target="_blank" rel="noopener noreferrer" aria-label="LeaseEditor auf GitHub öffnen"><strong>LeaseEditor</strong> v2.1.5 · veröffentlicht 2026-10-22</a>
<div class="modal-backdrop" id="modal"><form class="modal" id="form"><h2 id="modalTitle">Reservierung</h2><div class="fields"><div class="field"><label for="mac">MAC-Adresse *</label><input id="mac" required placeholder="aa:bb:cc:dd:ee:ff"></div><div class="field"><label for="ipv4">IPv4-Adresse *</label><input id="ipv4" required placeholder="192.168.1.20"></div><div class="field"><label for="ipv6">IPv6-Adresse (optional)</label><input id="ipv6" placeholder="fd00::20"></div><div class="field"><label for="hostname">Hostname</label><input id="hostname" placeholder="drucker"></div><div class="field full"><label for="lease">Lease-Dauer (leer lassen, wenn nicht gesetzt)</label><input id="lease" placeholder="infinite"></div></div><div class="modal-actions"><button class="btn" type="button" id="cancel">Abbrechen</button><button class="btn primary" type="submit">Übernehmen</button></div></form></div>
<script>
const $=id=>document.getElementById(id);let items=[],editing=-1,dirty=false;
function status(msg,kind='ok'){const el=$('status');el.textContent=msg;el.className='status '+kind;}
function markDirty(){dirty=true;$('save').textContent='Änderungen speichern *';}
function showModal(index=-1){editing=index;const it=index>=0?items[index]:{mac:'',ipv4:'',ipv6:'',hostname:'',lease:'infinite'};$('modalTitle').textContent=index>=0?'Reservierung bearbeiten':'Neue Reservierung';for(const k of ['mac','ipv4','ipv6','hostname','lease'])$(k).value=it[k]||'';$('modal').classList.add('open');$('mac').focus();}
function hideModal(){$('modal').classList.remove('open');}
function esc(s){return String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
function render(){const q=$('search').value.toLowerCase();const tbody=$('rows');tbody.innerHTML='';let visible=0;items.forEach((it,i)=>{if(it.kind==='entry'){const hay=[it.mac,it.ipv4,it.ipv6,it.hostname,it.lease].join(' ').toLowerCase();if(q&&!hay.includes(q))return;visible++;const tr=document.createElement('tr');tr.innerHTML="<td>"+esc(it.mac)+"</td><td>"+esc(it.ipv4)+"</td><td>"+esc(it.ipv6)+"</td><td>"+esc(it.hostname)+"</td><td>"+esc(it.lease)+"</td><td class=\"actions\"><button class=\"btn\" data-edit=\""+i+"\">Bearbeiten</button><button class=\"btn danger\" data-del=\""+i+"\">Löschen</button></td>";tbody.appendChild(tr);}else{const raw=it.raw||'';if(q&&!raw.toLowerCase().includes(q))return;const tr=document.createElement('tr');tr.className='section';tr.innerHTML="<td colspan=\"6\">"+(it.kind==='blank'?'':esc(raw))+" "+(it.kind==='raw'?'(unbekannte Zeile – wird unverändert erhalten)':'')+"</td>";tbody.appendChild(tr);}});$('empty').hidden=visible!==0;$('count').textContent=visible+' Reservierung(en) · '+items.filter(x=>x.kind==='comment').length+' Kommentarzeile(n)';}
async function load(){if(dirty&&!confirm('Ungespeicherte Änderungen verwerfen und Datei neu laden?'))return;try{const r=await fetch('/api/reservations');if(!r.ok)throw new Error(await r.text());const d=await r.json();items=d.items||[];dirty=false;$('save').textContent='Änderungen speichern';render();status('Datei geladen.','ok');}catch(e){status('Laden fehlgeschlagen: '+e.message,'err');}}
$('rows').addEventListener('click',e=>{const ed=e.target.closest('[data-edit]'),del=e.target.closest('[data-del]');if(ed)showModal(Number(ed.dataset.edit));if(del){const i=Number(del.dataset.del);if(confirm('Diese Reservierung entfernen?')){items.splice(i,1);markDirty();render();}}});
$('form').addEventListener('submit',e=>{e.preventDefault();const it={kind:'entry',mac:$('mac').value.trim(),ipv4:$('ipv4').value.trim(),ipv6:$('ipv6').value.trim(),hostname:$('hostname').value.trim(),lease:$('lease').value.trim()};if(editing>=0)items[editing]=it;else items.push(it);markDirty();hideModal();render();});
$('cancel').onclick=hideModal;$('modal').addEventListener('click',e=>{if(e.target===$('modal'))hideModal()});$('add').onclick=()=>showModal();$('reload').onclick=load;$('search').oninput=render;
$('save').onclick=async()=>{if(!confirm('Die bestehende dnsmasq.dhcphosts-Datei jetzt überschreiben? Eine .bak-Sicherung wird angelegt.'))return;try{$('save').disabled=true;const r=await fetch('/api/reservations',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({items})});if(!r.ok){let msg=await r.text();try{msg=JSON.parse(msg).error||msg}catch{}throw new Error(msg)}dirty=false;$('save').textContent='Änderungen speichern';status('Datei erfolgreich gespeichert. Prüfe anschließend, ob dnsmasq die Änderungen neu laden muss.','ok');await load();}catch(e){status('Speichern fehlgeschlagen: '+e.message,'err')}finally{$('save').disabled=false;}};
window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});load();
</script></body></html>`
