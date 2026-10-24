# WLM LeaseEditor

Eigenständiger Go-Webeditor für dnsmasq DHCP-Reservierungen in `dnsmasq.dhcphosts`. Das Projekt orientiert sich am einfachen Go-Aufbau von LeaseLens, ist aber ein separates Programm.

## Funktionen im ersten Stand

- WLM-Logo und dunkle, responsive Weboberfläche
- Reservierungen anzeigen, suchen, hinzufügen, bearbeiten und löschen
- MAC-, IPv4- und optionale IPv6-Adresse sowie Hostname und Lease-Feld bearbeiten
- Kommentare, Leerzeilen und unbekannte Zeilen beim Speichern beibehalten
- Validierung von MAC-, IPv4- und IPv6-Adressen sowie doppelte MAC-/IPv4-Adressen
- Vor dem Überschreiben wird eine Sicherung `dnsmasq.dhcphosts.bak` erstellt
- Atomisches Schreiben über temporäre Datei
- Go-Standardbibliothek, keine externen Go-Abhängigkeiten

## Lokal starten

```bash
export DNSMASQ_HOSTS_FILE="$PWD/dnsmasq.dhcphosts"
export HOST=127.0.0.1
export PORT=3010
go test ./...
go run .
```

Öffne danach `http://127.0.0.1:3010`.

## Docker Compose

1. Passe in `compose.yaml` den Host-Pfad links vom Doppelpunkt an den tatsächlichen Speicherort der Datei an.
2. Stelle sicher, dass der Container-Benutzer (UID/GID des Images: nicht-root `leaseeditor`) die Datei lesen und das Verzeichnis beschreiben darf. Für Sicherung und atomisches Ersetzen werden Schreibrechte im Verzeichnis benötigt.
3. Starte das Programm:

```bash
docker compose up -d --build
```

Öffne `http://<server-ip>:3010`.

## Wichtige Hinweise

- Die Weboberfläche besitzt im ersten Stand keine Anmeldung. Veröffentliche den Port nicht im Internet; verwende ihn nur in einem vertrauenswürdigen LAN oder hinter einer authentifizierten Reverse-Proxy-Lösung.
- Speichern schreibt die Datei direkt. Eine bestehende Sicherung `.bak` wird bei jedem Speichern aktualisiert.
- Das Programm führt keinen dnsmasq-Neustart oder Reload aus. Prüfe nach dem Speichern, ob deine dnsmasq-Installation die Änderungen neu einlesen muss.
- Kommentare und Zeilen, die das Programm nicht als einfache Reservierung erkennt, werden als Kommentar-/Rohzeilen erhalten. Wenn du eine solche Zeile bearbeiten möchtest, ändere sie manuell in der Originaldatei.
- Dies ist ein erster Programmaufbau. Vor dem produktiven Einsatz sollte er zunächst mit einer Kopie deiner Datei getestet werden.
