#!/bin/bash
set -euo pipefail
BIN=${1:-./deaddrop-server}
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
DEPLOY_DIR=$(cd "$SCRIPT_DIR/.." && pwd)
if [ "$(id -u)" -ne 0 ]; then echo "run as root" >&2; exit 1; fi
apt-get update
apt-get install -y tor sqlite3 curl ufw ca-certificates logrotate
id deaddrop >/dev/null 2>&1 || useradd -r -s /usr/sbin/nologin deaddrop
install -o deaddrop -g deaddrop -m 0750 "$BIN" /usr/local/bin/deaddrop-server
install -d -o deaddrop -g deaddrop -m 0700 /var/lib/deaddrop
install -d -o deaddrop -g adm -m 0750 /var/log/deaddrop
if ! grep -q 'HiddenServiceDir /var/lib/tor/deaddrop/' /etc/tor/torrc; then cat "$DEPLOY_DIR/tor/torrc.server.snippet" >> /etc/tor/torrc; fi
systemctl reload tor || systemctl restart tor
sleep 5
install -m 0755 "$DEPLOY_DIR/scripts/wait-for-tor" /usr/local/bin/wait-for-tor
install -m 0755 "$DEPLOY_DIR/scripts/deaddrop-vacuum" /usr/local/bin/deaddrop-vacuum
install -m 0644 "$DEPLOY_DIR/systemd/deaddrop-server.service" /etc/systemd/system/deaddrop-server.service
install -m 0644 "$DEPLOY_DIR/logrotate/deaddrop" /etc/logrotate.d/deaddrop
install -m 0644 "$DEPLOY_DIR/scripts/deaddrop.cron" /etc/cron.d/deaddrop
install -m 0644 "$DEPLOY_DIR/sysctl/99-deaddrop-server.conf" /etc/sysctl.d/99-deaddrop-server.conf
sysctl --system >/dev/null || true
systemctl daemon-reload
systemctl enable --now deaddrop-server
cat /var/lib/tor/deaddrop/hostname
