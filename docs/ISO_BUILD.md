# ISO build

## Requirements

- clean Debian 12 host/container;
- `live-build`;
- built DEADDROP binaries in `artifacts/`.

## Build binaries

```bash
./tools/build-binaries.sh
```

## Build ISO

```bash
./iso/build-iso.sh
```

Outputs:

```text
artifacts/deaddrop-os.iso
artifacts/deaddrop-os.iso.sha256
```

## Active E2E ISO

```bash
EXTRA_BOOTAPPEND='deaddrop.e2e=1 deaddrop.e2e_url=http://<test-onion>' ./iso/build-iso.sh
```

When booted, the guest service `deaddrop-e2e.service` runs `deaddrop-e2e-guest`, which:

- waits for Tor;
- checks IPv6 disabled;
- checks firewall default DROP;
- runs optional client self-test if `deaddrop.e2e_url` is provided;
- triggers panic wipe;
- verifies temporary key removal;
- powers off.

## Reproducibility

A production release requires two independent clean builds producing the same ISO SHA-256. This cannot be satisfied by local source checks alone.
