# DEADDROP-OS implementation status

This repository contains a release-candidate implementation aligned to the approved DEADDROP-OS v2.0.0 production engineering specification where code can satisfy the requirement directly. Human release gates and lab-only ISO/QEMU validation remain external blockers.

## Implemented

### Backend

- Go backend in `server/` and `cmd/deaddrop-server`
- Endpoints: `POST /drop`, `GET /drops`, `DELETE /drop/:id`, `GET /health`
- SQLite WAL schema with hint/expires indexes
- In-process TTL hard-delete loop
- Loopback-only listener guard for onion-only deployment
- Access logging policy that excludes client IPs, hints, user agents, bodies, and payloads
- Endpoint/host rate limiting with expired-bucket pruning
- Stateless proof-of-work fallback to reduce shared Tor-loopback bucket starvation

### Client crypto

- RSA-OAEP-SHA256 DEK wrapping
- AES-256-GCM content encryption
- Addendum B compact JSON + base64url wire envelope
- Generic decryption failure errors
- PEM key generation/parsing helpers

### Client UI

- Full `tview` TUI in `client/tui/app.go`
- Compose, retrieve, decrypt, public-key display screens
- CLI fallback via `deaddrop-client -cli`
- Non-interactive active lifecycle test via `deaddrop-client -self-test`

### Panic daemon

- Signal and inactivity triggers
- Unix datagram activity socket at `/run/deaddrop/activity.sock`
- Client activity pings reset the inactivity timer
- Best-effort wipe of tmpfs key/session paths
- systemd `Restart=on-failure` to avoid wipe loops

### Firewall and platform config

- Canonical Addendum A iptables/ip6tables script
- IPv6 sysctl disablement
- client/server Tor snippets
- systemd units for firewall, TUI, backend, panic daemon, and E2E test hook
- volatile journald config for client
- server logrotate policy without unsafe signals

### Build/deploy/release

- `tools/build-binaries.sh`
- `tools/release-check.sh`
- `tools/production-gate.sh`
- Debian live-build ISO scaffolding under `iso/`
- server deployment script and wait-for-Tor gate
- backup, vacuum, and E2E guest scripts
- release manifest and readiness documentation

### Tests

- crypto round trip, tamper detection, wrong-key failure, nonce uniqueness, PEM parse
- backend API, TTL, delete, hint isolation, health, concurrent drops, logging policy
- rate limiter port stripping and pruning tests
- proof-of-work retry test
- panic wipe and inactivity reset tests
- static security invariant checks
- QEMU passive capture harness and guest active lifecycle hook

## Verified in this workspace

The release check has been run successfully with the pinned Go 1.25.11 toolchain:

```text
gofmt: PASS
go test ./...: PASS
go vet ./...: PASS
static security checks: PASS
govulncheck ./...: PASS
binary build: PASS
backend smoke/log-redaction test: PASS
```

## Not fully verifiable inside this sandbox

The following require an external clean Debian 12 live-build/QEMU environment:

- final bootable ISO assembly;
- independent ISO hash reproduction;
- active QEMU packet-capture lifecycle with a real test onion service;
- production deployment signoff.

## Human gates still open

Per §1.3 of `deaddroposdesigndoc.md`, production release remains blocked until named humans close gates G-1 through G-6, including ADR-05 resolution, backup/restore tabletop, incident contacts, onion key audit, and reproducible ISO hash publication.
