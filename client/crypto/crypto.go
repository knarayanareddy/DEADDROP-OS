package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
)

const genericDecryptError = "decryption failed"

type Payload struct {
	Version    string `json:"version"`
	EncDEK     string `json:"enc_dek"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	Tag        string `json:"tag"`
}

func GenerateKeyPair(bits int) (*rsa.PrivateKey, error) {
	if bits == 0 {
		bits = 4096
	}
	return rsa.GenerateKey(rand.Reader, bits)
}

func Encrypt(pub *rsa.PublicKey, plaintext []byte) (string, error) {
	dek := make([]byte, 32)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return "", err
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	ct, tag := sealed[:len(sealed)-gcm.Overhead()], sealed[len(sealed)-gcm.Overhead():]
	encDEK, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, dek, nil)
	if err != nil {
		return "", err
	}
	p := Payload{Version: "1", EncDEK: b64(encDEK), Nonce: b64(nonce), Ciphertext: b64(ct), Tag: b64(tag)}
	j, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return b64(j), nil
}

func Decrypt(priv *rsa.PrivateKey, envelope string) ([]byte, error) {
	outer, err := unb64(envelope)
	if err != nil {
		return nil, errors.New(genericDecryptError)
	}
	var p Payload
	if err := json.Unmarshal(outer, &p); err != nil || p.Version != "1" {
		return nil, errors.New(genericDecryptError)
	}
	encDEK, err1 := unb64(p.EncDEK)
	nonce, err2 := unb64(p.Nonce)
	ct, err3 := unb64(p.Ciphertext)
	tag, err4 := unb64(p.Tag)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil, errors.New(genericDecryptError)
	}
	dek, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encDEK, nil)
	if err != nil {
		return nil, errors.New(genericDecryptError)
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, errors.New(genericDecryptError)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New(genericDecryptError)
	}
	sealed := append(append([]byte{}, ct...), tag...)
	pt, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, errors.New(genericDecryptError)
	}
	return pt, nil
}

func EncodePrivateKeyPEM(k *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
}
func EncodePublicKeyPEM(k *rsa.PublicKey) ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(k)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b}), nil
}
func ParsePrivateKeyPEM(b []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("invalid private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
func ParsePublicKeyPEM(b []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("invalid public key")
	}
	v, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := v.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pub, nil
}
func b64(b []byte) string            { return base64.RawURLEncoding.EncodeToString(b) }
func unb64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
