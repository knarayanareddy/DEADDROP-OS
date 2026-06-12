package panic

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Trigger string

const (
	TriggerSignal     Trigger = "signal"
	TriggerInactivity Trigger = "inactivity"
)

type Daemon struct {
	Paths      []string
	Inactivity time.Duration
	OnTrigger  func(Trigger)
}

func (d *Daemon) Run(ctx context.Context, trigger <-chan Trigger) {
	d.RunWithActivity(ctx, trigger, nil)
}

func (d *Daemon) RunWithActivity(ctx context.Context, trigger <-chan Trigger, activity <-chan struct{}) {
	var timer *time.Timer
	var timerC <-chan time.Time
	if d.Inactivity > 0 {
		timer = time.NewTimer(d.Inactivity)
		timerC = timer.C
		defer timer.Stop()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-trigger:
			d.Wipe(t)
			return
		case <-timerC:
			d.Wipe(TriggerInactivity)
			return
		case <-activity:
			if timer != nil {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(d.Inactivity)
			}
		}
	}
}

func (d *Daemon) Wipe(t Trigger) {
	if d.OnTrigger != nil {
		d.OnTrigger(t)
	}
	for _, p := range d.Paths {
		_ = wipePath(p)
	}
}

func ListenActivity(ctx context.Context, socketPath string) (<-chan struct{}, error) {
	_ = os.Remove(socketPath)
	addr := net.UnixAddr{Name: socketPath, Net: "unixgram"}
	conn, err := net.ListenUnixgram("unixgram", &addr)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(socketPath, 0600)
	ch := make(chan struct{}, 1)
	go func() {
		defer conn.Close()
		defer close(ch)
		buf := make([]byte, 16)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			_, _, err := conn.ReadFromUnix(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					select {
					case <-ctx.Done():
						return
					default:
						continue
					}
				}
				return
			}
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}()
	return ch, nil
}

func NotifyActivity(socketPath string) error {
	addr := net.UnixAddr{Name: socketPath, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, &addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte("a"))
	return err
}

func wipePath(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			_ = wipePath(filepath.Join(path, e.Name()))
		}
		return os.RemoveAll(path)
	}
	if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		zeros := make([]byte, 1024*1024)
		left := info.Size()
		for left > 0 {
			n := int64(len(zeros))
			if left < n {
				n = left
			}
			_, _ = f.Write(zeros[:n])
			left -= n
		}
		_ = f.Sync()
		_ = f.Close()
	}
	return os.Remove(path)
}
