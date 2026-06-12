#!/bin/bash
# Final production-tag gate. This intentionally requires explicit human attestations.
set -euo pipefail
required=(
  DEADDROP_GATE_G1_SPEC_OWNER_SECURITY_LEAD
  DEADDROP_GATE_G2_ADR05_RESOLVED
  DEADDROP_GATE_G3_BACKUP_RESTORE_SIGNED
  DEADDROP_GATE_G4_IR_TABLETOP_DONE
  DEADDROP_GATE_G5_ONION_KEY_AUDITED
  DEADDROP_GATE_G6_ISO_HASH_PUBLISHED
  DEADDROP_QEMU_SECURITY_TESTS_PASS
)
missing=0
for v in "${required[@]}"; do
  if [ "${!v:-}" != "CLOSED" ]; then
    echo "BLOCKED: $v must be set to CLOSED" >&2
    missing=1
  fi
done
if [ ! -s artifacts/deaddrop-os.iso.sha256 ]; then
  echo "BLOCKED: artifacts/deaddrop-os.iso.sha256 is missing; run ./iso/build-iso.sh on Debian 12 live-build host" >&2
  missing=1
fi
if [ "$missing" -ne 0 ]; then
  echo "Production release is blocked by open human/ISO gates." >&2
  exit 1
fi
./tools/release-check.sh
echo "All production gates closed. Ready to tag."
