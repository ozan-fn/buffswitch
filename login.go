package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// defaultCredsPath/defaultFreebuff point at the user's own config dir
// (~/.config/manicode/), so they work on Windows, macOS and Linux alike
// instead of a hardcoded home.
func defaultCredsPath() string {
	return filepath.Join(mustHome(), ".config", "manicode", "credentials.json")
}

func defaultFreebuff() string {
	return filepath.Join(mustHome(), ".config", "manicode", "freebuff")
}

func mustHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return home
}

// freebuffBin returns the freebuff binary path (env override supported).
func freebuffBin() string {
	if b := os.Getenv("FREEBUFF_BIN"); b != "" {
		return b
	}
	return defaultFreebuff()
}

// preLoginState is the account snapshot persisted between the TUI run and
// the resumed run, so new accounts can be detected after `freebuff login`.
type preLoginState struct {
	Emails []string `json:"emails"`
	Token  string   `json:"token"`
}

func statePath(credsPath string) string {
	return filepath.Join(filepath.Dir(credsPath), ".buffswitch-login-state.json")
}

func savePreLogin(credsPath string, emails []string, token string) error {
	data, err := json.MarshalIndent(preLoginState{Emails: emails, Token: token}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(credsPath), append(data, '\n'), 0o600)
}

func loadPreLogin(credsPath string) (preLoginState, error) {
	var s preLoginState
	data, err := os.ReadFile(statePath(credsPath))
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func removePreLogin(credsPath string) {
	_ = os.Remove(statePath(credsPath))
}

// runFreebuff hands the terminal over to the freebuff binary itself (no
// subcommand), so the user lands straight in the CLI with the picked account.
func runFreebuff() error {
	cmd := exec.Command(freebuffBin())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runLoginInteractive hands the terminal over to `freebuff login` (the login
// URL is printed by the CLI itself), waits for it to finish, then re-reads
// credentials.json and reports what happened.
func runLoginInteractive(credsPath string, pre preLoginState) (string, error) {
	cmd := exec.Command(freebuffBin(), "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	st, err := Load(credsPath)
	if err != nil {
		return "", err
	}
	preEmails := map[string]bool{}
	for _, e := range pre.Emails {
		preEmails[e] = true
	}
	added := st.IngestAfterLogin(preEmails)
	if err := st.Save(); err != nil {
		return "", err
	}

	d, has := st.Default()
	switch {
	case len(added) > 0 && has:
		return fmt.Sprintf("Akun baru aktif: %s <%s>", d.Name, d.Email), nil
	case has && pre.Token != "" && d.AuthToken != pre.Token:
		return "Sesi diperbarui untuk: " + d.Name, nil
	case has:
		return "Tidak ada akun baru (login dibatalkan?).", nil
	default:
		return "Tidak ada akun aktif.", nil
	}
}
