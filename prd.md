# PRD — buffswitch (`bs`)

> Dokumen kebutuhan produk. Disusun dari hasil scan kode (main.go, store.go, login.go, postinstall.mjs, package.json, store_test.go) dan README.
> Versi package: 0.1.3 · Lisensi: MIT · Repo: `ozan-fn/buffswitch`

---

## 1. Ringkasan

`bs` adalah **TUI pemilih akun untuk Freebuff** yang berjalan di terminal, ditulis dalam Go (Bubble Tea + Lipgloss).

**Masalah:** Freebuff hanya membaca key `"default"` di `~/.config/manicode/credentials.json`. Pengguna dengan banyak akun harus mengedit file JSON manual untuk berpindah akun.

**Solusi:** `bs` menukar isi key `"default"` dengan akun pilihan, dan memarkir akun lama ke slot yang di-key dengan **email** akun (slot lain tidak pernah dibaca Freebuff).

```text
npm i -g buffswitch
bs
```

---

## 2. Pengguna & Use Case

| Pengguna | Use case |
|---|---|
| Pengguna Freebuff dengan ≥2 akun | Berpindah akun aktif dalam 2 tombol tanpa edit JSON |
| Pengguna baru (0 akun) | Login pertama langsung dari TUI |
| Sesuatu berubah di luar `bs` | Muat ulang daftar dari disk |

---

## 3. Fitur (v0.1.3 — sudah ada)

### 3.1 F1 — Daftar akun
- Semua akun unik (dedup by email), akun aktif ditandai `(aktif)`.
- Urutan: aktif dulu, lalu nama (case-insensitive), lalu email.
- Jendela scrolling jika daftar > tinggi terminal; lebar 80 kolom, auto-shrink (min 20) di terminal sempit.

### 3.2 F2 — Switch akun (`Enter` → langsung `freebuff`)
- Isi `"default"` (akun lama) disalin ke slot email-nya; isi slot akun pilihan disalin ke `"default"`.
- **Setelah switch, `bs` keluar dari TUI dan langsung menjalankan binary `freebuff`** (hand-off terminal, pola yang sama dengan flow login). Saat `freebuff` selesai, `bs` berhenti.
- Switch ke akun yang sudah aktif = tetap lanjut launch `freebuff` (tanpa menulis ulang file).

### 3.3 F3 — Tambah akun (`a`)
- Parkir akun aktif dulu, snapshot email+token ke file state sementara (`.buffswitch-login-state.json`, 0600, auto-clean).
- TUI keluar (alt-screen mati), terminal diserahkan ke `freebuff login` (binary resmi; override via env `FREEBUFF_BIN`). URL login tampil langsung di terminal — tidak ada auto-open browser.
- Setelah CLI selesai: file dibaca ulang, lalu rekonsiliasi:
  - akun **baru** → dijadikan `default`;
  - email **sama** → sesi di-refresh, token baru ikut disalin ke slot parkir;
  - tidak berubah (login dibatalkan) → `default` dibiarkan utuh.
- Tekan Enter → TUI relaunch dengan daftar terbaru (loop di `main()`).

### 3.4 F4 — Hapus akun (`d`)
- Konfirmasi `y/n` (`n`/`Esc`/`N` batal). Menghapus slot email; jika akun yang dihapus adalah aktif, key `"default"` ikut dihapus.

### 3.5 F5 — Reload (`r`)
- Baca ulang `credentials.json` dari disk. Kursor tetap di akun yang sama bila masih ada.

### 3.7 F7 — Statistik pemakaian (`buffswitch-stats.json`)
- File terpisah di `~/.config/manicode/buffswitch-stats.json` — format `credentials.json` tidak diubah.
- Per akun (key = email): `totalSeconds`, `count`, `lastUsed` (plus `startedAt` selama sesi terbuka).
- **Sesi** = dari `Enter` sampai proses `freebuff` selesai; sesi dibuka saat Enter (`Begin`) dan ditutup saat `freebuff` exit (`End`). Crash = durasi sesi itu hangus, sesi berikutnya mulai baru.
- Daftar TUI diurutkan **paling jarang dipakai dulu** (count asc → nama → email).
- Tiap baris akun menampilkan ringkasan: `12x · 3j · 2j lalu`.

### 3.8 F8 — Normalisasi & keamanan file (non-UI, selalu jalan)
- Key liar hasil login (mis. `defaults`) di-re-key berdasarkan email akun saat `Save`.
- Entry tanpa email dibuang (tidak bisa di-key).
- Penulisan **atomik**: tulis `.tmp` + rename; mode file asli dipertahankan (default 0600).
- Isi field akun (`authToken`, `fingerprint*`) **tidak pernah diubah** — hanya dipindah antar key.
- Setelah setiap `Save`, invariant file: `default` = akun aktif, sisanya slot email, dan `default` selalu punya salinan parkir yang tersinkron.

---

## 4. Out of Scope (Non-Goals)

- Mengedit isi field akun (token, fingerprint) secara manual.
- Membuka browser otomatis atau menangkap output login (CLI login punya terminal).
- Mengelola credentials untuk tool selain Freebuff.
- GUI / versi non-terminal.

---

## 5. Persyaratan Fungsional

| ID | Persyaratan |
|---|---|
| FR-1 | Tampilkan semua akun dari `credentials.json`, aktif ditandai. |
| FR-2 | `Enter` mengaktifkan akun terpilih (parkir lama → slot email). |
| FR-3 | `a` menjalankan `freebuff login` via hand-off terminal, lalu merekonsiliasi hasil (baru/refresh/batal). |
| FR-4 | `d` menghapus akun dengan konfirmasi; hapus `default` jika akun aktif. |
| FR-5 | `r` memuat ulang file dari disk tanpa keluar dari TUI. |
| FR-6 | File selalu dinormalisasi (email-keyed) dan ditulis atomik dengan mode asli. |
| FR-7 | File state login sementara dibersihkan otomatis setelah dipakai. |

## 6. Persyaratan Non-Fungsional

| ID | Persyaratan |
|---|---|
| NFR-1 | Keamanan: file credentials selalu mode 0600 (atau mode asli); state sementara 0600; token tidak pernah dimodifikasi/di-log. |
| NFR-2 | Ketahanan: penulisan atomik (temp+rename) — file tidak pernah setengah-tertulis; unduhan binary postinstall juga via file sementara. |
| NFR-3 | Portabilitas: 8 platform (linux x64/arm64/arm/386, darwin x64/arm64, win32 x64/arm64), binary statis `CGO_ENABLED=0` (satu build Linux jalan di glibc & musl). |
| NFR-4 | Ringan: npm tarball tanpa binary (stub saja); postinstall mengunduh **satu** binary sesuai OS/arch (beberapa MB, bukan ~48 MB). |
| NFR-5 | UX terminal: alt-screen, auto-shrink layout, minimal ~20 kolom. |
| NFR-6 | Kualitas: `gofmt` rapi; `go vet` bersih; golangci-lint v2 = 0 issues (baseline project). `go test ./...` tersedia untuk logika store + stats (15 test). |

---

## 7. Antarmuka

### 7.1 CLI
```text
bs [-creds /path/credentials.json]
```
- `-creds` — override path credentials (default `~/.config/manicode/credentials.json`).
- Env: `FREEBUFF_BIN` (path binary `freebuff`), `BUFFSW_CLI_BINARY_URL` (override URL unduhan postinstall).

### 7.2 Keymap TUI
| Tombol | Aksi |
|---|---|
| `↑`/`↓` atau `j`/`k` | pilih akun |
| `Enter` | aktifkan akun → `bs` exit → jalankan `freebuff` |
| `a` | tambah akun (login) |
| `d` | hapus akun (konfirmasi y/n) |
| `r` | reload dari disk |
| `q` / `Ctrl+C` | keluar |

### 7.3 Format `credentials.json` (invariant setelah `Save`)
```json
{
  "default": { "id", "name", "email", "authToken", "fingerprintId", "fingerprintHash" },
  "<email-1>": { ... },
  "<email-2>": { ... }
}
```
Key selain `default` = email akun (slot parkir, tidak dibaca Freebuff).

---

## 8. Distribusi & Rilis

1. Naikkan `"version"` di `package.json`, buat tag `v<versi>` — **tag harus sama dengan versi package** (URL unduhan postinstall disusun dari versi).
2. Build 8 platform ke `dist/` (loop `GOOS`/`GOARCH` di README; binary gitignored).
3. `gh release upload v<versi> dist/*`.
4. `npm publish` (package `buffswitch`, bin `bs` → `bin/bs.exe`; tarball hanya stub + postinstall).
5. `postinstall.mjs` mengunduh binary sesuai `process.platform:process.arch`, chmod 0755, verifikasi dengan `--help`.

---

## 9. Verifikasi

- `gofmt -l .` → kosong.
- golangci-lint v2 → 0 issues (satu-satunya perintah verifikasi standar project).
- `go test ./...` → unit test logika store: load stray keys, switch round-trip, switch/delete email tak dikenal, ingest (akun baru / stray key / login batal / refresh token / login pertama), delete (parkir/aktif), file hilang, mode file dipertahankan, file tersimpan valid & email-keyed.
- Unit test stats: file hilang → zero usage; sesi round-trip (Begin ×2, reload, End mengakumulasi durasi); `End` tanpa sesi = no-op.

---

## 10. Roadmap (kandidat, belum ada di kode)

- Kandidat: mode non-interaktif (`bs -list`, `bs -switch email@x.com`), edit/rename nama akun, pencarian/filter daftar, file lock, backup `.bak` per Save, copy email ke clipboard, flag `--version`.
- Belum ada issue tracker/CI di repo — pipeline rilis masih manual via `gh`.
