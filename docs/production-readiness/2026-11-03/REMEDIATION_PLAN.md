# DEADDROP-OS remediation plan

Applies to pinned `38b8e706304ea551d59b77fd6bc9f2ef7ed89bb9`; findings refer to [REVIEW.md](REVIEW.md). All actions below are **future work**, not executed. Use synthetic keys, hints and messages in a segregated lab; keep sensitive logs and captures controlled. Do not weaken existing gates to obtain a pass.

## Phases and dependencies

| Phase | Entry | Exit | Tasks |
|---|---|---|---|
| 0 Baseline | Owner approves isolated reproduction | Failure cause and expected checks recorded | T1 |
| 1 Correctness and security | T1 baseline; owner agrees response and backup policy | Failure-path defects fixed with negative tests | T2, T3 |
| 2 Meaningful automation | T1–T3 outputs | Real behavior and deliberate breakage both detected | T4 |
| 3 Image and operations | T3–T4; lab and recovery custodians available | Independent image, network and restore evidence collected | T5 |
| 4 Independent acceptance | T2–T5; owners available | Independent review signs evidence and remaining risks | T6 |

Dependency map: T1 → T2/T3 → T4 → T5 → T6; T3 must precede any backup drill, and T5 is also contingent on owner hardware and lab access. Do not release merely because individual task tests pass.

## T1 — Establish reproducible baseline

**Finding:** D04. **Prerequisite:** clean isolated checkout and pinned SHA. **Observed paths:** `.github/workflows/ci.yml`, `tools/check-go-version.sh`, `release/PRODUCTION_READINESS.md`.

1. Inspect the failed job logs and installed versus requested Go versions; record exact failure without assuming its cause.
2. In an approved clean Linux builder, reproduce the observed version check, then the workflow's existing `go test ./...`, `go vet ./...`, build and static/security steps when prerequisites are available; preserve separate outputs and skipped steps.
3. Correct a configuration/toolchain mismatch only if step 1 establishes it as the cause, selecting an owner-approved supported version. Otherwise repair the evidenced cause or leave the gate explicitly blocked with the missing evidence recorded. Rerun the full workflow at the candidate SHA without removing the check.

**Deliverable/verification:** baseline and candidate run links with SHAs, step logs and artifact inventory. **DoD:** [ ] cause reproduced or explicitly unresolved; [ ] every planned gate ran or is marked blocked; [ ] no skipped check is called a pass.

## T2 — Repair false acknowledgments

**Findings:** D02, D03. **Prerequisite:** T1 baseline. **Observed paths:** `server/server.go`, `server/storage.go`, `server/server_test.go`, `client/tui/main.go`, `client/tui/app.go`.

1. Define API behavior for deleted/nonexistent IDs versus database failure, and define valid submit response and bounded response size.
2. Propagate deletion storage errors without exposing identifiers; reject failed reads, malformed JSON and empty or invalid IDs before displaying submission success.
3. Add handler-level tests with an independently induced database failure and client transport tests for valid 201, malformed/truncated 201, empty ID and non-201 responses; exercise the TUI status boundary.

**Deliverable/verification:** patch and observable tests. Deliberately restore each ignored error in a disposable test branch: its associated negative test must fail. **DoD:** [ ] normal/nonexistent delete contract preserved; [ ] DB failure is not 204; [ ] malformed success cannot report submitted.

## T3 — Secure backup and failure cleanup

**Finding:** D01. **Prerequisite:** T1; owner chooses encryption recipient, permissions, retention and recovery access. **Observed path:** `deploy/scripts/backup-deaddrop.sh`; **observed runbook:** `runbooks/BACKUP_RESTORE.md` (contents not reviewed).

1. Threat-model SQLite consistency and onion-key confidentiality; select encrypted streaming or tightly permissioned temporary staging with explicit cleanup and filesystem limitations.
2. Implement fail-closed cleanup across backup, archive and encryption failures, and document that overwrite-based erasure is not guaranteed on SSD, snapshots or copy-on-write storage.
3. In a disposable lab, inject failures at each stage and inspect staging, file permissions and exit status; encrypt synthetic data, restore to a separate target and compare database contents and onion identity.

**Deliverable/verification:** hardened script, updated recovery procedure and drill record. **DoD:** [ ] plaintext staging is avoided or observable temporary files are cleaned up under every tested failure mode, with permissions verified; [ ] encrypted restore independently observed; [ ] SSD, snapshot and copy-on-write recoverability limits are documented, without treating file deletion as proof of secure erasure; [ ] exceptions and residual storage risks documented.

## T4 — Build security and interface regression gates

**Findings:** D05–D06; preventive dependency and UX requirements. **Prerequisite:** T1–T3. **Observed paths:** `client/firewall/firewall.sh`, `tests/qemu/run-qemu-security-tests.sh`, `.github/workflows/ci.yml`, `client/tui/app.go`; any new test path is proposed, not existing.

1. Specify boot/firewall fail-closed invariants, key-directory and memory-lock policy, plus expected terminal behavior for connectivity and failure messages.
2. Add independent tests for malformed responses, terminal keyboard navigation, small terminals and activity behavior; pin reviewed CI action/tool versions and define a repeatable dependency scan without presuming the current scan passed.
3. Strengthen capture validation to require readable, nonempty evidence and guest boot markers; run deliberate broken-rule, dropped-boot and invalid-capture controls in an isolated lab to prove each gate fails for the intended reason.

**Deliverable/verification:** test and CI changes with candidate run outputs and negative-control results. **DoD:** [ ] no mirror-only tests; [ ] failures trigger the intended gates; [ ] keyboard and assistive-technology findings are recorded, not assumed fixed.

## T5 — Image, network, hardware and recovery evidence

**Findings:** D05–D06 and operational acceptance. **Prerequisite:** T3–T4, separate clean Debian builders, isolated test onion, capture network and approved representative hardware. **Observed paths:** `iso/build-iso.sh`, `tests/qemu/run-qemu-security-tests.sh`, `release/PRODUCTION_READINESS.md`, `deploy/scripts/backup-deaddrop.sh`.

1. Build binaries and release ISO independently on two clean builders using the repository's documented process; compare image hashes, investigate mismatches and preserve provenance. An ISO build is not a boot test.
2. Boot the produced image in a VM; capture boot, Tor bootstrap and active synthetic submit/retrieve lifecycle. Observe DNS, IPv4, IPv6 and TCP egress during normal startup, Tor failure, firewall failure/restart and shutdown; test panic and tmpfs state against the agreed threat model. Confirm capture completeness and apply T4 negative controls.
3. Repeat applicable boot, networking and amnesia checks on approved representative physical hardware; conduct an isolated encrypted backup restore and onion-key recovery, time it, and record signed evidence and unresolved limitations.

**Deliverable/verification:** two build manifests, matching hashes or documented blocker, packet/guest evidence, hardware matrix and restore report. **DoD:** [ ] image reproducibility observed; [ ] VM and physical results separately recorded; [ ] recovery actually restored and verified; [ ] missing hardware or network capture remains a blocker, not a pass.

## T6 — Independent acceptance and release

**Findings:** all; owner decisions in review. **Prerequisite:** T2–T5 completed and release/security owners appointed. **Observed paths:** `release/PRODUCTION_READINESS.md`, `SECURITY.md`; release/signing procedures must be inspected before use.

1. Independent reviewers trace each claim to candidate SHA and evidence, inspect crypto, panic, systemd, auth/authorization and supply chain boundaries, and review negative controls.
2. Resolve or formally accept remaining risks, designate vulnerability-reporting contact, complete incident tabletop and approve onion-key custody and TLS-on-onion decision.
3. Release manager verifies signed artifacts and provenance, confirms rollback and tested recovery procedures, and applies the repository's production gate only after each human and technical gate is evidenced.

**Deliverable/verification:** signed acceptance record and release decision, with evidence links and explicit exceptions. **DoD:** [ ] independent reviewer approves scope and evidence; [ ] all blockers closed or release withheld; [ ] release SHA and artifacts match reviewed evidence.

## Acceptance, approvals and rollback

- [ ] CI candidate run completes; independent tests detect intentionally broken behavior.
- [ ] Security review covers ciphertext handling, key protection, hint/ID authorization assumptions, injection, dependency integrity and retention.
- [ ] ISO reproducibility, VM egress/panic checks, physical hardware checks and restore drill have distinct observed records.
- [ ] Owners approve release signing/custody, target platform, risk acceptance, incident response and recovery access before deployment or policy changes.

Rollback means stop distribution, preserve evidence, revoke or rotate compromised release/onion keys as appropriate, and restore a previously verified image and encrypted backup on an isolated target. Validate service and onion identity after restore; never assume a database backup alone contains the onion key. Changes to security invariants, firewall policy, key handling, credentials, target infrastructure or production release require explicit owner approval.
