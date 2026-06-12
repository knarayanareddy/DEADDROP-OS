# Response to external production gap report

The external review identified a valid 4/10 pre-alpha readiness state. This branch addresses the actionable code/configuration defects while keeping non-code human gates explicitly blocked.

## Fixed critical bugs

### 1. Rate-limit memory leak and ephemeral-port bypass

- Reworked `server/ratelimit.go` to strip ephemeral source ports from `RemoteAddr`.
- Added expired-bucket pruning to prevent unbounded map growth.
- Added tests for port stripping and pruning.

Note: true per-Tor-circuit rate limiting is not exposed to the Go backend by a standard onion-service TCP mapping. The current backend behavior is safer than per-port accounting and effectively per-endpoint/per-loopback-origin for onion deployment. Tor-level rate limiting remains a future hardening item.

### 2. Logrotate SIGUSR1 crash

- Removed `systemctl kill -s USR1 deaddrop-server.service` from logrotate.
- Switched logrotate entries to `copytruncate` for systemd append logs.
- Added static invariant checks to prevent regression.

### 3. Inactivity panic daemon unconditional wipe

- Added a Unix datagram activity socket: `/run/deaddrop/activity.sock`.
- The panic daemon resets the inactivity timer whenever the client/TUI emits activity.
- Added unit test proving activity resets the inactivity timer.

### 4. Panic daemon restart loop

- Changed `Restart=always` to `Restart=on-failure`.
- A successful wipe exits cleanly and does not immediately restart into a wipe loop.
- Added static invariant checks to prevent regression.

### 5. Redundant TTL expiry loops

- Removed cron-driven TTL expiry.
- The Go backend owns TTL deletion via its in-process maintenance loop.
- Cron now only performs daily WAL checkpoint/VACUUM.

## Fixed architectural gap

### TUI implementation

- Added `github.com/rivo/tview v0.42.0`.
- `cmd/deaddrop-client` now launches a real terminal UI by default.
- The previous line-oriented interface remains available with `-cli` as a fallback.

## Still not claimable as production-complete

- Full QEMU/ISO packet-capture tests still require a real Debian live-build/QEMU environment and an actual ISO artifact.
- Human gates G-1 through G-6 remain blockers by design.


## DEADDROP-OSv2 residual-risk follow-up

### Global rate-limit starvation

The v2 review correctly noted that stripping Tor loopback source ports turns the limiter into a shared loopback bucket. To reduce starvation without reintroducing ephemeral-port bypasses, the backend now issues a stateless proof-of-work challenge after a bucket is exhausted:

- response header: `X-Deaddrop-Pow-Challenge`
- retry header: `X-Deaddrop-Pow`
- challenge is HMAC-bound to endpoint, expiry, and difficulty
- client helpers solve and retry automatically for submit/retrieve

This does not replace true Tor-circuit-level accounting, but it prevents one cheap burst from fully starving honest clients that can spend a small amount of CPU.

### Active ISO lifecycle testing

Added a guest-side E2E hook and client self-test mode:

- `deaddrop-client -self-test` performs generate/encrypt/submit/retrieve/decrypt non-interactively.
- `deaddrop-e2e.service` runs inside the ISO only when `deaddrop.e2e=1` is present on the kernel command line.
- `deaddrop-e2e-guest` validates firewall/IPv6 state, optionally runs the client lifecycle against `deaddrop.e2e_url=...`, triggers panic wipe, and powers off.

The host QEMU harness remains topology-dependent for full onion test-server wiring, but the ISO now contains the active lifecycle hook needed for an automated lab run.
