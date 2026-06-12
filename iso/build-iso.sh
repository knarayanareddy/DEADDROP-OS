#!/bin/bash
# Build DEADDROP-OS ISO on a clean Debian 12 host with live-build installed.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
WORK=${WORK:-$ROOT/iso-work}
ART=${ART:-$ROOT/artifacts}
mkdir -p "$WORK" "$ART"
cd "$WORK"
lb clean || true
lb config \
  --distribution bookworm \
  --architectures amd64 \
  --binary-images iso-hybrid \
  --bootappend-live "boot=live components noeject ipv6.disable=1 noswap ${EXTRA_BOOTAPPEND:-}" \
  --debian-installer false \
  --archive-areas "main"
mkdir -p config/package-lists config/includes.chroot/usr/local/bin config/includes.chroot/usr/local/lib/deaddrop \
  config/includes.chroot/etc/systemd/system config/includes.chroot/etc/sysctl.d \
  config/includes.chroot/etc/tor config/includes.chroot/etc/systemd/journald.conf.d config/hooks/normal
cp "$ROOT/iso/package-lists-deaddrop.list.chroot" config/package-lists/deaddrop.list.chroot
cp "$ART/deaddrop-client" config/includes.chroot/usr/local/bin/deaddrop-client
cp "$ART/deaddrop-panic" config/includes.chroot/usr/local/bin/deaddrop-panic
cp "$ROOT/deploy/scripts/wait-for-tor" config/includes.chroot/usr/local/bin/wait-for-tor
cp "$ROOT/deploy/scripts/deaddrop-e2e-guest" config/includes.chroot/usr/local/bin/deaddrop-e2e-guest
cp "$ROOT/client/firewall/firewall.sh" config/includes.chroot/usr/local/lib/deaddrop/firewall.sh
cp "$ROOT/client/firewall/99-deaddrop-no-ipv6.conf" config/includes.chroot/etc/sysctl.d/99-deaddrop-no-ipv6.conf
cp "$ROOT/deploy/tor/torrc.client" config/includes.chroot/etc/tor/torrc
cp "$ROOT/deploy/systemd/deaddrop-firewall.service" config/includes.chroot/etc/systemd/system/deaddrop-firewall.service
cp "$ROOT/deploy/systemd/deaddrop-tui.service" config/includes.chroot/etc/systemd/system/deaddrop-tui.service
cp "$ROOT/deploy/systemd/deaddrop-panic.service" config/includes.chroot/etc/systemd/system/deaddrop-panic.service
cp "$ROOT/deploy/systemd/deaddrop-e2e.service" config/includes.chroot/etc/systemd/system/deaddrop-e2e.service
cp "$ROOT/deploy/systemd/journald.conf.d-deaddrop.conf" config/includes.chroot/etc/systemd/journald.conf.d/deaddrop.conf
cp "$ROOT/iso/live-config/hooks/010-deaddrop.chroot" config/hooks/normal/010-deaddrop.chroot
chmod +x config/hooks/normal/010-deaddrop.chroot
lb build
ISO=$(ls -1 *.iso | head -1)
cp "$ISO" "$ART/deaddrop-os.iso"
(cd "$ART" && sha256sum deaddrop-os.iso > deaddrop-os.iso.sha256)
