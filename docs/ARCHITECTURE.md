# Architecture

## Overview

DEADDROP-OS is split into two main deployment zones:

1. **Client live OS** — amnesic Debian Live environment with Tor-only network enforcement, TUI messenger, cryptographic client code, and panic wipe daemon.
2. **Persistent onion server** — Debian server running Tor hidden service and a loopback-only Go HTTP backend backed by SQLite.

```text
Client Live ISO
  ├─ deaddrop-firewall.service
  ├─ tor.service
  ├─ deaddrop-panic.service
  └─ deaddrop-tui.service / deaddrop-client
          │
          │ onion HTTP
          ▼
Server
  ├─ tor hidden service :80 -> 127.0.0.1:8080
  ├─ deaddrop-server
  └─ /var/lib/deaddrop/drops.db
```

## Client components

| Component | Path | Responsibility |
|---|---|---|
| Crypto layer | `client/crypto` | RSA-OAEP/AES-GCM encryption and decryption |
| TUI | `client/tui`, `cmd/deaddrop-client` | Compose/retrieve/decrypt drops |
| Panic daemon | `client/panic`, `cmd/deaddrop-panic` | Signal/inactivity-triggered wipe |
| Firewall | `client/firewall/firewall.sh` | iptables/ip6tables Tor-only policy |
| Secure memory | `client/securemem` | Best-effort `mlockall` |

## Server components

| Component | Path | Responsibility |
|---|---|---|
| HTTP API | `server/server.go` | `/drop`, `/drops`, `/drop/:id`, `/health` |
| Storage | `server/storage.go` | SQLite schema, WAL, TTL delete |
| Rate limiting | `server/ratelimit.go` | shared endpoint buckets with pruning |
| PoW | `server/pow.go` | stateless challenge fallback after rate-limit exhaustion |
| Entrypoint | `cmd/deaddrop-server` | loopback-only listener and expiry loop |

## Boot/service ordering

Client service ordering follows the design invariants:

```text
firewall -> network-pre.target -> tor -> TUI
panic daemon starts after basic.target independently
```

Important units:

- `deploy/systemd/deaddrop-firewall.service`
- `deploy/systemd/deaddrop-tui.service`
- `deploy/systemd/deaddrop-panic.service`
- `deploy/systemd/deaddrop-e2e.service`

## Data flow: submit

1. User opens TUI compose form.
2. TUI loads recipient public key from pasted PEM.
3. Crypto layer generates random DEK and nonce.
4. AES-GCM encrypts plaintext.
5. RSA-OAEP encrypts DEK.
6. Client serializes Addendum B envelope and base64url encodes it.
7. Client POSTs to onion server `/drop`.
8. Server validates hint/payload and stores ciphertext envelope.

## Data flow: retrieve

1. User enters hint.
2. Client GETs `/drops?hint=...`.
3. Server returns all unexpired ciphertext envelopes for the hint.
4. Client decrypts DEK with local private key.
5. Client decrypts AES-GCM payload.
6. Plaintext is displayed in terminal only.

## Design constraints

- Server never holds content encryption keys.
- Server only stores encrypted payload envelopes.
- Client keys are intended to live under tmpfs paths.
- Backend refuses non-loopback listeners.
- Client firewall denies non-Tor paths.
