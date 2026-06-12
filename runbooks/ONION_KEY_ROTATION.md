# Onion key rotation runbook

## When to rotate

- server compromise;
- suspected onion private key exposure;
- planned address migration;
- incident response directive.

## Consequence

Rotating the onion key changes the `.onion` address. Users must receive the new address out-of-band.

## Procedure

```bash
sudo systemctl stop deaddrop-server
sudo systemctl stop tor
sudo tar czf onion-key-old-$(date +%Y%m%d).tar.gz /var/lib/tor/deaddrop
# Encrypt archive immediately with operator key, then shred plaintext archive.
sudo rm -rf /var/lib/tor/deaddrop/private_key /var/lib/tor/deaddrop/hostname
sudo systemctl start tor
sleep 5
sudo cat /var/lib/tor/deaddrop/hostname
sudo systemctl start deaddrop-server
```

Verify health over the new onion address and distribute it out-of-band.
