# HTTP API

The server is intended to be reachable only as an onion service. The Go backend listens only on `127.0.0.1:8080` by default and refuses non-loopback listeners.

## Common rules

- JSON request/response bodies.
- Hints must match `[A-Za-z0-9_-]{1,128}`.
- Hints are never logged.
- Payloads must be base64url-encoded Addendum B envelopes.
- Empty retrieval returns `{"drops":[]}` rather than `404`.
- Delete returns `204` even if the ID does not exist.

## POST `/drop`

Request:

```json
{
  "hint": "opaque_hint",
  "payload": "base64url-envelope"
}
```

Responses:

```http
201 Created
```

```json
{
  "id": "uuid-v4",
  "expires_at": "2026-06-15T19:08:38Z"
}
```

Errors:

```json
{"error":"invalid_payload","message":"invalid payload"}
```

## GET `/drops?hint=...`

Response:

```json
{
  "drops": [
    {
      "id": "uuid-v4",
      "payload": "base64url-envelope",
      "created_at": "2026-06-12T19:08:38Z",
      "expires_at": "2026-06-15T19:08:38Z"
    }
  ]
}
```

## DELETE `/drop/:id`

Response:

```http
204 No Content
```

## GET `/health`

Response:

```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

## Rate limiting and proof of work

The backend strips ephemeral ports from `RemoteAddr` and keys rate limits by endpoint and host. Because onion-service traffic appears as loopback, clients may share a bucket. When exhausted, server returns a stateless proof-of-work challenge:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
X-Deaddrop-Pow-Challenge: <challenge>
```

Official clients solve and retry with:

```http
X-Deaddrop-Pow: <solution>
```

The challenge is HMAC-bound to endpoint, expiry, and difficulty.
