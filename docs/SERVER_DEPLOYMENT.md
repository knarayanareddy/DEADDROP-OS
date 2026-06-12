# Server deployment

## Target platform

- Debian 12 bookworm
- root/sudo access
- Tor available from Debian repositories
- persistent disk for SQLite and Tor onion key

## Build binary

```bash
./tools/build-binaries.sh
```

## Deploy

```bash
sudo deploy/scripts/deploy.sh artifacts/deaddrop-server
```

The script installs packages, user, directories, Tor hidden-service config, systemd service, logrotate, cron vacuum, and sysctl hardening.

## Onion hostname

```bash
sudo cat /var/lib/tor/deaddrop/hostname
```

Distribute the onion address out-of-band.

## Important paths

| Path | Purpose |
|---|---|
| `/usr/local/bin/deaddrop-server` | backend binary |
| `/var/lib/deaddrop/drops.db` | SQLite DB |
| `/var/lib/tor/deaddrop/` | onion private key and hostname |
| `/var/log/deaddrop/` | access/error/maintenance logs |
| `/etc/systemd/system/deaddrop-server.service` | service unit |

## Service commands

```bash
sudo systemctl status deaddrop-server
sudo systemctl restart deaddrop-server
sudo journalctl -u deaddrop-server
```

## Health check over Tor

```bash
curl --socks5-hostname 127.0.0.1:9050 \
  http://$(sudo cat /var/lib/tor/deaddrop/hostname)/health
```
