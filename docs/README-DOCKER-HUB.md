# LeaseEditor-Dnsmasq

**Your network. Your reservations. Clearly managed.**

LeaseEditor-Dnsmasq is a lightweight Go web application for editing static DHCP reservations in a `dnsmasq` hosts file. Manage reservations, comments, and groups through a web interface instead of editing the file manually.

**Source code:** https://github.com/anonymous-writer/LeaseEditor-Dnsmasq  
**License:** MIT — see [LICENSE](https://github.com/anonymous-writer/LeaseLens-Dnsmasq/blob/main/LICENSE).

![Screenshot](https://raw.githubusercontent.com/anonymous-writer/leaseeditor-dnsmasq/main/docs/screenshot1.png)

## Features

- Add, edit, and delete static DHCP reservations.
- Manage comment lines and group headings; move comments up and down.
- Search reservations and comments, and filter by group.
- Preserve supported comments and unrecognized lines when saving.
- Create a `.bak` backup before saving.
- Dark/light theme and English, German, and Spanish interface options.
- `DEBUG`, `INFO`, `WARN`, and `ERROR` logging levels.
- Multi-architecture support: `linux/amd64` and `linux/arm64`.

LeaseEditor edits the static DHCP hosts file (usually `dnsmasq.dhcphosts`). It does not directly rewrite `dnsmasq.leases`, which is maintained by dnsmasq.

## Quick start

Create a directory for your configuration and prepare `config.yaml`:

```yaml
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

Change the host directory to the location of your `dnsmasq.dhcphosts` file. The directory must be writable by the container's application user so LeaseEditor can save changes and create `.bak` backups.

Start the container:

```bash
docker compose up -d
```

Then open **http://YOUR_HOST:3009**.

You can also pull the stable GHCR image directly:

```bash
docker pull ghcr.io/anonymous-writer/leaseeditor-dnsmasq:latest
```

## Configuration

| Setting | Default | Description |
|---|---|---|
| `hosts_file` | `/var/lib/dnsmasq/dnsmasq.dhcphosts` | Static DHCP hosts file to edit |
| `host` | `127.0.0.1` | Web server listen address |
| `port` | `3009` | Web server port |
| `LOG_LEVEL` | `INFO` | Minimum logging level: `DEBUG`, `INFO`, `WARN`, or `ERROR` |

## Logs

```bash
docker compose logs -f leaseeditor-dnsmasq
```

Set `LOG_LEVEL: "DEBUG"` temporarily for more detailed troubleshooting. The interface and API do not include built-in authentication. Keep this application on a trusted network and do not expose it directly to the public internet.

## Important notes

- The **Restart dnsmasq** button is for native Linux/systemd installations only; it cannot restart the host service from inside the container.
- The container runs as a non-root user. Configure file and directory permissions deliberately rather than making the hosts file world-writable.
- This application edits static reservations, not the active lease database.

## License

MIT — see [LICENSE](https://github.com/anonymous-writer/LeaseLens-Dnsmasq/blob/main/LICENSE).
