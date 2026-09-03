package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// DefaultKey is the only key in credentials.json that Freebuff reads.
const DefaultKey = "default"

// Account is one Freebuff credential entry.
type Account struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	AuthToken       string `json:"authToken"`
	FingerprintID   string `json:"fingerprintId"`
	FingerprintHash string `json:"fingerprintHash"`
}

// Store wraps the credentials file. Slots are keyed by the account email;
// "default" always mirrors the active account.
type Store struct {
	Path     string
	Raw      map[string]Account
	KeyOrder []string // file key order, for "newest last" detection
}

// Load reads the credentials file (or returns an empty store if missing).
func Load(path string) (*Store, error) {
	s := &Store{Path: path, Raw: map[string]Account{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("%s: parse: %w", path, err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("%s: expected a JSON object", path)
	}
	for dec.More() {
		ktok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("%s: parse: %w", path, err)
		}
		key, ok := ktok.(string)
		if !ok {
			return nil, fmt.Errorf("%s: invalid key in object", path)
		}
		var a Account
		if err := dec.Decode(&a); err != nil {
			return nil, fmt.Errorf("%s: entry %q: %w", path, key, err)
		}
		if _, exists := s.Raw[key]; !exists {
			s.KeyOrder = append(s.KeyOrder, key)
		}
		s.Raw[key] = a
	}
	return s, nil
}

// Default returns the active account stored under the "default" key.
func (s *Store) Default() (Account, bool) {
	d, ok := s.Raw[DefaultKey]
	return d, ok
}

// Accounts lists every unique account (by email), active one first,
// then the rest sorted by name.
func (s *Store) Accounts() []Account {
	d, hasD := s.Default()
	var list []Account
	for _, a := range s.Raw {
		if a.Email == "" {
			continue
		}
		if hasD && a.Email == d.Email {
			continue // the default copy is the freshest
		}
		list = append(list, a)
	}
	if hasD && d.Email != "" {
		list = append(list, d)
	}
	sort.Slice(list, func(i, j int) bool {
		di := list[i].Email == d.Email
		dj := list[j].Email == d.Email
		if di != dj {
			return di
		}
		ni, nj := strings.ToLower(list[i].Name), strings.ToLower(list[j].Name)
		if ni != nj {
			return ni < nj
		}
		return list[i].Email < list[j].Email
	})
	return list
}

// SwitchTo makes the account with the given email the active "default".
// The previously active account is parked under its own email key.
func (s *Store) SwitchTo(email string) error {
	if email == "" {
		return errors.New("email kosong")
	}
	s.normalize() // fold any stray keys (e.g. "defaults") into email slots first
	d, hasD := s.Default()
	if hasD && d.Email == email {
		return nil // already active
	}
	if _, ok := s.Raw[email]; !ok {
		return fmt.Errorf("akun %s tidak ditemukan", email)
	}
	if hasD && d.Email != "" {
		s.Raw[d.Email] = d // park the outgoing default
	}
	s.Raw[DefaultKey] = s.Raw[email]
	return s.Save()
}

// normalize re-keys every entry by account email, keeping "default" as the
// active account and a synced parked copy under the active email.
func (s *Store) normalize() {
	d, hasD := s.Raw[DefaultKey]
	keys := make([]string, 0, len(s.Raw))
	for k := range s.Raw {
		if k != DefaultKey {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := map[string]Account{}
	for _, k := range keys {
		a := s.Raw[k]
		if a.Email == "" {
			continue // unkeyable without an email
		}
		out[a.Email] = a
	}
	if hasD {
		if d.Email != "" {
			out[d.Email] = d // keep the parked copy in sync
		}
		out[DefaultKey] = d
	}
	s.Raw = out
}

// Delete removes an account (parked or the active default). The email-keyed
// slot is removed; if that account was the active "default", the default key
// is removed too.
func (s *Store) Delete(email string) error {
	if email == "" {
		return errors.New("email kosong")
	}
	s.normalize()
	d, hasD := s.Default()
	if _, ok := s.Raw[email]; !ok {
		return fmt.Errorf("akun %s tidak ditemukan", email)
	}
	delete(s.Raw, email)
	if hasD && d.Email == email {
		delete(s.Raw, DefaultKey)
	}
	return s.Save()
}

// ParkDefault copies the current default into its email-keyed slot.
// Call this before launching `freebuff login` so the active account is
// preserved when login overwrites the "default" key.
func (s *Store) ParkDefault() {
	if d, ok := s.Default(); ok && d.Email != "" {
		s.Raw[d.Email] = d
	}
}

// IngestAfterLogin reconciles the file after a login run. pre is the set of
// emails that existed before login. Stray keys are re-keyed by email, the
// default is refreshed/updated, and if a brand-new account appeared it is
// promoted to active. Returns the emails of accounts added by the login.
func (s *Store) IngestAfterLogin(pre map[string]bool) (added []string) {
	dflt, hasDflt := s.Default()

	slots := map[string]Account{}
	var order []string
	for _, k := range s.KeyOrder {
		if k == DefaultKey {
			continue
		}
		a, ok := s.Raw[k]
		if !ok || a.Email == "" {
			continue
		}
		if _, exists := slots[a.Email]; !exists {
			order = append(order, a.Email)
		}
		slots[a.Email] = a
	}

	seen := map[string]bool{}
	addIfNew := func(email string) {
		if email == "" || seen[email] {
			return
		}
		seen[email] = true
		if !pre[email] {
			added = append(added, email)
		}
	}
	for _, e := range order {
		addIfNew(e)
	}
	if hasDflt {
		addIfNew(dflt.Email)
	}

	var active Account
	have := false
	switch {
	case hasDflt && dflt.Email != "" && !pre[dflt.Email]:
		// login wrote a brand-new account straight into "default"
		active, have = dflt, true
	case len(added) > 0:
		// default unchanged but a new account appeared: promote the newest
		last := added[len(added)-1]
		if a, ok := slots[last]; ok {
			active, have = a, true
		}
	case hasDflt:
		// nothing new: keep the existing default (refreshed copy wins)
		active, have = dflt, true
	}

	out := map[string]Account{}
	for e, a := range slots {
		out[e] = a
	}
	if have {
		out[DefaultKey] = active
	}
	s.Raw = out
	s.KeyOrder = nil
	return added
}

// Save normalizes all keys (email-keyed slots + the "default" key), syncs a
// copy of the active account into its email slot, and writes atomically with
// 0600 perms (or the file's existing perms).
func (s *Store) Save() error {
	s.normalize()

	data, err := json.MarshalIndent(s.Raw, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	mode := os.FileMode(0o600)
	if fi, err := os.Stat(s.Path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
