# Incident response runbook

## Severity levels

| Severity | Examples | Response |
|---|---|---|
| SEV-1 | server compromise, onion key leak, clearnet leak | immediate |
| SEV-2 | crypto bug, high/critical reachable CVE | <4h |
| SEV-3 | Tor/bootstrap failure, DB corruption | <24h |
| SEV-4 | docs/minor bugs | next sprint |

## SEV-1 server compromise

1. Isolate host from network.
2. Preserve evidence if safe.
3. Assume onion key compromised.
4. Notify users out-of-band: stop using onion address.
5. Provision clean server.
6. Deploy from verified build.
7. Generate new onion key/address.
8. Restore DB only from known-clean backup if acceptable.
9. Publish post-mortem.

## SEV-1 client clearnet leak

1. Halt release/distribution.
2. Quarantine artifact.
3. Identify firewall/Tor failure.
4. Patch and add regression test.
5. Re-run QEMU packet-capture tests.
6. Publish advisory if artifact was distributed.

## Critical dependency CVE

1. Run `govulncheck ./...`.
2. Determine reachability.
3. Patch dependency/toolchain.
4. Re-run release checks.
5. Release patched artifacts and hashes.
