package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	dd "github.com/knarayanareddy/DEADDROP-OS/server"
)

func main() {
	addr := flag.String("addr", getenv("DEADDROP_LISTEN", "127.0.0.1:8080"), "loopback listen address")
	dbPath := flag.String("db", getenv("DEADDROP_DB", "/var/lib/deaddrop/drops.db"), "SQLite database path")
	flag.Parse()
	host, _, err := net.SplitHostPort(*addr)
	if err != nil || host != "127.0.0.1" {
		log.Fatalf("refusing non-loopback listener %q; onion service must front the backend", *addr)
	}
	store, err := dd.OpenStore(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	logger := log.New(os.Stdout, "", 0)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	dd.StartExpiryLoop(ctx, store, 15*time.Minute, logger)
	srv := &http.Server{Addr: *addr, Handler: dd.New(store, logger).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(c)
	}()
	logger.Printf("deaddrop-server listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
