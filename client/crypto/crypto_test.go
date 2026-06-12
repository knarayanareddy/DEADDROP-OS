package crypto

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	k, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	env, err := Encrypt(&k.PublicKey, []byte("secret message"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := Decrypt(k, env)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "secret message" {
		t.Fatalf("got %q", pt)
	}
}

func TestTamperDetection(t *testing.T) {
	k, _ := GenerateKeyPair(2048)
	env, err := Encrypt(&k.PublicKey, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	outer, _ := base64.RawURLEncoding.DecodeString(env)
	var p Payload
	if err := json.Unmarshal(outer, &p); err != nil {
		t.Fatal(err)
	}
	ct, _ := base64.RawURLEncoding.DecodeString(p.Ciphertext)
	ct[0] ^= 0xff
	p.Ciphertext = base64.RawURLEncoding.EncodeToString(ct)
	outer, _ = json.Marshal(p)
	env = base64.RawURLEncoding.EncodeToString(outer)
	if _, err := Decrypt(k, env); err == nil || err.Error() != genericDecryptError {
		t.Fatalf("expected generic decrypt error, got %v", err)
	}
}

func TestWrongKeyDecrypt(t *testing.T) {
	k1, _ := GenerateKeyPair(2048)
	k2, _ := GenerateKeyPair(2048)
	env, err := Encrypt(&k1.PublicKey, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(k2, env); err == nil || err.Error() != genericDecryptError {
		t.Fatalf("expected generic decrypt error, got %v", err)
	}
}

func TestNonceUniqueness(t *testing.T) {
	k, _ := GenerateKeyPair(2048)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		env, err := Encrypt(&k.PublicKey, []byte("x"))
		if err != nil {
			t.Fatal(err)
		}
		outer, _ := base64.RawURLEncoding.DecodeString(env)
		var p Payload
		_ = json.Unmarshal(outer, &p)
		if seen[p.Nonce] {
			t.Fatalf("nonce collision at %d", i)
		}
		seen[p.Nonce] = true
	}
}

func TestPEMEncodeParse(t *testing.T) {
	k, _ := GenerateKeyPair(2048)
	priv, err := ParsePrivateKeyPEM(EncodePrivateKeyPEM(k))
	if err != nil {
		t.Fatal(err)
	}
	pubPEM, err := EncodePublicKeyPEM(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePublicKeyPEM(pubPEM); err != nil {
		t.Fatal(err)
	}
}
