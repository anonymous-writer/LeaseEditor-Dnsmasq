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

## Fehleranalyse und Log-Level

LeaseEditor schreibt strukturierte Logmeldungen mit Zeitstempel und Schweregrad nach stdout (bei systemd über `journalctl`, bei Docker über `docker logs`). Das Mindest-Level wird über `LOG_LEVEL` eingestellt:

| `LOG_LEVEL` | Standard | Beschreibung |
| ----------- | -------- | ------------ |
| `DEBUG` | | Detaillierte Diagnoseinformationen, z. B. Anzahl geladener DHCP-Hosteinträge |
| `INFO` | Ja | Normale Betriebsinformationen und erfolgreiche Speichervorgänge |
| `WARN` | | Warnungen, die geprüft werden sollten |
| `ERROR` | | Fehler, bei denen ein Vorgang nicht abgeschlossen werden konnte |

Ist `LOG_LEVEL` nicht gesetzt oder ungültig, wird `INFO` verwendet. Höhere Schweregrade bleiben immer sichtbar. Einzelne MAC-Adressen, Hostnamen und vollständige DHCP-Dateiinhalte werden nicht protokolliert.

Docker Compose verwendet standardmäßig `LOG_LEVEL: "INFO"`. Für temporäre Diagnose kann dies in `compose.yaml` auf `DEBUG` gesetzt werden und der Container anschließend neu erstellt werden. Native Installation: `LOG_LEVEL=DEBUG` in der systemd-Service-Umgebung setzen und danach `sudo systemctl daemon-reload && sudo systemctl restart leaseeditor-dnsmasq` ausführen. Logs anzeigen: `journalctl -u leaseeditor-dnsmasq -f`. Docker-Logs: `docker compose logs -f`.

## Tests und Sicherheit

```bash
go test ./...
go vet ./...
```

Die Weboberfläche hat keine Anmeldung. Stelle sie nicht ungeschützt ins Internet und nutze sie nur in einem vertrauenswürdigen LAN oder hinter einer authentifizierten Reverse-Proxy-Lösung. Der Button „dnsmasq neu starten“ ist für native systemd-Installationen vorgesehen. Installiere dafür die mitgelieferte `leaseeditor-dnsmasq.sudoers` als `/etc/sudoers.d/leaseeditor-dnsmasq` mit Modus `0440` (nur wenn dein Dienstbenutzer `pi` ist). Entferne in der systemd-Service-Datei `NoNewPrivileges=true` (in dieser Version bereits angepasst), lade systemd neu und starte LeaseEditor neu. Der Button führt ausschließlich `systemctl restart dnsmasq` aus. In Docker kann der Host-Dienst nicht auf diese Weise neu gestartet werden.


## Projektstruktur und Versionsanzeige

Analog zu LeaseLens: `main.go` ist der Einstiegspunkt, `internal/app/` enthält die Programmlogik, `templates/index.html` die Oberfläche und `static/app.css`, `static/app.js` und `static/icon.svg` die Web-Assets. Alle Webdateien sind in die native Binary eingebettet.

Die Version erscheint als Link im vorhandenen Fußtext, nicht mehr als separate Statusbox. Standardmäßig `1.0.0-Beta`; Release-Builds können Version und Veröffentlichungsdatum setzen:

```sh
go build -ldflags "-X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildVersion=1.0.0-Beta -X github.com/anonymous-writer/wlm-leaseeditor/internal/app.BuildReleaseDate=2026-10-24" -o leaseeditor-dnsmasq .
```
