package tui

import (
	"fmt"
	"os"
	"path/filepath"

	ddcrypto "github.com/knarayanareddy/DEADDROP-OS/client/crypto"
	ddpanic "github.com/knarayanareddy/DEADDROP-OS/client/panic"
	"github.com/rivo/tview"
)

func RunApp(baseURL, keyDir, activitySocket string) error {
	if err := ensureKeys(keyDir); err != nil {
		return err
	}
	client := New(baseURL)
	app := tview.NewApplication()
	pages := tview.NewPages()
	status := tview.NewTextView().SetDynamicColors(true).SetText("[green]Ready[-]")
	setStatus := func(format string, args ...any) { status.SetText(fmt.Sprintf(format, args...)) }
	ping := func() { _ = ddpanic.NotifyActivity(activitySocket) }

	showMain := func() {}
	showDrop := func() {}
	showRetrieve := func() {}
	showPublic := func() {}

	main := tview.NewForm().
		AddButton("Compose Drop", func() { ping(); showDrop() }).
		AddButton("Retrieve Drops", func() { ping(); showRetrieve() }).
		AddButton("Show Public Key", func() { ping(); showPublic() }).
		AddButton("Quit", func() { app.Stop() })
	main.SetBorder(true).SetTitle(" DEADDROP-OS Messenger ")

	layout := func(title string, primitive tview.Primitive) tview.Primitive {
		return tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(primitive, 0, 1, true).
			AddItem(status.SetTitle(title), 1, 0, false)
	}

	showMain = func() { pages.SwitchToPage("main") }
	showDrop = func() {
		var hint, pubPEM, message string
		form := tview.NewForm().
			AddInputField("Hint", "", 40, nil, func(v string) { hint = v; ping() }).
			AddTextArea("Recipient public key PEM", "", 0, 8, 0, func(v string) { pubPEM = v; ping() }).
			AddTextArea("Message", "", 0, 8, 0, func(v string) { message = v; ping() }).
			AddButton("Submit", func() {
				ping()
				payload, err := EncryptMessage([]byte(pubPEM), message)
				if err != nil {
					setStatus("[red]encrypt failed: %v[-]", err)
					return
				}
				id, err := client.Submit(hint, payload)
				if err != nil {
					setStatus("[red]submit failed: %v[-]", err)
					return
				}
				setStatus("[green]submitted drop %s[-]", id)
			}).
			AddButton("Back", showMain)
		form.SetBorder(true).SetTitle(" Compose encrypted drop ")
		pages.AddAndSwitchToPage("drop", layout("compose", form), true)
	}
	showRetrieve = func() {
		var hint string
		results := tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
		form := tview.NewForm().
			AddInputField("Hint", "", 40, nil, func(v string) { hint = v; ping() }).
			AddButton("Retrieve", func() {
				ping()
				privBytes, err := os.ReadFile(filepath.Join(keyDir, "private.pem"))
				if err != nil {
					setStatus("[red]key read failed: %v[-]", err)
					return
				}
				priv, err := ddcrypto.ParsePrivateKeyPEM(privBytes)
				if err != nil {
					setStatus("[red]key parse failed[-]")
					return
				}
				drops, err := client.Drops(hint)
				if err != nil {
					setStatus("[red]retrieve failed: %v[-]", err)
					return
				}
				results.Clear()
				if len(drops) == 0 {
					results.SetText("[yellow]No drops found.[-]")
					return
				}
				for _, d := range drops {
					pt, err := ddcrypto.Decrypt(priv, d.Payload)
					if err != nil {
						fmt.Fprintf(results, "[red]--- %s: decryption failed ---[-]\n", d.ID)
						continue
					}
					fmt.Fprintf(results, "[green]--- %s ---[-]\n%s\n\n", d.ID, string(pt))
				}
				setStatus("[green]retrieved %d drop(s)[-]", len(drops))
			}).
			AddButton("Back", showMain)
		form.SetBorder(true).SetTitle(" Retrieve drops ")
		flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(form, 8, 0, true).AddItem(results, 0, 1, false)
		pages.AddAndSwitchToPage("retrieve", layout("retrieve", flex), true)
	}
	showPublic = func() {
		b, err := os.ReadFile(filepath.Join(keyDir, "public.pem"))
		if err != nil {
			setStatus("[red]public key read failed: %v[-]", err)
			return
		}
		text := tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetText(string(b))
		text.SetBorder(true).SetTitle(" Session public key — share out-of-band ")
		pages.AddAndSwitchToPage("public", layout("public key", text), true)
	}
	pages.AddPage("main", layout("main", main), true, true)
	return app.SetRoot(pages, true).EnableMouse(false).Run()
}

func ensureKeys(keyDir string) error {
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return err
	}
	privPath, pubPath := filepath.Join(keyDir, "private.pem"), filepath.Join(keyDir, "public.pem")
	if _, err := os.Stat(privPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	k, err := ddcrypto.GenerateKeyPair(4096)
	if err != nil {
		return err
	}
	if err := os.WriteFile(privPath, ddcrypto.EncodePrivateKeyPEM(k), 0600); err != nil {
		return err
	}
	pub, err := ddcrypto.EncodePublicKeyPEM(&k.PublicKey)
	if err != nil {
		return err
	}
	return os.WriteFile(pubPath, pub, 0644)
}
