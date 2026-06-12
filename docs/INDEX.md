# DEADDROP-OS documentation index

This directory contains the implementation-facing documentation for DEADDROP-OS. The canonical product/specification source remains `deaddroposdesigndoc.md`; these documents explain how the codebase implements, builds, operates, and verifies that specification.

## Core engineering docs

| Document | Purpose |
|---|---|
| `ARCHITECTURE.md` | End-to-end component architecture and boot/service ordering |
| `SECURITY_MODEL.md` | Trust boundaries, threat model, invariants, residual risks |
| `CRYPTOGRAPHY.md` | Wire format, RSA-OAEP/AES-GCM flow, key handling |
| `API.md` | Server HTTP API contract, errors, rate limiting, PoW |
| `CLIENT.md` | TUI, CLI fallback, self-test mode, key locations |
| `SERVER.md` | Backend internals, SQLite schema, expiry, logging |
| `NETWORK_ENFORCEMENT.md` | Firewall, Tor, DNS, IPv6 disablement |
| `PANIC_DAEMON.md` | Panic triggers, activity socket, wipe behavior |

## Build/test/ops docs

| Document | Purpose |
|---|---|
| `ISO_BUILD.md` | Debian live-build ISO process and E2E mode |
| `TESTING.md` | Unit, integration, static, vulnerability, QEMU tests |
| `SERVER_DEPLOYMENT.md` | Onion server deployment on Debian 12 |
| `OPERATIONS.md` | Routine maintenance and operational procedures |
| `RELEASE_PROCESS.md` | Release checks and production gate procedure |
| `TROUBLESHOOTING.md` | Common failures and fixes |
| `COMPLIANCE_MATRIX.md` | Mapping from spec requirements to implementation |
