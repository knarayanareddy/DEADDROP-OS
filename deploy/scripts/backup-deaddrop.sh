#!/bin/bash
set -euo pipefail
: "${GPG_RECIPIENT:?set GPG_RECIPIENT to the operator backup recipient}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR=${BACKUP_DIR:-/var/backups/deaddrop}
DB_FILE=${DEADDROP_DB:-/var/lib/deaddrop/drops.db}
ONION_KEY_DIR=${ONION_KEY_DIR:-/var/lib/tor/deaddrop}
mkdir -p "$BACKUP_DIR"
sqlite3 "$DB_FILE" ".backup $BACKUP_DIR/drops-$TIMESTAMP.db"
tar czf "$BACKUP_DIR/onion-key-$TIMESTAMP.tar.gz" "$ONION_KEY_DIR"
gpg --recipient "$GPG_RECIPIENT" --encrypt "$BACKUP_DIR/drops-$TIMESTAMP.db"
gpg --recipient "$GPG_RECIPIENT" --encrypt "$BACKUP_DIR/onion-key-$TIMESTAMP.tar.gz"
shred -u "$BACKUP_DIR/drops-$TIMESTAMP.db" "$BACKUP_DIR/onion-key-$TIMESTAMP.tar.gz"
find "$BACKUP_DIR" -name "*.gpg" -mtime +14 -delete
