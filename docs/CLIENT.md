# Client

## Modes

`deaddrop-client` supports three modes:

1. **TUI mode** — default `tview` interface.
2. **CLI fallback** — `-cli`, useful for debugging/minimal terminals.
3. **Self-test** — `-self-test`, non-interactive submit/retrieve/decrypt lifecycle.

## TUI mode

```bash
DEADDROP_SERVER=http://example.onion deaddrop-client
```

Screens:

- compose encrypted drop;
- retrieve drops by hint;
- display session public key;
- status line for errors/results.

## CLI fallback

```bash
deaddrop-client -cli -server http://example.onion
```

Commands:

- `d` compose and submit;
- `r` retrieve and decrypt;
- `p` print public key;
- `q` quit.

## Self-test mode

```bash
deaddrop-client -self-test -server http://127.0.0.1:8080 -key-dir /tmp/dd-selftest
```

This mode generates a temporary keypair, submits an encrypted message, retrieves it, decrypts it, verifies content, and exits.

## Activity socket

The client sends best-effort activity pings to:

```text
/run/deaddrop/activity.sock
```

This resets the panic daemon inactivity timer. Failure to ping is non-fatal because the daemon may not be running in local development.

## Key directory

Default:

```text
/run/user/1000/deaddrop
```

Files:

```text
private.pem
public.pem
```

In the live ISO this must be tmpfs-backed.
