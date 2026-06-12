package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const Version = "0.1.0"

var hintRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type Server struct {
	store  *Store
	rl     *RateLimiter
	pow    *Pow
	logger *log.Logger
}

func New(store *Store, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Server{store: store, rl: NewRateLimiter(), pow: NewPow(18), logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.withLog("/health", s.health))
	mux.HandleFunc("/drop", s.withLog("/drop", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.postDrop(w, r)
		default:
			methodNotAllowed(w)
		}
	}))
	mux.HandleFunc("/drops", s.withLog("/drops", s.getDrops))
	mux.HandleFunc("/drop/", s.withLog("/drop/:id", s.deleteDrop))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLog(path string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: 200}
		h(rw, r)
		// Spec-compliant access log: no RemoteAddr, no User-Agent, no hint, no body.
		s.logger.Printf("%s | %s | %s | %d | %d | %d", time.Now().UTC().Format(time.RFC3339), r.Method, path, rw.status, time.Since(start).Milliseconds(), rw.bytes)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status, bytes int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *statusWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (s *Server) rate(w http.ResponseWriter, r *http.Request, endpoint string, limit int) bool {
	ok, wait := s.rl.Allow(connectionKey(r, endpoint), limit, time.Minute)
	if ok {
		return true
	}
	if s.pow.ValidRequest(r, endpoint) {
		return true
	}
	if wait < time.Second {
		wait = time.Second
	}
	challenge := s.pow.Challenge(endpoint)
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", wait.Seconds()))
	w.Header().Set(powChallengeHeader, challenge)
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":     "rate_limited",
		"message":   fmt.Sprintf("retry after %.0f seconds or retry with proof-of-work", wait.Seconds()),
		"challenge": challenge,
	})
	return false
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !s.rate(w, r, "health", 60) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": Version})
}

type postDropReq struct {
	Hint    string `json:"hint"`
	Payload string `json:"payload"`
}
type postDropResp struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}

func (s *Server) postDrop(w http.ResponseWriter, r *http.Request) {
	if !s.rate(w, r, "post", 10) {
		return
	}
	var req postDropReq
	if err := decodeJSON(r, &req, 1<<20); err != nil {
		writeError(w, 400, "invalid_payload", "invalid JSON")
		return
	}
	if err := validateHint(req.Hint); err != nil {
		writeError(w, 400, "invalid_payload", "invalid hint")
		return
	}
	if err := validatePayload(req.Payload); err != nil {
		writeError(w, 400, "invalid_payload", "invalid payload")
		return
	}
	d, err := s.store.CreateDrop(r.Context(), req.Hint, req.Payload, DefaultTTL)
	if err != nil {
		writeError(w, 500, "server_error", "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, postDropResp{ID: d.ID, ExpiresAt: d.ExpiresAt.Format(time.RFC3339)})
}

func (s *Server) getDrops(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !s.rate(w, r, "get", 30) {
		return
	}
	hint := r.URL.Query().Get("hint")
	if err := validateHint(hint); err != nil {
		writeJSON(w, 200, map[string]any{"drops": []any{}})
		return
	}
	drops, err := s.store.ListDrops(r.Context(), hint, time.Now())
	if err != nil {
		writeError(w, 500, "server_error", "internal error")
		return
	}
	items := make([]map[string]string, 0, len(drops))
	for _, d := range drops {
		items = append(items, map[string]string{"id": d.ID, "payload": d.Payload, "created_at": d.CreatedAt.Format(time.RFC3339), "expires_at": d.ExpiresAt.Format(time.RFC3339)})
	}
	writeJSON(w, 200, map[string]any{"drops": items})
}

func (s *Server) deleteDrop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	if !s.rate(w, r, "delete", 20) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/drop/")
	if id != "" {
		_ = s.store.DeleteDrop(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func StartExpiryLoop(ctx context.Context, store *Store, interval time.Duration, logger *log.Logger) {
	t := time.NewTicker(interval)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n, err := store.ExpireDrops(ctx, time.Now())
				if logger != nil {
					if err != nil {
						logger.Printf("expiry error: %v", err)
					} else {
						logger.Printf("expired drops: %d", n)
					}
				}
			}
		}
	}()
}

func validateHint(h string) error {
	if !hintRE.MatchString(h) {
		return errors.New("invalid hint")
	}
	return nil
}
func validatePayload(p string) error {
	if p == "" || len(p) > 1<<20 {
		return errors.New("invalid size")
	}
	_, err := base64.RawURLEncoding.DecodeString(p)
	return err
}
func decodeJSON(r *http.Request, v any, max int64) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, max)).Decode(v)
}
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, code int, errCode, msg string) {
	writeJSON(w, code, map[string]string{"error": errCode, "message": msg})
}
func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}
