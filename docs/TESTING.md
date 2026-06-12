# Testing

## Unit and integration tests

```bash
go test ./...
```

Coverage includes:

- crypto round trip;
- tamper detection;
- wrong-key generic error;
- nonce uniqueness smoke test;
- API submit/retrieve/delete;
- TTL expiry;
- hint isolation;
- concurrent drops;
- rate limiter pruning and port stripping;
- proof-of-work retry;
- panic wipe and inactivity reset.

## Static security checks

```bash
python3 tests/security/check_static_invariants.py
```

Checks include:

- firewall default drop;
- IPv6 drop/disable settings;
- Tor tmpfs config;
- wait-for-Tor gates;
- no dangerous logrotate signal;
- no panic restart loop;
- no duplicate cron TTL expiry;
- no full `RemoteAddr` rate-limit key;
- no forbidden DB schema fields.

## Vulnerability scanning

```bash
govulncheck ./...
```

Production release requires no reachable vulnerabilities.

## Release check

```bash
./tools/release-check.sh
```

Runs formatting, tests, vet, static checks, govulncheck, binary build, backend smoke test, and log redaction test.

## QEMU tests

```bash
tests/qemu/run-qemu-security-tests.sh artifacts/deaddrop-os.iso
```

The passive harness asserts no IPv6 and no non-DHCP UDP in boot capture. Active lifecycle validation uses the guest E2E ISO mode described in `ISO_BUILD.md`.
