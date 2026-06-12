# Server internals

## Entrypoint

`cmd/deaddrop-server` starts the backend. Defaults:

```text
listen: 127.0.0.1:8080
db:     /var/lib/deaddrop/drops.db
```

The server refuses non-loopback bind addresses.

## SQLite schema

```sql
CREATE TABLE drops (
 id TEXT PRIMARY KEY,
 hint TEXT NOT NULL,
 payload BLOB NOT NULL,
 created_at INTEGER NOT NULL,
 expires_at INTEGER NOT NULL
);
CREATE INDEX idx_drops_hint ON drops(hint);
CREATE INDEX idx_drops_expires_at ON drops(expires_at);
```

No plaintext, sender, recipient, or client IP columns exist.

## WAL mode

Storage initialization enables:

```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
```

## TTL expiry

The Go backend runs an in-process expiry loop every 15 minutes:

```sql
DELETE FROM drops WHERE expires_at < ?;
```

Cron does not perform TTL deletion; cron only does daily vacuum/checkpoint.

## Logging

Access logs contain:

```text
timestamp | method | path | status_code | response_time_ms | bytes_sent
```

Logs never include:

- client IP;
- hint;
- request body;
- payload;
- user-agent;
- plaintext.

## Rate limiting

`server/ratelimit.go` provides endpoint/host keyed buckets with pruning. `server/pow.go` provides proof-of-work fallback after bucket exhaustion.
