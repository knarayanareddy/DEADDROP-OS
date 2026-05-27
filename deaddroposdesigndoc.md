💀 DEADDROP OS
Comprehensive Engineering Design Document
Version 1.0 | Amnesic Live OS + Tor-Forced Networking + Encrypted Dead Drop Messenger
TABLE OF CONTENTS

    Project Overview & Vision
    Goals, Non-Goals & Constraints
    System Architecture
    Module Breakdown
        4.1 Base OS Layer (Debian Live + Amnesia)
        4.2 Network Enforcement Layer (Tor Transparent Proxy)
        4.3 Dead Drop Messenger (Encrypted Bulletin Board)
        4.4 Onion Service Backend (Go + bine)
        4.5 Cryptography Engine
        4.6 Panic Mode Daemon
        4.7 Boot & Init System
        4.8 Build Pipeline (live-build + GitHub Actions)
    Data Models & Schemas
    API Specifications
    Directory Structure
    Configuration System
    Cryptography & Key Management
    Network Enforcement Deep Dive
    Dead Drop Messenger Deep Dive
    Onion Service Backend Deep Dive
    Panic Mode Deep Dive
    Live OS & Amnesia System Deep Dive
    Security Model & Threat Boundaries
    Storage & Persistence
    Logging & Observability
    Testing Strategy
    Build, Packaging & Installation
    Platform Support Matrix
    Performance Targets & Benchmarks
    Error Handling Strategy
    Dependency Registry
    Milestone & Phased Rollout Plan
    Open Questions & Future Work

1. Project Overview & Vision
1.1 What is DeadDrop OS?

DeadDrop OS is a minimal, amnesic, Tor-forced live operating system designed for high-sensitivity communication under adversarial conditions. It runs entirely from RAM via a bootable USB drive, leaves no trace on the host machine, and routes every packet through the Tor anonymity network. It includes a built-in encrypted messaging system called Dead Drop Messenger — a bulletin-board-style asynchronous communication tool that uses public-key hybrid encryption over a Tor onion service, meaning neither sender nor receiver ever exchange messages directly.

DeadDrop OS is built on four interlocking pillars:
Pillar	Description
Amnesia	All writes go to tmpfs. Shutdown destroys the session entirely. No disk persistence.
Forced Anonymity	Every network packet — without exception — is routed through Tor via a transparent proxy + iptables firewall. No clearnet traffic is possible.
Encrypted Dead Drop	Messages are posted to a shared onion-service bulletin board as encrypted blobs. Only the holder of the correct private key can decrypt them.
Panic Mode	A hardware hotkey combination instantly stops sensitive processes, drops caches, and switches the UI to a plausible decoy workspace.
1.2 The Problem Being Solved

Modern secure communication tools assume a cooperative environment: a persistent device, a stable identity, and a network that can be partially trusted. DeadDrop OS is built for the opposite scenario:

    The device may be confiscated before, during, or after a session
    The network is assumed to be fully monitored
    Sender and receiver may never be online simultaneously
    Any persistent artifact — log file, cached key, browsing history — is a liability
    A physical interruption (search, checkpoint, confrontation) must not expose the active session

DeadDrop OS gives operators a boot-and-burn tool: boot from USB, communicate, shut down, hand the machine to anyone. Nothing remains.
1.3 Design Philosophy
Principle	Description
Amnesic by default	Persistence is never the default. It must be explicitly, consciously added.
Network enforcement is a hard guarantee	Not a setting. Not a toggle. The firewall rules are applied before any user process can reach the network.
Encryption is mandatory, not optional	The messenger has no plaintext mode. There is no "send unencrypted" option.
Panic is a first-class feature	Panic Mode is not an afterthought. It is designed, tested, and documented as a primary security control.
Minimal attack surface	The OS ships only what is needed. No desktop environment beyond a minimal window manager. No unnecessary daemons.
Transparency	Every network decision, every message action is logged to RAM (not disk) and visible in the TUI.
2. Goals, Non-Goals & Constraints
2.1 Goals (In Scope)

    Bootable ISO image built on Debian Live (bookworm, amd64)
    All filesystem writes routed to tmpfs (RAM); zero writes to host disk by default
    Tor transparent proxy with iptables/nftables enforcement covering all egress TCP/DNS traffic
    IPv6 completely disabled at kernel and firewall level (no leak surface)
    Dead Drop Messenger: a Textual-based TUI for composing, posting, fetching, and decrypting messages
    Hybrid encryption (RSA-OAEP key encapsulation + AES-256-GCM message encryption)
    Go-based onion service backend (embedded Tor via bine, SQLite with WAL mode, TTL-based expiry)
    Shared onion service model: one long-running .onion address serves all participants
    Panic Mode daemon: hotkey-triggered process kill, decoy launch, cache drop
    systemd units for Tor, firewall, onion backend, messenger
    live-build config with automated ISO artifact via GitHub Actions
    Per-boot ephemeral identity (no persistent keypair unless operator explicitly provisions one)
    Platform target: x86_64 hardware with BIOS/UEFI boot

2.2 Non-Goals (Explicitly Out of Scope)

    ❌ Persistent storage of any kind in default configuration
    ❌ Full desktop environment (GNOME, KDE, etc.)
    ❌ Clearnet browsing or split-tunnel networking
    ❌ Any feature that requires the OS to phone home
    ❌ Mobile device support
    ❌ Graphical (non-TUI) messenger interface
    ❌ Automatic key exchange or key discovery (keys are provisioned out-of-band)
    ❌ Multi-hop or cascaded onion service routing (single Tor circuit is sufficient and correct)
    ❌ "Secure RAM wipe" guarantees via software alone (hardware cold-boot attacks are out of scope)
    ❌ Forward secrecy via ratcheting (static RSA keypairs are used; this is a design tradeoff noted in §25)

2.3 Constraints

    Must boot on hardware with ≥ 2GB RAM (4GB recommended for comfortable tmpfs headroom)
    ISO size must be ≤ 1.5GB to fit comfortably on a standard USB drive
    Tor must be fully bootstrapped before any user-space process is allowed network access
    The iptables rules must be applied by a pre-network systemd unit that fires before network.target
    The onion backend private key must persist across reboots of the server (not the client OS); this requires a provisioned key on the server, not on user machines
    All Python code targets Python 3.11+ (Debian bookworm default)
    All Go code targets Go 1.22+
    The messenger TUI must be fully operable via keyboard; no mouse required

3. System Architecture
3.1 High-Level Architecture Diagram

text

┌──────────────────────────────────────────────────────────────────────────┐
│                          DEADDROP OS (RAM Only)                          │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                     BASE OS LAYER (tmpfs)                           │ │
│  │  Debian Live bookworm │ Openbox WM │ systemd │ No swap │ No logs   │ │
│  └────────────────────────────────┬────────────────────────────────────┘ │
│                                   │                                      │
│  ┌────────────────────────────────▼────────────────────────────────────┐ │
│  │                  NETWORK ENFORCEMENT LAYER                          │ │
│  │                                                                     │ │
│  │  ┌─────────────────────────┐   ┌─────────────────────────────────┐ │ │
│  │  │   iptables/nftables     │   │          Tor Daemon             │ │ │
│  │  │   (drop all non-Tor)    │◄──│  TransPort 9040  DNSPort 5353  │ │ │
│  │  │   IPv6: DISABLED        │   │  SocksPort 9050                 │ │ │
│  │  └─────────────────────────┘   └──────────────┬──────────────────┘ │ │
│  └────────────────────────────────────────────────│────────────────────┘ │
│                                                   │                      │
│  ┌────────────────────────────────────────────────▼────────────────────┐ │
│  │                    APPLICATION LAYER                                │ │
│  │                                                                     │ │
│  │  ┌──────────────────────┐    ┌───────────────────────────────────┐ │ │
│  │  │  Dead Drop Messenger  │    │     Onion Service Backend         │ │ │
│  │  │  (Python + Textual)  │◄──►│     (Go + bine + SQLite)          │ │ │
│  │  │  TUI on /dev/tty1    │    │     .onion:80 → localhost:8080    │ │ │
│  │  └──────────────────────┘    └───────────────────────────────────┘ │ │
│  │                                                                     │ │
│  │  ┌──────────────────────┐    ┌───────────────────────────────────┐ │ │
│  │  │    Panic Mode Daemon │    │      Decoy Applications           │ │ │
│  │  │    (Python/pynput)   │    │  (LibreOffice, Firefox → news)    │ │ │
│  │  └──────────────────────┘    └───────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │ All traffic: Tor only
                                     ▼
                         ┌───────────────────────┐
                         │     TOR NETWORK        │
                         │  (.onion services)     │
                         └───────────────────────┘

3.2 Boot Sequence (Step by Step)

text

Step 1:  BIOS/UEFI loads GRUB from USB
Step 2:  GRUB loads kernel + initrd with parameters:
         boot=live components live-config.noautologin=false
         live-config.username=user live-config.hostname=deaddrop
         net.ifnames=0 biosdevname=0 ipv6.disable=1
Step 3:  live-boot mounts squashfs root + overlays tmpfs (union mount)
         ALL writes from this point go to RAM
Step 4:  systemd starts. Unit order:
         deaddrop-firewall.service (before network.target)
           → Applies DROP-all iptables rules BEFORE any NIC is brought up
Step 5:  Tor daemon starts (deaddrop-tor.service)
           → Bootstraps to Tor network
           → Signals readiness via ControlPort "BOOTSTRAP PROGRESS=100"
Step 6:  Post-bootstrap: iptables rules updated to allow Tor egress by UID
Step 7:  deaddrop-onion.service starts (Go backend)
           → Creates/loads .onion keypair
           → Binds onion listener
           → Initializes SQLite (:memory: or tmpfs-backed)
Step 8:  deaddrop-messenger.service starts
           → Presents Textual TUI on /dev/tty1
Step 9:  deaddrop-panic.service starts
           → pynput listener active (Ctrl+Alt+P)
Step 10: User session begins. No disk has been touched.

3.3 Component Ownership
Component	Language	Primary Owns
Base OS	Debian Live / live-build	Filesystem, boot, amnesia
Firewall	bash / nftables	Pre-boot traffic enforcement
Tor daemon	C (upstream)	Anonymity routing
Onion Backend	Go	Message storage, onion service
Dead Drop Messenger	Python 3 + Textual	TUI, message composition, crypto
Cryptography Engine	Python (cryptography lib)	Key gen, encrypt/decrypt
Panic Mode	Python + pynput	Hotkey listen, decoy launch
Build System	live-build + GitHub Actions	ISO production
4. Module Breakdown
4.1 Base OS Layer (Debian Live + Amnesia)
Purpose

The Base OS layer is the foundation. It provides a minimal, read-only root filesystem that is overlaid with a RAM-backed writable layer at boot time. When the machine shuts down or the USB is removed, the overlay is gone. There is nothing left on the host machine.
Amnesia Mechanism

Debian Live uses live-boot which, at boot time, mounts the compressed squashfs image read-only, then creates a writable overlay using tmpfs. All changes — file writes, installs, config modifications — go to this overlay, which lives entirely in RAM.

text

/         ← union mount (overlayfs)
├── lower ← squashfs (USB, read-only, the OS)
└── upper ← tmpfs (RAM, writable, ephemeral)

On shutdown or power cut, the upper layer (tmpfs) vanishes. The squashfs is unmodified. The host disk is never touched unless explicitly mounted by the operator.
Key Amnesia Properties Required
Property	Implementation
No swap	Swap disabled entirely. /etc/fstab contains no swap entries. swapoff -a in firewall pre-script.
No journald persistence	Storage=volatile in /etc/systemd/journald.conf
No disk auto-mount	udisks2 removed. automount units masked. udev rules suppress auto-mount.
No core dumps	ulimit -c 0 in /etc/profile.d/deaddrop.sh. kernel.core_pattern=/dev/null in sysctl.
tmpfs for /tmp and /run	Standard on Debian Live; explicitly verified in live-build config.
live-build Configuration

Bash

# config/build.sh — run once to initialize live-build config

lb config \
  --distribution bookworm \
  --archive-areas "main contrib non-free non-free-firmware" \
  --binary-images iso-hybrid \
  --bootappend-live "boot=live live-config.noautologin=false \
    live-config.username=user live-config.hostname=deaddrop \
    live-config.locales=en_US.UTF-8 live-config.timezone=UTC \
    live-config.noeject ipv6.disable=1 net.ifnames=0 biosdevname=0 \
    systemd.journald.storage=volatile" \
  --debian-installer false \
  --memtest none \
  --win32-loader false \
  --firmware-binary true \
  --firmware-chroot true \
  --mirror-bootstrap http://deb.debian.org/debian \
  --mirror-chroot http://deb.debian.org/debian

Package List

text

# config/package-lists/deaddrop.list.chroot

# Core minimal system
openbox
obconf
xorg
xinit
xterm

# Tor
tor
tor-geoipdb

# Python messenger stack
python3
python3-pip
python3-venv

# Build tools (for Go onion backend, stripped in final ISO)
# golang-go -- installed via hook, binary copied, toolchain removed

# Utilities
curl
wget
nftables
iptables
iproute2
dnsutils
git
vim

# Decoy applications (Panic Mode)
libreoffice-calc
firefox-esr
feh

# Remove unnecessary services
# (Done via hooks — see config/hooks/)

Hardening Hooks

Bash

# config/hooks/live/0010-harden.hook.chroot

#!/bin/bash
set -e

# Disable IPv6 system-wide
echo "net.ipv6.conf.all.disable_ipv6 = 1" >> /etc/sysctl.conf
echo "net.ipv6.conf.default.disable_ipv6 = 1" >> /etc/sysctl.conf
echo "net.ipv6.conf.lo.disable_ipv6 = 1" >> /etc/sysctl.conf

# Disable core dumps
echo "kernel.core_pattern = /dev/null" >> /etc/sysctl.conf
echo "* hard core 0" >> /etc/security/limits.conf
echo "* soft core 0" >> /etc/security/limits.conf

# Journal to volatile only
mkdir -p /etc/systemd/journald.conf.d
cat > /etc/systemd/journald.conf.d/volatile.conf <<EOF
[Journal]
Storage=volatile
Compress=no
MaxRetentionSec=0
EOF

# Mask unused services
systemctl mask bluetooth.service
systemctl mask cups.service
systemctl mask avahi-daemon.service
systemctl mask ModemManager.service
systemctl mask NetworkManager.service  # We use systemd-networkd directly

# Remove suid bits from unnecessary binaries
chmod -s /usr/bin/sudo || true
chmod -s /bin/su || true

# Lock root password
passwd -l root

echo "[deaddrop] Hardening hooks applied."

4.2 Network Enforcement Layer (Tor Transparent Proxy)
Purpose

The Network Enforcement Layer's job is a single, absolute guarantee: no IP packet from any process reaches the internet without passing through Tor. This is enforced at the kernel level via iptables/nftables rules that are applied before any network interface is brought up.
Tor Configuration (torrc)

text

# /etc/tor/torrc

# Transparent proxy
TransPort 127.0.0.1:9040
TransListenAddress 127.0.0.1

# DNS proxy
DNSPort 127.0.0.1:5353
DNSListenAddress 127.0.0.1

# Onion address mapping
AutomapHostsOnResolve 1
AutomapHostsSuffixes .onion,.exit
VirtualAddrNetworkIPv4 10.192.0.0/10

# SOCKS for apps that support it
SocksPort 127.0.0.1:9050

# Control port (for bootstrap detection)
ControlPort 127.0.0.1:9051
CookieAuthentication 1

# Logging (RAM only — /tmp is tmpfs)
Log notice file /tmp/tor.log

# Strict node settings (no exit via .exit notation)
AllowDotExit 0
ExitPolicy reject *:*

# Data directory on tmpfs
DataDirectory /tmp/tor-data

Firewall Enforcement Script

Bash

# /usr/local/lib/deaddrop/firewall-init.sh
# Applied by deaddrop-firewall.service BEFORE network.target

#!/bin/bash
set -euo pipefail

TOR_UID=$(id -u debian-tor)
TRANS_PORT=9040
DNS_PORT=5353
LOOPBACK="127.0.0.1"
VIRT_ADDR="10.192.0.0/10"

echo "[firewall] Initializing DeadDrop OS firewall..."

# ── 1. ENSURE ALL EXISTING RULES ARE FLUSHED ──────────────────────────────
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X

# ── 2. SET DEFAULT POLICIES TO DROP ───────────────────────────────────────
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT DROP

# ── 3. ALLOW LOOPBACK UNCONDITIONALLY ─────────────────────────────────────
iptables -A INPUT  -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# ── 4. ALLOW TOR DAEMON OUTBOUND TRAFFIC (by UID) ─────────────────────────
# Tor itself must reach the internet to bootstrap
iptables -A OUTPUT -m owner --uid-owner "$TOR_UID" -j ACCEPT

# ── 5. ALLOW ESTABLISHED / RELATED INBOUND (for Tor circuits) ─────────────
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# ── 6. NAT: REDIRECT DNS TO TOR'S DNSPORT ─────────────────────────────────
iptables -t nat -A OUTPUT -p udp --dport 53 \
  -j REDIRECT --to-ports "$DNS_PORT"
iptables -t nat -A OUTPUT -p tcp --dport 53 \
  -j REDIRECT --to-ports "$DNS_PORT"

# ── 7. NAT: REDIRECT VIRTUAL .ONION ADDRESSES TO TRANS PORT ───────────────
iptables -t nat -A OUTPUT -d "$VIRT_ADDR" -p tcp \
  -j REDIRECT --to-ports "$TRANS_PORT"

# ── 8. NAT: REDIRECT ALL OTHER TCP TO TOR TRANS PORT ──────────────────────
iptables -t nat -A OUTPUT -p tcp \
  ! -d "$LOOPBACK" \
  -j REDIRECT --to-ports "$TRANS_PORT"

# ── 9. DROP ALL UDP (Tor does not proxy UDP; no leaks permitted) ───────────
iptables -A OUTPUT -p udp ! -d "$LOOPBACK" -j DROP

# ── 10. DROP ALL ICMP (prevents ping-based fingerprinting) ────────────────
iptables -A OUTPUT -p icmp -j DROP

# ── 11. DROP EVERYTHING ELSE OUTBOUND ─────────────────────────────────────
iptables -A OUTPUT ! -o lo -j DROP

# ── 12. DISABLE IPv6 AT FIREWALL LEVEL (belt-and-suspenders) ──────────────
ip6tables -P INPUT   DROP
ip6tables -P FORWARD DROP
ip6tables -P OUTPUT  DROP
ip6tables -F
ip6tables -X

echo "[firewall] All rules applied. IPv6 hard-blocked. Tor-only egress enforced."

systemd Unit: Firewall

ini

# /etc/systemd/system/deaddrop-firewall.service

[Unit]
Description=DeadDrop OS Firewall (Tor-Only Enforcement)
DefaultDependencies=no
Before=network-pre.target network.target
After=local-fs.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/lib/deaddrop/firewall-init.sh
ExecStop=/usr/local/lib/deaddrop/firewall-flush.sh

[Install]
WantedBy=multi-user.target

systemd Unit: Tor

ini

# /etc/systemd/system/deaddrop-tor.service

[Unit]
Description=DeadDrop OS Tor Daemon
Requires=deaddrop-firewall.service
After=deaddrop-firewall.service

[Service]
Type=forking
User=debian-tor
Group=debian-tor
ExecStart=/usr/bin/tor -f /etc/tor/torrc
ExecStartPost=/usr/local/lib/deaddrop/wait-for-tor.sh
PIDFile=/tmp/tor-data/tor.pid
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target

Tor Bootstrap Wait Script

Bash

# /usr/local/lib/deaddrop/wait-for-tor.sh
# Blocks systemd unit startup until Tor reports 100% bootstrap

#!/bin/bash
set -euo pipefail

MAX_WAIT=120
ELAPSED=0

echo "[tor-wait] Waiting for Tor to bootstrap..."

while [ "$ELAPSED" -lt "$MAX_WAIT" ]; do
  PROGRESS=$(python3 -c "
import socket, time
try:
    s = socket.socket()
    s.connect(('127.0.0.1', 9051))
    s.send(b'AUTHENTICATE \"\"\r\n')
    s.recv(256)
    s.send(b'GETINFO status/bootstrap-phase\r\n')
    r = s.recv(1024).decode()
    s.close()
    import re
    m = re.search(r'PROGRESS=(\d+)', r)
    print(m.group(1) if m else '0')
except:
    print('0')
" 2>/dev/null)

  if [ "$PROGRESS" = "100" ]; then
    echo "[tor-wait] Tor fully bootstrapped."
    exit 0
  fi

  echo "[tor-wait] Bootstrap at ${PROGRESS}%... (${ELAPSED}s elapsed)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
done

echo "[tor-wait] ERROR: Tor did not bootstrap within ${MAX_WAIT}s."
exit 1

4.3 Dead Drop Messenger (Encrypted Bulletin Board TUI)
Purpose

The Dead Drop Messenger is a Python + Textual TUI application that allows operators to compose, encrypt, post, fetch, and decrypt messages via a shared onion service bulletin board. Messages are encrypted to a recipient's public key before being posted. The server stores only ciphertext blobs — it cannot read any message.

The "dead drop" metaphor is intentional: sender and receiver never interact directly or simultaneously. The sender leaves an encrypted blob; the receiver picks it up. The server is the dead drop location.
Architecture

text

┌─────────────────────────────────────────────────────────┐
│             Dead Drop Messenger TUI                      │
│                                                         │
│  ┌─────────────────┐    ┌──────────────────────────┐   │
│  │  Compose Panel  │    │  Inbox Panel              │   │
│  │  - Recipient PEM│    │  - Fetch from onion       │   │
│  │  - Message body │    │  - Try decrypt each blob  │   │
│  │  - Encrypt btn  │    │  - Show decrypted msgs    │   │
│  └────────┬────────┘    └────────────┬─────────────┘   │
│           │                          │                  │
│  ┌────────▼──────────────────────────▼─────────────┐   │
│  │             Cryptography Engine                  │   │
│  │  RSA-4096 keygen │ Hybrid encrypt │ Hybrid decrypt│  │
│  └────────────────────────────┬────────────────────┘   │
│                                │                        │
│  ┌─────────────────────────────▼──────────────────┐   │
│  │          Onion Transport Layer                   │   │
│  │   HTTP over SOCKS5 → 127.0.0.1:9050 → .onion   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘

Textual TUI Implementation

Python

# src/messenger/app.py

from textual.app import App, ComposeResult
from textual.widgets import (
    Header, Footer, Input, Button, TextArea,
    DataTable, Label, Static, TabbedContent, TabPane
)
from textual.containers import Container, Horizontal, Vertical
from textual.reactive import reactive
from textual import work

from .crypto import CryptoEngine
from .transport import OnionTransport
from .models import Message, DropPayload

import asyncio
import json


class DeadDropApp(App):
    """DeadDrop OS Messenger — Encrypted Bulletin Board"""

    CSS = """
    Screen {
        background: #0d1117;
        color: #c9d1d9;
    }
    .panel {
        border: solid #30363d;
        padding: 1 2;
        margin: 1;
    }
    .panel-title {
        color: #58a6ff;
        text-style: bold;
        margin-bottom: 1;
    }
    Button {
        margin-top: 1;
    }
    Button.danger {
        background: #da3633;
    }
    Button.success {
        background: #238636;
    }
    .status-ok   { color: #3fb950; }
    .status-err  { color: #f85149; }
    .status-warn { color: #d29922; }
    DataTable {
        height: 15;
    }
    """

    BINDINGS = [
        ("ctrl+p", "panic", "PANIC"),
        ("ctrl+q", "quit", "Quit"),
        ("ctrl+n", "new_keypair", "New Keypair"),
    ]

    tor_status: reactive[str] = reactive("checking...")
    onion_address: reactive[str] = reactive("unknown")

    def __init__(self, config: dict):
        super().__init__()
        self.config = config
        self.crypto = CryptoEngine(config["keystore_path"])
        self.transport = OnionTransport(
            socks_host="127.0.0.1",
            socks_port=9050,
            onion_url=config["onion_url"],
        )
        self._inbox: list[Message] = []

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        with TabbedContent():
            with TabPane("📬 Compose", id="compose"):
                yield self._compose_panel()
            with TabPane("📥 Inbox", id="inbox"):
                yield self._inbox_panel()
            with TabPane("🔑 Keys", id="keys"):
                yield self._keys_panel()
            with TabPane("⚙️ Status", id="status"):
                yield self._status_panel()
        yield Footer()

    def _compose_panel(self) -> Container:
        return Vertical(
            Static("📬 COMPOSE ENCRYPTED MESSAGE", classes="panel-title"),
            Label("Recipient Public Key (PEM):"),
            TextArea(id="recipient_pubkey", language=None),
            Label("Hint (optional — visible to server, keep vague):"),
            Input(placeholder="e.g. 'msg-alpha'", id="hint"),
            Label("Message:"),
            TextArea(id="message_body", language=None),
            Horizontal(
                Button("🔐 Encrypt & Send", id="btn_send", classes="success"),
                Button("🔑 Load My Pubkey", id="btn_load_pubkey"),
            ),
            Static("", id="compose_status"),
            classes="panel",
        )

    def _inbox_panel(self) -> Container:
        table = DataTable(id="inbox_table")
        table.add_columns("ID", "Hint", "Posted", "Status")
        return Vertical(
            Static("📥 INBOX — ENCRYPTED DROPS", classes="panel-title"),
            Button("🔄 Fetch Drops", id="btn_fetch", classes="success"),
            table,
            Static("", id="inbox_status"),
            TextArea(id="decrypted_body", language=None),
            classes="panel",
        )

    def _keys_panel(self) -> Container:
        return Vertical(
            Static("🔑 KEY MANAGEMENT", classes="panel-title"),
            Static("⚠️  Keys exist only in RAM this session unless you export them.",
                   classes="status-warn"),
            Button("🔑 Generate New RSA-4096 Keypair", id="btn_genkey"),
            Static("", id="pubkey_display"),
            Button("📋 Export Public Key (copy to clipboard)", id="btn_export_pubkey"),
            classes="panel",
        )

    def _status_panel(self) -> Container:
        return Vertical(
            Static("⚙️ SYSTEM STATUS", classes="panel-title"),
            Static("Tor Status:", classes="panel-title"),
            Static(id="tor_status_display"),
            Static("Onion Address:", classes="panel-title"),
            Static(id="onion_address_display"),
            classes="panel",
        )

    async def on_mount(self) -> None:
        self.check_tor_status()
        self.fetch_onion_address()

    @work(exclusive=True)
    async def check_tor_status(self) -> None:
        try:
            status = await self.transport.check_tor()
            self.query_one("#tor_status_display", Static).update(
                f"[green]✅ Connected[/] — Circuit established"
                if status else f"[red]❌ Not connected[/]"
            )
        except Exception as e:
            self.query_one("#tor_status_display", Static).update(
                f"[red]❌ Error: {e}[/]"
            )

    @work(exclusive=True)
    async def fetch_onion_address(self) -> None:
        try:
            addr = await self.transport.get_onion_address()
            self.query_one("#onion_address_display", Static).update(addr)
        except Exception as e:
            self.query_one("#onion_address_display", Static).update(f"[red]{e}[/]")

    async def on_button_pressed(self, event: Button.Pressed) -> None:
        match event.button.id:
            case "btn_send":
                await self._send_message()
            case "btn_fetch":
                await self._fetch_inbox()
            case "btn_genkey":
                await self._generate_keypair()
            case "btn_load_pubkey":
                await self._load_own_pubkey()
            case "btn_export_pubkey":
                await self._export_pubkey()

    @work(exclusive=False)
    async def _send_message(self) -> None:
        status = self.query_one("#compose_status", Static)
        try:
            pubkey_pem = self.query_one("#recipient_pubkey", TextArea).text.strip()
            body = self.query_one("#message_body", TextArea).text.strip()
            hint = self.query_one("#hint", Input).value.strip()

            if not pubkey_pem or not body:
                status.update("[red]❌ Recipient public key and message are required.[/]")
                return

            status.update("[yellow]🔐 Encrypting...[/]")
            payload = self.crypto.encrypt_message(pubkey_pem, body)

            status.update("[yellow]📡 Sending via Tor...[/]")
            drop_id = await self.transport.post_drop(
                payload=payload,
                hint=hint,
            )

            status.update(f"[green]✅ Sent! Drop ID: {drop_id[:12]}...[/]")
            # Clear compose fields
            self.query_one("#recipient_pubkey", TextArea).clear()
            self.query_one("#message_body", TextArea).clear()
            self.query_one("#hint", Input).value = ""

        except Exception as e:
            status.update(f"[red]❌ Error: {e}[/]")

    @work(exclusive=False)
    async def _fetch_inbox(self) -> None:
        status = self.query_one("#inbox_status", Static)
        table = self.query_one("#inbox_table", DataTable)
        try:
            status.update("[yellow]📡 Fetching drops via Tor...[/]")
            drops = await self.transport.fetch_drops()
            table.clear()
            status.update(f"[green]Fetched {len(drops)} drops. Attempting decryption...[/]")

            decrypted_count = 0
            for drop in drops:
                try:
                    plaintext = self.crypto.decrypt_message(drop.payload)
                    table.add_row(
                        drop.id[:8] + "...",
                        drop.hint or "(none)",
                        drop.timestamp[:16],
                        "✅ Decrypted",
                    )
                    decrypted_count += 1
                    # Show first decrypted message in preview
                    if decrypted_count == 1:
                        self.query_one("#decrypted_body", TextArea).load_text(plaintext)
                except Exception:
                    table.add_row(
                        drop.id[:8] + "...",
                        drop.hint or "(none)",
                        drop.timestamp[:16],
                        "🔒 Not for us",
                    )

            status.update(
                f"[green]✅ {decrypted_count} message(s) decrypted "
                f"of {len(drops)} total drops.[/]"
            )
        except Exception as e:
            status.update(f"[red]❌ Fetch failed: {e}[/]")

    @work(exclusive=True)
    async def _generate_keypair(self) -> None:
        self.crypto.generate_keypair()
        pubkey_pem = self.crypto.get_public_key_pem()
        self.query_one("#pubkey_display", Static).update(
            f"[green]✅ New RSA-4096 keypair generated.[/]\n\n"
            f"Your Public Key:\n[yellow]{pubkey_pem[:80]}...[/]\n\n"
            f"[red]⚠️ This key exists in RAM only. Export it now if you need it.[/]"
        )

    @work(exclusive=True)
    async def _load_own_pubkey(self) -> None:
        pubkey_pem = self.crypto.get_public_key_pem()
        self.query_one("#compose_status", Static).update(
            f"[green]Your public key loaded above for sharing.[/]"
        )

    async def _export_pubkey(self) -> None:
        pubkey_pem = self.crypto.get_public_key_pem()
        import subprocess
        subprocess.run(["xclip", "-selection", "clipboard"],
                       input=pubkey_pem.encode(), check=False)
        self.query_one("#compose_status", Static).update(
            "[green]✅ Public key copied to clipboard.[/]"
        )

    def action_panic(self) -> None:
        """Triggered by Ctrl+P — also triggers the panic daemon externally."""
        import subprocess
        subprocess.Popen(["systemctl", "start", "deaddrop-panic-now.service"])
        self.exit()


if __name__ == "__main__":
    import sys
    config = {
        "keystore_path": "/tmp/deaddrop-keys",
        "onion_url": sys.argv[1] if len(sys.argv) > 1 else "http://placeholder.onion",
    }
    app = DeadDropApp(config)
    app.run()

4.4 Onion Service Backend (Go + bine)
Purpose

The Onion Service Backend is a Go application that creates and manages a Tor v3 onion service, accepting POST and GET requests for encrypted message blobs. It is the "dead drop location" — a server with a stable .onion address that operators post to and fetch from.

This service runs on the operator's own infrastructure (a VPS or dedicated machine), not on every client's DeadDrop OS instance. Client machines only boot, post/fetch, and shut down. The server is the persistent piece.
Why a Separate Server?

The distinction is critical:
Component	Where it runs	Persistent?
DeadDrop OS (client)	Operator's USB-booted laptop	No — amnesic
Onion Service Backend	Operator's dedicated server	Yes — needs stable .onion address

The onion backend's private key MUST persist across reboots (otherwise the .onion address changes and recipients lose it). This key lives on the server only, never on client machines.
Go Backend Implementation

Go

// src/onion/main.go

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil/ed25519"
	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
)

const (
	localPort      = 8080
	onionPort      = 80
	keyFile        = "/var/lib/deaddrop/onion_key.pem"
	dbFile         = "/var/lib/deaddrop/drops.db"
	maxPayloadSize = 64 * 1024  // 64KB per drop
	maxHintLen     = 64
	defaultTTLDays = 7
)

type Drop struct {
	ID        string    `json:"id"`
	Payload   string    `json:"payload"`   // base64 ciphertext
	Hint      string    `json:"hint"`      // cleartext routing hint
	Timestamp time.Time `json:"timestamp"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Server struct {
	db *sql.DB
}

func main() {
	ctx := context.Background()

	// ── 1. INITIALIZE DATABASE ────────────────────────────────────────────
	db, err := initDB(dbFile)
	if err != nil {
		log.Fatalf("[onion] DB init failed: %v", err)
	}
	defer db.Close()

	srv := &Server{db: db}

	// ── 2. START LOCAL HTTP SERVER ────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("POST /drop", srv.handlePostDrop)
	mux.HandleFunc("GET /drops", srv.handleGetDrops)
	mux.HandleFunc("DELETE /drop/{id}", srv.handleDeleteDrop)
	mux.HandleFunc("GET /health", srv.handleHealth)

	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	httpServer := &http.Server{
		Addr:         localAddr,
		Handler:      loggingMiddleware(rateLimitMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── 3. LOAD OR GENERATE ONION SERVICE KEY ─────────────────────────────
	key, err := loadOrGenerateKey(keyFile)
	if err != nil {
		log.Fatalf("[onion] Key load/generate failed: %v", err)
	}

	// ── 4. START TOR + CREATE ONION SERVICE ───────────────────────────────
	log.Println("[onion] Starting embedded Tor...")
	t, err := tor.Start(ctx, &tor.StartConf{
		DataDir:    "/var/lib/deaddrop/tor-data",
		TempDataDirBase: "/tmp",
	})
	if err != nil {
		log.Fatalf("[onion] Tor start failed: %v", err)
	}
	defer t.Close()

	// Wait for Tor to bootstrap
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer dialCancel()

	log.Println("[onion] Waiting for Tor to bootstrap...")
	dialer, err := t.Dialer(dialCtx, nil)
	if err != nil {
		log.Fatalf("[onion] Tor dialer failed: %v", err)
	}
	_ = dialer

	// Create onion service
	onion, err := t.Listen(ctx, &tor.ListenConf{
		LocalPort:  localPort,
		RemotePorts: []int{onionPort},
		Key:        key,
		Version3:   true,
	})
	if err != nil {
		log.Fatalf("[onion] Onion listen failed: %v", err)
	}
	defer onion.Close()

	log.Printf("[onion] ✅ Onion service ready: http://%v.onion", onion.ID)

	// Write address to file for system status queries
	_ = os.WriteFile("/var/lib/deaddrop/onion_address.txt",
		[]byte(onion.ID+".onion"), 0600)

	// ── 5. START BACKGROUND EXPIRY CLEANER ────────────────────────────────
	go srv.runExpiryLoop(ctx)

	// ── 6. SERVE ──────────────────────────────────────────────────────────
	log.Printf("[onion] HTTP server listening on %s", localAddr)
	if err := httpServer.Serve(onion); err != nil && err != net.ErrClosed {
		log.Fatalf("[onion] Serve error: %v", err)
	}
}

// ── HANDLERS ──────────────────────────────────────────────────────────────

func (s *Server) handlePostDrop(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > maxPayloadSize {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	var input struct {
		Payload   string `json:"payload"`
		Hint      string `json:"hint"`
		TTLDays   int    `json:"ttl_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Payload == "" {
		http.Error(w, "payload required", http.StatusBadRequest)
		return
	}

	// Sanitize hint
	if len(input.Hint) > maxHintLen {
		input.Hint = input.Hint[:maxHintLen]
	}

	ttl := defaultTTLDays
	if input.TTLDays > 0 && input.TTLDays <= 30 {
		ttl = input.TTLDays
	}

	drop := Drop{
		ID:        uuid.New().String(),
		Payload:   input.Payload,
		Hint:      input.Hint,
		Timestamp: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().AddDate(0, 0, ttl),
	}

	_, err := s.db.Exec(
		`INSERT INTO drops (id, payload, hint, timestamp, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		drop.ID, drop.Payload, drop.Hint, drop.Timestamp, drop.ExpiresAt,
	)
	if err != nil {
		log.Printf("[onion] DB insert error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": drop.ID})
}

func (s *Server) handleGetDrops(w http.ResponseWriter, r *http.Request) {
	hint := r.URL.Query().Get("hint")

	var rows *sql.Rows
	var err error

	if hint != "" {
		rows, err = s.db.Query(
			`SELECT id, payload, hint, timestamp, expires_at FROM drops
			 WHERE hint = ? AND expires_at > ? ORDER BY timestamp DESC LIMIT 100`,
			hint, time.Now().UTC(),
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, payload, hint, timestamp, expires_at FROM drops
			 WHERE expires_at > ? ORDER BY timestamp DESC LIMIT 100`,
			time.Now().UTC(),
		)
	}

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	drops := make([]Drop, 0)
	for rows.Next() {
		var d Drop
		if err := rows.Scan(&d.ID, &d.Payload, &d.Hint, &d.Timestamp, &d.ExpiresAt); err != nil {
			continue
		}
		drops = append(drops, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drops)
}

func (s *Server) handleDeleteDrop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := s.db.Exec("DELETE FROM drops WHERE id = ?", id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) runExpiryLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := s.db.Exec(
				"DELETE FROM drops WHERE expires_at < ?", time.Now().UTC(),
			)
			if err == nil {
				n, _ := result.RowsAffected()
				if n > 0 {
					log.Printf("[onion] Expired %d drops.", n)
				}
			}
		}
	}
}

// ── KEY MANAGEMENT ────────────────────────────────────────────────────────

func loadOrGenerateKey(path string) (ed25519.KeyPair, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[onion] No key found at %s — generating new keypair...", path)
		return generateAndSaveKey(path)
	}
	return loadKey(path)
}

func generateAndSaveKey(path string) (ed25519.KeyPair, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	kp, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	data, err := kp.PrivateKey().Marshal()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}
	log.Printf("[onion] New keypair generated and saved to %s", path)
	return kp, nil
}

func loadKey(path string) (ed25519.KeyPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pk, err := ed25519.PrivateKeyFromBytes(data)
	if err != nil {
		return nil, err
	}
	return ed25519.FromCrypto(pk.AsEd25519())
}

// ── DATABASE ──────────────────────────────────────────────────────────────

func initDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS drops (
			id         TEXT PRIMARY KEY,
			payload    TEXT NOT NULL,
			hint       TEXT NOT NULL DEFAULT '',
			timestamp  DATETIME NOT NULL,
			expires_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_drops_hint    ON drops(hint);
		CREATE INDEX IF NOT EXISTS idx_drops_expires ON drops(expires_at);
		CREATE INDEX IF NOT EXISTS idx_drops_ts      ON drops(timestamp);
	`)
	return db, err
}

4.5 Cryptography Engine
Purpose

The Cryptography Engine provides all cryptographic operations for the messenger. The critical design decision is to use hybrid encryption — a standard pattern for public-key message encryption:

    Generate a random 256-bit symmetric key (the "data encryption key" or DEK)
    Encrypt the message with AES-256-GCM using the DEK → produces ciphertext
    Encrypt the DEK with the recipient's RSA-4096 public key using OAEP → produces encrypted key
    Package both together as the final payload

This solves the fundamental limitation of raw RSA-OAEP: messages of any length can be encrypted.
Why Not Raw RSA-OAEP?

With RSA-4096 and SHA-256 OAEP, the maximum plaintext is: 512 - 2*32 - 2 = 446 bytes. A practical message easily exceeds this. Hybrid encryption has no message length limit (limited only by AES-GCM's ~64GB per nonce, which is irrelevant here).
Wire Format

text

payload (base64-encoded JSON):
{
  "version": 1,
  "enc_key": "<base64 RSA-OAEP encrypted DEK>",
  "iv":      "<base64 AES-GCM 12-byte nonce>",
  "ct":      "<base64 AES-GCM ciphertext>",
  "tag":     "<base64 AES-GCM 16-byte auth tag>"
}

Implementation

Python

# src/messenger/crypto.py

import os
import json
import base64
from pathlib import Path
from typing import Optional

from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.primitives.asymmetric.rsa import (
    RSAPrivateKey, RSAPublicKey
)
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.exceptions import InvalidTag

PAYLOAD_VERSION = 1
RSA_KEY_SIZE = 4096
AES_KEY_SIZE = 32   # 256 bits
GCM_NONCE_SIZE = 12 # bytes
OAEP_HASH = hashes.SHA256()
MGF_HASH = hashes.SHA256()


class CryptoEngine:
    """
    Hybrid encryption engine for DeadDrop Messenger.

    Encryption: RSA-4096-OAEP (key encapsulation) + AES-256-GCM (message)
    All operations are in-memory. Keys stored under keystore_path (tmpfs in live OS).
    """

    def __init__(self, keystore_path: str):
        self.keystore_path = Path(keystore_path)
        self.keystore_path.mkdir(parents=True, exist_ok=True, mode=0o700)
        self._private_key: Optional[RSAPrivateKey] = None
        self._auto_load()

    def _auto_load(self) -> None:
        """Auto-load existing keypair from keystore if present."""
        priv_path = self.keystore_path / "private.pem"
        if priv_path.exists():
            with open(priv_path, "rb") as f:
                self._private_key = serialization.load_pem_private_key(
                    f.read(), password=None
                )

    def generate_keypair(self) -> None:
        """Generate a new RSA-4096 keypair and store it in keystore_path."""
        self._private_key = rsa.generate_private_key(
            public_exponent=65537,
            key_size=RSA_KEY_SIZE,
        )
        priv_path = self.keystore_path / "private.pem"
        pub_path = self.keystore_path / "public.pem"

        priv_bytes = self._private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        )
        pub_bytes = self._private_key.public_key().public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo,
        )

        priv_path.write_bytes(priv_bytes)
        priv_path.chmod(0o600)
        pub_path.write_bytes(pub_bytes)
        pub_path.chmod(0o644)

    def get_public_key_pem(self) -> str:
        """Return PEM-encoded public key for sharing with correspondents."""
        if self._private_key is None:
            raise RuntimeError("No keypair loaded. Generate one first.")
        pub_bytes = self._private_key.public_key().public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo,
        )
        return pub_bytes.decode("utf-8")

    def encrypt_message(self, recipient_pubkey_pem: str, plaintext: str) -> str:
        """
        Encrypt a plaintext message for the given recipient.

        Returns: base64-encoded JSON payload (the 'drop').

        Algorithm:
          1. Generate random 256-bit AES key (DEK)
          2. Encrypt plaintext with AES-256-GCM → (ciphertext, nonce, tag)
          3. Encrypt DEK with recipient's RSA-4096-OAEP public key → enc_key
          4. Return JSON payload containing all of the above
        """
        # 1. Load recipient public key
        recipient_key: RSAPublicKey = serialization.load_pem_public_key(
            recipient_pubkey_pem.encode("utf-8")
        )

        # 2. Generate random DEK
        dek = os.urandom(AES_KEY_SIZE)

        # 3. Encrypt message with AES-256-GCM
        nonce = os.urandom(GCM_NONCE_SIZE)
        aesgcm = AESGCM(dek)
        # AESGCM.encrypt returns ciphertext + 16-byte tag appended
        ct_with_tag = aesgcm.encrypt(nonce, plaintext.encode("utf-8"), None)
        ciphertext = ct_with_tag[:-16]
        tag = ct_with_tag[-16:]

        # 4. Encrypt DEK with RSA-OAEP
        enc_key = recipient_key.encrypt(
            dek,
            padding.OAEP(
                mgf=padding.MGF1(algorithm=MGF_HASH),
                algorithm=OAEP_HASH,
                label=None,
            ),
        )

        # 5. Assemble and encode payload
        payload = {
            "version": PAYLOAD_VERSION,
            "enc_key": base64.b64encode(enc_key).decode(),
            "iv":      base64.b64encode(nonce).decode(),
            "ct":      base64.b64encode(ciphertext).decode(),
            "tag":     base64.b64encode(tag).decode(),
        }
        payload_bytes = json.dumps(payload).encode("utf-8")
        return base64.b64encode(payload_bytes).decode("utf-8")

    def decrypt_message(self, payload_b64: str) -> str:
        """
        Attempt to decrypt a drop payload with our private key.

        Raises:
          RuntimeError: No private key loaded
          DecryptionError: Payload was not encrypted to us (expected — caller should catch)
          InvalidTag: AES-GCM authentication failed (tampered or corrupted)
        """
        if self._private_key is None:
            raise RuntimeError("No private key loaded.")

        # 1. Decode outer base64 + JSON
        try:
            payload_bytes = base64.b64decode(payload_b64)
            payload = json.loads(payload_bytes)
        except Exception as e:
            raise ValueError(f"Malformed payload: {e}")

        if payload.get("version") != PAYLOAD_VERSION:
            raise ValueError(f"Unknown payload version: {payload.get('version')}")

        enc_key   = base64.b64decode(payload["enc_key"])
        nonce     = base64.b64decode(payload["iv"])
        ciphertext = base64.b64decode(payload["ct"])
        tag       = base64.b64decode(payload["tag"])

        # 2. Decrypt DEK with our RSA private key
        try:
            dek = self._private_key.decrypt(
                enc_key,
                padding.OAEP(
                    mgf=padding.MGF1(algorithm=MGF_HASH),
                    algorithm=OAEP_HASH,
                    label=None,
                ),
            )
        except ValueError:
            # RSA decrypt failed → this message was not encrypted to us
            raise DecryptionError("Message not addressed to this keypair.")

        # 3. Decrypt message with AES-256-GCM
        aesgcm = AESGCM(dek)
        try:
            plaintext_bytes = aesgcm.decrypt(nonce, ciphertext + tag, None)
        except InvalidTag:
            raise InvalidTag("AES-GCM authentication tag mismatch — message may be tampered.")

        return plaintext_bytes.decode("utf-8")


class DecryptionError(Exception):
    """Raised when decryption fails due to key mismatch (not tamper)."""
    pass

Transport Layer (SOCKS5 over Tor)

Python

# src/messenger/transport.py

import asyncio
import aiohttp
import aiohttp_socks
from typing import Optional
from .models import Drop
import base64
import json


class OnionTransport:
    """
    HTTP client that routes all requests through Tor SOCKS5 proxy.
    Never makes a clearnet connection under any circumstance.
    """

    def __init__(self, socks_host: str, socks_port: int, onion_url: str):
        self.socks_host = socks_host
        self.socks_port = socks_port
        self.onion_url = onion_url.rstrip("/")
        self._connector = aiohttp_socks.ProxyConnector.from_url(
            f"socks5://{socks_host}:{socks_port}"
        )

    async def _session(self) -> aiohttp.ClientSession:
        return aiohttp.ClientSession(
            connector=self._connector,
            timeout=aiohttp.ClientTimeout(total=120),
        )

    async def check_tor(self) -> bool:
        """Verify Tor is working by hitting a known .onion or check.torproject.org."""
        try:
            async with await self._session() as s:
                async with s.get("https://check.torproject.org/api/ip") as r:
                    data = await r.json()
                    return data.get("IsTor", False)
        except Exception:
            return False

    async def get_onion_address(self) -> str:
        """Read the onion address from the system status file."""
        try:
            with open("/var/lib/deaddrop/onion_address.txt") as f:
                return f.read().strip()
        except FileNotFoundError:
            return self.onion_url

    async def post_drop(self, payload: str, hint: str = "", ttl_days: int = 7) -> str:
        """Post an encrypted payload to the onion bulletin board."""
        async with await self._session() as s:
            async with s.post(
                f"{self.onion_url}/drop",
                json={
                    "payload": payload,
                    "hint": hint,
                    "ttl_days": ttl_days,
                },
            ) as r:
                r.raise_for_status()
                data = await r.json()
                return data["id"]

    async def fetch_drops(self, hint: Optional[str] = None) -> list[Drop]:
        """Fetch all unexpired drops from the bulletin board."""
        params = {}
        if hint:
            params["hint"] = hint

        async with await self._session() as s:
            async with s.get(
                f"{self.onion_url}/drops",
                params=params,
            ) as r:
                r.raise_for_status()
                raw = await r.json()
                return [Drop(**d) for d in raw]

    async def delete_drop(self, drop_id: str) -> None:
        """Delete a drop after successful pickup (clean up bulletin board)."""
        async with await self._session() as s:
            async with s.delete(f"{self.onion_url}/drop/{drop_id}") as r:
                r.raise_for_status()

4.6 Panic Mode Daemon
Purpose

The Panic Mode daemon monitors for a user-defined hotkey combination. When triggered, it immediately:

    Kills sensitive processes (messenger, Tor, onion backend)
    Requests the kernel to drop clean page caches
    Launches plausible decoy applications
    Changes the desktop wallpaper and theme

Critical design note: Panic Mode is a UI deception and rapid process termination feature. It is NOT a secure RAM wipe. Writing 3 to /proc/sys/vm/drop_caches drops clean kernel caches (pagecache, dentries, inodes) but does not zero sensitive resident memory pages. Operators must understand this. For RAM sanitization, the correct action is a hard power-off (hold power button), not a software wipe attempt.
Implementation

Python

# src/panic/daemon.py

import os
import sys
import signal
import subprocess
import logging
from pathlib import Path
from pynput import keyboard

logger = logging.getLogger("deaddrop.panic")


class PanicDaemon:
    """
    Panic Mode Daemon — listens for hotkey, triggers deception response.

    Hotkey: Ctrl+Alt+P (configurable via /etc/deaddrop/panic.conf)

    WHAT THIS DOES:
    - Kills sensitive processes (messenger, tor, onion backend)
    - Requests kernel to drop clean page caches (NOT a RAM wipe)
    - Launches decoy applications
    - Changes desktop appearance

    WHAT THIS DOES NOT DO:
    - Securely zero RAM (not possible in pure software with this timing)
    - Guarantee no forensic recovery of session data
    - Protect against hardware-level attacks
    """

    SENSITIVE_PROCESSES = [
        "deaddrop-messenger",
        "python3",          # messenger TUI
        "deaddrop-onion",   # Go onion backend
        "tor",
    ]

    DECOY_APPS = [
        ["libreoffice", "--calc", "/usr/local/share/deaddrop/decoy_budget.ods"],
        ["firefox-esr", "--new-window", "https://news.ycombinator.com"],
    ]

    DECOY_WALLPAPER = "/usr/local/share/deaddrop/decoy_wallpaper.jpg"

    def __init__(self, config: dict):
        self.config = config
        self._triggered = False
        hotkey_str = config.get("hotkey", "<ctrl>+<alt>+p")
        self._hotkey = keyboard.HotKey(
            keyboard.HotKey.parse(hotkey_str),
            self._on_trigger,
        )

    def start(self) -> None:
        logger.info("[panic] Daemon started. Hotkey: %s", self.config.get("hotkey"))
        with keyboard.Listener(
            on_press=self._for_canonical(self._hotkey.press),
            on_release=self._for_canonical(self._hotkey.release),
        ) as listener:
            listener.join()

    def _for_canonical(self, f):
        """Wrap key events to canonical form for HotKey matching."""
        listener_ref = None
        def wrapper(key):
            nonlocal listener_ref
            if listener_ref is None:
                listener_ref = keyboard.Listener.__new__(keyboard.Listener)
            canonical = listener_ref.canonical(key) if hasattr(listener_ref, 'canonical') else key
            f(canonical)
        return wrapper

    def _on_trigger(self) -> None:
        if self._triggered:
            return  # Prevent double-trigger
        self._triggered = True
        logger.warning("[panic] ⚠️  PANIC TRIGGERED")
        self._execute_panic()

    def _execute_panic(self) -> None:
        """
        Execute panic response sequence.
        Order matters: kill sensitive processes FIRST before launching decoys.
        """
        # Step 1: Kill sensitive processes
        logger.info("[panic] Step 1: Killing sensitive processes...")
        self._kill_sensitive_processes()

        # Step 2: Drop kernel page cache (NOT a RAM wipe — see module docstring)
        logger.info("[panic] Step 2: Requesting kernel cache drop...")
        self._drop_caches()

        # Step 3: Clear clipboard
        logger.info("[panic] Step 3: Clearing clipboard...")
        self._clear_clipboard()

        # Step 4: Change desktop appearance
        logger.info("[panic] Step 4: Switching to decoy desktop...")
        self._apply_decoy_theme()

        # Step 5: Launch decoy applications
        logger.info("[panic] Step 5: Launching decoy applications...")
        self._launch_decoys()

        logger.info("[panic] Panic sequence complete.")

    def _kill_sensitive_processes(self) -> None:
        for proc_name in self.SENSITIVE_PROCESSES:
            try:
                result = subprocess.run(
                    ["pkill", "-9", "-f", proc_name],
                    capture_output=True,
                    timeout=5,
                )
                if result.returncode == 0:
                    logger.info("[panic] Killed: %s", proc_name)
            except Exception as e:
                logger.warning("[panic] Could not kill %s: %s", proc_name, e)

        # Also send SIGTERM to our own process group last
        # (we survive long enough to finish the sequence)
        try:
            subprocess.run(
                ["systemctl", "stop",
                 "deaddrop-messenger.service",
                 "deaddrop-onion.service",
                 "deaddrop-tor.service"],
                timeout=10,
                capture_output=True,
            )
        except Exception as e:
            logger.warning("[panic] systemctl stop failed: %s", e)

    def _drop_caches(self) -> None:
        """
        Request kernel to drop clean page caches.

        /proc/sys/vm/drop_caches values:
          1 = pagecache
          2 = dentries + inodes
          3 = pagecache + dentries + inodes

        NOTE: This drops CLEAN caches only. Dirty pages are flushed to disk first
        via sync(). For tmpfs, "disk" means RAM, so sync() doesn't help with
        confidentiality. This call primarily removes file content from kernel
        buffer cache — it does NOT zero process heap/stack memory.

        For maximum effect, the operator should power off immediately after panic.
        """
        try:
            subprocess.run(["sync"], timeout=5)
            with open("/proc/sys/vm/drop_caches", "w") as f:
                f.write("3")
            logger.info("[panic] Kernel cache drop requested.")
        except PermissionError:
            logger.warning("[panic] drop_caches requires root. Run daemon as root or via sudo.")
        except Exception as e:
            logger.warning("[panic] Cache drop failed: %s", e)

    def _clear_clipboard(self) -> None:
        """Clear X11 clipboard contents."""
        try:
            subprocess.run(
                ["xclip", "-selection", "clipboard"],
                input=b"",
                timeout=3,
            )
            subprocess.run(
                ["xclip", "-selection", "primary"],
                input=b"",
                timeout=3,
            )
        except Exception as e:
            logger.warning("[panic] Clipboard clear failed: %s", e)

    def _apply_decoy_theme(self) -> None:
        """Change wallpaper to decoy image."""
        if Path(self.DECOY_WALLPAPER).exists():
            try:
                subprocess.Popen(
                    ["feh", "--bg-fill", self.DECOY_WALLPAPER],
                    env={**os.environ, "DISPLAY": ":0"},
                )
            except Exception as e:
                logger.warning("[panic] Wallpaper change failed: %s", e)

    def _launch_decoys(self) -> None:
        """Launch plausible decoy applications."""
        for cmd in self.DECOY_APPS:
            try:
                subprocess.Popen(
                    cmd,
                    env={**os.environ, "DISPLAY": ":0"},
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )
                logger.info("[panic] Launched decoy: %s", cmd[0])
            except Exception as e:
                logger.warning("[panic] Decoy launch failed (%s): %s", cmd[0], e)


if __name__ == "__main__":
    import json
    config_path = "/etc/deaddrop/panic.conf"
    try:
        with open(config_path) as f:
            config = json.load(f)
    except FileNotFoundError:
        config = {"hotkey": "<ctrl>+<alt>+p"}

    logging.basicConfig(level=logging.INFO)
    daemon = PanicDaemon(config)
    daemon.start()

systemd Unit: Panic Daemon

ini

# /etc/systemd/system/deaddrop-panic.service

[Unit]
Description=DeadDrop OS Panic Mode Daemon
After=deaddrop-messenger.service graphical.target
Wants=deaddrop-messenger.service

[Service]
Type=simple
User=user
Group=user
Environment=DISPLAY=:0
ExecStart=/usr/local/bin/deaddrop-panic-daemon
Restart=always
RestartSec=3

[Install]
WantedBy=graphical.target

ini

# /etc/systemd/system/deaddrop-panic-now.service
# One-shot unit triggered by the TUI's action_panic() or external scripts

[Unit]
Description=DeadDrop OS Panic — Execute Immediately
After=deaddrop-firewall.service

[Service]
Type=oneshot
User=root
ExecStart=/usr/local/lib/deaddrop/panic-now.sh

4.7 Boot & Init System
Openbox Autostart

Bash

# /etc/xdg/openbox/autostart

# Set background
feh --bg-fill /usr/local/share/deaddrop/wallpaper.jpg &

# Start panic daemon
systemctl --user start deaddrop-panic.service &

# Launch messenger TUI in xterm
xterm -title "DeadDrop Messenger" \
      -bg "#0d1117" -fg "#c9d1d9" \
      -fn "DejaVu Sans Mono:size=11" \
      -fullscreen \
      -e /usr/local/bin/deaddrop-messenger &

.xinitrc for TTY1 login

Bash

# /home/user/.xinitrc

#!/bin/bash
exec openbox-session

4.8 Build Pipeline (live-build + GitHub Actions)
GitHub Actions Workflow

YAML

# .github/workflows/build-iso.yml

name: Build DeadDrop OS ISO

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-22.04
    timeout-minutes: 90

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Install build dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y \
            live-build \
            debootstrap \
            squashfs-tools \
            xorriso \
            isolinux \
            syslinux-efi \
            grub-pc-bin \
            grub-efi-amd64-bin \
            golang-go \
            python3 \
            python3-pip \
            tinygo || true

      - name: Initialize live-build configuration
        run: |
          mkdir -p build && cd build
          bash ../config/build.sh

      - name: Copy package lists and hooks
        run: |
          cp -r config/ build/

      - name: Build Go onion backend
        run: |
          cd src/onion
          go build -ldflags="-s -w" -o ../../build/config/includes.chroot/usr/local/bin/deaddrop-onion .
          cd ../..

      - name: Install Python packages into chroot includes
        run: |
          pip3 install \
            --target build/config/includes.chroot/usr/local/lib/deaddrop-py \
            textual \
            cryptography \
            aiohttp \
            aiohttp-socks \
            pynput

      - name: Build ISO
        run: |
          cd build
          sudo lb build 2>&1 | tee ../build.log

      - name: Verify ISO produced
        run: |
          ls -lh build/*.iso || ls -lh build/*.hybrid.iso

      - name: Generate SHA-256 checksum
        run: |
          ISO=$(find build/ -name "*.iso" | head -1)
          sha256sum "$ISO" > "${ISO}.sha256"
          echo "ISO: $ISO"
          cat "${ISO}.sha256"

      - name: Sign ISO (release only)
        if: startsWith(github.ref, 'refs/tags/')
        run: |
          # Requires GPG_PRIVATE_KEY secret set in repo
          echo "${{ secrets.GPG_PRIVATE_KEY }}" | gpg --import
          ISO=$(find build/ -name "*.iso" | head -1)
          gpg --detach-sign --armor "$ISO"

      - name: Upload ISO artifact
        uses: actions/upload-artifact@v4
        with:
          name: deaddrop-os-${{ github.sha }}
          path: |
            build/*.iso
            build/*.sha256
            build/*.asc
          retention-days: 30

      - name: Upload build log
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: build-log-${{ github.sha }}
          path: build.log
          retention-days: 7

      - name: Create GitHub Release
        if: startsWith(github.ref, 'refs/tags/')
        uses: softprops/action-gh-release@v1
        with:
          files: |
            build/*.iso
            build/*.sha256
            build/*.asc
          body: |
            ## DeadDrop OS ${{ github.ref_name }}

            ### Verification
            1. Download the ISO and `.sha256` file
            2. Run: `sha256sum -c deaddrop-os.iso.sha256`
            3. Verify GPG signature: `gpg --verify deaddrop-os.iso.asc`

            ### Boot Instructions
            1. Write ISO to USB: `dd if=deaddrop-os.iso of=/dev/sdX bs=4M status=progress`
            2. Boot from USB
            3. All traffic is forced through Tor. No disk writes occur.

5. Data Models & Schemas
5.1 Onion Backend SQLite Schema

SQL

-- Server-side schema (persistent on server, NOT on client)

CREATE TABLE IF NOT EXISTS drops (
    id         TEXT PRIMARY KEY,           -- UUID v4
    payload    TEXT NOT NULL,              -- base64-encoded JSON ciphertext
    hint       TEXT NOT NULL DEFAULT '',   -- cleartext routing hint (keep vague)
    timestamp  DATETIME NOT NULL,          -- UTC ISO-8601
    expires_at DATETIME NOT NULL           -- TTL expiry; deleted by background loop
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_drops_hint    ON drops(hint);
CREATE INDEX IF NOT EXISTS idx_drops_expires ON drops(expires_at);
CREATE INDEX IF NOT EXISTS idx_drops_ts      ON drops(timestamp DESC);

5.2 Drop Payload Wire Format

JSON

{
  "version": 1,
  "enc_key": "<base64: RSA-4096-OAEP encrypted 256-bit AES key>",
  "iv":      "<base64: 12-byte AES-GCM nonce>",
  "ct":      "<base64: AES-256-GCM ciphertext of UTF-8 message>",
  "tag":     "<base64: 16-byte AES-GCM authentication tag>"
}

(The entire object is base64-encoded again and stored as payload in the database.)
5.3 Python Data Models

Python

# src/messenger/models.py

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional


@dataclass
class Drop:
    """A single encrypted drop as returned by the onion backend."""
    id: str
    payload: str           # base64-encoded encrypted JSON
    hint: str
    timestamp: str         # ISO-8601 string
    expires_at: str        # ISO-8601 string

    def timestamp_dt(self) -> datetime:
        return datetime.fromisoformat(self.timestamp)


@dataclass
class Message:
    """A decrypted message ready for display."""
    drop_id: str
    plaintext: str
    hint: str
    received_at: datetime = field(default_factory=datetime.utcnow)


@dataclass
class DropPost:
    """Payload for posting a new drop."""
    payload: str      # base64-encoded ciphertext
    hint: str = ""
    ttl_days: int = 7


@dataclass
class SystemStatus:
    """Status snapshot for the Status panel."""
    tor_connected: bool
    tor_circuit_count: int
    onion_address: str
    firewall_active: bool
    keypair_loaded: bool
    drops_fetched: int
    drops_decrypted: int

5.4 API Request/Response Types

Go

// src/onion/types.go

package main

// POST /drop
type PostDropRequest struct {
    Payload string `json:"payload"`    // base64 ciphertext (required)
    Hint    string `json:"hint"`       // cleartext hint (optional)
    TTLDays int    `json:"ttl_days"`   // 1-30 (optional, default 7)
}

type PostDropResponse struct {
    ID string `json:"id"`
}

// GET /drops → []Drop (defined in Drop struct above)

// GET /health
type HealthResponse struct {
    Status  string `json:"status"`   // "ok"
    Version string `json:"version"`
    Drops   int    `json:"drops"`    // current unexpired drop count
    Uptime  string `json:"uptime"`
}

6. API Specifications
6.1 Onion Service HTTP API

All endpoints are served at http://<onion_address>.onion/ on port 80. All communication goes through Tor SOCKS5. No TLS is needed — Tor provides transport-level encryption between client and onion service.

text

POST   /drop
       Content-Type: application/json
       Body: { "payload": "<b64>", "hint": "<str>", "ttl_days": <int> }
       Returns 201: { "id": "<uuid>" }
       Returns 413: payload too large (> 64KB)
       Returns 400: missing payload

GET    /drops
       Query: ?hint=<str>  (optional — filter by hint)
       Returns 200: [ { "id", "payload", "hint", "timestamp", "expires_at" }, ... ]
       Note: Returns max 100 most-recent unexpired drops

DELETE /drop/{id}
       Returns 204: deleted
       Returns 404: not found (or already expired)

GET    /health
       Returns 200: { "status": "ok", "version": "...", "drops": <int>, "uptime": "..." }

6.2 Rate Limiting Policy

Implemented in middleware on the Go server:
Endpoint	Limit
POST /drop	10 requests/minute per Tor exit (best effort)
GET /drops	60 requests/minute per Tor exit
DELETE /drop	30 requests/minute per Tor exit

Note: Tor exit node IP rate limiting is coarse (many clients share an exit). The primary anti-abuse control is payload size limits and TTL expiry.
6.3 Tor Control Port API (Internal)

Used by the bootstrap wait script and status display:

text

AUTHENTICATE ""
GETINFO status/bootstrap-phase
→ 250+status/bootstrap-phase=NOTICE BOOTSTRAP PROGRESS=100 TAG=done ...

GETINFO circuit-status
→ Returns active circuits for display in Status panel

7. Directory Structure

text

deaddrop-os/
│
├── .github/
│   └── workflows/
│       └── build-iso.yml              # CI/CD: build ISO artifact
│
├── config/                            # live-build configuration
│   ├── build.sh                       # lb config invocation script
│   ├── package-lists/
│   │   └── deaddrop.list.chroot       # Packages to install into squashfs
│   ├── hooks/
│   │   ├── live/
│   │   │   ├── 0010-harden.hook.chroot    # OS hardening (sysctl, masking, etc.)
│   │   │   ├── 0020-install-go-bin.hook.chroot # Copy pre-built Go binary
│   │   │   └── 0030-python-deps.hook.chroot    # Install Python packages
│   │   └── normal/                    # Hooks for standard (non-live) builds
│   └── includes.chroot/               # Files overlaid into chroot root
│       ├── etc/
│       │   ├── tor/
│       │   │   └── torrc              # Tor transparent proxy config
│       │   ├── systemd/
│       │   │   ├── system/
│       │   │   │   ├── deaddrop-firewall.service
│       │   │   │   ├── deaddrop-tor.service
│       │   │   │   ├── deaddrop-onion.service
│       │   │   │   ├── deaddrop-messenger.service
│       │   │   │   ├── deaddrop-panic.service
│       │   │   │   └── deaddrop-panic-now.service
│       │   │   └── journald.conf.d/
│       │   │       └── volatile.conf  # volatile storage for journal
│       │   ├── sysctl.d/
│       │   │   └── 99-deaddrop.conf   # IPv6 disable, core dump disable
│       │   ├── xdg/
│       │   │   └── openbox/
│       │   │       └── autostart      # Openbox session startup
│       │   └── deaddrop/
│       │       ├── config.conf        # Main system config
│       │       └── panic.conf         # Panic mode config
│       ├── usr/
│       │   ├── local/
│       │   │   ├── bin/
│       │   │   │   ├── deaddrop-messenger       # Python TUI entry point
│       │   │   │   ├── deaddrop-panic-daemon    # Panic daemon entry point
│       │   │   │   └── deaddrop-onion           # Go binary (built by CI)
│       │   │   ├── lib/
│       │   │   │   └── deaddrop/
│       │   │   │       ├── firewall-init.sh     # iptables enforcement script
│       │   │   │       ├── firewall-flush.sh    # iptables flush (stop)
│       │   │   │       ├── wait-for-tor.sh      # Tor bootstrap wait
│       │   │   │       └── panic-now.sh         # One-shot panic script
│       │   │   └── share/
│       │   │       └── deaddrop/
│       │   │           ├── wallpaper.jpg         # Normal wallpaper
│       │   │           ├── decoy_wallpaper.jpg   # Panic mode wallpaper
│       │   │           └── decoy_budget.ods      # LibreOffice decoy file
│       └── home/
│           └── user/
│               ├── .xinitrc                      # startx → openbox
│               └── .bashrc
│
├── src/
│   ├── messenger/                     # Python Dead Drop Messenger TUI
│   │   ├── __init__.py
│   │   ├── app.py                     # Textual App, all UI panels
│   │   ├── crypto.py                  # Hybrid encryption engine
│   │   ├── transport.py               # Tor SOCKS5 HTTP transport
│   │   ├── models.py                  # Data models (Drop, Message, etc.)
│   │   └── __main__.py                # Entry: python3 -m messenger
│   │
│   ├── onion/                         # Go onion service backend
│   │   ├── main.go                    # Tor init, onion listen, HTTP serve
│   │   ├── handlers.go                # HTTP handler functions
│   │   ├── db.go                      # SQLite init, migrations
│   │   ├── keys.go                    # Ed25519 key load/generate
│   │   ├── middleware.go              # Rate limiting, logging, size limits
│   │   ├── types.go                   # Request/response types
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── panic/                         # Python Panic Mode daemon
│       ├── __init__.py
│       ├── daemon.py                  # PanicDaemon class
│       └── __main__.py                # Entry: python3 -m panic
│
├── tests/
│   ├── unit/
│   │   ├── test_crypto.py             # CryptoEngine unit tests
│   │   ├── test_transport.py          # Transport layer tests (mocked SOCKS)
│   │   └── test_panic.py              # Panic daemon unit tests
│   ├── integration/
│   │   ├── test_drop_roundtrip.py     # Full encrypt → post → fetch → decrypt
│   │   └── test_firewall_rules.sh     # iptables rule verification
│   └── conftest.py
│
├── scripts/
│   ├── write-usb.sh                   # dd ISO to USB with verification
│   ├── verify-iso.sh                  # sha256 + GPG verify
│   └── test-vm.sh                     # Boot ISO in QEMU for testing
│
├── Makefile
├── pyproject.toml                     # Python deps (messenger + panic)
├── README.md
└── THREAT-MODEL.md                    # Explicit threat model document

8. Configuration System
8.1 System Config (/etc/deaddrop/config.conf)

ini

# /etc/deaddrop/config.conf
# DeadDrop OS System Configuration
# All paths are on tmpfs unless marked [SERVER]

[tor]
trans_port = 9040
dns_port = 5353
socks_port = 9050
control_port = 9051
data_dir = /tmp/tor-data
log_file = /tmp/tor.log
bootstrap_timeout_sec = 120

[network]
virtual_addr_range = 10.192.0.0/10
disable_ipv6 = true
disable_swap = true

[onion_backend]
# Address of the shared onion service (operator pre-configures this)
# Must be set before building ISO or overridden at first boot
onion_url = http://REPLACE_WITH_YOUR_ONION_ADDRESS.onion
local_port = 8080
# [SERVER] key file path — only relevant when running the backend server
key_file = /var/lib/deaddrop/onion_key.pem
# [SERVER] database path — use :memory: for ephemeral, or path for persistent
db_file = /var/lib/deaddrop/drops.db

[messenger]
keystore_path = /tmp/deaddrop-keys
default_ttl_days = 7
max_fetch_count = 100
socks_host = 127.0.0.1
socks_port = 9050

[panic]
hotkey = <ctrl>+<alt>+p
decoy_wallpaper = /usr/local/share/deaddrop/decoy_wallpaper.jpg
decoy_budget_file = /usr/local/share/deaddrop/decoy_budget.ods
decoy_url = https://news.ycombinator.com

8.2 Panic Config (/etc/deaddrop/panic.conf)

JSON

{
  "hotkey": "<ctrl>+<alt>+p",
  "sensitive_processes": [
    "deaddrop-messenger",
    "deaddrop-onion",
    "python3",
    "tor"
  ],
  "decoy_apps": [
    ["libreoffice", "--calc", "/usr/local/share/deaddrop/decoy_budget.ods"],
    ["firefox-esr", "--new-window", "https://news.ycombinator.com"]
  ],
  "decoy_wallpaper": "/usr/local/share/deaddrop/decoy_wallpaper.jpg",
  "drop_caches": true,
  "clear_clipboard": true
}

8.3 Config Loader (Python)

Python

# src/messenger/config.py

import configparser
import os
from pathlib import Path
from dataclasses import dataclass


@dataclass
class MessengerConfig:
    onion_url: str
    keystore_path: str
    socks_host: str
    socks_port: int
    default_ttl_days: int
    max_fetch_count: int


def load_config(path: str = "/etc/deaddrop/config.conf") -> MessengerConfig:
    cfg = configparser.ConfigParser()
    cfg.read(path)

    return MessengerConfig(
        onion_url=cfg.get("onion_backend", "onion_url",
                          fallback=os.environ.get("DEADDROP_ONION_URL", "")),
        keystore_path=cfg.get("messenger", "keystore_path",
                              fallback="/tmp/deaddrop-keys"),
        socks_host=cfg.get("messenger", "socks_host", fallback="127.0.0.1"),
        socks_port=cfg.getint("messenger", "socks_port", fallback=9050),
        default_ttl_days=cfg.getint("messenger", "default_ttl_days", fallback=7),
        max_fetch_count=cfg.getint("messenger", "max_fetch_count", fallback=100),
    )

9. Cryptography & Key Management
9.1 Keypair Lifecycle (Client)

text

Boot
  │
  ▼
Check /tmp/deaddrop-keys/private.pem
  │
  ├─── Exists? ───► Load keypair into memory
  │
  └─── Missing? ──► User generates via Keys tab in TUI
                         │
                         ▼
                    RSA-4096 generated
                    Saved to /tmp/deaddrop-keys/ (tmpfs)
                         │
                         ▼
                    User exports public key (copy to clipboard)
                    → Shares out-of-band with correspondents
                         │
                    (Session ends)
                         │
                         ▼
                    Shutdown → tmpfs wiped → key gone

9.2 Key Provisioning Options
Option	How	Persistence	Use Case
Generate per session	Via TUI Keys tab	No — lost on shutdown	Maximum anonymity
Pre-baked in ISO	Built into squashfs	No — read-only, same for all	Shared team key
Loaded from USB at boot	Hook script reads from second USB	Session-only	Operator-specific key
Loaded from encrypted container	Manual operator step	Session-only	High-assurance ops
9.3 Server Key Management

The onion service backend runs on a persistent server. Its Ed25519 private key:

text

/var/lib/deaddrop/onion_key.pem
  permissions: 0600
  owner: deaddrop-server (dedicated service user)
  group: deaddrop-server

This file must be:

    Backed up securely (losing it means losing the .onion address)
    Protected by the server's full-disk encryption
    Never copied to client machines or included in the ISO

9.4 Cryptographic Primitives Reference
Operation	Algorithm	Parameters	Library
Key generation	RSA	4096-bit, e=65537	Python cryptography
Key encapsulation	RSA-OAEP	SHA-256 hash, SHA-256 MGF1	Python cryptography
Message encryption	AES-GCM	256-bit key, 96-bit nonce	Python cryptography
Authentication	GCM tag	128-bit (16 bytes)	Built into AES-GCM
Onion service key	Ed25519	Standard	bine / Go crypto
Onion address	SHA-3 (Tor v3)	Derived from Ed25519 public key	Tor protocol
10. Network Enforcement Deep Dive
10.1 IPv6 Handling

IPv6 is disabled at three independent layers. Any one of these alone is sufficient; all three together provide defense in depth:
Layer	Mechanism
Kernel boot parameter	ipv6.disable=1 in GRUB bootappend
sysctl	net.ipv6.conf.all.disable_ipv6 = 1 etc.
ip6tables	Default DROP policy on all chains, all rules flushed

The firewall script explicitly applies ip6tables DROP-all even though IPv6 should be disabled, because:

    A kernel module could be loaded post-boot that re-enables IPv6
    Certain hardware might ignore the boot parameter
    Defense in depth costs nothing and prevents surprises

10.2 DNS Leak Prevention

DNS is the most common source of leaks in "Tor-transparent" setups. DeadDrop OS prevents this via:

    systemd-resolved is disabled — its stub listener on 127.0.0.53 is masked
    /etc/resolv.conf points to Tor's DNSPort: nameserver 127.0.0.1 with options port:5353
    iptables redirects all UDP/53 to Tor's DNS port — even if a process ignores /etc/resolv.conf
    AutomapHostsOnResolve 1 in torrc ensures .onion hostnames resolve to virtual IPs without leaking

Bash

# config/includes.chroot/etc/resolv.conf
# Static — systemd-resolved is masked

nameserver 127.0.0.1
options port:5353 edns0 trust-ad

Bash

# Mask systemd-resolved in harden hook
systemctl mask systemd-resolved.service
systemctl mask systemd-resolved.socket

10.3 Tor Bootstrap Failure Handling

If Tor fails to bootstrap within the timeout:

ini

# deaddrop-tor.service — OnFailure handler

[Unit]
OnFailure=deaddrop-tor-failure.service

# deaddrop-tor-failure.service — notifies user, firewall stays active

[Service]
Type=oneshot
ExecStart=/usr/local/lib/deaddrop/tor-failure-notify.sh

Bash

# /usr/local/lib/deaddrop/tor-failure-notify.sh

#!/bin/bash
# Show error on TTY1 and keep firewall ACTIVE (no fallback to clearnet)
DISPLAY=:0 xmessage -center \
  "❌ Tor failed to bootstrap within the timeout.\n\n\
All network access remains BLOCKED.\n\n\
Possible causes:\n\
  • No internet connection on host network\n\
  • Tor is censored on this network (try Tor bridges)\n\
  • Hardware issue\n\n\
The firewall is still active. No clearnet traffic is possible." \
  -buttons "Shut Down:1,Retry:2" \
  -default "Shut Down"

exit_code=$?
if [ "$exit_code" = "1" ]; then
  systemctl poweroff
elif [ "$exit_code" = "2" ]; then
  systemctl restart deaddrop-tor.service
fi

10.4 Network Topology Diagram

text

[DeadDrop OS Client]
        │
        │ All TCP → iptables REDIRECT → Tor TransPort 9040
        │ All UDP/53 → iptables REDIRECT → Tor DNSPort 5353
        │ All UDP (non-53) → DROP
        │ All ICMP → DROP
        │ IPv6 → DROP (kernel + ip6tables)
        ▼
[Tor Daemon (127.0.0.1:9040)]
        │
        │ 3-hop Tor circuit (encrypted)
        ▼
[Tor Guard] → [Tor Middle] → [Tor Exit/Rendezvous]
        │
        │ Onion service rendezvous (no exit node for .onion)
        ▼
[Onion Service Backend]
[.onion address:80]
[Go HTTP server]
[SQLite drops table]

11. Dead Drop Messenger Deep Dive
11.1 Message Composition Flow

text

User opens Compose tab
        │
        ▼
User pastes recipient's RSA-4096 public key (PEM format)
        │
        ▼
User types message body (any length)
        │
        ▼
User optionally adds hint (e.g. "batch-7")
        │
        ▼
User clicks "Encrypt & Send"
        │
        ▼
CryptoEngine.encrypt_message(pubkey_pem, body):
  1. Parse recipient RSA public key
  2. Generate random 256-bit AES key (DEK)
  3. Generate random 12-byte nonce
  4. AES-256-GCM encrypt body → (ciphertext, tag)
  5. RSA-OAEP encrypt DEK → enc_key
  6. Assemble JSON payload
  7. Base64-encode entire payload
        │
        ▼
OnionTransport.post_drop(payload, hint, ttl):
  HTTP POST → SOCKS5(127.0.0.1:9050) → .onion/drop
        │
        ▼
Server stores drop in SQLite
        │
        ▼
TUI shows success: "✅ Sent! Drop ID: abc123..."

11.2 Message Fetch & Decrypt Flow

text

User opens Inbox tab
        │
        ▼
User clicks "Fetch Drops"
        │
        ▼
OnionTransport.fetch_drops(hint=None):
  HTTP GET → SOCKS5 → .onion/drops
  Returns: [ {id, payload, hint, timestamp, expires_at}, ... ]
        │
        ▼
For each drop in results:
        │
        ├── CryptoEngine.decrypt_message(drop.payload):
        │     1. Base64-decode outer payload
        │     2. JSON parse → {version, enc_key, iv, ct, tag}
        │     3. RSA-OAEP decrypt enc_key → DEK
        │        ├── Success → proceed to AES
        │        └── ValueError → "Not for us" — skip
        │     4. AES-256-GCM decrypt (ct + tag) with DEK + iv
        │        ├── Success → plaintext string
        │        └── InvalidTag → message tampered — mark error
        │
        ├── ✅ Decrypted: add to table as "✅ Decrypted", show preview
        └── 🔒 Failed: add to table as "🔒 Not for us"
        │
        ▼
TUI shows: "3 messages decrypted of 12 total drops"

11.3 Hint Field Security Considerations

The hint field is stored and transmitted in cleartext on the server. This is a deliberate design choice (it allows filtering without server-side decryption), but operators must understand its implications:
Risk	Mitigation
Server operator sees hints	Use opaque hints (e.g. "msg-17", not "for-alice")
Traffic analysis of hint patterns	Rotate hint schemes per operation
Hint enumeration by attacker	Server does not expose hint list; only exact-match filter

Recommendation: Use short, random-looking identifiers as hints. Never use personal names or operational context.
12. Onion Service Backend Deep Dive
12.1 Server vs. Client Distinction

This is the most important architectural decision to understand:

text

┌─────────────────────────────────────────────────────────────────┐
│  CLIENT (DeadDrop OS — booted from USB)                         │
│  - Amnesic: no disk writes                                      │
│  - Tor transparent proxy running                                │
│  - Messenger TUI: composes + fetches drops                      │
│  - Does NOT run the onion backend                               │
│  - Knows only the .onion address (baked into ISO at build time)  │
└──────────────────────────────────┬──────────────────────────────┘
                                   │ HTTP POST/GET over Tor
                                   │ (SOCKS5 → .onion)
┌──────────────────────────────────▼──────────────────────────────┐
│  SERVER (Operator's persistent server — e.g. VPS)               │
│  - Running 24/7 (drops must be pickable-up asynchronously)      │
│  - Ed25519 key stored persistently (stable .onion address)      │
│  - SQLite database (persistent, backed up)                      │
│  - Go onion backend process                                     │
│  - Full-disk encryption on server is strongly recommended       │
└─────────────────────────────────────────────────────────────────┘

12.2 Middleware (Rate Limiting + Logging)

Go

// src/onion/middleware.go

package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// ── LOGGING MIDDLEWARE ────────────────────────────────────────────────────

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		// Log without logging actual payloads or client IP
		// (client is a Tor circuit, IP is meaningless)
		log.Printf("[onion] %s %s → %d (%dms)",
			r.Method, r.URL.Path, wrapped.status,
			time.Since(start).Milliseconds(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// ── RATE LIMIT MIDDLEWARE ────────────────────────────────────────────────
// Note: Tor clients share exit nodes; IP-based rate limiting is approximate.
// This is a basic anti-flood measure, not a robust anti-abuse system.

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientState
}

type clientState struct {
	count    int
	resetAt  time.Time
}

var globalRateLimiter = &rateLimiter{
	clients: make(map[string]*clientState),
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		globalRateLimiter.mu.Lock()
		state, ok := globalRateLimiter.clients[ip]
		if !ok || time.Now().After(state.resetAt) {
			state = &clientState{count: 0, resetAt: time.Now().Add(time.Minute)}
			globalRateLimiter.clients[ip] = state
		}
		state.count++
		count := state.count
		globalRateLimiter.mu.Unlock()

		limit := rateLimitFor(r.Method, r.URL.Path)
		if count > limit {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitFor(method, path string) int {
	switch {
	case method == "POST" && path == "/drop":
		return 10
	case method == "GET" && path == "/drops":
		return 60
	case method == "DELETE":
		return 30
	default:
		return 120
	}
}

12.3 Anti-Abuse Controls
Control	Implementation
Payload size limit	64KB hard limit in handler
TTL enforcement	Background loop expires old drops hourly
Max drops returned	100 per GET /drops request
Rate limiting	Per-circuit (best effort) per minute
No unauthenticated admin	DELETE requires drop ID — no list-then-delete amplification

Future consideration (§25): Token-based authentication for POST (operators share a bearer token) to prevent anonymous public spam. Currently out of scope.
13. Panic Mode Deep Dive
13.1 What Panic Mode Realistically Achieves
Action	What it does	What it does NOT do
pkill -9 sensitive processes	Terminates processes; their open file handles are closed	Does not zero process heap/stack memory before free
systemctl stop services	Clean service shutdown + marks as stopped	Does not prevent memory forensics of freed pages
sync + drop_caches = 3	Flushes dirty pages; drops clean kernel buffer cache	Does not zero RAM contents
Clipboard clear	Removes clipboard contents	Does not prevent X11 clipboard history recovery
Decoy app launch	Shows plausible alternative activity	Does not change hardware-level evidence
Wallpaper change	Visual deception for casual observer	Does not fool digital forensics

The correct follow-up to Panic Mode is immediate power-off. Panic Mode buys time and creates visual confusion. It is not a substitute for physical security (removing the USB, powering off, leaving the area).
13.2 Threat Model for Panic Mode
Threat	Panic Mode effective?
Casual observer glances at screen	✅ Yes — decoy apps are visible
Colleague/family member checks device	✅ Yes — messenger not visible
Physical checkpoint with brief device inspection	⚠️ Partial — depends on inspection depth
Digital forensics lab with RAM imaging	❌ No — RAM still contains session data
Hardware keylogger / DMA attack	❌ No — hardware-level, unaffected
Network-level traffic correlation	❌ No — traffic already happened
13.3 Improved Panic Sequence (Production Recommendation)

For higher-assurance deployments, the panic sequence should also trigger:

Bash

# /usr/local/lib/deaddrop/panic-now.sh
# One-shot panic script (run as root for drop_caches)

#!/bin/bash

# 1. Kill sensitive processes
pkill -9 -f deaddrop-messenger
pkill -9 -f deaddrop-onion
systemctl stop deaddrop-messenger deaddrop-onion deaddrop-tor

# 2. Clear tmpfs keystore (overwrite then remove)
if [ -d /tmp/deaddrop-keys ]; then
    find /tmp/deaddrop-keys -type f -exec shred -vuz {} \;
    rm -rf /tmp/deaddrop-keys
fi

# 3. Overwrite Tor data directory
if [ -d /tmp/tor-data ]; then
    find /tmp/tor-data -type f -exec shred -uz {} \;
    rm -rf /tmp/tor-data
fi

# 4. Drop caches
sync
echo 3 > /proc/sys/vm/drop_caches

# 5. Clear clipboard
DISPLAY=:0 xclip -selection clipboard /dev/null 2>/dev/null || true
DISPLAY=:0 xclip -selection primary   /dev/null 2>/dev/null || true

# 6. Apply decoy desktop
DISPLAY=:0 feh --bg-fill /usr/local/share/deaddrop/decoy_wallpaper.jpg &

# 7. Launch decoys
DISPLAY=:0 libreoffice --calc /usr/local/share/deaddrop/decoy_budget.ods &
DISPLAY=:0 firefox-esr --new-window https://news.ycombinator.com &

echo "[panic] Sequence complete."

14. Live OS & Amnesia System Deep Dive
14.1 What "Amnesic" Means Precisely
Claim	True?	Detail
No writes to host disk in default config	✅ Yes	overlayfs upper = tmpfs; no disk mount
Shutdown destroys the session	✅ Yes	tmpfs unmounted/cleared by kernel
RAM contents gone after power cut	⚠️ Mostly	Cold-boot attacks can recover RAM for seconds–minutes after power-off on some hardware
RAM cleared on normal shutdown	⚠️ Partial	Kernel does not zero all pages; some data remains briefly
Process memory is unrecoverable after pkill	❌ No	Freed pages may sit in RAM until overwritten
Swap never used	✅ Yes (by design)	swap disabled; no swap partition exists
Logs are not written to disk	✅ Yes	journald volatile mode; all logs in tmpfs
14.2 Preventing Accidental Disk Writes

Bash

# config/hooks/live/0010-harden.hook.chroot (continued)

# Remove udisks2 entirely (prevents auto-mount GUI)
apt-get remove -y --purge udisks2 gvfs || true

# Mask automount
systemctl mask proc-sys-fs-binfmt_misc.automount || true

# udev rule: deny all non-USB-boot disk write access
cat > /etc/udev/rules.d/99-deaddrop-nodisk.rules <<EOF
# Block all internal disk writes
# USB boot device is identified by kernel; internal disks are locked
KERNEL=="sd[a-z]", ATTR{removable}=="0", RUN+="/bin/sh -c 'blockdev --setro /dev/%k'"
KERNEL=="nvme*", RUN+="/bin/sh -c 'blockdev --setro /dev/%k'"
EOF

14.3 Memory Considerations at Shutdown

For maximum assurance, operators can configure a shutdown-time memory overwrite:

Bash

# /etc/systemd/system/deaddrop-memwipe.service
# Runs late in shutdown sequence — attempts to fill free RAM

[Unit]
Description=DeadDrop OS Shutdown Memory Fill
DefaultDependencies=no
Before=shutdown.target reboot.target
After=deaddrop-messenger.service

[Service]
Type=oneshot
RemainAfterExit=no
# Fill free RAM with zeros to overwrite freed pages (best effort, not guaranteed)
ExecStart=/bin/sh -c "dd if=/dev/zero of=/tmp/memfill bs=4M 2>/dev/null; rm -f /tmp/memfill"
TimeoutStartSec=10

[Install]
WantedBy=shutdown.target reboot.target

⚠️ Important caveat: This is a best-effort measure. The kernel manages memory allocation and the scheduler may not overwrite all sensitive pages within the timeout window. Physical memory remanence (cold-boot attacks) cannot be fully defeated in software.
15. Security Model & Threat Boundaries
15.1 Threat Model (Explicit)

DeadDrop OS is designed for the following adversary profile:
Adversary Capability	In Scope?	Notes
Network traffic monitoring (full ISP/nation-state level)	✅ Yes	Tor provides anonymity
Physical device confiscation after shutdown	✅ Yes	Amnesia + no disk writes
Casual physical inspection during active session	✅ Partial	Panic Mode provides deception
Active server compromise (onion backend)	⚠️ Partial	Server stores only ciphertext
RAM imaging during/after session (cold-boot)	❌ Out of scope	Hardware-level attack
Compromised ISO (supply chain)	❌ Out of scope	Operator must verify ISO signature
Keylogger on host hardware	❌ Out of scope	Hardware-level attack
Compromised recipient (key theft)	❌ Out of scope	Key management is operator responsibility
Tor deanonymization (global traffic correlation)	❌ Out of scope	Nation-state level; Tor's known limitation
15.2 Trust Zones

text

┌────────────────────────────────────────────┐
│  TRUSTED (RAM, in-session)                 │
│  - tmpfs filesystem                        │
│  - Running processes                       │
│  - Keys in /tmp/deaddrop-keys              │
│  - Tor daemon                              │
└─────────────────────┬──────────────────────┘
                      │
┌─────────────────────▼──────────────────────┐
│  SEMI-TRUSTED (Onion service server)       │
│  - Stores only ciphertext — cannot decrypt │
│  - Knows: hints, drop timestamps, sizes    │
│  - Operator-controlled, off the client     │
└─────────────────────┬──────────────────────┘
                      │
┌─────────────────────▼──────────────────────┐
│  UNTRUSTED (Everything else)               │
│  - Host hardware (potential keyloggers)    │
│  - Network (assumed fully monitored)       │
│  - Tor relays (assume malicious subset)    │
│  - Host disk (never touched)               │
└────────────────────────────────────────────┘

15.3 What the Server Learns

Even with a fully honest and uncompromised server, the onion backend learns:
Data Point	Visible to Server
Message content	❌ No — encrypted ciphertext only
Sender's Tor circuit	❌ No — onion service, no exit node
Recipient identity	❌ No — recipient fetches anonymously
Number of drops posted	✅ Yes
Timing of post/fetch	✅ Yes
Hint value	✅ Yes (cleartext)
Drop size (approximate)	✅ Yes

Operators should design their hint scheme and operational tempo to minimize metadata inference from timing and size.
15.4 CA Key / Key Material Storage
Material	Location	Protection
Client RSA private key	/tmp/deaddrop-keys/private.pem (tmpfs)	OS-level isolation; gone on shutdown
Client RSA public key	/tmp/deaddrop-keys/public.pem (tmpfs)	Non-sensitive
Server Ed25519 private key	/var/lib/deaddrop/onion_key.pem	0600 permissions; server FDE
Drop ciphertext	SQLite on server	Encrypted; server cannot decrypt
16. Storage & Persistence
16.1 Client Storage Policy
Path	Backend	Persists after shutdown?	Contains
/tmp/*	tmpfs	❌ No	Keys, Tor data, logs, all runtime data
/run/*	tmpfs	❌ No	PID files, sockets
/var/log/*	tmpfs	❌ No	Journald volatile logs
/home/user/*	overlayfs/tmpfs	❌ No	User config, shell history
USB drive	squashfs (read-only)	✅ Yes (read-only)	OS filesystem
16.2 Server Storage Policy
Path	Backend	Notes
/var/lib/deaddrop/drops.db	SQLite (WAL)	Persistent; contains ciphertext blobs
/var/lib/deaddrop/onion_key.pem	Filesystem	Persistent; protected by FDE
/var/lib/deaddrop/tor-data/	Filesystem	Tor's data directory; persistent
/var/log/deaddrop/	Filesystem	Server logs (optional; rotate frequently)
16.3 Server Database Maintenance

Bash

# /etc/cron.daily/deaddrop-db-maintenance

#!/bin/bash
# Run daily: expire old drops, vacuum database

sqlite3 /var/lib/deaddrop/drops.db <<'EOF'
DELETE FROM drops WHERE expires_at < datetime('now');
VACUUM;
PRAGMA wal_checkpoint(TRUNCATE);
EOF

echo "[maintenance] Drop cleanup complete: $(date)"

17. Logging & Observability
17.1 Client Logging Policy

All logs on the client are ephemeral (tmpfs only):
Component	Log destination	Level
Tor daemon	/tmp/tor.log	notice
Firewall script	journald (volatile)	info
Onion backend	stdout → journald (volatile)	info
Messenger TUI	/tmp/deaddrop-messenger.log	warning
Panic daemon	/tmp/deaddrop-panic.log	info
systemd journal	RAM only (Storage=volatile)	info

No log file is ever written to the host disk. All logs are lost on shutdown.
17.2 What Is Never Logged

The following are explicitly never written to any log:

    Message plaintext
    RSA private key material
    AES session keys
    Decrypted payload contents
    User-typed message body

17.3 Server Logging Policy

text

/var/log/deaddrop/server.log
  - HTTP access log: method, path, status, duration (no body, no payload)
  - NO client IPs logged (meaningless for Tor; would only log exit nodes)
  - Rotation: daily, retain 7 days
  - Level: info

17.4 Status Display in TUI

The Status tab in the Textual TUI shows a live readout of:

text

⚙️ SYSTEM STATUS
─────────────────────────────────────────
Tor:      ✅ Connected — Circuit established
          Bootstrap: 100%   Circuits: 3 active
Onion:    http://abc123...xyz.onion
          Backend: ✅ Reachable
Firewall: ✅ Active (Tor-only enforcement)
IPv6:     ✅ Disabled

Keys:     ✅ RSA-4096 keypair loaded
          Fingerprint: SHA256:abc123...

Session:  Drops fetched: 12  Decrypted: 3
          Posted: 2  Failed: 0

18. Testing Strategy
18.1 Test Pyramid

text

                    ┌─────────────────────┐
                    │  ISO Boot Tests (5%) │
                    │  QEMU VM, smoke test │
                    └──────────┬──────────┘
               ┌───────────────┴────────────────┐
               │   Integration Tests (25%)       │
               │   Drop roundtrip, Tor SOCKS,    │
               │   Firewall rule verification    │
               └───────────────┬────────────────┘
          ┌────────────────────┴─────────────────────┐
          │           Unit Tests (70%)                │
          │   CryptoEngine, models, config, panic     │
          └───────────────────────────────────────────┘

18.2 Crypto Unit Tests

Python

# tests/unit/test_crypto.py

import pytest
import tempfile
import os
from src.messenger.crypto import CryptoEngine, DecryptionError
from cryptography.hazmat.primitives.ciphers.aead import AESGCM


@pytest.fixture
def tmp_keystore(tmp_path):
    return str(tmp_path / "keys")


@pytest.fixture
def engine(tmp_keystore):
    e = CryptoEngine(tmp_keystore)
    e.generate_keypair()
    return e


@pytest.fixture
def alice(tmp_path):
    e = CryptoEngine(str(tmp_path / "alice-keys"))
    e.generate_keypair()
    return e


@pytest.fixture
def bob(tmp_path):
    e = CryptoEngine(str(tmp_path / "bob-keys"))
    e.generate_keypair()
    return e


class TestKeyGeneration:
    def test_generates_rsa_4096_keypair(self, engine, tmp_keystore):
        priv_path = os.path.join(tmp_keystore, "private.pem")
        pub_path = os.path.join(tmp_keystore, "public.pem")
        assert os.path.exists(priv_path)
        assert os.path.exists(pub_path)

    def test_private_key_permissions(self, engine, tmp_keystore):
        priv_path = os.path.join(tmp_keystore, "private.pem")
        mode = oct(os.stat(priv_path).st_mode)[-3:]
        assert mode == "600"

    def test_public_key_pem_format(self, engine):
        pem = engine.get_public_key_pem()
        assert pem.startswith("-----BEGIN PUBLIC KEY-----")
        assert pem.strip().endswith("-----END PUBLIC KEY-----")

    def test_no_keypair_raises_on_pubkey_get(self, tmp_keystore):
        fresh = CryptoEngine(tmp_keystore + "-empty")
        with pytest.raises(RuntimeError, match="No keypair loaded"):
            fresh.get_public_key_pem()


class TestHybridEncryption:
    def test_encrypt_decrypt_roundtrip(self, alice, bob):
        """Alice encrypts to Bob; Bob decrypts successfully."""
        bob_pubkey = bob.get_public_key_pem()
        plaintext = "Hello, Bob. This is a secret message."
        payload = alice.encrypt_message(bob_pubkey, plaintext)
        recovered = bob.decrypt_message(payload)
        assert recovered == plaintext

    def test_cannot_decrypt_with_wrong_key(self, alice, bob):
        """Alice encrypts to Bob; Alice cannot decrypt (different key)."""
        bob_pubkey = bob.get_public_key_pem()
        payload = alice.encrypt_message(bob_pubkey, "secret")
        with pytest.raises(DecryptionError):
            alice.decrypt_message(payload)

    def test_long_message_roundtrip(self, alice, bob):
        """Hybrid encryption handles arbitrarily long messages."""
        bob_pubkey = bob.get_public_key_pem()
        long_message = "A" * 100_000  # 100KB — far beyond RSA-OAEP limit
        payload = alice.encrypt_message(bob_pubkey, long_message)
        recovered = bob.decrypt_message(payload)
        assert recovered == long_message

    def test_payload_is_valid_base64(self, alice, bob):
        import base64, json
        bob_pubkey = bob.get_public_key_pem()
        payload = alice.encrypt_message(bob_pubkey, "test")
        decoded = base64.b64decode(payload)
        obj = json.loads(decoded)
        assert "version" in obj
        assert "enc_key" in obj
        assert "iv" in obj
        assert "ct" in obj
        assert "tag" in obj

    def test_tampered_ciphertext_fails_authentication(self, alice, bob):
        import base64, json
        from cryptography.exceptions import InvalidTag
        bob_pubkey = bob.get_public_key_pem()
        payload = alice.encrypt_message(bob_pubkey, "secret")
        # Tamper with ciphertext
        decoded = json.loads(base64.b64decode(payload))
        ct_bytes = bytearray(base64.b64decode(decoded["ct"]))
        ct_bytes[0] ^= 0xFF  # flip first byte
        decoded["ct"] = base64.b64encode(bytes(ct_bytes)).decode()
        tampered = base64.b64encode(json.dumps(decoded).encode()).decode()
        with pytest.raises(InvalidTag):
            bob.decrypt_message(tampered)

    def test_encrypt_to_invalid_key_raises(self, alice):
        with pytest.raises(Exception):
            alice.encrypt_message("not-a-pem-key", "message")

    def test_decrypt_malformed_payload_raises(self, alice):
        with pytest.raises(ValueError, match="Malformed payload"):
            alice.decrypt_message("this is not base64 json")

    def test_unicode_message_roundtrip(self, alice, bob):
        bob_pubkey = bob.get_public_key_pem()
        message = "Привет мир 🌍 مرحبا بالعالم"
        payload = alice.encrypt_message(bob_pubkey, message)
        recovered = bob.decrypt_message(payload)
        assert recovered == message

18.3 Integration Tests

Python

# tests/integration/test_drop_roundtrip.py
# Requires: running onion backend (can be on localhost for CI), Tor SOCKS5

import pytest
import asyncio
from src.messenger.crypto import CryptoEngine
from src.messenger.transport import OnionTransport


@pytest.mark.integration
@pytest.mark.asyncio
async def test_full_drop_roundtrip(alice_engine, bob_engine, transport):
    """
    Full end-to-end: Alice posts an encrypted drop, Bob fetches and decrypts it.
    """
    bob_pubkey = bob_engine.get_public_key_pem()
    original_message = "Integration test message — confidential."

    # Alice encrypts and posts
    payload = alice_engine.encrypt_message(bob_pubkey, original_message)
    drop_id = await transport.post_drop(payload=payload, hint="test-integration")
    assert len(drop_id) == 36  # UUID format

    # Bob fetches and decrypts
    drops = await transport.fetch_drops(hint="test-integration")
    assert any(d.id == drop_id for d in drops)

    our_drop = next(d for d in drops if d.id == drop_id)
    recovered = bob_engine.decrypt_message(our_drop.payload)
    assert recovered == original_message

    # Cleanup
    await transport.delete_drop(drop_id)

18.4 Firewall Rule Tests

Bash

# tests/integration/test_firewall_rules.sh
# Run inside a test VM with DeadDrop OS firewall active

#!/bin/bash
set -e

PASS=0
FAIL=0

check() {
    local desc="$1"
    local cmd="$2"
    local expect_exit="$3"

    eval "$cmd" > /dev/null 2>&1
    actual=$?
    if [ "$actual" = "$expect_exit" ]; then
        echo "✅ PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "❌ FAIL: $desc (expected exit $expect_exit, got $actual)"
        FAIL=$((FAIL + 1))
    fi
}

# Clearnet TCP should be blocked
check "Clearnet TCP 80 blocked" \
    "timeout 5 curl -s --max-time 3 http://1.1.1.1/" 1

# Clearnet HTTPS should be blocked
check "Clearnet TCP 443 blocked" \
    "timeout 5 curl -s --max-time 3 https://1.1.1.1/" 1

# DNS via clearnet should be blocked (non-Tor DNS)
check "Direct UDP/53 blocked" \
    "timeout 5 dig @8.8.8.8 example.com" 1

# IPv6 should be completely blocked
check "IPv6 outbound blocked" \
    "timeout 5 ping6 -c 1 2001:4860:4860::8888" 1

# ICMP should be blocked
check "ICMP outbound blocked" \
    "timeout 5 ping -c 1 -W 3 8.8.8.8" 1

# Loopback should work
check "Loopback allowed" \
    "ping -c 1 127.0.0.1" 0

# Tor SOCKS5 should be accessible
check "Tor SOCKS5 accessible" \
    "timeout 5 curl -s --max-time 5 --socks5-hostname 127.0.0.1:9050 https://check.torproject.org/api/ip" 0

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" = "0" ] && exit 0 || exit 1

18.5 ISO Boot Smoke Test (QEMU)

Bash

# scripts/test-vm.sh

#!/bin/bash
# Boot DeadDrop OS ISO in QEMU and verify basic smoke tests

ISO=$(find . -name "*.iso" | head -1)

if [ -z "$ISO" ]; then
    echo "No ISO found. Build first."
    exit 1
fi

echo "Testing ISO: $ISO"

# Boot in QEMU with serial console output to file
timeout 120 qemu-system-x86_64 \
    -m 2048 \
    -cdrom "$ISO" \
    -boot d \
    -nographic \
    -serial file:/tmp/deaddrop-boot.log \
    -netdev user,id=net0 \
    -device virtio-net-pci,netdev=net0 \
    &

QEMU_PID=$!
sleep 90

# Check boot log for expected strings
PASS=0
FAIL=0

check_log() {
    local desc="$1"
    local pattern="$2"
    if grep -q "$pattern" /tmp/deaddrop-boot.log; then
        echo "✅ PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "❌ FAIL: $desc (pattern: $pattern)"
        FAIL=$((FAIL + 1))
    fi
}

check_log "Firewall rules applied" "firewall.*rules applied"
check_log "Tor started" "deaddrop-tor.*started"
check_log "Tor bootstrapped" "Tor fully bootstrapped"
check_log "Onion service ready" "Onion service ready"

kill $QEMU_PID 2>/dev/null || true

echo "Smoke test results: $PASS passed, $FAIL failed"
[ "$FAIL" = "0" ] && exit 0 || exit 1

19. Build, Packaging & Installation
19.1 Makefile

Makefile

# Makefile

.PHONY: all build build-go build-python build-iso test test-unit test-integration \
        clean verify-iso write-usb lint fmt

# ── CONFIG ────────────────────────────────────────────────────────────────
VERSION     := $(shell git describe --tags --always --dirty)
ISO_NAME    := deaddrop-os-$(VERSION).iso
BUILD_DIR   := build

# ── PRIMARY TARGETS ───────────────────────────────────────────────────────

all: build

build: build-go build-python build-iso

build-go:
	@echo "==> Building Go onion backend..."
	cd src/onion && \
	CGO_ENABLED=1 go build \
	  -ldflags="-s -w -X main.Version=$(VERSION)" \
	  -o ../../$(BUILD_DIR)/config/includes.chroot/usr/local/bin/deaddrop-onion .
	@echo "==> Go binary built."

build-python:
	@echo "==> Installing Python dependencies into chroot includes..."
	pip3 install \
	  --target $(BUILD_DIR)/config/includes.chroot/usr/local/lib/deaddrop-py \
	  --upgrade \
	  textual \
	  cryptography \
	  aiohttp \
	  aiohttp-socks \
	  pynput
	@echo "==> Python deps installed."

build-iso:
	@echo "==> Initializing live-build config..."
	mkdir -p $(BUILD_DIR) && cd $(BUILD_DIR) && bash ../config/build.sh
	@echo "==> Copying config files..."
	rsync -av --exclude='.git' config/ $(BUILD_DIR)/config/
	@echo "==> Building ISO (requires sudo)..."
	cd $(BUILD_DIR) && sudo lb build 2>&1 | tee ../build.log
	@echo "==> Generating checksum..."
	ISO=$$(find $(BUILD_DIR) -name "*.iso" | head -1); \
	sha256sum "$$ISO" > "$$ISO.sha256"; \
	echo "Built: $$ISO"

# ── TESTING ───────────────────────────────────────────────────────────────

test: test-unit

test-unit:
	@echo "==> Running unit tests..."
	python3 -m pytest tests/unit/ -v --tb=short

test-integration:
	@echo "==> Running integration tests (requires running backend)..."
	python3 -m pytest tests/integration/ -v --tb=short -m integration

test-firewall:
	@echo "==> Running firewall rule tests (requires root, active firewall)..."
	sudo bash tests/integration/test_firewall_rules.sh

test-vm:
	@echo "==> Booting ISO in QEMU for smoke test..."
	bash scripts/test-vm.sh

# ── CODE QUALITY ──────────────────────────────────────────────────────────

lint:
	python3 -m ruff check src/ tests/
	cd src/onion && go vet ./...

fmt:
	python3 -m ruff format src/ tests/
	cd src/onion && gofmt -w .

# ── USB WRITING ───────────────────────────────────────────────────────────

write-usb:
	@echo "==> Finding ISO..."
	@ISO=$$(find $(BUILD_DIR) -name "*.iso" | head -1); \
	echo "ISO: $$ISO"; \
	echo "WARNING: This will DESTROY all data on the target device."; \
	read -p "Target device (e.g. /dev/sdb): " DEV; \
	echo "Writing $$ISO to $$DEV..."; \
	sudo dd if="$$ISO" of="$$DEV" bs=4M status=progress conv=fsync; \
	echo "Done. Verifying..."; \
	sha256sum -c "$$ISO.sha256"

# ── VERIFY ────────────────────────────────────────────────────────────────

verify-iso:
	@ISO=$$(find $(BUILD_DIR) -name "*.iso" | head -1); \
	echo "Verifying $$ISO..."; \
	sha256sum -c "$$ISO.sha256"; \
	if [ -f "$$ISO.asc" ]; then \
	  gpg --verify "$$ISO.asc" "$$ISO"; \
	fi

# ── CLEAN ─────────────────────────────────────────────────────────────────

clean:
	sudo lb clean --all || true
	rm -rf $(BUILD_DIR)/.build
	rm -f build.log

clean-all: clean
	rm -rf $(BUILD_DIR)

19.2 Python Package Definition

toml

# pyproject.toml

[project]
name = "deaddrop-messenger"
version = "1.0.0"
description = "DeadDrop OS Encrypted Messenger TUI"
requires-python = ">=3.11"

dependencies = [
    "textual>=0.59.0",
    "cryptography>=42.0.0",
    "aiohttp>=3.9.0",
    "aiohttp-socks>=0.8.0",
    "pynput>=1.7.0",
]

[project.scripts]
deaddrop-messenger = "messenger.__main__:main"
deaddrop-panic-daemon = "panic.__main__:main"

[tool.ruff]
target-version = "py311"
line-length = 100

[tool.pytest.ini_options]
markers = [
    "integration: marks integration tests (require running backend)"
]

19.3 Server Deployment (Onion Backend)

Bash

# scripts/deploy-server.sh
# Deploy onion backend on a fresh Debian bookworm server

#!/bin/bash
set -euo pipefail

echo "==> Installing dependencies..."
apt-get update
apt-get install -y tor sqlite3 golang-go

echo "==> Creating service user..."
useradd -r -s /bin/false -d /var/lib/deaddrop deaddrop-server || true
mkdir -p /var/lib/deaddrop/tor-data
chown -R deaddrop-server:deaddrop-server /var/lib/deaddrop

echo "==> Building onion backend..."
cd /opt/deaddrop-server/src/onion
go build -ldflags="-s -w" -o /usr/local/bin/deaddrop-onion .

echo "==> Installing systemd unit..."
cat > /etc/systemd/system/deaddrop-onion.service <<EOF
[Unit]
Description=DeadDrop OS Onion Service Backend
After=network.target

[Service]
Type=simple
User=deaddrop-server
Group=deaddrop-server
ExecStart=/usr/local/bin/deaddrop-onion
Restart=always
RestartSec=5
Environment=DEADDROP_DB=/var/lib/deaddrop/drops.db
Environment=DEADDROP_KEY=/var/lib/deaddrop/onion_key.pem

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable deaddrop-onion
systemctl start deaddrop-onion

echo "==> Waiting for onion address..."
sleep 30
ONION_ADDR=$(cat /var/lib/deaddrop/onion_address.txt 2>/dev/null || echo "not yet generated")
echo ""
echo "==================================================================="
echo "✅ DeadDrop Onion Backend is running!"
echo ""
echo "Onion address: $ONION_ADDR"
echo ""
echo "Add this to your DeadDrop OS ISO config:"
echo "  onion_url = http://$ONION_ADDR"
echo "==================================================================="

20. Platform Support Matrix
Feature	Client: x86_64 BIOS	Client: x86_64 UEFI	Client: ARM64	Server: Debian Linux
ISO boot	✅	✅	⚠️ Experimental	N/A
Tor transparent proxy	✅	✅	✅	✅
Amnesia (tmpfs)	✅	✅	✅	N/A
Messenger TUI	✅	✅	✅	✅
Onion backend	N/A	N/A	N/A	✅
Panic Mode	✅	✅	✅	N/A
IPv6 disable	✅	✅	✅	✅
Build via GitHub Actions	✅ (ubuntu-22.04)	✅	❌ Cross-compile needed	N/A

Note: The client OS is currently targeted at x86_64. ARM64 support (e.g., running from USB on Apple Silicon via virtualization) is marked experimental because Debian Live live-build ARM64 configuration requires additional work. This is a roadmap item.
21. Performance Targets & Benchmarks
Metric	Target	Notes
Boot to usable session	< 90 seconds	Includes Tor bootstrap (network-dependent)
Tor bootstrap time (good network)	< 60 seconds	Depends on Tor network conditions
Tor bootstrap time (censored network)	Operator configures bridges	Out of default scope
Message encryption (1KB)	< 50ms	RSA-OAEP + AES-GCM in Python
Message encryption (1MB)	< 500ms	AES-GCM scales linearly
Message post (over Tor)	< 30 seconds	Tor adds latency
Message fetch (100 drops)	< 60 seconds	Tor circuit dependent
Firewall rule application	< 1 second	iptables ruleset is small
Panic Mode trigger to decoys visible	< 5 seconds	Process kill + app launch
ISO size	≤ 1.5 GB	Debian Live minimal target
RAM usage at rest	< 400 MB	Well within 2GB minimum
22. Error Handling Strategy
22.1 Error Categories
Category	Examples	Policy
Tor bootstrap failure	No network, Tor censored	Block UI with error + retry/shutdown options
Firewall failure	iptables not available	Hard stop — do not allow networking
Crypto error (encrypt)	Invalid public key	Surface in TUI — do not send
Crypto error (decrypt)	Wrong key	Expected — mark drop as "Not for us"
Tampered message (InvalidTag)	AES-GCM auth fail	Surface warning — do not display content
Transport failure	Onion backend unreachable	Retry with backoff, show error
Backend server error	5xx response	Surface in TUI, retry once
Panic daemon failure	pynput error	Restart daemon; log to /tmp
22.2 Critical Rule: Network Failure Must Not Fall Back to Clearnet

Bash

# /usr/local/lib/deaddrop/firewall-check.sh
# Run before allowing any user-space networking

#!/bin/bash
# Verify firewall is in Tor-only state before proceeding

DEFAULT_POLICY=$(iptables -S OUTPUT | grep "^-P OUTPUT" | awk '{print $3}')

if [ "$DEFAULT_POLICY" != "DROP" ]; then
    echo "CRITICAL: iptables OUTPUT policy is not DROP. Refusing to proceed."
    echo "Network access is not safe. Halting."
    systemctl stop deaddrop-tor
    systemctl stop deaddrop-messenger
    exit 1
fi

echo "Firewall verified: OUTPUT policy = DROP. Safe to proceed."

22.3 Messenger Error Handling Pattern

Python

# src/messenger/app.py (error handling pattern)

@work(exclusive=False)
async def _send_message(self) -> None:
    status = self.query_one("#compose_status", Static)
    try:
        # ... encryption and posting
    except DecryptionError as e:
        # Expected non-fatal — wrong key
        status.update(f"[yellow]⚠️ Could not decrypt: {e}[/]")
    except InvalidTag:
        # Authentication failure — tampered message
        status.update("[red]❌ Message authentication failed — may be tampered.[/]")
    except aiohttp.ClientConnectionError as e:
        # Onion backend unreachable
        status.update(f"[red]❌ Could not reach onion backend: {e}\n"
                      f"Is Tor fully bootstrapped?[/]")
    except aiohttp.ClientResponseError as e:
        # Server error
        status.update(f"[red]❌ Server error {e.status}: {e.message}[/]")
    except Exception as e:
        # Unexpected — log and show generic error
        import traceback
        with open("/tmp/deaddrop-messenger.log", "a") as f:
            traceback.print_exc(file=f)
        status.update(f"[red]❌ Unexpected error: {type(e).__name__}. "
                      f"See /tmp/deaddrop-messenger.log[/]")

23. Dependency Registry
23.1 Python Dependencies
Package	Version	Purpose
textual	≥ 0.59.0	TUI framework
cryptography	≥ 42.0.0	RSA-OAEP, AES-GCM, key management
aiohttp	≥ 3.9.0	Async HTTP client
aiohttp-socks	≥ 0.8.0	SOCKS5 proxy connector for aiohttp
pynput	≥ 1.7.0	Keyboard listener for Panic Mode
23.2 Go Dependencies

Go

// src/onion/go.mod

module deaddrop-onion

go 1.22

require (
    github.com/cretz/bine v0.2.0             // Tor control + onion service
    github.com/mattn/go-sqlite3 v1.14.22     // SQLite driver (CGO)
    github.com/google/uuid v1.6.0            // UUID generation
)

23.3 System Packages (Installed in ISO)
Package	Purpose
tor	Tor daemon
tor-geoipdb	Tor GeoIP database
openbox	Minimal window manager
xorg, xinit	X11 display server
xterm	Terminal for TUI
xclip	Clipboard management (panic clear)
feh	Wallpaper setter (panic decoy)
libreoffice-calc	Decoy application
firefox-esr	Decoy application
nftables, iptables	Firewall
iproute2	Network utilities
python3, python3-pip	Messenger + panic daemon
curl	Health checks
23.4 Build Tools Required
Tool	Version	Required For
live-build	3.x (Debian)	ISO build
debootstrap	Any	Debian chroot bootstrap
Go toolchain	≥ 1.22	Onion backend binary
Python	≥ 3.11	Messenger + panic deps
xorriso	Any	ISO image creation
squashfs-tools	Any	squashfs compression
QEMU	Any	ISO smoke testing
GPG	Any	Release signing
24. Milestone & Phased Rollout Plan
Phase 1 — Bootable Hardened OS (Weeks 1–4)

Goal: A bootable ISO that enforces
