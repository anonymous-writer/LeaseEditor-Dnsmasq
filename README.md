# LeaseEditor-Dnsmasq

**Your network. Your reservations. Clearly managed.**

LeaseEditor-Dnsmasq is a lightweight Go web application for editing static DHCP reservations in a `dnsmasq` hosts file. It provides a browser-based interface for managing reservations, comments, and groups without manually editing the file line by line.

- **GitHub repository:** https://github.com/anonymous-writer/LeaseEditor-Dnsmasq
- **Related project:** [LeaseLens-Dnsmasq](https://github.com/anonymous-writer/LeaseLens-Dnsmasq) — a dashboard for viewing and monitoring DHCP leases.
- **License:** MIT — see [LICENSE](https://github.com/anonymous-writer/LeaseLens-Dnsmasq/blob/main/LICENSE).

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/leaseeditor-dnsmasq/main/docs/screenshot1.png)

## Features

- Read and edit the configured `dnsmasq.dhcphosts` file.
- Add, edit, and delete static DHCP reservations.
- Manage comment lines and group headings, including moving comments up and down.
- Assign reservations to groups and filter the table by group.
- Search reservations and comments.
- Preserve comments and unrecognized lines when saving, where supported by the file parser.
- Suggest IPv4 addresses based on addresses found in the current reservation file.
- Dark/light theme and English, German, and Spanish interface options.
- Create a `.bak` backup before saving changes.
- Optional button to restart the host's `dnsmasq` service in native Linux/systemd installations.
- Configurable log levels: `DEBUG`, `INFO`, `WARN`, and `ERROR`.
- Native Linux binaries for ARM64 and AMD64, and a multi-architecture Docker image published to GitHub Container Registry on GitHub Releases.

> **Important:** LeaseEditor edits the static DHCP hosts file. It does not directly rewrite `dnsmasq.leases`; that file is maintained by dnsmasq and contains active DHCP lease information.

## Quick start — Docker

The release workflow publishes multi-architecture images to GitHub Container Registry (GHCR). For the latest stable release:

```bash
docker pull ghcr.io/anonymous-writer/leaseeditor-dnsmasq:latest
```

Create a working directory containing `config.yaml` and `compose.yaml`, and create sure the configured hosts file exists and is writable by the container. Example configuration:

```yaml
# config.yaml
hosts_file: "/var/lib/dnsmasq/dnsmasq.dhcphosts"
host: "0.0.0.0"
port: "3009"
```

Example `compose.yaml`:

```yaml
services:
  leaseeditor-dnsmasq:
    image: ghcr.io/anonymous-writer/leaseeditor-dnsmasq:latest
    container_name: leaseeditor-dnsmasq
    restart: unless-stopped
    environment:
      LOG_LEVEL: "INFO"
    ports:
      - "3009:3009"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - /var/lib/dnsmasq:/var/lib/dnsmasq:rw
```

Adjust `/var/lib/dnsmasq` to the directory that contains your `dnsmasq.dhcphosts` file. The directory is mounted read-write because LeaseEditor must save the file and create backups. Start the application:

```bash
docker compose up -d
```

Open **http://YOUR_HOST:3009**, replacing `YOUR_HOST` with the address of the Docker host.

The container runs as a non-root user. Ensure the mounted directory and hosts file permissions allow that user to write changes. Do not solve permission issues by making the file world-writable; use an appropriate group or ownership arrangement instead.

## Native Linux installation

Prebuilt ARM64 and AMD64 binaries are attached to published GitHub Releases when the native build workflow succeeds. Download the archive for your system from the repository's **Releases** page, extract it, and make the binary executable.

Example:

```bash
chmod +x leaseeditor-dnsmasq-linux-arm64
./leaseeditor-dnsmasq-linux-arm64 --config ./config.yaml
```

Example `config.yaml`:

```yaml
hosts_file: "/var/lib/dnsmasq/dnsmasq.dhcphosts"
host: "127.0.0.1"
port: "3009"
```

Use `host: "127.0.0.1"` to allow access only from the local machine. Use `host: "0.0.0.0"` only when you intend to allow access from other interfaces and have appropriate network controls in place.

The supplied systemd unit uses `/home/pi/leaseeditor-dnsmasq/config.yaml`; adjust it for your installation path and service account if needed.

## Configuration

The native binary and Docker image use the same flat YAML-style configuration file.

| Key | Default | Description |
|---|---|---|
| `hosts_file` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Path to the static DHCP hosts file that LeaseEditor edits |
| `host` | `127.0.0.1` | Address the web server listens on |
| `port` | `3009` | TCP port for the web interface |
| `log_level` | `INFO` | Minimum log level: `DEBUG`, `INFO`, `WARN`, or `ERROR` |

The `LOG_LEVEL` environment variable can be used to set the minimum log level. When both a config value and environment variable are supplied, the environment variable takes precedence.

## Logging and troubleshooting

LeaseEditor writes timestamped log messages with severity levels to standard output. Under systemd, view the logs with:

```bash
journalctl -u leaseeditor-dnsmasq -f
```

For Docker Compose:

```bash
docker compose logs -f leaseeditor-dnsmasq
```

Log levels:

| `LOG_LEVEL` | Messages | Typical use |
|---|---|---|
| `DEBUG` | Debug, info, warnings, and errors | Temporary troubleshooting |
| `INFO` | Info, warnings, and errors | Normal operation (default) |
| `WARN` | Warnings and errors | Quieter operation |
| `ERROR` | Errors only | Important failures only |

Example Docker Compose override:

```yaml
environment:
  LOG_LEVEL: "DEBUG"
```

Restart or recreate the service after changing its environment. Avoid sharing logs publicly without checking them for sensitive infrastructure details.

## Restarting dnsmasq

The **Restart dnsmasq** button is intended for native Linux installations using systemd. It runs the fixed command `systemctl restart dnsmasq` and requires the included sudoers rule to be installed and configured for the actual service account.

The button does not restart the host's dnsmasq service from inside Docker. Docker deployments should restart the service on the host using the host's normal administration tools.

## Security

LeaseEditor has no built-in user authentication. It can modify a network's DHCP reservation file, so treat it as an administrative tool:

- Keep it on a trusted LAN or behind an authenticated reverse proxy.
- Do not expose it directly to the public internet.
- Restrict write permissions to the required hosts file and directory.
- Review the sudoers rule before enabling the native restart button.
- Keep backups of your dnsmasq configuration.

## Development

Requires Go. Run the tests and static analysis, then build:

```bash
go test ./...
go vet ./...
go build -trimpath -o leaseeditor-dnsmasq .
```

The web templates and static assets are embedded in the executable.

## Contributing

Issues and pull requests are welcome. For bug reports, include reproduction steps and relevant logs with private network details removed.

## License

MIT — see [LICENSE](https://github.com/anonymous-writer/LeaseLens-Dnsmasq/blob/main/LICENSE).
