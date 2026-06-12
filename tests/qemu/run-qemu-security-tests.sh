#!/bin/bash
# QEMU packet-capture harness for ISO security invariants.
set -euo pipefail
ISO=${1:-artifacts/deaddrop-os.iso}
DURATION=${DURATION:-180}
PCAP=${PCAP:-artifacts/qemu-boot-capture.pcap}
[ -f "$ISO" ] || { echo "missing ISO: $ISO" >&2; exit 2; }
command -v qemu-system-x86_64 >/dev/null || { echo "qemu-system-x86_64 required" >&2; exit 2; }
command -v tcpdump >/dev/null || { echo "tcpdump required" >&2; exit 2; }
mkdir -p "$(dirname "$PCAP")"
set +e
timeout "$DURATION" qemu-system-x86_64 \
  -m 2048 -smp 2 -no-reboot -display none -serial mon:stdio \
  -cdrom "$ISO" \
  -netdev user,id=n1 \
  -device e1000,netdev=n1 \
  -object filter-dump,id=f1,netdev=n1,file="$PCAP"
QEMU_STATUS=$?
set -e
if [ "$QEMU_STATUS" -ne 124 ] && [ "$QEMU_STATUS" -ne 0 ]; then
  echo "qemu exited unexpectedly with status $QEMU_STATUS" >&2
  exit "$QEMU_STATUS"
fi
if tcpdump -nn -r "$PCAP" 'ip6' 2>/dev/null | grep -q .; then
  echo "FAIL: IPv6 packets observed" >&2
  exit 1
fi
if tcpdump -nn -r "$PCAP" 'udp and not (port 67 or port 68)' 2>/dev/null | grep -q .; then
  echo "FAIL: non-DHCP UDP packets observed" >&2
  exit 1
fi
echo "QEMU passive capture checks passed: $PCAP"
echo "For active client lifecycle testing, build an E2E ISO with:"
echo "  EXTRA_BOOTAPPEND='deaddrop.e2e=1 deaddrop.e2e_url=http://<test-onion>' ./iso/build-iso.sh"
echo "The guest then runs deaddrop-client -self-test and triggers panic wipe before shutdown."
