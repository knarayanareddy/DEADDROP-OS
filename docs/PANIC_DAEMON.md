# Panic daemon

## Purpose

The panic daemon wipes configured tmpfs session paths when triggered by signal or inactivity.

## Triggers

- `SIGUSR1` to daemon process/service;
- inactivity timeout;
- future hooks can feed the trigger channel.

## Activity reset

The daemon listens on a Unix datagram socket:

```text
/run/deaddrop/activity.sock
```

The client sends best-effort pings to reset the inactivity timer.

## Service unit

```ini
ExecStart=/usr/local/bin/deaddrop-panic \
  -paths /run/user/1000/deaddrop,/tmp/deaddrop \
  -inactivity 30m \
  -activity-socket /run/deaddrop/activity.sock
Restart=on-failure
```

`Restart=always` is intentionally prohibited because successful wipe should terminate the session rather than loop.

## Wipe behavior

For each configured path:

1. if directory: recursively wipe entries, then remove;
2. if file: overwrite with zero blocks, sync, close, remove.

This is best effort and assumes tmpfs. It is not a cold-boot defense.
