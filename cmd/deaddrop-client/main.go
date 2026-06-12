package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ddcrypto "github.com/knarayanareddy/DEADDROP-OS/client/crypto"
	ddpanic "github.com/knarayanareddy/DEADDROP-OS/client/panic"
	"github.com/knarayanareddy/DEADDROP-OS/client/securemem"
	"github.com/knarayanareddy/DEADDROP-OS/client/tui"
)

func main() {
	server := flag.String("server", os.Getenv("DEADDROP_SERVER"), "onion base URL, e.g. http://example.onion")
	keyDir := flag.String("key-dir", getenv("DEADDROP_KEY_DIR", "/run/user/1000/deaddrop"), "tmpfs key directory")
	activitySocket := flag.String("activity-socket", getenv("DEADDROP_ACTIVITY_SOCKET", "/run/deaddrop/activity.sock"), "panic daemon activity socket")
	cli := flag.Bool("cli", false, "use line-oriented CLI fallback instead of full TUI")
	selfTest := flag.Bool("self-test", false, "run non-interactive self-test submit/retrieve flow")
	flag.Parse()
	_ = securemem.LockProcessMemory() // best-effort; prevents key/plaintext swap when permitted
	if *server == "" {
		fmt.Fprintln(os.Stderr, "server URL required")
		os.Exit(2)
	}
	if *selfTest {
		if err := runSelfTest(*server, *keyDir, *activitySocket); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if !*cli {
		if err := tui.RunApp(*server, *keyDir, *activitySocket); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	_ = os.MkdirAll(*keyDir, 0700)
	privPath, pubPath := filepath.Join(*keyDir, "private.pem"), filepath.Join(*keyDir, "public.pem")
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		k, err := ddcrypto.GenerateKeyPair(4096)
		must(err)
		must(os.WriteFile(privPath, ddcrypto.EncodePrivateKeyPEM(k), 0600))
		pub, err := ddcrypto.EncodePublicKeyPEM(&k.PublicKey)
		must(err)
		must(os.WriteFile(pubPath, pub, 0644))
		fmt.Println("Generated session key pair in tmpfs. Share public.pem out-of-band.")
	}
	client := tui.New(*server)
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n[d]rop [r]etrieve [p]ublic key [q]uit > ")
		_ = ddpanic.NotifyActivity(*activitySocket)
		cmd, _ := in.ReadString('\n')
		switch strings.TrimSpace(cmd) {
		case "d":
			fmt.Print("hint> ")
			hint := read(in)
			fmt.Println("Paste recipient public key PEM, end with single '.' line:")
			pub := readBlock(in)
			fmt.Println("Message, end with single '.' line:")
			msg := readBlock(in)
			payload, err := tui.EncryptMessage([]byte(pub), msg)
			must(err)
			id, err := client.Submit(hint, payload)
			must(err)
			fmt.Println("submitted id:", id)
		case "r":
			fmt.Print("hint> ")
			hint := read(in)
			privBytes, err := os.ReadFile(privPath)
			must(err)
			priv, err := ddcrypto.ParsePrivateKeyPEM(privBytes)
			must(err)
			drops, err := client.Drops(hint)
			must(err)
			if len(drops) == 0 {
				fmt.Println("no drops")
			}
			for _, d := range drops {
				pt, err := ddcrypto.Decrypt(priv, d.Payload)
				if err != nil {
					fmt.Println(d.ID, "decryption failed")
				} else {
					fmt.Printf("--- %s ---\n%s\n", d.ID, string(pt))
				}
			}
		case "p":
			b, _ := os.ReadFile(pubPath)
			fmt.Print(string(b))
		case "q":
			return
		}
	}
}
func read(in *bufio.Reader) string { s, _ := in.ReadString('\n'); return strings.TrimSpace(s) }
func readBlock(in *bufio.Reader) string {
	var b strings.Builder
	for {
		s, _ := in.ReadString('\n')
		if strings.TrimSpace(s) == "." {
			break
		}
		b.WriteString(s)
	}
	return b.String()
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func runSelfTest(server, keyDir, activitySocket string) error {
	_ = ddpanic.NotifyActivity(activitySocket)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return err
	}
	k, err := ddcrypto.GenerateKeyPair(2048)
	if err != nil {
		return err
	}
	privPath, pubPath := filepath.Join(keyDir, "private.pem"), filepath.Join(keyDir, "public.pem")
	if err := os.WriteFile(privPath, ddcrypto.EncodePrivateKeyPEM(k), 0600); err != nil {
		return err
	}
	pub, err := ddcrypto.EncodePublicKeyPEM(&k.PublicKey)
	if err != nil {
		return err
	}
	if err := os.WriteFile(pubPath, pub, 0644); err != nil {
		return err
	}
	client := tui.New(server)
	hint := fmt.Sprintf("e2e_%d", time.Now().UnixNano())
	message := "deaddrop self-test"
	payload, err := ddcrypto.Encrypt(&k.PublicKey, []byte(message))
	if err != nil {
		return err
	}
	if _, err := client.Submit(hint, payload); err != nil {
		return err
	}
	_ = ddpanic.NotifyActivity(activitySocket)
	drops, err := client.Drops(hint)
	if err != nil {
		return err
	}
	for _, d := range drops {
		pt, err := ddcrypto.Decrypt(k, d.Payload)
		if err == nil && string(pt) == message {
			fmt.Println("self-test passed")
			return nil
		}
	}
	return fmt.Errorf("self-test failed: submitted drop was not retrieved/decrypted")
}
