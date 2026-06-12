# DEADDROP-OS

**DEADDROP-OS** is a Tor-only, amnesic dead-drop messaging system designed around the approved `deaddroposdesigndoc.md` production engineering specification. It provides an x86_64 Debian Live client environment, a persistent onion-only backend, end-to-end encrypted asynchronous drops, and operational tooling for deployment, verification, and release gating.

> **Current status:** Release Candidate codebase. The implementation passes local release checks, but a production tag remains blocked until the human/process gates in the specification are closed and the ISO/QEMU security tests are run in a real lab environment.

---

## 1. Security goals

DEADDROP-OS is designed to provide:

- **Amnesia on the client:** keys, plaintext, decrypted payloads, and session state live only in tmpfs/RAM.
- **Tor-only networking:** client firewall policy drops all traffic except Tor-mediated traffic and DNS redirection.
- **Onion-only server exposure:** backend listens only on `127.0.0.1`; Tor exposes the service.
- **Asynchronous delivery:** sender and recipient do not need to be online simultaneously.
- **End-to-end encryption:** server stores ciphertext only and never holds decryption keys.
- **Minimal metadata logging:** server logs exclude client IPs, hints, request bodies, payloads, user agents, and plaintext.
- **Verifiable release process:** release scripts, static invariant checks, vulnerability scans, build artifacts, and production gates are included.

Out-of-scope threats include compromised firmware, global Tor correlation, cold-boot RAM extraction, coerced disclosure, and supply-chain compromise if users do not verify published hashes.

---

## 2. Repository layout

```text
.
├── client/
│   ├── crypto/          RSA-OAEP + AES-256-GCM Addendum B wire format
│   ├── firewall/        Tor-only firewall + IPv6 disable config
│   ├── panic/           panic wipe daemon and activity socket logic
│   ├── securemem/       best-effort mlockall support
│   └── tui/             tview terminal UI and client HTTP helpers
├── cmd/
│   ├── deaddrop-client/ client TUI / CLI / self-test entrypoint
│   ├── deaddrop-panic/  panic daemon entrypoint
│   └── deaddrop-server/ onion-only backend entrypoint
├── server/              HTTP API, SQLite store, TTL expiry, rate limiting, PoW
├── deploy/
│   ├── systemd/         systemd service units
│   ├── tor/             Tor client/server snippets
│   ├── logrotate/       log rotation policy
│   ├── sysctl/          host hardening sysctls
│   └── scripts/         deploy, backup, maintenance, E2E guest scripts
├── iso/                 Debian live-build ISO scaffolding
├── tests/
│   ├── security/        static security invariant checks
│   └── qemu/            QEMU packet-capture harness
├── tools/               release, build, and production gate scripts
├── docs/                complete engineering and operational documentation
├── runbooks/            incident/backup/restore/operator procedures
├── release/             readiness, manifest, expert-feedback response
├── deaddroposdesigndoc.md canonical product/spec document
└── VERSION              implementation version
```

---

## 3. Quick start for development

### 3.1 Prerequisites

Production release checks require the pinned toolchain:

- Debian-like Linux environment
- Go **1.25.11** exactly for release checks
- GCC / libc development headers for `github.com/mattn/go-sqlite3`
- Python 3
- `curl`
- `govulncheck` for production checks

Install `govulncheck`:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### 3.2 Run tests

```bash
go test ./...
go vet ./...
python3 tests/security/check_static_invariants.py
govulncheck ./...
```

Or run the full release check:

```bash
./tools/release-check.sh
```

### 3.3 Build binaries

```bash
./tools/build-binaries.sh
cat artifacts/SHA256SUMS
```

Binaries produced:

```text
artifacts/deaddrop-server
artifacts/deaddrop-client
artifacts/deaddrop-panic
artifacts/SHA256SUMS
```

### 3.4 Run local backend

```bash
./artifacts/deaddrop-server -db ./drops.dev.db
```

Health check:

```bash
curl http://127.0.0.1:8080/health
```

### 3.5 Run client TUI locally

```bash
DEADDROP_SERVER=http://127.0.0.1:8080 ./artifacts/deaddrop-client
```

Line-oriented fallback mode:

```bash
./artifacts/deaddrop-client -cli -server http://127.0.0.1:8080
```

Non-interactive client self-test:

```bash
./artifacts/deaddrop-client \
  -self-test \
  -server http://127.0.0.1:8080 \
  -key-dir /tmp/deaddrop-selftest
```

---

## 4. Core components

### 4.1 Client crypto

Implemented in `client/crypto`:

- RSA-4096 key generation for normal sessions
- RSA-OAEP-SHA256 encrypted data encryption key (DEK)
- AES-256-GCM payload encryption
- CSPRNG nonce generation
- compact JSON payload schema from Addendum B
- base64url outer envelope
- generic decryption failure errors

See [`docs/CRYPTOGRAPHY.md`](docs/CRYPTOGRAPHY.md).

### 4.2 Server backend

Implemented in `server` and `cmd/deaddrop-server`:

- `POST /drop`
- `GET /drops?hint=...`
- `DELETE /drop/:id`
- `GET /health`
- SQLite WAL storage
- hard TTL expiry loop
- no plaintext columns
- no client IP/payload/hint logging
- shared-bucket rate limiting with proof-of-work fallback

See [`docs/API.md`](docs/API.md) and [`docs/SERVER.md`](docs/SERVER.md).

### 4.3 Client TUI

Implemented using `github.com/rivo/tview`:

- compose encrypted drop
- retrieve and decrypt drops
- show session public key
- activity pings to panic daemon
- CLI fallback for diagnostics

See [`docs/CLIENT.md`](docs/CLIENT.md).

### 4.4 Panic daemon

Implemented in `client/panic` and `cmd/deaddrop-panic`:

- signal-triggered wipe via `SIGUSR1`
- inactivity-triggered wipe
- activity reset socket at `/run/deaddrop/activity.sock`
- best-effort overwrite and removal of configured tmpfs paths
- systemd `Restart=on-failure` to avoid wipe loops

See [`docs/PANIC_DAEMON.md`](docs/PANIC_DAEMON.md).

### 4.5 Firewall and Tor enforcement

Implemented in `client/firewall` and `deploy/tor`:

- iptables default DROP
- ip6tables default DROP
- TCP redirect to Tor `TransPort`
- DNS redirect to Tor `DNSPort`
- UDP/ICMP blocks
- Tor data/log directories on tmpfs for the client

See [`docs/NETWORK_ENFORCEMENT.md`](docs/NETWORK_ENFORCEMENT.md).

---

## 5. API overview

### POST `/drop`

Submit an encrypted payload:

```json
{
  "hint": "shared_hint",
  "payload": "base64url-addendum-b-envelope"
}
```

Success:

```json
{
  "id": "uuid-v4",
  "expires_at": "2026-06-15T19:08:38Z"
}
```

### GET `/drops?hint=...`

Returns all unexpired drops for a hint:

```json
{
  "drops": [
    {
      "id": "uuid-v4",
      "payload": "base64url-addendum-b-envelope",
      "created_at": "2026-06-12T19:08:38Z",
      "expires_at": "2026-06-15T19:08:38Z"
    }
  ]
}
```

### DELETE `/drop/:id`

Always returns `204 No Content` for success or non-existent IDs, preventing enumeration.

### GET `/health`

```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

Detailed API: [`docs/API.md`](docs/API.md).

---

## 6. Proof-of-work fallback for shared onion rate limits

Tor hidden services map backend connections to loopback. Rate limiting by full `RemoteAddr` leaks memory and is bypassable because source ports change. Rate limiting by host avoids that but creates shared-bucket starvation.

DEADDROP-OS uses:

1. endpoint/loopback keyed rate limiting, plus
2. stateless proof-of-work challenge after a bucket is exhausted.

Server returns:

```http
HTTP/1.1 429 Too Many Requests
X-Deaddrop-Pow-Challenge: <challenge>
```

Client retries with:

```http
X-Deaddrop-Pow: <solution>
```

This is a mitigation, not a complete substitute for future Tor-circuit-aware accounting.

---

## 7. Build the ISO

ISO builds require a clean Debian 12 host/container with `live-build`.

```bash
./tools/build-binaries.sh
./iso/build-iso.sh
cat artifacts/deaddrop-os.iso.sha256
```

For active E2E test mode:

```bash
EXTRA_BOOTAPPEND='deaddrop.e2e=1 deaddrop.e2e_url=http://<test-onion>' ./iso/build-iso.sh
```

See [`docs/ISO_BUILD.md`](docs/ISO_BUILD.md).

---

## 8. Deploy the onion server

On a clean Debian 12 server:

```bash
./tools/build-binaries.sh
sudo deploy/scripts/deploy.sh artifacts/deaddrop-server
```

The deploy script installs:

- Tor
- SQLite
- service user `deaddrop`
- `/var/lib/deaddrop`
- `/var/log/deaddrop`
- Tor hidden-service mapping
- systemd service
- logrotate policy
- daily vacuum cron

Retrieve onion hostname:

```bash
sudo cat /var/lib/tor/deaddrop/hostname
```

See [`docs/SERVER_DEPLOYMENT.md`](docs/SERVER_DEPLOYMENT.md).

---

## 9. Release checks

Run:

```bash
./tools/release-check.sh
```

Checks include:

- exact Go toolchain check
- `gofmt`
- `go test ./...`
- `go vet ./...`
- static security invariant checks
- `govulncheck ./...`
- release-style binary build
- backend smoke test
- log redaction check

Final production gate:

```bash
DEADDROP_GATE_G1_SPEC_OWNER_SECURITY_LEAD=CLOSED \
DEADDROP_GATE_G2_ADR05_RESOLVED=CLOSED \
DEADDROP_GATE_G3_BACKUP_RESTORE_SIGNED=CLOSED \
DEADDROP_GATE_G4_IR_TABLETOP_DONE=CLOSED \
DEADDROP_GATE_G5_ONION_KEY_AUDITED=CLOSED \
DEADDROP_GATE_G6_ISO_HASH_PUBLISHED=CLOSED \
DEADDROP_QEMU_SECURITY_TESTS_PASS=CLOSED \
./tools/production-gate.sh
```

See [`docs/RELEASE_PROCESS.md`](docs/RELEASE_PROCESS.md).

---

## 10. QEMU and security invariant testing

Passive host-side capture:

```bash
tests/qemu/run-qemu-security-tests.sh artifacts/deaddrop-os.iso
```

Active guest lifecycle mode requires an ISO built with:

```bash
EXTRA_BOOTAPPEND='deaddrop.e2e=1 deaddrop.e2e_url=http://<test-onion>' ./iso/build-iso.sh
```

The guest then runs firewall checks, optional submit/retrieve/decrypt client self-test, panic wipe, and shutdown.

See [`docs/TESTING.md`](docs/TESTING.md).

---

## 11. Documentation index

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- [`docs/SECURITY_MODEL.md`](docs/SECURITY_MODEL.md)
- [`docs/CRYPTOGRAPHY.md`](docs/CRYPTOGRAPHY.md)
- [`docs/API.md`](docs/API.md)
- [`docs/CLIENT.md`](docs/CLIENT.md)
- [`docs/SERVER.md`](docs/SERVER.md)
- [`docs/SERVER_DEPLOYMENT.md`](docs/SERVER_DEPLOYMENT.md)
- [`docs/NETWORK_ENFORCEMENT.md`](docs/NETWORK_ENFORCEMENT.md)
- [`docs/PANIC_DAEMON.md`](docs/PANIC_DAEMON.md)
- [`docs/ISO_BUILD.md`](docs/ISO_BUILD.md)
- [`docs/TESTING.md`](docs/TESTING.md)
- [`docs/OPERATIONS.md`](docs/OPERATIONS.md)
- [`docs/RELEASE_PROCESS.md`](docs/RELEASE_PROCESS.md)
- [`docs/TROUBLESHOOTING.md`](docs/TROUBLESHOOTING.md)
- [`docs/COMPLIANCE_MATRIX.md`](docs/COMPLIANCE_MATRIX.md)
- [`runbooks/BACKUP_RESTORE.md`](runbooks/BACKUP_RESTORE.md)
- [`runbooks/INCIDENT_RESPONSE.md`](runbooks/INCIDENT_RESPONSE.md)
- [`runbooks/ONION_KEY_ROTATION.md`](runbooks/ONION_KEY_ROTATION.md)
- [`release/PRODUCTION_READINESS.md`](release/PRODUCTION_READINESS.md)
- [`release/MANIFEST.md`](release/MANIFEST.md)
- [`release/EXPERT_FEEDBACK_RESPONSE.md`](release/EXPERT_FEEDBACK_RESPONSE.md)

---

## 12. Production readiness statement

This repository contains a code-complete release candidate with automated checks. It should not be represented as production-approved until:

1. all human gates G-1 through G-6 in the design document are closed;
2. ADR-05 is formally resolved;
3. ISO hash reproducibility is proven on two clean builders;
4. QEMU active lifecycle packet-capture tests pass;
5. backup/restore and incident-response tabletop exercises are signed off.

Until then, the correct status is:

```text
Release Candidate / code-ready pending external production gates
```
