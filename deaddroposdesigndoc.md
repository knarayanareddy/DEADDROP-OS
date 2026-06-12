# DEADDROP-OS — Production Engineering Specification
### Single Source of Truth · v2.0.0 · Status: **APPROVED**

---

> **CANONICALITY RULE — READ FIRST**
> This document is the authoritative engineering specification for DEADDROP-OS.
> - Sections marked `[NORMATIVE]` define system contracts. Implementations MUST conform.
> - Sections marked `[ILLUSTRATIVE]` contain examples or reference implementations. If an illustrative block and a normative section conflict, the normative section wins.
> - If this document and code in the repository conflict, open a spec-deviation issue immediately. Do not assume code is correct. Do not assume the doc is correct. Resolve explicitly via the change control process in §1.5.
> - Addendum A (firewall ruleset) is canonical for network enforcement policy. If §7 (Network Enforcement) and Addendum A conflict, Addendum A wins until the next reconciliation date.
> - Addendum B (cryptographic wire format) is canonical for the drop payload schema. If §8 (Cryptographic Design) and Addendum B conflict, Addendum B wins.

---

## Table of Contents

```
§1   Document Governance
§2   Problem Statement
§3   Goals, Non-Goals & Constraints
§4   Success Metrics & Acceptance Criteria
§5   System Architecture
§6   Trust Model & Threat Boundaries
§7   Network Enforcement Model                     [NORMATIVE]
§8   Cryptographic Design                          [NORMATIVE]
§9   Interface Contracts                           [NORMATIVE]
§10  Data Model & Lifecycle                        [NORMATIVE]
§11  Key Lifecycle & Secrets Management            [NORMATIVE]
§12  Logging & Observability Policy                [NORMATIVE]
§13  Architecture Decision Records (ADRs)
§14  Platform Matrix & Build System
§15  Testing Strategy & Verification
§16  Operational Runbooks
§17  Dependency Registry
§18  Phased Rollout & Milestones
§19  Open Questions & Parking Lot

Addendum A  — Canonical Firewall Ruleset
Addendum B  — Canonical Cryptographic Wire Format
Addendum C  — Incident Response Playbooks
```

---

# §1 — Document Governance

## 1.1 Front Matter

| Field | Value |
|---|---|
| **Document title** | DEADDROP-OS Production Engineering Specification |
| **Version** | 2.0.0 |
| **Status** | APPROVED — Single Source of Truth |
| **Canonical location** | `main` branch · `/deaddroposdesigndoc.md` |
| **Spec owner** | Platform Lead (must be named before v1 prod release) |
| **Security reviewer** | Security Lead (must be named before v1 prod release) |
| **Last updated** | 2026-06-12 |
| **Next scheduled review** | 2026-09-12 (quarterly) |
| **Addendum A last reconciled** | 2026-06-12 |
| **Addendum B last reconciled** | 2026-06-12 |

---

## 1.2 Changelog

| Version | Date | Author | Summary |
|---|---|---|---|
| 2.0.0 | 2026-06-12 | Spec Owner | Full rewrite: added SSOT governance, ADRs, canonicality rules, expanded threat model, operational runbooks, incident response, backup/restore, error taxonomy, wire format addendum, firewall addendum |
| 1.0.0 | — | Spec Owner | Initial design document |

---

## 1.3 Pre-Production Human Action Gates

The following items MUST be resolved by a named human before any production deployment. This list is not advisory — it is a **release blocker**.

| # | Gate | Owner | Status |
|---|---|---|---|
| G-1 | Spec Owner and Security Lead named and recorded in §1.1 | Platform Lead | ⬜ OPEN |
| G-2 | ADR-05 (TLS-on-onion decision) formally resolved and recorded in §13 | Security Lead | ⬜ OPEN |
| G-3 | Backup/restore runbook (§16.4) tested end-to-end and signed off | Platform Lead | ⬜ OPEN |
| G-4 | Incident response playbook (Addendum C) reviewed and contacts populated | Security Lead | ⬜ OPEN |
| G-5 | Server onion key generation + storage procedure audited | Security Lead | ⬜ OPEN |
| G-6 | ISO build reproduced from clean environment and SHA-256 hash published | Platform Lead | ⬜ OPEN |

No CI/CD pipeline should gate-pass a production tag without confirming all gates above are CLOSED.

---

## 1.4 Reviewers & Approval

| Role | Name | Approved | Date |
|---|---|---|---|
| Platform Lead | TBD | ⬜ | — |
| Security Lead | TBD | ⬜ | — |
| Cryptography Reviewer | TBD | ⬜ | — |

---

## 1.5 Change Control Process

1. **Minor changes** (typo, clarification, non-normative addition): PR → 1 reviewer → merge. Bump patch version.
2. **Normative changes** (any `[NORMATIVE]` section, any ADR, any addendum): PR → Spec Owner + Security Lead review → explicit approval comment → merge. Bump minor version.
3. **Architectural changes** (changes to trust model, crypto protocol, boot invariants, data lifecycle): RFC doc → review period (minimum 5 business days) → both approvers sign off → merge. Bump major version.
4. After any addendum change, the reconciliation date in §1.1 MUST be updated in the same PR.
5. Stale addendum reconciliation dates (> 90 days) are treated as a **spec health incident**.

---

## 1.6 Scope of this Document

This document covers the full DEADDROP-OS system: the amnesic Debian Live client OS, the persistent onion-based server, the Go backend, the TUI messenger, the cryptographic protocol, operational runbooks, and build/CI pipeline. It does NOT cover:

- Downstream integrations or client applications wrapping the TUI
- Infrastructure beyond the single-server model described herein
- Post-MVP features listed in the parking lot (§19)

---

# §2 — Problem Statement

Whistleblowers, journalists, activists, and individuals operating under surveillance require a dead-drop communication channel with the following properties that no conventional tool guarantees simultaneously:

1. **Deniability** — the client device retains no forensic evidence of use after shutdown.
2. **Network anonymity** — all traffic is routed through Tor; no clearnet fallback is possible.
3. **Asynchronous delivery** — sender and recipient do not need to be online concurrently.
4. **End-to-end encryption** — the server operator cannot read message content.
5. **Minimal metadata exposure** — the server learns as little as possible about the parties.
6. **Verifiability** — the stated security properties are testable in CI, not assumed.

Existing tools fail at least one of these properties: Signal requires phone numbers, SecureDrop is complex to deploy, standard Tor Browser leaves artifacts, and commercial chat apps are clearnet-first.

DEADDROP-OS is a self-contained, bootable operating system that enforces all five properties at the OS level — not by convention, not by user discipline — by making violations technically impossible within the threat model defined in §6.

---

# §3 — Goals, Non-Goals & Constraints

## 3.1 Goals (In-Scope)

| ID | Goal | Notes |
|---|---|---|
| G-01 | Bootable Debian Live ISO (bookworm) with full tmpfs amnesia | No writes to physical media during session |
| G-02 | Tor transparent proxy enforced before any user-space network access | Firewall-level enforcement, not application-level |
| G-03 | IPv6 fully disabled at kernel and firewall level | IPv4-only; no dual-stack leaks |
| G-04 | TUI asynchronous messenger over onion service | Hint-based polling; no persistent connection required |
| G-05 | RSA-OAEP + AES-256-GCM hybrid encryption | Defined in Addendum B |
| G-06 | Go HTTP backend with SQLite TTL expiry | Onion-only; no clearnet listener |
| G-07 | Panic daemon with multi-trigger instant wipe | Power button, hotkey, inactivity |
| G-08 | Reproducible ISO build with published SHA-256 | CI artifact with hash |
| G-09 | Security properties expressed as automated CI checks | Not documentation assumptions |
| G-10 | Single-server deployment model | Described in §16 |

## 3.2 Non-Goals (Explicitly Out-of-Scope)

| ID | Non-Goal | Rationale |
|---|---|---|
| NG-01 | Persistence mode or optional "save session" | Contradicts amnesia guarantee |
| NG-02 | Clearnet fallback or split tunneling | Contradicts Tor-only guarantee |
| NG-03 | Perfect Forward Secrecy / Double Ratchet | ADR-01 — static RSA keys are explicit design choice |
| NG-04 | Auto key exchange or key agreement protocol | Out-of-band key exchange is intentional design |
| NG-05 | GUI / graphical desktop environment | TUI-only reduces attack surface |
| NG-06 | Cold boot RAM imaging attacks | Requires specialized hardware; out of threat model |
| NG-07 | Supply chain compromise of ISO | Requires reproducible build verification by user |
| NG-08 | Global traffic correlation by nation-state adversary | Tor does not protect against this |
| NG-09 | Multi-server federation or HA | Single-server model only in this spec |
| NG-10 | Mobile client | Desktop Linux only |

## 3.3 System Constraints

| ID | Constraint | Enforcement |
|---|---|---|
| C-01 | Firewall rules MUST be applied before networking starts | systemd unit ordering (§5.3) |
| C-02 | Tor bootstrap MUST complete before user-space network access | `ExecStartPre` gate on backend unit |
| C-03 | No data MUST ever be written to non-tmpfs paths during a session | auditd + CI check |
| C-04 | Server MUST NOT log client IP addresses | Logging policy §12.2 |
| C-05 | Server MUST NOT store plaintext message content | DB schema §10 + CI check |
| C-06 | Panic MUST complete wipe within 10 seconds of trigger | Acceptance criterion §4 |
| C-07 | ISO MUST boot on x86_64 hardware with ≥ 2 GB RAM | Platform matrix §14 |

---

# §4 — Success Metrics & Acceptance Criteria

## 4.1 Security Invariant Acceptance Criteria

These are binary pass/fail criteria. A build MUST NOT be released if any of these fail.

| ID | Criterion | Test Method |
|---|---|---|
| AC-01 | No TCP packet reaches clearnet during any session activity | `tcpdump` capture on physical NIC; zero non-Tor packets |
| AC-02 | No UDP packet (except DHCP pre-Tor) reaches any external host | Same capture |
| AC-03 | IPv6 is unreachable at kernel level | `ping6 ::1` returns `ENETUNREACH`; no `ip6` interface |
| AC-04 | DNS queries only resolve via Tor SOCKS / `DNSPort` | No UDP port 53 to external; `127.0.0.1:53` only |
| AC-05 | Panic wipe completes in ≤ 10 seconds from trigger | Timed automated test |
| AC-06 | Zero files written outside tmpfs during session | auditd rule covering all non-`/tmp`, non-`/run` writes |
| AC-07 | AES-GCM tamper detection: modified ciphertext fails decryption | Unit test with bit-flip |
| AC-08 | RSA-OAEP with wrong key fails decryption | Unit test |
| AC-09 | ISO SHA-256 matches published hash | CI artifact verification step |
| AC-10 | Server DB contains no plaintext message content | Schema check + integration test |

## 4.2 Performance Targets

| Metric | Target | Notes |
|---|---|---|
| Drop submission (LAN to onion) | < 3 seconds P95 | Tor latency included |
| Drop retrieval (single drop) | < 2 seconds P95 | |
| Panic wipe trigger to completion | ≤ 10 seconds | Hard requirement |
| ISO boot to TUI ready | < 90 seconds | On reference hardware |
| SQLite TTL expiry maintenance job | < 5 seconds for 10,000 rows | |

## 4.3 Operational Acceptance Criteria

| ID | Criterion |
|---|---|
| OA-01 | Server deploys from `deploy.sh` on clean Debian 12 with zero manual steps |
| OA-02 | ISO build produces identical hash on two independent clean builds (reproducibility) |
| OA-03 | Backup restore (§16.4) recovers drops not yet expired with no data loss |
| OA-04 | Incident response playbook (Addendum C) completed successfully in tabletop exercise |

---

# §5 — System Architecture

## 5.1 Component Overview (C4 Level 1)

```
┌─────────────────────────────────────────────────────────────────┐
│                        DEADDROP-OS SYSTEM                       │
│                                                                 │
│  ┌──────────────────────────────────────────┐                   │
│  │           CLIENT (Amnesic ISO)           │                   │
│  │                                          │                   │
│  │  ┌────────────┐   ┌──────────────────┐  │                   │
│  │  │  TUI       │   │  Crypto Layer    │  │  ◄── RAM only     │
│  │  │  Messenger │──►│  RSA-OAEP/AES   │  │      No disk      │
│  │  └────────────┘   └──────────────────┘  │      writes       │
│  │        │                                │                   │
│  │  ┌─────▼──────┐   ┌──────────────────┐  │                   │
│  │  │  Panic     │   │  iptables/nft    │  │                   │
│  │  │  Daemon    │   │  Tor Enforcement │  │                   │
│  │  └────────────┘   └──────────────────┘  │                   │
│  └──────────────────┬───────────────────────┘                   │
│                     │ Tor (onion transport)                     │
│  ┌──────────────────▼───────────────────────┐                   │
│  │           SERVER (Persistent)            │                   │
│  │                                          │                   │
│  │  ┌──────────────┐   ┌────────────────┐  │                   │
│  │  │  Go HTTP     │   │  SQLite DB     │  │                   │
│  │  │  Backend     │──►│  (ciphertext   │  │                   │
│  │  │  (onion only)│   │   only)        │  │                   │
│  │  └──────────────┘   └────────────────┘  │                   │
│  │  ┌──────────────┐                       │                   │
│  │  │  Tor (onion  │                       │                   │
│  │  │  service)    │                       │                   │
│  │  └──────────────┘                       │                   │
│  └──────────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

## 5.2 Component Responsibilities

| Component | Responsibility | Canonical location |
|---|---|---|
| **TUI Messenger** | User interaction, drop compose/retrieve, hint input | `client/tui/` |
| **Crypto Layer** | RSA-OAEP DEK encryption, AES-256-GCM payload encryption/decryption, key loading | `client/crypto/` |
| **Panic Daemon** | Monitor triggers (power button, hotkey, inactivity timer), execute wipe | `client/panic/` |
| **Firewall Enforcement** | iptables rules, default-drop policy, Tor redirect, IPv6 kill | Addendum A; `client/firewall/` |
| **Go Backend** | HTTP API (onion-only), drop storage/retrieval, TTL enforcement | `server/` |
| **SQLite DB** | Ciphertext storage, TTL expiry, indexed by hint | `server/data/drops.db` |
| **Server Tor** | Onion service hidden service listener | `/etc/tor/torrc` |
| **Build System** | ISO generation, CI, reproducibility | `build/` |

## 5.3 Boot Sequence & Ordering Invariants `[NORMATIVE]`

The following ordering MUST be enforced via systemd unit dependencies. Any deviation is a security regression.

```
Stage 0  BIOS/UEFI → Bootloader (GRUB)
Stage 1  Kernel init → tmpfs root mount
Stage 2  ┌─────────────────────────────────────────────┐
         │  deaddrop-firewall.service                  │
         │  [MUST complete before any other networking]│
         │  DefaultDependencies=no                     │
         │  Before=network-pre.target                  │
         └──────────────────┬──────────────────────────┘
                            │
Stage 3  ┌─────────────────▼──────────────────────────┐
         │  tor.service                                │
         │  After=deaddrop-firewall.service            │
         │  ExecStartPost → wait for bootstrap 100%   │
         └──────────────────┬──────────────────────────┘
                            │
Stage 4  ┌─────────────────▼──────────────────────────┐
         │  deaddrop-backend.service (server only)     │
         │  After=tor.service                          │
         └──────────────────┬──────────────────────────┘
                            │
Stage 5  ┌─────────────────▼──────────────────────────┐
         │  deaddrop-tui.service (client only)         │
         │  After=tor.service                          │
         │  ExecStartPre=/usr/local/bin/wait-for-tor   │
         └─────────────────────────────────────────────┘

Parallel:  deaddrop-panic.service
           After=basic.target
           (always running, independent of network)
```

**Invariant I-01**: If `deaddrop-firewall.service` fails, the system MUST NOT proceed to networking. `network-pre.target` MUST NOT be reached.

**Invariant I-02**: If Tor bootstrap does not complete within 120 seconds, the TUI MUST NOT start. The system MUST present an error and halt network access.

**Invariant I-03**: The panic daemon MUST be running within 10 seconds of `basic.target` regardless of any other service state.

---

# §6 — Trust Model & Threat Boundaries

## 6.1 Trust Zones

| Zone | Components | Trust Level | Rationale |
|---|---|---|---|
| **Zone 0: Trusted RAM** | All client-side data during session (keys, plaintext, session state) | Fully trusted | Exists only in volatile RAM; wiped on poweroff/panic |
| **Zone 1: Semi-trusted Server** | Go backend, SQLite DB, server Tor | Partially trusted — sees ciphertext + metadata only | Operator is assumed honest but potentially compromised. Protocol must not require trusting operator. |
| **Zone 2: Tor Network** | Relays, exit nodes (N/A — onion only) | Untrusted for confidentiality | Provides anonymization but not content security |
| **Zone 3: External / Adversary** | Physical hardware, BIOS, NIC, upstream ISP, nation-state | Untrusted | DEADDROP-OS provides no protection against these in defined out-of-scope scenarios |

## 6.2 Threat Model — In-Scope

| ID | Threat | Mitigation | Residual Risk |
|---|---|---|---|
| T-01 | Post-session forensic recovery from client device | tmpfs-only storage; panic wipe; no swap | Low — only if panic fails |
| T-02 | Network traffic analysis revealing communication | All traffic via Tor; no clearnet; DNS via Tor | Low within Tor anonymity set |
| T-03 | Server operator reading message content | AES-256-GCM encryption; server stores ciphertext only | None — operator cannot decrypt without recipient private key |
| T-04 | Server operator identifying sender/recipient | No identity stored; hint-based addressing; no client IPs logged | Low — timing/size metadata remains |
| T-05 | Ciphertext tampering by server | AES-GCM authentication tag detects modification | None — any tamper fails MAC |
| T-06 | Drop enumeration by unauthenticated actor | Hint-based retrieval; no list endpoint; rate limiting | Low |
| T-07 | Clearnet leak via misconfigured network stack | Firewall default-drop; iptables redirect; Tor transparent proxy | None if C-01/C-02 invariants hold |
| T-08 | IPv6 leak | IPv6 disabled at kernel + firewall level | None |
| T-09 | DNS leak | DNS redirected to Tor `DNSPort`; no UDP port 53 allowed | None |
| T-10 | Key material left in swap | Swap disabled; tmpfs only; `mlockall` in critical processes | Low |

## 6.3 Threat Model — Explicitly Out-of-Scope

| ID | Threat | Reason Out-of-Scope |
|---|---|---|
| OT-01 | Cold boot RAM attack (cryogenic DIMM imaging) | Requires physical access + specialized hardware; outside operational threat model |
| OT-02 | Compromised ISO supply chain | User responsibility to verify SHA-256 hash against published value |
| OT-03 | Global traffic correlation (nation-state Tor deanonymization) | Tor provides no protection; fundamental Tor limitation |
| OT-04 | Compromised BIOS/UEFI/firmware | Out of scope; requires trusted hardware |
| OT-05 | Shoulder surfing / physical observation of screen | Physical security is user responsibility |
| OT-06 | Compromised recipient private key | Key management is user responsibility; DEADDROP-OS has no key custody |
| OT-07 | Rubber hose / coercion | Out of scope |

## 6.4 What the Server Learns `[NORMATIVE]`

The server operator, even with full access to server storage, logs, and memory, learns ONLY:

- The existence of ciphertext blobs with associated hints (opaque tokens)
- The approximate size of each drop
- The timestamp of submission and retrieval
- The number of drops associated with each hint

The server NEVER learns:
- The plaintext content of any drop
- The identity of the sender or recipient
- The IP address of any client
- The private keys of any party
- The relationship between any two hints

---

# §7 — Network Enforcement Model `[NORMATIVE]`

> **Canonical source**: Addendum A. This section is a normative summary. If any discrepancy exists between §7 and Addendum A, Addendum A wins. Update reconciliation date in §1.1 after any change.

## 7.1 Core Policy

```
Default policy: DROP on INPUT, OUTPUT, FORWARD
```

No traffic flows unless explicitly permitted by the rules below. This is a **whitelist**, not a blacklist.

## 7.2 Permitted OUTPUT Rules

| Priority | Protocol | Destination | Port | Purpose |
|---|---|---|---|---|
| 1 | TCP | 127.0.0.1 | 9040 | Tor transparent proxy redirect |
| 2 | UDP | 127.0.0.1 | 5353 | Tor DNS port |
| 3 | TCP | 127.0.0.1 | 9050 | Tor SOCKS proxy (explicit) |
| 4 | TCP | any | 9001, 9030 | Tor relay connections |

All other OUTPUT is DROP.

## 7.3 Redirect Rules

```
All TCP OUTPUT not destined for 127.0.0.0/8 → redirect to 127.0.0.1:9040
All UDP port 53 → redirect to 127.0.0.1:5353
```

## 7.4 Hard Blocks

| Protocol | Action | Rationale |
|---|---|---|
| UDP (except DNS redirect above) | DROP | No UDP to clearnet; prevents WebRTC/NTP leaks |
| ICMP | DROP | Prevents traceroute/ping leaks |
| All IPv6 (ip6tables) | DROP all chains | IPv6 disabled at kernel; defense in depth |
| All FORWARD | DROP | No routing; client is not a router |

## 7.5 Tor Configuration Requirements `[NORMATIVE]`

```
TransPort 9040
DNSPort 5353
AutomapHostsOnResolve 1
VirtualAddrNetworkIPv4 10.192.0.0/10
DataDirectory /run/tor          # tmpfs
Log notice file /run/tor/tor.log  # tmpfs
```

`DataDirectory` and log path MUST be on tmpfs. Any configuration pointing to non-volatile paths is a security misconfiguration.

## 7.6 IPv6 Kernel Disable `[NORMATIVE]`

```
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
net.ipv6.conf.lo.disable_ipv6 = 1
```

These MUST be set via `/etc/sysctl.d/99-deaddrop-no-ipv6.conf`, applied before networking starts.

---

# §8 — Cryptographic Design `[NORMATIVE]`

> **Canonical source**: Addendum B. This section is a normative summary. If §8 and Addendum B conflict, Addendum B wins.

## 8.1 Algorithm Selection

| Purpose | Algorithm | Parameters | Rationale |
|---|---|---|---|
| DEK encryption | RSA-OAEP | 4096-bit key, SHA-256 hash | Asymmetric; recipient's public key encrypts DEK |
| Payload encryption | AES-256-GCM | 256-bit key, 96-bit nonce, 128-bit tag | Authenticated encryption; tamper detection |
| Key size | RSA 4096-bit | — | Conservative; PFS explicitly out-of-scope (ADR-01) |
| Nonce generation | CSPRNG | 96-bit | Go `crypto/rand`; never reused |

**I-04 (Nonce Uniqueness Invariant)**: AES-GCM nonces MUST be generated from `crypto/rand` and MUST NOT be reused under the same key. A random 96-bit nonce provides sufficient collision resistance for the message volume in scope.

## 8.2 Encryption Flow `[NORMATIVE]`

```
ENCRYPT (sender):
  1. Generate DEK: 32 random bytes via crypto/rand
  2. Generate Nonce: 12 random bytes via crypto/rand
  3. AES-256-GCM encrypt plaintext with DEK + Nonce → ciphertext + auth_tag
  4. RSA-OAEP encrypt DEK with recipient public key → encrypted_dek
  5. Serialize drop payload per Addendum B schema
  6. Base64url-encode full payload
  7. POST to /drop endpoint

DECRYPT (recipient):
  1. Retrieve drop payload (base64url-decode)
  2. RSA-OAEP decrypt encrypted_dek with recipient private key → DEK
  3. If step 2 fails: abort; display "decryption failed" (do NOT expose error detail)
  4. AES-256-GCM decrypt ciphertext with DEK + Nonce
  5. If auth tag verification fails: abort; display "message tampered or corrupted"
  6. Return plaintext
```

**I-05 (Error Opacity Invariant)**: Decryption failure messages MUST NOT distinguish between "wrong key," "tampered ciphertext," or "corrupt payload." All failure modes MUST return the same generic error string to prevent oracle attacks.

## 8.3 Key Storage `[NORMATIVE]`

| Key | Location | Persistence |
|---|---|---|
| Client RSA private key | `$HOME/.config/deaddrop/private.pem` on tmpfs | Session only; wiped on shutdown/panic |
| Client RSA public key | `$HOME/.config/deaddrop/public.pem` on tmpfs | Session only |
| Server onion private key | `/var/lib/tor/deaddrop/` on persistent disk | Persistent; loss = new onion address |
| DEK (per-drop) | In-memory only during encryption/decryption | Never written to disk or logged |
| AES Nonce | Embedded in Addendum B payload | Ciphertext only; not secret |

---

# §9 — Interface Contracts `[NORMATIVE]`

## 9.1 API Overview

All endpoints are served over the onion address only. There is no clearnet listener. Authentication is not required (onion transport provides anonymization). Rate limiting is applied per §9.6.

**Base URL**: `http://<onion-address>.onion/`

## 9.2 Endpoint: POST /drop

**Purpose**: Submit an encrypted drop.

**Request**

```
POST /drop
Content-Type: application/json

{
  "hint":    string,   // opaque token; max 128 chars; see §9.5
  "payload": string    // base64url-encoded Addendum B wire format
}
```

**Response — Success**

```
HTTP 201 Created
Content-Type: application/json

{
  "id":         string,  // server-assigned UUID
  "expires_at": string   // ISO 8601 UTC timestamp
}
```

**Response — Error**

```
HTTP 400  { "error": "invalid_payload", "message": "..." }
HTTP 429  { "error": "rate_limited",    "message": "retry after X seconds" }
HTTP 500  { "error": "server_error",    "message": "internal error" }
```

## 9.3 Endpoint: GET /drops

**Purpose**: Retrieve all unexpired drops for a hint.

**Request**

```
GET /drops?hint=<hint>
```

**Response — Success**

```
HTTP 200 OK
Content-Type: application/json

{
  "drops": [
    {
      "id":         string,
      "payload":    string,   // base64url-encoded Addendum B
      "created_at": string,   // ISO 8601 UTC
      "expires_at": string    // ISO 8601 UTC
    }
  ]
}
```

Empty `drops` array is returned (not 404) when no drops exist for the hint (prevents hint enumeration inference).

## 9.4 Endpoint: DELETE /drop/:id

**Purpose**: Explicit drop deletion by recipient after retrieval.

**Request**

```
DELETE /drop/:id
```

**Response**

```
HTTP 204 No Content   (success or if ID does not exist — same response to prevent enumeration)
HTTP 429              (rate limited)
```

## 9.5 Endpoint: GET /health

**Purpose**: Liveness check. Returns no sensitive information.

**Request**

```
GET /health
```

**Response**

```
HTTP 200 OK
Content-Type: application/json

{
  "status": "ok",
  "version": string   // semver of backend binary
}
```

## 9.5 Hint Specification `[NORMATIVE]`

- A hint is an opaque, pre-shared token agreed out-of-band between sender and recipient.
- Maximum length: 128 characters.
- Allowed characters: `[A-Za-z0-9_\-]` (URL-safe alphanumeric + underscore + hyphen).
- The server MUST NOT interpret hint structure. It is an opaque string.
- Hints MUST NOT be logged (§12.2).
- There is no hint enumeration endpoint. The server MUST NOT expose a list of active hints.

## 9.6 Rate Limiting `[NORMATIVE]`

| Endpoint | Limit | Window | Scope |
|---|---|---|---|
| POST /drop | 10 requests | 60 seconds | Per Tor circuit / connection |
| GET /drops | 30 requests | 60 seconds | Per Tor circuit / connection |
| DELETE /drop/:id | 20 requests | 60 seconds | Per Tor circuit / connection |
| GET /health | 60 requests | 60 seconds | Global |

On limit exceeded: HTTP 429 with `Retry-After` header (seconds).

Note: "per Tor circuit" means per distinct TCP connection origin. Because all clients use Tor, IP-based rate limiting is NOT used (would block all clients sharing an exit). Use connection-level or token-bucket per connection.

## 9.7 API Versioning Policy

- Current API version: `v1` (implicit; no version prefix required for MVP).
- When a breaking change is required, a versioned prefix (`/v2/`) will be introduced.
- The old version MUST remain available for 90 days after the new version is deployed.
- This spec MUST be updated with a compatibility matrix before any breaking change ships.

---

# §10 — Data Model & Lifecycle `[NORMATIVE]`

## 10.1 Database

- Engine: SQLite 3
- Location: `/var/lib/deaddrop/drops.db` (server persistent disk)
- WAL mode: ENABLED (`PRAGMA journal_mode=WAL`)
- No remote DB; no replication (single-server model)

## 10.2 Schema

```sql
-- [NORMATIVE] drops table
CREATE TABLE drops (
  id          TEXT    PRIMARY KEY,         -- UUID v4
  hint        TEXT    NOT NULL,            -- opaque token; indexed
  payload     BLOB    NOT NULL,            -- base64url Addendum B ciphertext
  created_at  INTEGER NOT NULL,            -- Unix timestamp (seconds)
  expires_at  INTEGER NOT NULL             -- Unix timestamp (seconds); TTL-driven
);

CREATE INDEX idx_drops_hint       ON drops(hint);
CREATE INDEX idx_drops_expires_at ON drops(expires_at);

-- No plaintext content column exists or will exist in this schema.
-- The payload column stores ONLY the serialized encrypted wire format.
```

**Schema Invariant I-06**: No additional column containing plaintext, sender identity, recipient identity, or client IP address SHALL be added to this table without a major spec version bump and Security Lead sign-off.

## 10.3 TTL & Retention Policy `[NORMATIVE]`

| Parameter | Value | Notes |
|---|---|---|
| Default TTL | 72 hours | From `created_at` |
| Maximum TTL | 168 hours (7 days) | Hard limit; server enforces |
| Expiry check interval | Every 15 minutes | Via cron or Go ticker |
| Expiry action | Hard DELETE | Row removed; not soft-deleted |
| Vacuum interval | Daily | `PRAGMA wal_checkpoint(TRUNCATE)` + `VACUUM` |

## 10.4 Expiry Maintenance Job

```go
// [ILLUSTRATIVE] — reference implementation
func expireDrops(db *sql.DB) error {
    _, err := db.Exec(
        `DELETE FROM drops WHERE expires_at < ?`,
        time.Now().Unix(),
    )
    return err
}
```

This job MUST be a separate goroutine (or cron entry) and MUST NOT block request handling.

## 10.5 Data Classification

| Data Element | Classification | Location | Never |
|---|---|---|---|
| Drop ciphertext payload | Sensitive (encrypted at rest) | SQLite `payload` column | Never in logs |
| Hint | Sensitive (opaque identifier) | SQLite `hint` column | Never in logs |
| Drop ID | Non-sensitive | SQLite `id` column | — |
| Timestamps | Non-sensitive | SQLite columns | — |
| Plaintext message content | FORBIDDEN in DB | Does not exist in schema | Never stored |
| Client IP addresses | FORBIDDEN in logs and DB | Does not exist | Never stored |
| Private keys | FORBIDDEN outside tmpfs | RAM only (client) | Never on disk (client) |

## 10.6 Right to Deletion

- Any drop can be explicitly deleted via `DELETE /drop/:id`.
- All drops are hard-deleted on TTL expiry.
- There is no backup that preserves expired drops post-vacuum (see §16.4).
- There is no "soft delete" or tombstone. Deletion is permanent.

---

# §11 — Key Lifecycle & Secrets Management `[NORMATIVE]`

## 11.1 Client Key Lifecycle

```
Session start →  Key generation (if no key imported)
                 OR Key import (paste/pipe from out-of-band channel)
                 │
                 │  Keys stored in tmpfs: /run/user/<uid>/deaddrop/
                 │
Session active → Keys accessible to TUI and crypto layer only
                 mlockall() called to prevent swap (if kernel permits)
                 │
Shutdown/panic → tmpfs wiped by kernel on poweroff
                 Panic daemon: explicit shred of key file paths before sync
```

## 11.2 Server Key Lifecycle

| Key | Path | Persistence | Rotation |
|---|---|---|---|
| Tor onion private key | `/var/lib/tor/deaddrop/` | Permanent (loss = address change) | Manual; documented in §16.5 |
| No other server-side keys exist | — | — | — |

**I-07 (Server Key Invariant)**: The server does NOT hold any encryption keys for drop contents. The server onion key is solely for Tor hidden service identity. Compromise of the server onion key results in a new onion address, not in decryption of stored drops.

## 11.3 What Is Never on Disk (Client) `[NORMATIVE]`

The following MUST NEVER be written to any non-tmpfs path on the client:

- RSA private key
- RSA public key
- Any DEK (data encryption key)
- Any AES-GCM nonce
- Any plaintext message content
- Any decrypted drop payload

Violation of this invariant is a critical security bug requiring immediate incident response (Addendum C, Severity 1).

---

# §12 — Logging & Observability Policy `[NORMATIVE]`

## 12.1 Client Logging Policy

| Parameter | Value |
|---|---|
| Log destination | systemd journal (volatile, tmpfs-backed) only |
| Log persistence across reboot | NONE — volatile journal |
| Log location | `/run/log/journal/` |
| journald storage setting | `Storage=volatile` in `/etc/systemd/journald.conf` |
| Log level default | `info` |
| Log level for crypto operations | `error` only — no debug crypto logging in production |

**What is NEVER logged on client (hard prohibition)**:

| Prohibited | Rationale |
|---|---|
| Plaintext message content | Privacy |
| RSA private key material | Security |
| DEK (data encryption key) | Security |
| AES nonce + ciphertext pair | Security |
| Typed keystrokes / input buffer | Privacy |
| Decrypted drop contents | Privacy |
| Recipient hint values | Privacy |

## 12.2 Server Logging Policy `[NORMATIVE]`

Server access logs MUST contain:

```
timestamp | method | path | status_code | response_time_ms | bytes_sent
```

Server logs MUST NEVER contain:

| Prohibited | Rationale |
|---|---|
| Client IP address | No logging of any `RemoteAddr` |
| Hint values | Privacy (even hashed hints are forbidden) |
| Payload contents | Security |
| User-Agent string | Fingerprinting |
| Request body content | Security |

## 12.3 Server Log Retention

| Log type | Retention | Notes |
|---|---|---|
| Access log | 7 days | Rotated daily; hard delete after 7 days |
| Error log | 30 days | Security events retained longer |
| DB maintenance log | 30 days | Expiry job run records only |

## 12.4 Metrics (Operational)

The following metrics SHOULD be collected for operational visibility. None of these metrics may be disaggregated by hint or by connection identity.

| Metric | Type | Labels |
|---|---|---|
| `deaddrop_drops_stored_total` | Counter | — |
| `deaddrop_drops_expired_total` | Counter | — |
| `deaddrop_drops_deleted_total` | Counter | — |
| `deaddrop_api_requests_total` | Counter | `endpoint`, `status_code` |
| `deaddrop_api_latency_seconds` | Histogram | `endpoint` |
| `deaddrop_db_size_bytes` | Gauge | — |
| `deaddrop_panic_trigger_total` | Counter | `trigger_type` |

---

# §13 — Architecture Decision Records (ADRs)

Each ADR is immutable once approved. To change a decision, supersede the ADR with a new one.

---

## ADR-01: Static RSA Keys — No Perfect Forward Secrecy

**Status**: ACCEPTED
**Date**: 2026-06-12

**Context**: Asynchronous dead-drop communication requires the sender to encrypt without the recipient being online. This makes session-based key agreement (e.g., Double Ratchet, ECDH) incompatible with the asynchronous delivery model.

**Decision**: Use static RSA-4096 public keys distributed out-of-band. No forward secrecy is implemented.

**Consequences**:
- (+) Fully asynchronous operation; sender needs no online coordination.
- (-) Compromise of a recipient's private key compromises all past drops that have not been deleted.
- (-) No ratcheting; no break-in recovery.

**Mitigations**:
- Drops are TTL-limited (max 7 days); reduce exposure window.
- Client private key exists only in tmpfs; wiped on session end.
- Users should rotate key pairs periodically and exchange new public keys out-of-band.

**Future path**: If synchronous "secure channel" mode is ever added, evaluate Signal Protocol. That is a separate feature, not an extension of this spec.

---

## ADR-02: Onion-Only Server — No TLS Layer

**Status**: ACCEPTED — REQUIRES ANNUAL REVIEW
**Date**: 2026-06-12
**Next review**: 2027-06-12

**Context**: Tor onion services provide end-to-end authenticated encryption between the Tor client and the hidden service. This renders TLS redundant for confidentiality and origin authentication within the onion circuit.

**Decision**: The Go HTTP server listens on the Tor onion address without TLS. No HTTPS is configured.

**Consequences**:
- (+) Simpler deployment; no certificate management; no certificate transparency exposure.
- (+) Onion transport provides stronger anonymization than TLS alone.
- (-) No defense-in-depth if Tor onion encryption is somehow bypassed (considered out-of-scope).
- (-) Some HTTP clients may warn about non-HTTPS connections.

**Mitigations**:
- The server MUST NOT listen on any non-loopback interface (only `127.0.0.1:PORT`; Tor handles external exposure).
- If a future Tor vulnerability bypasses onion encryption, TLS-on-onion can be added without breaking the API contract.

**Dissenting view**: Adding TLS-on-onion provides defense-in-depth at low operational cost. This decision should be revisited before any public production release.

---

## ADR-03: SQLite Over PostgreSQL

**Status**: ACCEPTED
**Date**: 2026-06-12

**Context**: The server model is explicitly single-server, single-instance. No replication, clustering, or multi-tenant access pattern is required.

**Decision**: SQLite in WAL mode for the drops database.

**Consequences**:
- (+) Zero-dependency database; embedded in Go binary; simpler deployment and backup.
- (+) Excellent read performance for the read-heavy retrieval pattern.
- (-) Not horizontally scalable; single-writer constraint.
- (-) If scaling is ever needed, migration to PostgreSQL requires schema migration and deployment changes.

**Trigger for revisiting**: > 10,000 concurrent hints or > 100 writes/second sustained.

---

## ADR-04: Hint-Based Addressing — No Identity System

**Status**: ACCEPTED
**Date**: 2026-06-12

**Context**: A dead-drop system should not require user accounts, usernames, or any identity tokens that could link communications to identities.

**Decision**: Addressing is entirely hint-based. Hints are opaque pre-shared tokens. No user registration, authentication, or identity is required or stored.

**Consequences**:
- (+) Zero identity metadata in the system.
- (+) No account compromise vector.
- (-) Out-of-band hint coordination is required between parties.
- (-) No way to notify recipient of new drop without polling or out-of-band signal.

---

## ADR-05: TLS-on-Onion Defense-in-Depth

**Status**: OPEN — REQUIRES RESOLUTION BEFORE PROD (Gate G-2)
**Date**: 2026-06-12

**Context**: ADR-02 defers TLS. This ADR is the formal resolution checkpoint.

**Question**: Should we add optional TLS-on-onion before initial production release for defense-in-depth?

**Options**:
- A: Ship without TLS as per ADR-02. Document explicitly.
- B: Add self-signed TLS; configure Go server with TLS; accept cert pinned in client.
- C: Add TLS using a long-lived self-generated CA; distribute CA cert as part of ISO build.

**Owner**: Security Lead (Gate G-2). Must resolve and update this ADR before production tag.

---

## ADR-06: No Clearnet Fallback — Ever

**Status**: ACCEPTED
**Date**: 2026-06-12

**Context**: It might seem user-friendly to allow clearnet access if Tor is unavailable.

**Decision**: Clearnet fallback is permanently prohibited. If Tor is unavailable, the system presents an error. No traffic reaches the internet via any path other than Tor.

**Consequences**:
- (+) Absolute guarantee of anonymization for all network traffic.
- (-) System is unusable if Tor is blocked or unavailable.

**Mitigation**: Users in Tor-blocking environments should use Tor bridges (future feature; see §19).

---

# §14 — Platform Matrix & Build System

## 14.1 Supported Platforms

| Platform | Status | Notes |
|---|---|---|
| x86_64 (AMD64) bare metal | SUPPORTED | Primary target |
| x86_64 virtual machine (QEMU/KVM) | SUPPORTED | CI testing target |
| x86_64 virtual machine (VirtualBox) | SUPPORTED | User testing |
| ARM64 (aarch64) | NOT SUPPORTED | Future consideration; §19 |
| 32-bit x86 | NOT SUPPORTED | No plan |
| UEFI boot | SUPPORTED | Recommended |
| Legacy BIOS boot | SUPPORTED | Maintained |
| Secure Boot | NOT SUPPORTED | Future consideration; §19 |

## 14.2 Hardware Requirements

| Component | Minimum | Recommended |
|---|---|---|
| RAM | 2 GB | 4 GB |
| CPU | x86_64, 1 core | 2+ cores |
| Storage | None (RAM only) | USB 3.0 boot drive |
| Network | 1 NIC | 1 NIC |
| Display | 80×24 terminal | 120×40 terminal |

## 14.3 Base OS

- Debian 12 (bookworm) live
- Kernel: default Debian bookworm kernel (tracking LTS)
- No custom kernel patches required

## 14.4 Build Pipeline `[NORMATIVE]`

```
Build inputs:
  - deaddropos.conf (live-build config)
  - hooks/ (customization scripts)
  - server/ (Go backend source)
  - client/ (TUI source)

Build steps:
  1. lb config (live-build configuration)
  2. Go build: GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w"
  3. lb build (ISO assembly)
  4. sha256sum deaddrop-os.iso > deaddrop-os.iso.sha256
  5. Publish artifact + hash

Build environment:
  - Clean Debian 12 container (no host state)
  - Pinned package versions (apt-pin or snapshot)
  - No network access during build (all packages pre-fetched)
```

**Reproducibility requirement**: Two independent clean builds from the same commit MUST produce identical SHA-256 hashes. Failure is a build system bug, not acceptable variance.

## 14.5 CI Checks (All MUST pass for merge to main)

| Check | Description |
|---|---|
| `test:unit` | Go unit tests (crypto, API handlers, expiry logic) |
| `test:integration` | Backend integration tests (DB, API, TTL) |
| `test:security` | Security invariant checks (firewall, IPv6, DNS) — see §15.3 |
| `build:iso` | Full ISO build |
| `verify:hash` | ISO hash matches previous build (if not a hash-bumping PR) |
| `lint:go` | `golangci-lint` with security linters enabled |
| `scan:deps` | Dependency vulnerability scan |

---

# §15 — Testing Strategy & Verification

## 15.1 Testing Pyramid

```
         /\
        /E2E\         ISO boot + full session in QEMU
       /------\
      / Integ  \      Backend API + DB + TTL + concurrent drops
     /----------\
    /    Unit    \    Crypto (encrypt/decrypt/tamper), handlers, panic logic
   /--------------\
```

## 15.2 Unit Tests

| Test | What It Verifies |
|---|---|
| `TestEncryptDecryptRoundTrip` | Full encrypt → decrypt produces original plaintext |
| `TestTamperDetection` | Bit-flip in ciphertext fails AES-GCM auth tag |
| `TestWrongKeyDecrypt` | Decryption with wrong RSA key returns generic error |
| `TestNonceUniqueness` | 10,000 encryptions produce zero nonce collisions |
| `TestDropTTLExpiry` | Drops past TTL are removed by maintenance job |
| `TestPanicTrigger` | Panic daemon responds within 1 second of each trigger type |
| `TestRateLimiter` | Requests exceeding limit return 429 |

## 15.3 Security Invariant Tests

These tests run in a QEMU boot environment and MUST pass for any release build.

| Test | Command | Pass Condition |
|---|---|---|
| `test:no-clearnet-tcp` | `tcpdump -i eth0 'tcp and not host 127.0.0.1'` during active session | Zero packets captured |
| `test:no-udp-external` | `tcpdump -i eth0 udp` (excluding port 53 redirect) | Zero packets |
| `test:ipv6-disabled` | `ping6 ::1` | `ENETUNREACH` returned |
| `test:dns-via-tor` | `strace -e trace=network curl http://example.onion` | No port 53 UDP to external |
| `test:no-disk-writes` | `auditd` rule monitoring non-tmpfs write syscalls | Zero audit events |
| `test:panic-speed` | Trigger panic; measure time to tmpfs wipe | ≤ 10 seconds |

## 15.4 Integration Tests

| Test | What It Verifies |
|---|---|
| `TestDropSubmitRetrieve` | POST /drop → GET /drops returns same payload |
| `TestDropDelete` | DELETE /drop/:id → GET /drops returns empty |
| `TestExpiredDropNotReturned` | Drop past TTL not returned by GET /drops |
| `TestHintIsolation` | Drop for hint A not visible under hint B |
| `TestConcurrentDrops` | 100 concurrent POST /drop; all persist correctly |
| `TestRateLimitIntegration` | Burst past limit; verify 429s; verify recovery after window |
| `TestHealthEndpoint` | /health returns 200 with version string |

## 15.5 End-to-End Tests

Executed against a full QEMU boot of the ISO + a running server instance:

1. Boot ISO → Tor connects → TUI opens
2. Generate RSA key pair
3. Submit encrypted drop via TUI
4. Retrieve and decrypt drop via TUI with correct key
5. Verify decryption fails with wrong key
6. Verify drop is deleted after explicit delete
7. Verify drop disappears after TTL expiry
8. Trigger panic; verify session state wiped; verify reboot shows no artifacts

---

# §16 — Operational Runbooks

## 16.1 Server Initial Deployment

**Prerequisite**: Clean Debian 12 (bookworm) server. Root or sudo access. Gate G-1 through G-6 closed.

```bash
# 1. Update system
apt-get update && apt-get upgrade -y

# 2. Install dependencies
apt-get install -y tor sqlite3 curl ufw

# 3. Create service user
useradd -r -s /usr/sbin/nologin deaddrop

# 4. Deploy backend binary
cp deaddrop-server /usr/local/bin/deaddrop-server
chmod 750 /usr/local/bin/deaddrop-server
chown deaddrop:deaddrop /usr/local/bin/deaddrop-server

# 5. Create data directory
mkdir -p /var/lib/deaddrop
chown deaddrop:deaddrop /var/lib/deaddrop
chmod 700 /var/lib/deaddrop

# 6. Configure Tor hidden service
cat >> /etc/tor/torrc << EOF
HiddenServiceDir /var/lib/tor/deaddrop/
HiddenServicePort 80 127.0.0.1:8080
EOF

# 7. Reload Tor (generates onion address)
systemctl reload tor
sleep 5

# 8. Retrieve onion address (RECORD THIS — see Gate G-5)
cat /var/lib/tor/deaddrop/hostname

# 9. Install and start systemd unit
cp deaddrop-server.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable deaddrop-server
systemctl start deaddrop-server

# 10. Verify
curl --socks5-hostname 127.0.0.1:9050 http://$(cat /var/lib/tor/deaddrop/hostname)/health
```

Expected: `{"status":"ok","version":"<semver>"}`

---

## 16.2 Server Update / Deployment

```bash
# 1. Build new binary
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o deaddrop-server ./server/

# 2. Copy to server (via SCP over Tor, or secure channel)
scp deaddrop-server user@server:/tmp/deaddrop-server-new

# 3. On server: swap binary with zero-downtime window
systemctl stop deaddrop-server
cp /tmp/deaddrop-server-new /usr/local/bin/deaddrop-server
chmod 750 /usr/local/bin/deaddrop-server
systemctl start deaddrop-server

# 4. Verify
curl --socks5-hostname 127.0.0.1:9050 \
  http://$(cat /var/lib/tor/deaddrop/hostname)/health

# 5. Rollback if health check fails (see §16.3)
```

**Downtime window**: < 10 seconds (stop → start).
**Drops**: Not lost; SQLite DB is preserved across restart.

---

## 16.3 Rollback Procedure

```bash
# If deploy fails or health check returns non-200:

# 1. Stop new binary
systemctl stop deaddrop-server

# 2. Restore previous binary from backup
cp /usr/local/bin/deaddrop-server.backup /usr/local/bin/deaddrop-server
chmod 750 /usr/local/bin/deaddrop-server

# 3. Restart
systemctl start deaddrop-server

# 4. Verify
curl --socks5-hostname 127.0.0.1:9050 \
  http://$(cat /var/lib/tor/deaddrop/hostname)/health
```

**Pre-deploy step (required)**: Before every deploy, copy the current binary to `.backup`:
```bash
cp /usr/local/bin/deaddrop-server /usr/local/bin/deaddrop-server.backup
```

---

## 16.4 Backup & Restore

### Backup

The only persistent server state requiring backup is:
1. The SQLite database (`/var/lib/deaddrop/drops.db`)
2. The Tor onion key (`/var/lib/tor/deaddrop/`)

```bash
#!/bin/bash
# backup-deaddrop.sh — run daily via cron
# Encrypted backup using recipient's public key

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="/var/backups/deaddrop"
DB_FILE="/var/lib/deaddrop/drops.db"
ONION_KEY_DIR="/var/lib/tor/deaddrop"

mkdir -p "$BACKUP_DIR"

# 1. SQLite online backup (safe with WAL mode; no lock required)
sqlite3 "$DB_FILE" ".backup $BACKUP_DIR/drops-$TIMESTAMP.db"

# 2. Tar onion key directory
tar czf "$BACKUP_DIR/onion-key-$TIMESTAMP.tar.gz" "$ONION_KEY_DIR"

# 3. Encrypt backup (operator's public key)
gpg --recipient <OPERATOR_KEY_FINGERPRINT> \
    --encrypt "$BACKUP_DIR/drops-$TIMESTAMP.db"
gpg --recipient <OPERATOR_KEY_FINGERPRINT> \
    --encrypt "$BACKUP_DIR/onion-key-$TIMESTAMP.tar.gz"

# 4. Remove unencrypted copies
shred -u "$BACKUP_DIR/drops-$TIMESTAMP.db"
shred -u "$BACKUP_DIR/onion-key-$TIMESTAMP.tar.gz"

# 5. Prune backups older than 14 days
find "$BACKUP_DIR" -name "*.gpg" -mtime +14 -delete
```

**Important**: Backup contains ciphertext only (not plaintext). Even a compromised backup cannot be decrypted without recipient private keys.

### Restore

```bash
# Restore drops DB from backup

# 1. Decrypt backup
gpg --decrypt drops-<TIMESTAMP>.db.gpg > drops-restore.db

# 2. Stop server
systemctl stop deaddrop-server

# 3. Replace DB
cp /var/lib/deaddrop/drops.db /var/lib/deaddrop/drops.db.pre-restore
cp drops-restore.db /var/lib/deaddrop/drops.db
chown deaddrop:deaddrop /var/lib/deaddrop/drops.db
chmod 600 /var/lib/deaddrop/drops.db

# 4. Restart server
systemctl start deaddrop-server

# 5. Verify
curl --socks5-hostname 127.0.0.1:9050 \
  http://$(cat /var/lib/tor/deaddrop/hostname)/health

# 6. Confirm drops present (if known hint available)
# NOTE: Drops that had expired BEFORE backup will not be restored.
# NOTE: Drops that expired BETWEEN backup and restore will be cleaned on next expiry run.
```

**Restore semantics**: Restoring resurrects all drops that had not yet expired at backup time. Drops that expired between backup and restore will be cleaned by the next TTL maintenance job. This is expected and acceptable behavior.

---

## 16.5 Onion Key Rotation

**When to rotate**: Server compromise, planned address change, or security incident.

**Consequence**: Rotating the onion key changes the `.onion` address. All clients must be given the new address out-of-band. Any drops stored under the old address are unreachable (they exist in the DB but clients won't know to query the new address).

```bash
# 1. Notify all known recipients of upcoming address change (out-of-band)
# 2. Wait for existing drops to expire (up to 7 days) OR accept data loss

# 3. Stop server
systemctl stop deaddrop-server

# 4. Backup old onion key (encrypted)
tar czf old-onion-key-$(date +%Y%m%d).tar.gz /var/lib/tor/deaddrop/
gpg --encrypt --recipient <FINGERPRINT> old-onion-key-$(date +%Y%m%d).tar.gz
shred -u old-onion-key-$(date +%Y%m%d).tar.gz

# 5. Remove old onion key (forces new key generation)
rm -rf /var/lib/tor/deaddrop/private_key
rm -rf /var/lib/tor/deaddrop/hostname

# 6. Restart Tor (generates new key)
systemctl restart tor
sleep 5

# 7. Record new onion address
cat /var/lib/tor/deaddrop/hostname
# → Distribute this address to all parties out-of-band

# 8. Restart server
systemctl start deaddrop-server
```

---

## 16.6 DB Maintenance

Runs automatically via cron. Documented here for manual execution if needed.

```bash
# Manual TTL expiry + vacuum
sqlite3 /var/lib/deaddrop/drops.db \
  "DELETE FROM drops WHERE expires_at < strftime('%s','now');"
sqlite3 /var/lib/deaddrop/drops.db "PRAGMA wal_checkpoint(TRUNCATE);"
sqlite3 /var/lib/deaddrop/drops.db "VACUUM;"
```

Cron entry (`/etc/cron.d/deaddrop`):
```
*/15 * * * * deaddrop /usr/local/bin/deaddrop-expire >> /var/log/deaddrop/expire.log 2>&1
0    3 * * * deaddrop /usr/local/bin/deaddrop-vacuum >> /var/log/deaddrop/vacuum.log 2>&1
```

---

# §17 — Dependency Registry

All dependencies must be pinned to exact versions. Version bumps require a PR with the update rationale.

## 17.1 Server (Go)

| Package | Version | Purpose | License |
|---|---|---|---|
| `github.com/mattn/go-sqlite3` | v1.14.x (pin) | SQLite driver | MIT |
| `github.com/google/uuid` | v1.x (pin) | UUID generation | BSD-3 |
| Standard library (`net/http`, `crypto/rsa`, `crypto/aes`) | Go stdlib | HTTP server, crypto | BSD |

## 17.2 Client (Go TUI)

| Package | Version | Purpose | License |
|---|---|---|---|
| `github.com/rivo/tview` OR `github.com/gdamore/tcell` | pin | TUI rendering | Apache 2.0 / Apache 2.0 |
| Standard library | Go stdlib | Crypto, networking | BSD |

## 17.3 System Packages (Debian Bookworm)

| Package | Purpose | Notes |
|---|---|---|
| `tor` | Tor daemon | Pin to Debian bookworm version |
| `iptables` | Firewall enforcement | Or `nftables`; ADR required if switching |
| `sqlite3` | DB CLI tool (server) | Runtime dependency |
| `live-build` | ISO build toolchain | Pin version |
| `golang` | Build toolchain | Pin to 1.21+ |

## 17.4 Dependency Audit Policy

- Automated vulnerability scan on every CI run (`govulncheck` or `osv-scanner`).
- Any HIGH or CRITICAL CVE in a direct dependency is a release blocker.
- Any HIGH or CRITICAL CVE in a transitive dependency is reviewed within 72 hours.
- Dependency updates are reviewed by Security Lead before merge to main.

---

# §18 — Phased Rollout & Milestones

## Phase 0 — Foundation (Current / Pre-Release)

| Deliverable | Status |
|---|---|
| Core crypto library (RSA-OAEP + AES-GCM) | ✅ |
| Go HTTP backend + SQLite | ✅ |
| Onion service configuration | ✅ |
| TUI messenger (basic) | ✅ |
| Firewall enforcement service | ✅ |
| Panic daemon | ✅ |
| ISO build pipeline | ✅ |
| Security invariant CI checks | ⬜ |
| Reproducible build verification | ⬜ |
| All pre-prod gates (§1.3) closed | ⬜ |

## Phase 1 — Production Hardening

| Deliverable | Notes |
|---|---|
| Full test suite (§15) passing in CI | Security invariant tests in QEMU |
| ADR-05 resolved (TLS decision) | Gate G-2 |
| Backup/restore runbook tested E2E | Gate G-3 |
| Incident response tabletop | Gate G-4 |
| Onion key procedure audited | Gate G-5 |
| Published SHA-256 for ISO v1.0 | Gate G-6 |
| Spec Owner + Security Lead named | Gate G-1 |

**Phase 1 exit criterion**: All gates G-1 through G-6 closed. All AC-01 through AC-10 passing.

## Phase 2 — Operational Maturity

| Deliverable | Notes |
|---|---|
| Metrics dashboard (Grafana or equivalent) | §12.4 metrics |
| Automated backup verification (restore test weekly) | Cron job |
| Quarterly spec review process active | §1.1 |
| Formal threat model review post-deployment | External review |

## Phase 3 — Post-MVP (Requires New Spec Addendum)

See §19 for items in the parking lot.

---

# §19 — Open Questions & Parking Lot

## 19.1 Open Questions (Must Resolve Before Phase 1)

| ID | Question | Owner | Due |
|---|---|---|---|
| OQ-01 | ADR-05: TLS-on-onion — yes or no? | Security Lead | Gate G-2 |
| OQ-02 | What is the rate-limit implementation? Token bucket per connection, or external middleware? | Platform Lead | Phase 1 |
| OQ-03 | Onion key backup storage location — same server (encrypted)? Offline? | Security Lead | Gate G-5 |
| OQ-04 | What is the notification mechanism for recipients? Polling-only, or optional out-of-band push? | Platform Lead | Phase 1 |

## 19.2 Parking Lot (Future Spec Items — Do Not Implement Without RFC)

| ID | Item | Rationale for Deferral |
|---|---|---|
| PL-01 | Tor bridge support for censored environments | Nice to have; adds complexity; separate UX flow |
| PL-02 | ARM64 / Raspberry Pi boot support | Hardware testing environment needed |
| PL-03 | Secure Boot (UEFI signed bootloader) | Key signing infrastructure needed |
| PL-04 | Optional Perfect Forward Secrecy mode (synchronous session) | Fundamentally changes the async architecture; new spec |
| PL-05 | Multi-server federation / HA | Changes the single-server model assumption; new ADR required |
| PL-06 | Steganographic drop payloads | Research-level; defer |
| PL-07 | GUI alternative to TUI | Increases attack surface; low priority |
| PL-08 | iOS / Android client | Requires separate OS security model; distinct spec |
| PL-09 | I2P support (alternative to Tor) | Different anonymization tradeoffs; ADR required |

---

# Addendum A — Canonical Firewall Ruleset

> **This addendum is the normative source for all firewall policy.**
> Last reconciled with §7: 2026-06-12
> If this addendum and §7 conflict, this addendum wins.
> Reconciliation date MUST be updated when either is changed.

```bash
#!/bin/bash
# /usr/local/lib/deaddrop/firewall.sh
# CANONICAL DEADDROP-OS FIREWALL RULESET v2.0.0
# DO NOT MODIFY WITHOUT SPEC VERSION BUMP + SECURITY LEAD SIGN-OFF

set -euo pipefail

TOR_UID=$(id -u debian-tor 2>/dev/null || id -u tor)
TRANS_PORT=9040
DNS_PORT=5353

# --- FLUSH ALL EXISTING RULES ---
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X

# --- IPv6: HARD DISABLE ---
ip6tables -F
ip6tables -X
ip6tables -P INPUT DROP
ip6tables -P OUTPUT DROP
ip6tables -P FORWARD DROP

# --- DEFAULT POLICIES: DROP EVERYTHING ---
iptables -P INPUT DROP
iptables -P OUTPUT DROP
iptables -P FORWARD DROP

# --- LOOPBACK ---
iptables -A INPUT  -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# --- ESTABLISHED CONNECTIONS ---
iptables -A INPUT  -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# --- TOR TRANSPARENT PROXY (NAT TABLE) ---
# Redirect all TCP output (except Tor daemon itself) to TransPort
iptables -t nat -A OUTPUT \
  ! -o lo \
  -m owner ! --uid-owner $TOR_UID \
  -p tcp \
  -j REDIRECT --to-ports $TRANS_PORT

# Redirect DNS UDP to Tor DNSPort
iptables -t nat -A OUTPUT \
  ! -o lo \
  -m owner ! --uid-owner $TOR_UID \
  -p udp --dport 53 \
  -j REDIRECT --to-ports $DNS_PORT

# --- ALLOW TOR DAEMON OUTPUT ---
iptables -A OUTPUT -m owner --uid-owner $TOR_UID -j ACCEPT

# --- BLOCK ALL OTHER UDP (no clearnet UDP leaks) ---
iptables -A OUTPUT \
  ! -o lo \
  -p udp \
  -j DROP

# --- BLOCK ICMP (no ping/traceroute leaks) ---
iptables -A OUTPUT -p icmp -j DROP

# --- LOG DROPPED PACKETS (for CI verification; volatile log) ---
iptables -A OUTPUT -j LOG --log-prefix "DEADDROP-DROP: " --log-level 4

echo "DEADDROP-OS firewall rules applied successfully."
echo "Default policy: DROP. Tor transparent proxy: ACTIVE."
```

---

# Addendum B — Canonical Cryptographic Wire Format

> **This addendum is the normative source for all payload serialization.**
> Last reconciled with §8: 2026-06-12
> If this addendum and §8 conflict, this addendum wins.

## B.1 Drop Payload Schema (JSON, then Base64url-encoded)

```json
{
  "version":       "1",
  "enc_dek":       "<base64url: RSA-OAEP encrypted 32-byte DEK>",
  "nonce":         "<base64url: 12-byte random AES-GCM nonce>",
  "ciphertext":    "<base64url: AES-256-GCM ciphertext>",
  "tag":           "<base64url: 16-byte AES-GCM authentication tag>"
}
```

## B.2 Field Specifications

| Field | Type | Encoding | Length | Notes |
|---|---|---|---|---|
| `version` | string | literal `"1"` | — | Bumped on format change; old versions supported for 90 days |
| `enc_dek` | bytes | base64url (no padding) | RSA-4096 output = 512 bytes → 684 base64 chars | RSA-OAEP-SHA256 encrypted DEK |
| `nonce` | bytes | base64url (no padding) | 12 bytes → 16 base64 chars | Must be unique per encryption |
| `ciphertext` | bytes | base64url (no padding) | variable | AES-256-GCM ciphertext (same length as plaintext) |
| `tag` | bytes | base64url (no padding) | 16 bytes → 22 base64 chars | AES-GCM authentication tag |

## B.3 Full Wire Format (Outer Envelope)

The JSON object above is serialized to compact JSON (no extra whitespace), then base64url-encoded (no padding), and the resulting string is the value of the `payload` field in the API request/response.

```
OUTER ENVELOPE:
  POST /drop body:
  {
    "hint":    "<opaque string>",
    "payload": "<base64url(compact_json(B.1 object))>"
  }
```

## B.4 Version Migration Policy

When a new wire format version is needed:
1. Increment `version` field value.
2. Define new format in a new Addendum B revision.
3. Server MUST accept both old and new versions for 90 days post-migration.
4. Client MUST emit new version from migration date.
5. After 90 days, server MAY reject old version.

---

# Addendum C — Incident Response Playbooks

> Pre-production gate G-4: This addendum MUST be reviewed, contacts populated, and a tabletop exercise completed before production release.

## C.1 Severity Definitions

| Severity | Definition | Response Time |
|---|---|---|
| SEV-1 | Active compromise: server breached, onion key leaked, clearnet leak confirmed | Immediate (< 1 hour) |
| SEV-2 | Security regression in build: CI checks failing, new crypto vulnerability in dep | < 4 hours |
| SEV-3 | Service degraded: high latency, DB corruption, Tor bootstrap failing | < 24 hours |
| SEV-4 | Non-urgent: docs out of date, minor bug, performance regression | Next sprint |

## C.2 Contacts

| Role | Name | Contact | Backup |
|---|---|---|---|
| Incident Commander | TBD | TBD | TBD |
| Security Lead | TBD | TBD | TBD |
| Platform Lead | TBD | TBD | TBD |

*Contacts MUST be populated before production. This is Gate G-4.*

## C.3 SEV-1 Playbook: Server Compromise

```
T+0:00  DETECT — alert fires OR anomaly noticed
T+0:15  CONTAIN — isolate server from network (firewall all inbound/outbound)
T+0:30  ASSESS — determine scope: was onion key exfiltrated? DB copied?
T+1:00  NOTIFY — alert all known users via out-of-band channel:
                  "Address compromised; do not use; new address forthcoming"
T+2:00  REMEDIATE:
         1. Provision new clean server
         2. Deploy from clean build (verify ISO hash)
         3. Generate new onion key (new address)
         4. Restore DB from last clean backup IF backup was pre-compromise
            OTHERWISE: start fresh DB (accept data loss)
T+4:00  RECOVER — distribute new onion address to users out-of-band
T+24:00 POST-MORTEM — document root cause, timeline, and remediation
```

## C.4 SEV-1 Playbook: Clearnet Leak Confirmed on Client

```
T+0:00  DETECT — CI test AC-01/AC-02 fails OR user reports clearnet traffic
T+0:15  HALT — do not release ISO build; quarantine artifact
T+0:30  IDENTIFY — determine which firewall rule failed or was bypassed
T+1:00  FIX — patch Addendum A firewall ruleset; increment spec patch version
T+2:00  VERIFY — re-run full security invariant test suite (§15.3)
T+3:00  RE-RELEASE — only after all AC-01 through AC-10 pass
T+24:00 POST-MORTEM — if build was ever distributed, publish advisory
```

## C.5 SEV-2 Playbook: Critical CVE in Dependency

```
T+0:00  DETECT — govulncheck / osv-scanner reports HIGH or CRITICAL CVE
T+1:00  ASSESS — determine if CVE is exploitable in DEADDROP-OS usage context
T+4:00  PATCH — update dependency version; run full test suite
T+8:00  RELEASE — if tests pass, release patched build with new ISO hash
T+24:00 DISCLOSE — if shipped versions are affected, publish advisory
```

## C.6 Post-Mortem Template

```
# Incident Post-Mortem

Date:
Severity:
Duration:
Incident Commander:

## Summary
[2-3 sentence summary of what happened]

## Timeline
[Chronological events]

## Root Cause
[Technical root cause]

## Impact
[What was affected; any data or anonymity implications]

## Remediation
[What was done to fix it]

## Prevention
[What spec / code / process changes prevent recurrence]

## Action Items
[Specific tasks with owners and due dates]
```

---

*End of DEADDROP-OS Production Engineering Specification v2.0.0*

*To propose changes: follow §1.5 change control. To report a spec deviation: open an issue tagged `spec-deviation`. To report a security issue: contact Security Lead directly via out-of-band channel.*
