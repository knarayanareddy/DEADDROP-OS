#!/usr/bin/env python3
"""Static security invariant checks for DEADDROP-OS source/config."""
from pathlib import Path
import re, sys
root = Path(__file__).resolve().parents[2]
failures = []

def require(path, needle):
    text = (root / path).read_text()
    if needle not in text:
        failures.append(f"{path}: missing {needle!r}")

fw = (root / 'client/firewall/firewall.sh').read_text()
for needle in [
    'iptables -P INPUT DROP', 'iptables -P OUTPUT DROP', 'iptables -P FORWARD DROP',
    'ip6tables -P INPUT DROP', 'ip6tables -P OUTPUT DROP', 'ip6tables -P FORWARD DROP',
    '--to-ports "$TRANS_PORT"', '--to-ports "$DNS_PORT"', '-p icmp -j DROP'
]:
    if needle not in fw:
        failures.append(f'firewall missing {needle}')

require('client/firewall/99-deaddrop-no-ipv6.conf', 'net.ipv6.conf.all.disable_ipv6 = 1')
require('deploy/tor/torrc.client', 'DataDirectory /run/tor')
require('deploy/tor/torrc.client', 'Log notice file /run/tor/tor.log')
require('deploy/systemd/deaddrop-firewall.service', 'Before=network-pre.target')
require('deploy/systemd/deaddrop-server.service', 'ExecStartPre=/usr/local/bin/wait-for-tor')

server_text = (root / 'server/server.go').read_text()
if re.search(r'logger\.Printf\([^\n]*(RemoteAddr|User-Agent|Hint|Payload)', server_text):
    failures.append('server logging may include prohibited metadata')


logrotate = (root / 'deploy/logrotate/deaddrop').read_text()
if 'SIGUSR1' in logrotate or 'systemctl kill' in logrotate:
    failures.append('logrotate must not signal the Go server')
if 'copytruncate' not in logrotate:
    failures.append('logrotate should use copytruncate for systemd append logs')

panic_unit = (root / 'deploy/systemd/deaddrop-panic.service').read_text()
if 'Restart=always' in panic_unit:
    failures.append('panic daemon must not Restart=always after successful wipe')
if 'activity.sock' not in panic_unit:
    failures.append('panic daemon service must expose activity socket for inactivity reset')

cron = (root / 'deploy/scripts/deaddrop.cron').read_text()
if 'deaddrop-expire' in cron:
    failures.append('TTL expiry should not be duplicated in cron and backend')

rl = (root / 'server/ratelimit.go').read_text()
if 'r.RemoteAddr }' in rl or 'endpoint + ":" + r.RemoteAddr' in rl:
    failures.append('rate limiter must not key on full RemoteAddr with ephemeral port')
if 'pruneLocked' not in rl:
    failures.append('rate limiter must prune expired buckets')

schema = (root / 'server/storage.go').read_text()
for forbidden in ['plaintext', 'sender', 'recipient', 'client_ip', 'ip_address']:
    if re.search(r'CREATE TABLE IF NOT EXISTS drops[\s\S]*' + forbidden, schema, re.I):
        failures.append(f'schema contains forbidden field {forbidden}')

if failures:
    print('STATIC SECURITY CHECKS FAILED')
    for f in failures: print('-', f)
    sys.exit(1)
print('static security checks passed')
