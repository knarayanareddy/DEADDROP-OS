package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	powChallengeHeader = "X-Deaddrop-Pow-Challenge"
	powHeader          = "X-Deaddrop-Pow"
)

type Pow struct {
	secret     []byte
	difficulty int
	ttl        time.Duration
	now        func() time.Time
}

func NewPow(difficulty int) *Pow {
	if difficulty <= 0 {
		difficulty = 18
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic(err)
	}
	return &Pow{secret: secret, difficulty: difficulty, ttl: 2 * time.Minute, now: time.Now}
}

func (p *Pow) Challenge(endpoint string) string {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		panic(err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	expires := p.now().Add(p.ttl).Unix()
	mac := p.mac(endpoint, nonce, expires, p.difficulty)
	return fmt.Sprintf("%s:%d:%d:%s", nonce, expires, p.difficulty, mac)
}

func (p *Pow) ValidRequest(r *http.Request, endpoint string) bool {
	return p.Valid(endpoint, r.Header.Get(powHeader))
}

func (p *Pow) Valid(endpoint, token string) bool {
	parts := strings.Split(token, ":")
	if len(parts) != 5 {
		return false
	}
	nonce := parts[0]
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || p.now().Unix() > expires {
		return false
	}
	difficulty, err := strconv.Atoi(parts[2])
	if err != nil || difficulty != p.difficulty || difficulty < 1 || difficulty > 32 {
		return false
	}
	macValue := parts[3]
	counter := parts[4]
	if !hmac.Equal([]byte(macValue), []byte(p.mac(endpoint, nonce, expires, difficulty))) {
		return false
	}
	sum := sha256.Sum256([]byte(endpoint + ":" + nonce + ":" + parts[1] + ":" + parts[2] + ":" + counter))
	return leadingZeroBits(sum[:]) >= difficulty
}

func (p *Pow) mac(endpoint, nonce string, expires int64, difficulty int) string {
	h := hmac.New(sha256.New, p.secret)
	_, _ = fmt.Fprintf(h, "%s:%s:%d:%d", endpoint, nonce, expires, difficulty)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func leadingZeroBits(b []byte) int {
	count := 0
	for _, v := range b {
		if v == 0 {
			count += 8
			continue
		}
		for i := 7; i >= 0; i-- {
			if v&(1<<uint(i)) == 0 {
				count++
			} else {
				return count
			}
		}
	}
	return count
}
