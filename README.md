# buffsw-cli — account selector `bs`

> **English** · [Bahasa Indonesia](README.id.md)

A terminal account selector for **Freebuff**, written in Go with
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

`bs` manages multiple Freebuff accounts stored in
`~/.config/manicode/credentials.json`. Freebuff **only reads the `default`
key** in that file — so `bs` simply swaps the contents of `default` with
the account you pick, parking the previously active account in its own
slot.

---

## Quick Start

Install the package globally from npm, then run `bs`:

```bash
npm i -g buffsw-cli
bs
```

> **Dev mode** — run straight from the source tree (Go 1.27+ and an
> interactive terminal required):
>
> ```bash
> cd ~/projects/buffswitch
> go run .
> ```

Inside the TUI:

```text
↑/↓ or j/k select · Enter activate · a add · d delete · r reload · q quit
```

- **Switch account** — select one, press `Enter`.
- **Add account** — press `a`: the TUI steps aside and hands the terminal
  over to `freebuff login`; its output — including the login URL — appears
  right in your terminal. Finish the login there, then press Enter to
  return to the refreshed account list.
- Different credentials file? `bs -creds /path/credentials.json`

Full details below.

---

## Table of Contents

- [Features](#features)
- [How it works](#how-it-works)
- [File layout](#file-layout)
- [Requirements](#requirements)
- [Usage](#usage)
- [Add-account flow](#add-account-flow)
- [credentials.json format](#credentialsjson-format)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

---

## Features

- **Account list** — every stored Freebuff account, active one marked
  `(active)`, in an 80-column window (auto-shrinks on narrow terminals).
- **Switch account (`Enter`)** — the selected account is copied into the
  `default` key; the previously active account is parked under its
  **email** key.
- **Add account (`a`)** — runs the official `freebuff login` binary by
  handing over the terminal, so its output and the login URL are shown
  right there; when it finishes, `bs` returns to the TUI with the new
  account active.
- **Delete account (`d`)** — removes the selected account (parked or the
  active one) with a `y/n` confirmation; if it was the active account, the
  `default` key is cleared too.
- **Reload (`r`)** — re-reads `credentials.json` from disk (useful if you
  log in outside `bs`).
- **Automatic normalization** — stray/random keys left by login (e.g.
  `defaults`) are re-keyed by email; the file is never left messy.
- **Slot sync** — refreshed sessions (new token for the same email) are
  copied into the parked slot too.
- Safe: atomic writes (temp + rename) and the file's original permissions
  are preserved.

---

## How it works

`credentials.json` is a JSON map. Only one key matters to Freebuff:

```text
"default": { ... the currently active account ... }
```

Every other key is **not read by Freebuff** — those are `bs`'s parking
slots. `bs` uses the **account email as the slot key**, so each account is
easy to find and can never be mixed up.

Core operations:

| Operation | What happens to the file |
|---|---|
| **Switch to B** | contents of `default` (account A) are copied to A's email slot; contents of B's email slot are copied into `default` |
| **Before login (park)** | the current `default` is copied to its email slot first, so it is not lost when `freebuff login` overwrites `default` |
| **After login** | the file is re-read, stray keys are re-keyed by email, a brand-new account becomes `default` (or a same-email session is refreshed), and slots are synced |

Invariant kept after every `Save`:

```text
{
  "default":     <active account>,
  "<email-1>":   <parked account 1>,
  "<email-2>":   <parked account 2>,
  ...
}
```

---

## File layout

```text
buffswitch/            repo; npm package name is `buffsw-cli`
├── go.mod            module bs (bubbletea + lipgloss)
├── main.go           Bubble Tea TUI: account list, keymap, login loop
├── store.go          credentials.json logic: load/save/normalize/
│                     switch/park/ingest-after-login
├── login.go          terminal hand-off: runs `freebuff login`, ingests
│                     the result, manages the temporary login-state file
├── store_test.go     unit tests for the store logic
├── package.json      npm wrapper: `bin.bs` → `bin/bs.exe` (tarball ships
│                     no binaries — only the stub + postinstall)
├── postinstall.mjs   downloads the binary matching the user's OS/arch
│                     from the GitHub Release → `bin/bs.exe`
├── bin/
│   └── bs.exe        plain-text placeholder; replaced by postinstall
└── dist/              built binaries, one per OS/arch (gitignored) —
                      upload these to the GitHub Release with `gh`
```

### Flow when you press `a`

```text
[a] pressed
  → ParkDefault() + Save()          (park the active account first)
  → snapshot emails/token → temporary state file (auto-cleaned)
  → TUI quits (alt screen off)
  → `freebuff login` runs in your terminal (URL shown there)
  → CLI exits after the browser login finishes
  → credentials.json re-read → IngestAfterLogin(pre) → Save()
  → press Enter → TUI relaunches with the updated account list
```

---

## Requirements

- **To install & run:** Node.js/npm (for `npm i -g buffsw-cli`)
- **Dev mode / building from source:** Go 1.27+ (module `bs` is written with Go 1.27)
- The Freebuff binary inside the manicode config directory (override with
  the `FREEBUFF_BIN` env var)
- `credentials.json` inside the manicode config directory (override with
  the `-creds` flag)
- An interactive terminal (the TUI uses an alternate screen)

---

## Usage

| Key | Action |
|---|---|
| `↑` / `↓` or `j` / `k` | select account |
| `Enter` | activate the selected account (switch `default`) |
| `a` | add account: exit the TUI, run `freebuff login` in the terminal, then return |
| `r` | reload the list from disk |
| `d` | delete the selected account (confirm with `y`) |
| `q` / `Ctrl+C` | quit |

---

## Add-account flow

1. Press `a` in the account list.
2. `bs` parks the active account in its email slot (data is not lost) and
   snapshots the current accounts to a temporary state file.
3. The TUI closes and `freebuff login` takes over the terminal. Its output
   — including the login URL — is shown right there. Copy the URL, open it
   in your browser, and finish the login. Nothing is auto-opened.
4. When the CLI exits, `bs` re-reads `credentials.json`, then:
   - a **new** account (email not seen before) → becomes `default`;
   - the **same** email as an existing account → the session is refreshed
     (the new token is copied into the slot too);
   - nothing changed (login canceled) → `default` is left untouched.
5. A one-line result is printed (e.g. `Akun baru aktif: ...`). Press
   **Enter** to relaunch the TUI with the updated account list.

The temporary state file is removed automatically. `bs` never tries to
open a browser or capture the login output itself — `freebuff login` owns
the terminal, which is the most reliable way to run its interactive login.

---

## credentials.json format

Example structure after `bs` has managed the file — **all values below are
placeholders, not real data**:

```json
{
  "default": {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "Active Account Name",
    "email": "main-account@example.com",
    "authToken": "secret-auth-token",
    "fingerprintId": "enhanced-xxxxxxxx",
    "fingerprintHash": "hash-xxxxxxxx"
  },
  "other-account@example.com": {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "Another Account Name",
    "email": "other-account@example.com",
    "authToken": "secret-auth-token",
    "fingerprintId": "enhanced-xxxxxxxx",
    "fingerprintHash": "hash-xxxxxxxx"
  }
}
```

Rules `bs` enforces:

- Every key other than `default` is the account's **email**.
- Stray keys left by login (e.g. `defaults`) are re-keyed by email on
  `Save`.
- `default` always has a synced parked copy under the same email slot.
- The file is written atomically (`.tmp` + rename) keeping the original
  mode (0600 if the file does not exist yet).
- Account field contents (`authToken`, `fingerprint*`) are never modified —
  they are only moved between keys.

---

## Development

> Dev mode runs the Go source directly instead of the installed binary:

```bash
cd ~/projects/buffswitch

go run .                                   # dev mode: run the TUI (needs a TTY)

gofmt -l .                                 # format check (empty = tidy)

"$(go env GOBIN)/golangci-lint" run ./...  # lint (baseline: 0 issues)
```

**Linting is the only test/verification command this project uses**
(golangci-lint v2, installed via
`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).
`go vet`/`go test` are not part of the standard dev flow.

> Store-logic unit tests exist in `store_test.go` and can be run anytime
> with `go test ./...` — but they are not part of the standard dev flow.

### Release binaries (multi-OS)

The npm package ships **no binaries** at all. Each platform binary lives
as a **GitHub Release asset** (repo `ozan-fn/buffswitch`, tag
`v<version>`). During install, `postinstall.mjs` downloads **only the one
matching the user's OS/arch** and saves it as `bin/bs.exe` — so users
download a few MB, never all eight (~48 MB). `bin/bs.exe` in the tarball
is a plain-text stub (no shebang) that only prints an error if
postinstall never ran — it is a file, never a shell script, so Windows
never tries to run `sh`.

Build every platform into `dist/` before a release:

```bash
for t in "linux amd64 buffsw-cli-linux-x64" \
         "linux arm64 buffsw-cli-linux-arm64" \
         "linux arm buffsw-cli-linux-arm" \
         "linux 386 buffsw-cli-linux-386" \
         "darwin amd64 buffsw-cli-darwin-x64" \
         "darwin arm64 buffsw-cli-darwin-arm64" \
         "windows amd64 buffsw-cli-win32-x64.exe" \
         "windows arm64 buffsw-cli-win32-arm64.exe"; do
  set -- $t
  GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -trimpath -o "dist/$3" .
done
```

`CGO_ENABLED=0` produces **static** binaries, so a single Linux build runs
on both **glibc** and **musl**. Binaries are gitignored (`dist/`).

Optional: shrink with UPX before uploading (Linux and Windows-x64 only;
darwin and win-arm64 are not supported):

```bash
upx --best --lzma dist/buffsw-cli-linux-* dist/buffsw-cli-win32-x64.exe
```

Publish a release (a single npm package):

1. Make the release tag match the package version — bump `"version"` in
   `package.json`, then `git tag v0.1.1` (create the tag for the version
   you are releasing).
2. Upload the built binaries to the GitHub release:
   `gh release upload v0.1.1 dist/*`
   (create the release first with `gh release create v0.1.1` if needed).
3. Publish the npm package: `npm publish`.

`postinstall.mjs` builds its download URL from the package version
(`https://github.com/ozan-fn/buffswitch/releases/download/v0.1.1/buffsw-cli-linux-x64`),
so the release tag and the package version must always match. Override the
URL per install with the `BUFFSW_CLI_BINARY_URL` env var (e.g. a mirror).

File map for development:

| File | Responsibility |
|---|---|
| `main.go` | UI: Bubble Tea model, keymap, list rendering, login loop |
| `store.go` | Data rules: `Load`, `Save`, `normalize`, `SwitchTo`, `ParkDefault`, `Delete`, `IngestAfterLogin` |
| `login.go` | Terminal hand-off: runs `freebuff login`, ingests the result, manages the temporary login-state file |

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `Gagal membaca ... : expected a JSON object` | `credentials.json` is broken/invalid — check the JSON |
| `Gagal menjalankan freebuff login` | check `FREEBUFF_BIN` or that the `freebuff` binary exists |
| `State login tidak ditemukan` | the temporary state file was deleted while `bs` was closed — press `r` after relaunching to reload |
| An account is missing from the list | press `r` to reload from disk |
| Login finished but "No new account" | you logged in with an already-registered email — the session was refreshed, not added |
| Layout breaks in a narrow terminal | the window auto-shrinks; minimum width is ~20 columns |
| `Error: bs is not installed correctly.` / Windows `The system cannot find the path specified.` | postinstall never ran or the download failed — reinstall without `--ignore-scripts` (e.g. `npm i -g --force buffsw-cli`), check the release tag has the assets (see Release binaries), or run `node postinstall.mjs` inside the installed package |
| Install from a git clone/CI fails | `dist/` is gitignored and the release may not have the assets yet — build the binary for your OS into `dist/` and upload it to the release (see Release binaries) |

---

## License

MIT.
