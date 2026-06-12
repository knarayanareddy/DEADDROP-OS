package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	ddpanic "github.com/knarayanareddy/DEADDROP-OS/client/panic"
)

func main() {
	paths := flag.String("paths", "/run/user/1000/deaddrop,/tmp/deaddrop", "comma-separated tmpfs paths to wipe")
	idle := flag.Duration("inactivity", 30*time.Minute, "inactivity timeout reset by client activity")
	activitySocket := flag.String("activity-socket", "/run/deaddrop/activity.sock", "unix datagram socket for client activity pings")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan ddpanic.Trigger, 1)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sig
		if s == syscall.SIGUSR1 {
			ch <- ddpanic.TriggerSignal
		} else {
			cancel()
		}
	}()
	_ = os.MkdirAll(filepath.Dir(*activitySocket), 0700)
	activity, err := ddpanic.ListenActivity(ctx, *activitySocket)
	if err != nil {
		log.Printf("activity socket disabled: %v", err)
	}
	d := &ddpanic.Daemon{Paths: strings.Split(*paths, ","), Inactivity: *idle, OnTrigger: func(t ddpanic.Trigger) { log.Printf("panic trigger: %s", t) }}
	d.RunWithActivity(ctx, ch, activity)
}
