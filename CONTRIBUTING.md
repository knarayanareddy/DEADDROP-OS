# Contributing

## Ground rules

- `deaddroposdesigndoc.md` is the canonical specification.
- Normative security changes require explicit review under the spec change-control process.
- Do not add persistent client storage.
- Do not add clearnet fallback.
- Do not log hints, payloads, request bodies, client IPs, user agents, plaintext, or key material.

## Before submitting changes

```bash
./tools/release-check.sh
```

## Style

- Go code must be `gofmt` clean.
- Keep dependencies pinned in `go.mod`/`go.sum`.
- Add tests for regressions and security invariants.

## Security-sensitive areas

Changes to these require extra review:

- `client/crypto`
- `client/firewall`
- `server/pow.go`
- `server/ratelimit.go`
- systemd units
- Tor configs
- ISO build hooks
