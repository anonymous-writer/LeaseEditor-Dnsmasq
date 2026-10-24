#!/usr/bin/env bash
set -Eeuo pipefail

# Installiert ein manuell heruntergeladenes Linux-ARM64-Release.
# Das Skript lädt absichtlich nichts aus dem Internet herunter.

BASE_DIR="/media/pi/Ablage/Datenablage"
ARCHIVE_NAME="leaseeditor-dnsmasq-linux-arm64.tar.gz"
BINARY_NAME="leaseeditor-dnsmasq-linux-arm64"
ARCHIVE_PATH="${BASE_DIR}/${ARCHIVE_NAME}"
EXTRACTED_PATH="${BASE_DIR}/${BINARY_NAME}"
INSTALL_PATH="/usr/bin/leaseeditor-dnsmasq"
SERVICE_NAME="leaseeditor-dnsmasq.service"
SERVICE_SOURCE="${BASE_DIR}/${SERVICE_NAME}"
SERVICE_DEST="/etc/systemd/system/${SERVICE_NAME}"
CONFIG_DIR="/home/pi/leaseeditor-dnsmasq"
CONFIG_PATH="${CONFIG_DIR}/config.yaml"
SUMS_PATH="${BASE_DIR}/SHA256SUMS"

usage() {
  cat <<'USAGE'
Usage: ./update-leaseeditor-dnsmasq-arm64.sh

Lege das Release-Archiv hier ab:
  /media/pi/Ablage/Datenablage/leaseeditor-dnsmasq-linux-arm64.tar.gz

Optional: SHA256SUMS und leaseeditor-dnsmasq.service im selben Verzeichnis.
Die vorhandene Konfiguration bleibt erhalten. Falls sie fehlt, wird eine
config.yaml mit den Projekt-Standardwerten angelegt.
Das Skript lädt nichts aus dem Internet herunter. Starte es als normaler
Benutzer (z.B. pi), nicht mit sudo.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then usage; exit 0; fi
if [[ "$#" -gt 0 ]]; then echo "Error: Keine Argumente erlaubt." >&2; usage >&2; exit 2; fi
if [[ "$EUID" -eq 0 ]]; then echo "Error: Bitte ohne sudo als normaler Benutzer ausführen." >&2; exit 1; fi
case "$(uname -m)" in aarch64|arm64) ;; *) echo "Error: Nur ARM64 unterstützt; erkannt: $(uname -m)" >&2; exit 1 ;; esac
for cmd in tar sha256sum sudo install systemctl grep; do command -v "$cmd" >/dev/null 2>&1 || { echo "Error: Programm fehlt: $cmd" >&2; exit 1; }; done
[[ -d "$BASE_DIR" ]] || { echo "Error: Verzeichnis fehlt: $BASE_DIR" >&2; exit 1; }
[[ -f "$ARCHIVE_PATH" ]] || { echo "Error: Release-Archiv fehlt: $ARCHIVE_PATH" >&2; exit 1; }
if [[ -f "$SUMS_PATH" ]]; then
  echo "Prüfe SHA256SUMS..."
  (cd "$BASE_DIR" && sha256sum --check --ignore-missing "$SUMS_PATH") || { echo "Checksum-Prüfung fehlgeschlagen; Installation abgebrochen." >&2; exit 1; }
else
  echo "Warnung: SHA256SUMS nicht gefunden; Prüfsumme wird nicht geprüft." >&2
fi
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
ARCHIVE_MEMBER="$(tar -tzf "$ARCHIVE_PATH" | grep -E '(^|/)leaseeditor-dnsmasq-linux-arm64$' | head -n 1 || true)"
[[ -n "$ARCHIVE_MEMBER" ]] || { echo "Archiv enthält ${BINARY_NAME} nicht." >&2; tar -tzf "$ARCHIVE_PATH" >&2 || true; exit 1; }
tar -xOzf "$ARCHIVE_PATH" "$ARCHIVE_MEMBER" > "$TMP_DIR/$BINARY_NAME"
chmod 0755 "$TMP_DIR/$BINARY_NAME"
install -m 0755 "$TMP_DIR/$BINARY_NAME" "${EXTRACTED_PATH}.new"
mv -f "${EXTRACTED_PATH}.new" "$EXTRACTED_PATH"
if [[ -e "$INSTALL_PATH" ]]; then
  BACKUP_PATH="${INSTALL_PATH}.bak.$(date +%Y%m%d-%H%M%S)"
  echo "Sichere vorhandene Binary nach $BACKUP_PATH"
  sudo cp -a "$INSTALL_PATH" "$BACKUP_PATH"
fi
sudo install -m 0755 "$TMP_DIR/$BINARY_NAME" "${INSTALL_PATH}.new"
sudo mv -f "${INSTALL_PATH}.new" "$INSTALL_PATH"

# Konfiguration nicht überschreiben, damit lokale Einstellungen erhalten bleiben.
sudo install -d -o pi -g pi -m 0755 "$CONFIG_DIR"
if [[ ! -f "$CONFIG_PATH" ]]; then
  cat > "$TMP_DIR/config.yaml" <<'CFG'
# Web server
hosts_file: "/var/lib/dnsmasq/dnsmasq.dhcphosts"
host: "127.0.0.1"
port: "3009"
CFG
  sudo install -o pi -g pi -m 0644 "$TMP_DIR/config.yaml" "$CONFIG_PATH"
  echo "Standardkonfiguration angelegt: $CONFIG_PATH"
fi
if [[ -f "$SERVICE_SOURCE" ]]; then
  sudo install -m 0644 "$SERVICE_SOURCE" "$SERVICE_DEST"
else
  echo "Hinweis: $SERVICE_SOURCE nicht gefunden; vorhandene systemd-Service-Datei bleibt unverändert."
fi
sudo systemctl daemon-reload
if sudo systemctl cat "$SERVICE_NAME" >/dev/null 2>&1; then
  sudo systemctl enable "$SERVICE_NAME"
  sudo systemctl restart "$SERVICE_NAME"
  echo "Service-Status: sudo systemctl status $SERVICE_NAME"
else
  echo "Service nicht installiert. Lege $SERVICE_NAME in $BASE_DIR ab und führe das Skript erneut aus." >&2
fi

echo
echo "Update abgeschlossen."
echo "Binary:       $INSTALL_PATH"
echo "Konfiguration: $CONFIG_PATH"
echo "Service:      $SERVICE_DEST"
