# LeaseEditor-Dnsmasq 1.0.4 Beta — Prüfergebnis

Prüfdatum: 2026-10-31

## Durchgeführte Prüfungen

- Go-Tests: `go test ./...` — bestanden.
- Go-Static-Analysis: `go vet ./...` — bestanden.
- Go-Build: `go build -trimpath` — bestanden.
- JavaScript-Syntax: `node --check static/app.js` — bestanden.
- GitHub-Repository und README im Browser geprüft.

Hinweis zur lokalen Go-Umgebung: Das Paket fordert Go 1.27. Die bereitgestellte Prüfumgebung enthält Go 1.23.2 und kann Go 1.27 nicht herunterladen (kein Netzwerkzugriff). Für Tests, Vet und Build wurde deshalb ausschließlich in der temporären Prüfumgebung die `go`-Direktive auf 1.23 gesetzt und danach die originale `go.mod` unverändert wiederhergestellt. Die CI-/Docker-Konfiguration des Projekts fordert weiterhin Go 1.27.

## Änderungen in diesem korrigierten Paket

1. Gruppenbezeichnungen werden vor dem Schreiben auf CR-, LF- und NUL-Steuerzeichen geprüft. Dadurch kann ein Gruppenname keine zusätzlichen Zeilen oder beschädigte Inhalte in die dnsmasq-Konfigurationsdatei einschleusen.
2. Die eingebettete Standard-Versionsanzeige wurde von `1.0.0-Beta` auf `1.0.4-Beta` angehoben.
3. Tippfehler im README („create sure“) korrigiert.
4. Dieser Prüfbericht wurde ergänzt.

## Hinweise zum GitHub-Repository

- Das öffentliche Repository `https://github.com/anonymous-writer/LeaseEditor-Dnsmasq` ist erreichbar; die sichtbare README entspricht inhaltlich dem geprüften Paket.
- Die README verlinkt die MIT-Lizenz auf das separate Repository `LeaseLens-Dnsmasq`. Im bereitgestellten ZIP liegt keine eigene `LICENSE`-Datei. Vor einer Veröffentlichung sollte geklärt werden, welche Lizenz für dieses Repository gelten soll, und die passende Lizenzdatei im Repository-Root abgelegt werden.
- Auf der geprüften GitHub-Repository-Seite waren keine Releases/Packages aufgeführt. Daher konnte die README-Aussage, dass ein `latest`-Image unter GHCR verfügbar ist, nicht durch einen veröffentlichten Release dieses Repositories bestätigt werden. Vor Verwendung des `docker pull ...:latest`-Befehls bitte prüfen, ob das Image tatsächlich publiziert wurde.
- Das Webinterface besitzt laut README keine eingebaute Benutzeranmeldung. Da es DHCP-Reservierungen ändern und (bei nativer Installation mit sudoers-Regel) dnsmasq neu starten kann, sollte es ausschließlich in einem vertrauenswürdigen LAN oder hinter einem korrekt abgesicherten Reverse-Proxy betrieben werden. Nicht direkt ins Internet exponieren.

## Nicht durchgeführt

Es wurde nichts auf GitHub hochgeladen oder dort verändert. Es wurden keine Live-Änderungen an dnsmasq, Docker-Containern oder Systemdiensten vorgenommen. Die Korrekturen betreffen ausschließlich dieses ZIP-Paket.
