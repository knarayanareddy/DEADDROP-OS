# DEADDROP-OS production readiness checklist

This checklist maps the implementation to the v2.0.0 design document acceptance criteria.

## Automated gates

Run from a clean checkout:

```bash
./tools/release-check.sh
```

Expected checks:

- `gofmt` clean
- exact Go 1.25.11 toolchain check
- `go test ./...`
- `go vet ./...`
- static security invariant checks
- `govulncheck ./...` with no reachable vulnerabilities
- Linux AMD64 binary build
- artifact SHA-256 generation
- backend smoke test for health/drop/retrieve/delete/log redaction

## ISO release gate

On a clean Debian 12 builder with `live-build` installed:

```bash
./tools/build-binaries.sh
./iso/build-iso.sh
cat artifacts/deaddrop-os.iso.sha256
```

A production release must reproduce the same ISO SHA-256 on two independent clean builders.

## Human gates from §1.3

These remain release blockers and cannot be closed by code:

- G-1 named Spec Owner and Security Lead
- G-2 ADR-05 TLS-on-onion decision resolved
- G-3 backup/restore runbook tested and signed off
- G-4 incident response contacts populated and tabletop completed
- G-5 onion key generation/storage procedure audited
- G-6 ISO reproduced from clean environment and SHA-256 published

## QEMU/security invariant gates

The scripts under `tests/qemu/` document the required commands and expected results for AC-01 through AC-06. They must be run against the produced ISO on hardware/VM networking that allows packet capture.


## Final production tag command

After human gates are closed and the ISO hash exists, the release manager must run:

```bash
DEADDROP_GATE_G1_SPEC_OWNER_SECURITY_LEAD=CLOSED DEADDROP_GATE_G2_ADR05_RESOLVED=CLOSED DEADDROP_GATE_G3_BACKUP_RESTORE_SIGNED=CLOSED DEADDROP_GATE_G4_IR_TABLETOP_DONE=CLOSED DEADDROP_GATE_G5_ONION_KEY_AUDITED=CLOSED DEADDROP_GATE_G6_ISO_HASH_PUBLISHED=CLOSED DEADDROP_QEMU_SECURITY_TESTS_PASS=CLOSED ./tools/production-gate.sh
```

## External gap report fixes

See `release/EXPERT_FEEDBACK_RESPONSE.md` for the remediation mapping.


## Proof-of-work starvation mitigation

If a shared Tor-loopback bucket is exhausted, the server returns a proof-of-work challenge. Official client helpers solve the challenge and retry automatically. This is a mitigation for shared-bucket starvation, not a substitute for future Tor-circuit-aware accounting.

## Active ISO E2E mode

Build a test ISO with an active guest lifecycle hook:

```bash
EXTRA_BOOTAPPEND='deaddrop.e2e=1 deaddrop.e2e_url=http://<test-onion>' ./iso/build-iso.sh
```

The guest runs `deaddrop-client -self-test`, validates firewall state, triggers panic wipe, and powers off.
