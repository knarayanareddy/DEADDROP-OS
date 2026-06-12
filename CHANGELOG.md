# Changelog

## 0.1.0-rc2

- Implemented tview client TUI.
- Added CLI fallback and self-test mode.
- Implemented panic daemon activity socket and inactivity reset.
- Fixed panic restart loop.
- Fixed logrotate SIGUSR1 crash risk.
- Fixed rate limiter ephemeral-port bypass and bucket pruning.
- Added proof-of-work fallback for shared onion rate-limit buckets.
- Consolidated TTL expiry into backend loop.
- Added active ISO E2E guest hook.
- Added release checks, production gate, documentation, and runbooks.

## 0.1.0-rc1

- Initial Go backend, SQLite storage, crypto layer, firewall, panic daemon, and build scaffolding.
