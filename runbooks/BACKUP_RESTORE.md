# Backup and restore runbook

## Backup scope

Back up only persistent server state:

1. `/var/lib/deaddrop/drops.db`
2. `/var/lib/tor/deaddrop/`

The database contains ciphertext only. The onion directory contains the server onion identity key and must be protected.

## Backup command

Set GPG recipient:

```bash
export GPG_RECIPIENT='operator@example'
sudo -E deploy/scripts/backup-deaddrop.sh
```

Backups are written to `/var/backups/deaddrop` by default and unencrypted intermediates are shredded.

## Restore database

```bash
gpg --decrypt drops-YYYYMMDD-HHMMSS.db.gpg > drops-restore.db
sudo systemctl stop deaddrop-server
sudo cp /var/lib/deaddrop/drops.db /var/lib/deaddrop/drops.db.pre-restore
sudo cp drops-restore.db /var/lib/deaddrop/drops.db
sudo chown deaddrop:deaddrop /var/lib/deaddrop/drops.db
sudo chmod 600 /var/lib/deaddrop/drops.db
sudo systemctl start deaddrop-server
```

Verify:

```bash
curl --socks5-hostname 127.0.0.1:9050 http://<onion>/health
```

## Restore onion key

Only restore onion keys from trusted backups. Stop Tor and backend before replacing `/var/lib/tor/deaddrop/`.
