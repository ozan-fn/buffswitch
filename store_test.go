package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCreds(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func loadRaw(t *testing.T, path string) map[string]Account {
	t.Helper()
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return s.Raw
}

func emails(accs []Account) []string {
	var out []string
	for _, a := range accs {
		out = append(out, a.Email)
	}
	return out
}

// The real file uses stray keys like "default"/"defaults"; email keys must be
// derived from the account contents.
func TestLoadWithStrayKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{
  "default": {"id":"1","name":"AKHMAD FAUZAN","email":"ozan6825@gmail.com","authToken":"t0"},
  "defaults": {"id":"2","name":"gim","email":"2586ozan@gmail.com","authToken":"t1"}
}`)

	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := emails(s.Accounts()); len(got) != 2 || got[0] != "ozan6825@gmail.com" {
		t.Fatalf("accounts = %v, want active ozan6825@gmail.com first", got)
	}

	// Switching to the parked account must work even though the file key is
	// "defaults" and not the email address.
	if err := s.SwitchTo("2586ozan@gmail.com"); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}
	d, ok := s.Default()
	if !ok || d.Email != "2586ozan@gmail.com" {
		t.Fatalf("default after switch = %+v", d)
	}
	raw := loadRaw(t, path)
	if _, ok := raw["defaults"]; ok {
		t.Fatalf("stray key 'defaults' not folded away: %v", raw)
	}
	if _, ok := raw["2586ozan@gmail.com"]; !ok {
		t.Fatalf("parked slot missing after switch: %v", raw)
	}
}

func TestSwitchRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{
  "default": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "b@x.com": {"id":"2","name":"B","email":"b@x.com","authToken":"t1"}
}`)

	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SwitchTo("b@x.com"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Default()
	if d.Email != "b@x.com" {
		t.Fatalf("active = %s, want b@x.com", d.Email)
	}
	// old default parked under its own email, with the data it had
	parked, ok := s.Raw["a@x.com"]
	if !ok || parked.AuthToken != "t0" {
		t.Fatalf("parked A = %+v", parked)
	}

	if err := s.SwitchTo("a@x.com"); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Default()
	if d.Email != "a@x.com" {
		t.Fatalf("active = %s, want a@x.com back", d.Email)
	}
	// switching to the active account is a no-op
	before := loadRaw(t, path)
	if err := s.SwitchTo("a@x.com"); err != nil {
		t.Fatal(err)
	}
	after := loadRaw(t, path)
	if len(before) != len(after) {
		t.Fatalf("no-op switch changed the file")
	}
}

func TestSwitchUnknownEmail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SwitchTo("nobody@x.com"); err == nil {
		t.Fatal("expected error for unknown email")
	}
}

func TestIngestNewDefaultAccount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)

	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	pre := map[string]bool{}
	for _, a := range s.Accounts() {
		pre[a.Email] = true
	}
	// park the active account like the add flow does before login
	s.ParkDefault()
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	// freebuff login wrote the brand-new account straight into "default"
	writeCreds(t, path, `{
  "a@x.com": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "default": {"id":"3","name":"C","email":"c@x.com","authToken":"t9"}
}`)
	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	added := s2.IngestAfterLogin(pre)
	if len(added) != 1 || added[0] != "c@x.com" {
		t.Fatalf("added = %v, want [c@x.com]", added)
	}
	if err := s2.Save(); err != nil {
		t.Fatal(err)
	}
	d, _ := s2.Default()
	if d.Email != "c@x.com" {
		t.Fatalf("active = %s, want c@x.com", d.Email)
	}
	if got := emails(s2.Accounts()); len(got) != 2 {
		t.Fatalf("accounts = %v, want old A and new C", got)
	}
}

func TestIngestStrayNewAccountPromoted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	pre := map[string]bool{}
	for _, a := range s.Accounts() {
		pre[a.Email] = true
	}

	// login appended the new account under a stray key and left "default" alone
	writeCreds(t, path, `{
  "default": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "straykey": {"id":"4","name":"D","email":"d@x.com","authToken":"t4"}
}`)
	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	added := s2.IngestAfterLogin(pre)
	if len(added) != 1 || added[0] != "d@x.com" {
		t.Fatalf("added = %v, want [d@x.com]", added)
	}
	if err := s2.Save(); err != nil {
		t.Fatal(err)
	}
	d, _ := s2.Default()
	if d.Email != "d@x.com" {
		t.Fatalf("active = %s, want promoted d@x.com", d.Email)
	}
	raw := loadRaw(t, path)
	if _, ok := raw["straykey"]; ok {
		t.Fatalf("stray key survived Save: %v", raw)
	}
}

func TestIngestCanceledLoginKeepsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	pre := map[string]bool{}
	for _, a := range s.Accounts() {
		pre[a.Email] = true
	}
	s.ParkDefault()
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	// login never wrote anything (user canceled); file unchanged on disk
	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	added := s2.IngestAfterLogin(pre)
	if len(added) != 0 {
		t.Fatalf("added = %v, want none", added)
	}
	if err := s2.Save(); err != nil {
		t.Fatal(err)
	}
	d, ok := s2.Default()
	if !ok || d.Email != "a@x.com" || d.AuthToken != "t0" {
		t.Fatalf("default = %+v, want unchanged A", d)
	}
}

func TestIngestRefreshSameEmail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	pre := map[string]bool{}
	for _, a := range s.Accounts() {
		pre[a.Email] = true
	}
	s.ParkDefault()
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	// re-login of the SAME email refreshed the token in "default"
	writeCreds(t, path, `{
  "a@x.com": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "default": {"id":"1","name":"A","email":"a@x.com","authToken":"t2"}
}`)
	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	added := s2.IngestAfterLogin(pre)
	if len(added) != 0 {
		t.Fatalf("added = %v, want none for a refresh", added)
	}
	if err := s2.Save(); err != nil {
		t.Fatal(err)
	}
	d, _ := s2.Default()
	if d.AuthToken != "t2" {
		t.Fatalf("default token = %s, want refreshed t2", d.AuthToken)
	}
	// parked copy synced with the fresh token
	slot, ok := s2.Raw["a@x.com"]
	if !ok || slot.AuthToken != "t2" {
		t.Fatalf("slot token = %+v, want synced t2", slot)
	}
}

func TestIngestFirstEverLogin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	// empty store, first login ever
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	pre := map[string]bool{}
	s.ParkDefault()
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	writeCreds(t, path, `{"default":{"id":"5","name":"E","email":"e@x.com","authToken":"t5"}}`)
	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	added := s2.IngestAfterLogin(pre)
	if len(added) != 1 || added[0] != "e@x.com" {
		t.Fatalf("added = %v, want [e@x.com]", added)
	}
	if err := s2.Save(); err != nil {
		t.Fatal(err)
	}
	d, _ := s2.Default()
	if d.Email != "e@x.com" {
		t.Fatalf("active = %s", d.Email)
	}
}

func TestDeleteParked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{
  "default": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "b@x.com": {"id":"2","name":"B","email":"b@x.com","authToken":"t1"}
}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("b@x.com"); err != nil {
		t.Fatal(err)
	}
	if got := emails(s.Accounts()); len(got) != 1 || got[0] != "a@x.com" {
		t.Fatalf("accounts after delete = %v, want only a@x.com", got)
	}
	if _, ok := loadRaw(t, path)["b@x.com"]; ok {
		t.Fatalf("parked account still present after delete")
	}
}

func TestDeleteActiveClearsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{
  "default": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "b@x.com": {"id":"2","name":"B","email":"b@x.com","authToken":"t1"}
}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("a@x.com"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Default(); ok {
		t.Fatal("default should be cleared after deleting the active account")
	}
	raw := loadRaw(t, path)
	if _, ok := raw["a@x.com"]; ok {
		t.Fatalf("active account still present after delete")
	}
	if len(emails(s.Accounts())) != 1 {
		t.Fatalf("accounts = %v, want only b@x.com", emails(s.Accounts()))
	}
}

func TestDeleteUnknownEmail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("nobody@x.com"); err == nil {
		t.Fatal("expected error for unknown email")
	}
}

func TestLoadMissingFile(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts()) != 0 {
		t.Fatalf("expected empty store, got %v", s.Accounts())
	}
}

func TestSavePreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{"default":{"id":"1","name":"A","email":"a@x.com","authToken":"t0"}}`)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %o, want 644", got)
	}
}

func TestSavedFileIsValidAndEmailKeyed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	writeCreds(t, path, `{
  "default": {"id":"1","name":"A","email":"a@x.com","authToken":"t0"},
  "defaults": {"id":"2","name":"B","email":"b@x.com","authToken":"t1"}
}`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, "\"defaults\"") {
		t.Fatalf("stray key still present after Save: %s", body)
	}
	for _, want := range []string{"\"a@x.com\"", "\"b@x.com\"", "\"default\""} {
		if !strings.Contains(body, want) {
			t.Fatalf("saved file missing %s: %s", want, body)
		}
	}
}
