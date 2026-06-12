package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func testServer(t *testing.T) (*Store, http.Handler) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	s := NewStore(db)
	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv := New(s, log.New(io.Discard, "", 0))
	srv.pow.difficulty = 12
	return s, srv.Handler()
}

func TestDropSubmitRetrieve(t *testing.T) {
	_, h := testServer(t)
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"version":"1"}`))
	body, _ := json.Marshal(map[string]string{"hint": "abc_123", "payload": payload})
	r := httptest.NewRequest(http.MethodPost, "/drop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("post: %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, "/drops?hint=abc_123", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d", w.Code)
	}
	var out struct {
		Drops []struct{ ID, Payload string } `json:"drops"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Drops) != 1 || out.Drops[0].Payload != payload {
		t.Fatalf("unexpected drops: %+v", out.Drops)
	}
}

func TestDropDelete(t *testing.T) {
	_, h := testServer(t)
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"version":"1"}`))
	body, _ := json.Marshal(map[string]string{"hint": "hint", "payload": payload})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/drop", bytes.NewReader(body)))
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/drop/"+out.ID, nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/drops?hint=hint", nil))
	if strings.Contains(w.Body.String(), out.ID) {
		t.Fatal("deleted drop returned")
	}
}

func TestExpiredDropNotReturnedAndExpiry(t *testing.T) {
	s, _ := testServer(t)
	d, err := s.CreateDrop(context.Background(), "h", "cGF5bG9hZA", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.DB().Exec(`UPDATE drops SET expires_at=? WHERE id=?`, time.Now().Add(-time.Hour).Unix(), d.ID)
	if err != nil {
		t.Fatal(err)
	}
	drops, err := s.ListDrops(context.Background(), "h", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(drops) != 0 {
		t.Fatal("expired returned")
	}
	n, err := s.ExpireDrops(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expired %d", n)
	}
}

func TestHintIsolation(t *testing.T) {
	s, _ := testServer(t)
	_, _ = s.CreateDrop(context.Background(), "A", "cGF5", time.Hour)
	drops, _ := s.ListDrops(context.Background(), "B", time.Now())
	if len(drops) != 0 {
		t.Fatal("hint isolation failed")
	}
}

func TestInvalidHintEmptyDrops(t *testing.T) {
	_, h := testServer(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/drops?hint=bad!", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "[]") {
		t.Fatalf("unexpected: %d %s", w.Code, w.Body.String())
	}
}

func TestHealthEndpoint(t *testing.T) {
	_, h := testServer(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), Version) {
		t.Fatalf("bad health: %d %s", w.Code, w.Body.String())
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 2; i++ {
		if ok, _ := rl.Allow("k", 2, time.Minute); !ok {
			t.Fatal("unexpected reject")
		}
	}
	if ok, _ := rl.Allow("k", 2, time.Minute); ok {
		t.Fatal("expected reject")
	}
}

func TestNoIPInAccessLog(t *testing.T) {
	s, h := testServer(t)
	_ = s
	var buf bytes.Buffer
	h = New(s, log.New(&buf, "", 0)).Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if strings.Contains(buf.String(), "192.") || strings.Contains(buf.String(), "RemoteAddr") || strings.Contains(buf.String(), "User-Agent") {
		t.Fatalf("sensitive log: %q", buf.String())
	}
}

func TestConcurrentDrops(t *testing.T) {
	_, h := testServer(t)
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"version":"1"}`))
	errCh := make(chan error, 100)
	for i := 0; i < 100; i++ {
		go func(i int) {
			body, _ := json.Marshal(map[string]string{"hint": "concurrent", "payload": payload})
			r := httptest.NewRequest(http.MethodPost, "/drop", bytes.NewReader(body))
			r.RemoteAddr = "10.0.0." + strconv.Itoa(i+1) + ":12345"
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusCreated {
				errCh <- fmt.Errorf("post %d returned %d: %s", i, w.Code, w.Body.String())
				return
			}
			errCh <- nil
		}(i)
	}
	for i := 0; i < 100; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/drops?hint=concurrent", nil)
	r.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out struct {
		Drops []struct {
			ID string `json:"id"`
		} `json:"drops"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Drops) != 100 {
		t.Fatalf("got %d drops", len(out.Drops))
	}
}

func TestRateLimiterStripsEphemeralPortAndPrunes(t *testing.T) {
	rl := NewRateLimiter()
	base := time.Unix(1000, 0)
	rl.now = func() time.Time { return base }
	rl.lastPrune = base
	if ok, _ := rl.Allow("post:127.0.0.1", 1, time.Second); !ok {
		t.Fatal("first request rejected")
	}
	if ok, _ := rl.Allow("post:127.0.0.1", 1, time.Second); ok {
		t.Fatal("same host different port equivalent should be limited")
	}
	base = base.Add(2 * time.Second)
	if ok, _ := rl.Allow("other:127.0.0.1", 1, time.Second); !ok {
		t.Fatal("new bucket rejected")
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if len(rl.buckets) > 1 {
		t.Fatalf("expired buckets not pruned: %d", len(rl.buckets))
	}
}

func TestConnectionKeyStripsPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.RemoteAddr = "127.0.0.1:54321"
	if got := connectionKey(r, "health"); got != "health:127.0.0.1" {
		t.Fatalf("got %q", got)
	}
}

func TestProofOfWorkAllowsRequestAfterRateLimit(t *testing.T) {
	_, h := testServer(t)
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"version":"1"}`))
	body, _ := json.Marshal(map[string]string{"hint": "pow", "payload": payload})
	var challenge string
	for i := 0; i < 11; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/drop", bytes.NewReader(body))
		r.RemoteAddr = "127.0.0.1:10000"
		h.ServeHTTP(w, r)
		if i < 10 && w.Code != http.StatusCreated {
			t.Fatalf("request %d: %d", i, w.Code)
		}
		if i == 10 {
			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("expected 429, got %d", w.Code)
			}
			challenge = w.Header().Get(powChallengeHeader)
		}
	}
	if challenge == "" {
		t.Fatal("missing pow challenge")
	}
	// Use the server's live verifier by solving against its challenge. The helper
	// is intentionally local to avoid exporting client internals into server tests.
	parts := strings.Split(challenge, ":")
	if len(parts) != 4 {
		t.Fatalf("bad challenge %q", challenge)
	}
	difficulty, _ := strconv.Atoi(parts[2])
	var token string
	for counter := 0; ; counter++ {
		c := strconv.Itoa(counter)
		sum := sha256.Sum256([]byte("post:" + parts[0] + ":" + parts[1] + ":" + parts[2] + ":" + c))
		if leadingZeroBits(sum[:]) >= difficulty {
			token = challenge + ":" + c
			break
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/drop", bytes.NewReader(body))
	r.RemoteAddr = "127.0.0.1:10001"
	r.Header.Set(powHeader, token)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("pow retry failed: %d %s", w.Code, w.Body.String())
	}
}
