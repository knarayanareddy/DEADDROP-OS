# Operations

## Routine checks

```bash
systemctl status deaddrop-server
curl --socks5-hostname 127.0.0.1:9050 http://<onion>/health
sqlite3 /var/lib/deaddrop/drops.db 'SELECT COUNT(*) FROM drops;'
```

## Logs

```text
/var/log/deaddrop/access.log
/var/log/deaddrop/error.log
/var/log/deaddrop/vacuum.log
```

Logs are rotated by `deploy/logrotate/deaddrop`. Access logs retain 7 rotations; error/vacuum logs retain 30.

## Maintenance

Daily cron performs:

```bash
/usr/local/bin/deaddrop-vacuum
```

TTL expiry is handled by the Go backend.

## Updates

1. Build new binary.
2. Back up old binary.
3. Stop service.
4. Replace binary.
5. Start service.
6. Verify health.

## Backups

Back up:

- SQLite database;
- Tor onion key directory.

Use `deploy/scripts/backup-deaddrop.sh` with `GPG_RECIPIENT` set.
