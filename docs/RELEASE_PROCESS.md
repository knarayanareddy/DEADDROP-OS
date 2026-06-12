# Release process

## Release candidate check

```bash
./tools/release-check.sh
```

## Containerized builder

```bash
podman build -f Containerfile.release -t deaddrop-release .
podman run --rm -v "$PWD:/src" -w /src deaddrop-release ./tools/release-check.sh
```

ISO build requires privileges/live-build support:

```bash
podman run --rm --privileged -v "$PWD:/src" -w /src deaddrop-release ./iso/build-iso.sh
```

## Production gate

Production tag requires explicit closure of human gates:

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

The production gate also requires `artifacts/deaddrop-os.iso.sha256`.

## Artifact manifest

Update `release/MANIFEST.md` with final binary and ISO hashes.
