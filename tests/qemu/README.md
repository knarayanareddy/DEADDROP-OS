# QEMU security invariant tests

These tests require a built `artifacts/deaddrop-os.iso`, QEMU/KVM, root packet capture permissions, and an isolated test network.

They correspond to design-doc §15.3 / AC-01 through AC-06:

- no clearnet TCP packets
- no external UDP packets
- IPv6 disabled
- DNS via Tor only
- no non-tmpfs disk writes
- panic wipe within 10 seconds

The `run-qemu-security-tests.sh` script is intentionally conservative and exits if prerequisites are absent.
