# LeaseEditor-Dnsmasq

Die Weboberfläche heißt **LeaseEditor**. Der Versionshinweis rechts in der Kopfzeile verlinkt auf das [GitHub-Projekt](https://github.com/anonymous-writer/LeaseEditor-Dnsmasq).

Eigenständiger Go-Webeditor für dnsmasq-DHCP-Reservierungen. Native Programmversion und Docker verwenden dieselbe `config.yaml`.

## Konfiguration

```yaml
# Web server
hosts_file: "/var/lib/dnsmasq/dnsmasq.dhcphosts"
host: "127.0.0.1"
port: "3009"
```

- `hosts_file`: Pfad zur dnsmasq-Hosts-Datei
- `host`: Bind-Adresse des Webservers. Für Zugriff aus dem LAN in der Regel `0.0.0.0` verwenden; `127.0.0.1` erlaubt nur lokale Zugriffe.
- `port`: TCP-Port des Webservers

## Native Version

Lege `config.yaml` neben die Programmdatei und starte:

```bash
./leaseeditor-dnsmasq-linux-arm64 --config ./config.yaml
```

Der systemd-Dienst und das Update-Skript verwenden `/home/pi/leaseeditor-dnsmasq/config.yaml`.

## Docker Compose

Lege `config.yaml` und `compose.yaml` in dasselbe Verzeichnis. Passe bei Bedarf den Host-Pfad für `/var/lib/dnsmasq` an und starte:

```bash
docker compose up -d --build
```

Compose bindet dieselbe `config.yaml` schreibgeschützt nach `/app/config.yaml` ein. Wenn du `port` änderst, passe auch das Port-Mapping in `compose.yaml` an.

## Tests und Sicherheit

```bash
go test ./...
go vet ./...
```

Die Weboberfläche hat keine Anmeldung. Stelle sie nicht ungeschützt ins Internet und nutze sie nur in einem vertrauenswürdigen LAN oder hinter einer authentifizierten Reverse-Proxy-Lösung. Das Programm führt keinen dnsmasq-Neustart aus.


## Projektstruktur und Versionsanzeige

Analog zu LeaseLens: `main.go` ist der Einstiegspunkt, `internal/app/` enthält die Programmlogik, `templates/index.html` die Oberfläche und `static/app.css`, `static/app.js` und `static/icon.svg` die Web-Assets. Alle Webdateien sind in die native Binary eingebettet.

Die Version erscheint als Link im vorhandenen Fußtext, nicht mehr als separate Statusbox. Standardmäßig `1.0.0-Beta`; Release-Builds können Version und Veröffentlichungsdatum setzen:

```sh
go build -ldflags "-X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildVersion=1.0.0-Beta -X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildReleaseDate=2026-10-24" -o leaseeditor-dnsmasq .
```
