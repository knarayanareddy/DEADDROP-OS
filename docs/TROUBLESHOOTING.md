# Troubleshooting

## `DEADDROP-OS production builds require go1.25.11`

Install the exact pinned Go toolchain or use `Containerfile.release`.

## `server URL required`

Set `DEADDROP_SERVER` or pass `-server`:

```bash
deaddrop-client -server http://127.0.0.1:8080
```

## Client TUI fails in minimal terminal

Use CLI fallback:

```bash
deaddrop-client -cli -server http://127.0.0.1:8080
```

## `decryption failed`

This is intentionally generic. Possible causes:

- wrong private key;
- malformed payload;
- tampered ciphertext;
- wrong recipient public key used by sender.

## Server refuses to start on non-loopback

The backend intentionally refuses non-loopback listeners. Use Tor hidden service mapping to expose externally.

## 429 rate limited

Official clients automatically solve proof-of-work challenges. Manual clients must read `X-Deaddrop-Pow-Challenge`, solve, and retry with `X-Deaddrop-Pow`.

## Panic daemon wipes during active use

Confirm the activity socket exists and client can write to it:

```bash
ls -l /run/deaddrop/activity.sock
systemctl status deaddrop-panic
```

## Tor wait fails

Check:

```bash
journalctl -u tor
cat /run/tor/tor.log
```
