# DEADDROP-OS release manifest

Version: 0.1.0
Spec: DEADDROP-OS Production Engineering Specification v2.0.0
Build toolchain: Go 1.25.11 linux/amd64
Build command: `./tools/release-check.sh`

## Binary artifacts

```text
ee77b939f9d872f22185676f93efc07f54730c160db74b7d01438f0da02dae28  deaddrop-client
3408f652893b0f855f4e4b3595fbbf37470bd982a54bef2095133079fd1be222  deaddrop-panic
194aa7ba29cf769f12c4ab12ce98f206c37c90b6298ff675c01b555d8810e029  deaddrop-server
```

## Verification performed in this workspace

- `go test ./...`: PASS
- `go vet ./...`: PASS
- static security invariant check: PASS
- `govulncheck ./...`: PASS / no vulnerabilities found
- release-style binary build: PASS
- backend health/drop/retrieve/delete/log-redaction smoke test: PASS

## Release blockers not closable by automation

The project is code-complete for an MVP release candidate, but a production tag still requires the human/process gates in `deaddroposdesigndoc.md` §1.3 and ISO/QEMU validation in §15.3.
