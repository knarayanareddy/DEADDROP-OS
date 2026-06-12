# Cryptography

## Algorithms

| Purpose | Algorithm | Parameters |
|---|---|---|
| Content encryption | AES-GCM | AES-256, 96-bit nonce, 128-bit tag |
| DEK encryption | RSA-OAEP | RSA 4096-bit in normal sessions, SHA-256 |
| Randomness | CSPRNG | Go `crypto/rand` |
| Encoding | base64url | no padding |

## Payload schema

The inner payload is compact JSON:

```json
{
  "version": "1",
  "enc_dek": "base64url-rsa-oaep-dek",
  "nonce": "base64url-12-byte-nonce",
  "ciphertext": "base64url-aes-gcm-ciphertext",
  "tag": "base64url-aes-gcm-tag"
}
```

The compact JSON is then base64url-encoded and placed in the API `payload` field.

## Encrypt flow

1. Generate 32-byte DEK with `crypto/rand`.
2. Generate 12-byte nonce with `crypto/rand`.
3. AES-256-GCM encrypt plaintext.
4. Split ciphertext and 16-byte tag.
5. RSA-OAEP-SHA256 encrypt DEK with recipient public key.
6. Serialize JSON payload.
7. base64url encode outer envelope.

## Decrypt flow

1. base64url decode outer envelope.
2. Parse JSON and require `version == "1"`.
3. base64url decode fields.
4. RSA-OAEP-SHA256 decrypt DEK with local private key.
5. AES-GCM open ciphertext/tag.
6. Return plaintext or a generic error.

## Error opacity

All decryption failures return:

```text
decryption failed
```

This avoids distinguishing wrong key, corrupt payload, tampering, or malformed envelope.

## Key handling

Normal client key generation uses RSA-4096. Client self-test mode may use smaller temporary test keys for speed. Production user/session keys should remain RSA-4096.

Client keys are stored under the configured key directory, normally:

```text
/run/user/1000/deaddrop/private.pem
/run/user/1000/deaddrop/public.pem
```

This path must be tmpfs-backed in the live ISO.
