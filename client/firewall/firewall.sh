#!/bin/bash
# /usr/local/lib/deaddrop/firewall.sh
# CANONICAL DEADDROP-OS FIREWALL RULESET v2.0.0
# DO NOT MODIFY WITHOUT SPEC VERSION BUMP + SECURITY LEAD SIGN-OFF
set -euo pipefail
TOR_UID=$(id -u debian-tor 2>/dev/null || id -u tor)
TRANS_PORT=9040
DNS_PORT=5353
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X
ip6tables -F
ip6tables -X
ip6tables -P INPUT DROP
ip6tables -P OUTPUT DROP
ip6tables -P FORWARD DROP
iptables -P INPUT DROP
iptables -P OUTPUT DROP
iptables -P FORWARD DROP
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -t nat -A OUTPUT ! -o lo -m owner ! --uid-owner "$TOR_UID" -p tcp -j REDIRECT --to-ports "$TRANS_PORT"
iptables -t nat -A OUTPUT ! -o lo -m owner ! --uid-owner "$TOR_UID" -p udp --dport 53 -j REDIRECT --to-ports "$DNS_PORT"
iptables -A OUTPUT -m owner --uid-owner "$TOR_UID" -j ACCEPT
iptables -A OUTPUT ! -o lo -p udp -j DROP
iptables -A OUTPUT -p icmp -j DROP
iptables -A OUTPUT -j LOG --log-prefix "DEADDROP-DROP: " --log-level 4
echo "DEADDROP-OS firewall rules applied successfully."
echo "Default policy: DROP. Tor transparent proxy: ACTIVE."
