# Compliance matrix

| Spec area | Requirement | Implementation | Status |
|---|---|---|---|
| §5.2 TUI | Go TUI messenger | `client/tui/app.go`, `tview` | implemented |
| §7 Firewall | Tor-only default DROP | `client/firewall/firewall.sh` | implemented |
| §7 IPv6 | kernel/sysctl + ip6tables disable | `99-deaddrop-no-ipv6.conf`, firewall | implemented |
| §8 Crypto | RSA-OAEP + AES-256-GCM | `client/crypto` | implemented |
| Addendum B | base64url JSON envelope | `client/crypto.Payload` | implemented |
| §9 API | `/drop`, `/drops`, delete, health | `server/server.go` | implemented |
| §9 Rate limit | limit endpoints | `server/ratelimit.go`, `server/pow.go` | implemented with PoW fallback |
| §10 DB | SQLite WAL schema | `server/storage.go` | implemented |
| §10 TTL | hard delete expiry | backend loop | implemented |
| §11 Keys | client tmpfs session keys | default `/run/user/1000/deaddrop` | implemented, ISO must enforce tmpfs |
| §12 Logs | no IP/hint/payload/body | access logger + release smoke test | implemented |
| §14 Build | Go binaries + live-build | `tools/`, `iso/` | implemented scaffolding |
| §15 Unit/integration | crypto/API/panic tests | `*_test.go` | implemented |
| §15 QEMU | security invariant tests | `tests/qemu`, E2E guest hook | partially environment-dependent |
| §16 Deploy | Debian server deploy | `deploy/scripts/deploy.sh` | implemented |
| §1.3 Gates | human approvals | `tools/production-gate.sh` | blocked until humans close |
