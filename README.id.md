# buffsw-cli — account selector `bs`

> [English](README.md) · **Bahasa Indonesia**

TUI pemilih akun untuk **Freebuff**, ditulis dalam Go memakai
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

`bs` mengelola banyak akun Freebuff yang tersimpan di
`~/.config/manicode/credentials.json`. Freebuff **hanya membaca key
`default`** di file itu — jadi tugas `bs` hanyalah menukar isi key
`default` dengan akun yang kamu pilih, lalu memarkir akun lama ke slot-nya
masing-masing.

---

## Quick Start

Pasang package secara global dari npm, lalu jalankan `bs`:

```bash
npm i -g buffsw-cli
bs
```

> **Mode dev** — jalankan langsung dari source (butuh Go 1.27+ dan
> terminal interaktif):
>
> ```bash
> cd ~/projects/buffswitch
> go run .
> ```

Di dalam TUI:

```text
↑/↓ atau j/k pilih · Enter aktifkan · a tambah · d hapus · r reload · q keluar
```

- **Switch akun** — pilih akun, tekan `Enter`.
- **Tambah akun** — tekan `a`: TUI menyingkir dan menyerahkan terminal ke
  `freebuff login`; output-nya — termasuk URL login — tampil langsung di
  terminal. Selesaikan login di sana, lalu tekan Enter untuk kembali ke
  daftar akun yang sudah diperbarui.
- File credentials lain? `bs -creds /path/credentials.json`

Detail selengkapnya di bawah.

---

## Daftar Isi

- [Fitur](#fitur)
- [Bagaimana cara kerjanya](#bagaimana-cara-kerjanya)
- [Struktur file](#struktur-file)
- [Persyaratan](#persyaratan)
- [Penggunaan](#penggunaan)
- [Alur tambah akun (add account)](#alur-tambah-akun-add-account)
- [Format credentials.json](#format-credentialsjson)
- [Pengembangan (dev)](#pengembangan-dev)
- [Pemecahan masalah](#pemecahan-masalah)

---

## Fitur

- **Daftar akun** — semua akun Freebuff yang tersimpan, akun aktif ditandai
  `(aktif)`, dalam jendela 80-kolom (otomatis menyusut di terminal sempit).
- **Switch akun (`Enter`)** — akun pilihan disalin ke key `default`; akun
  yang tadinya aktif diparkir ke slot key **email**-nya.
- **Tambah akun (`a`)** — menjalankan binary resmi `freebuff login` dengan
  menyerahkan terminal, jadi output & URL login tampil langsung di sana;
  setelah selesai, `bs` kembali ke TUI dengan akun baru aktif.
- **Hapus akun (`d`)** — menghapus akun yang dipilih (akun parkir atau akun
  aktif) dengan konfirmasi `y/n`; kalau itu akun aktif, key `default` ikut
  dibersihkan.
- **Muat ulang (`r`)** — baca ulang `credentials.json` dari disk (berguna
  kalau kamu login manual di luar `bs`).
- **Normalisasi otomatis** — key acak/liar hasil login (mis. `defaults`)
  dirapikan menjadi key berdasarkan email; file tidak pernah rusak.
- **Sinkronisasi slot** — sesi yang di-refresh (token baru untuk email yang
  sama) ikut disalin ke slot parkir.
- Aman: penulisan atomik (temp + rename) dan mode file asli dipertahankan.

---

## Bagaimana cara kerjanya

`credentials.json` berbentuk peta JSON. Kunci yang berarti bagi Freebuff
hanya satu:

```text
"default": { ... akun yang sedang aktif ... }
```

Semua kunci lain **tidak dibaca Freebuff** — itu slot parkir milik `bs`.
`bs` memakai **email akun sebagai kunci slot**, sehingga setiap akun mudah
dicari dan tidak mungkin tertukar.

Operasi inti:

| Operasi | Yang terjadi pada file |
|---|---|
| **Switch ke B** | isi `default` (akun A) disalin ke slot email A; isi slot email B disalin ke `default` |
| **Sebelum login (parkir)** | `default` saat ini disalin ke slot email-nya dulu, supaya tidak hilang saat `freebuff login` menimpa `default` |
| **Setelah login** | file dibaca ulang, key liar dirapikan ke email, akun baru yang muncul dijadikan `default` (atau sesi email sama di-refresh), slot ikut disinkronkan |

Invariant yang dijaga setelah setiap `Save`:

```text
{
  "default":     <akun aktif>,
  "<email-1>":   <akun parkir 1>,
  "<email-2>":   <akun parkir 2>,
  ...
}
```

---

## Struktur file

```text
buffswitch/            repo; nama package npm-nya `buffsw-cli`
├── go.mod            module bs (bubbletea + lipgloss)
├── main.go           TUI Bubble Tea: daftar akun, keymap, loop login
├── store.go          logika credentials.json: load/save/normalisasi/
│                     switch/parkir/ingest setelah login
├── login.go          hand-off terminal: menjalankan `freebuff login`,
│                     ingest hasil, kelola file state login sementara
├── store_test.go     unit test logika store
├── package.json      wrapper npm: `bin.bs` → `bin/bs.exe`
├── postinstall.mjs   salin binary platform yang cocok → `bin/bs.exe`
├── bin/
│   └── bs.exe        placeholder (`echo`); diganti postinstall
└── platform/          binary hasil build, satu per OS/arch
    └── buffsw-cli-<os>-<arch>/   binary
```

### Alur saat menekan `a`

```text
[a] ditekan
  → ParkDefault() + Save()          (akun aktif diparkir dulu)
  → snapshot email/token → file state sementara (dibersihkan otomatis)
  → TUI keluar (alt-screen mati)
  → `freebuff login` jalan di terminalmu (URL tampil di sana)
  → CLI selesai setelah login di browser
  → credentials.json dibaca ulang → IngestAfterLogin(pre) → Save()
  → tekan Enter → TUI buka lagi dengan daftar akun terbaru
```

---

## Persyaratan

- **Untuk instalasi & pakai:** Node.js/npm (untuk `npm i -g buffsw-cli`)
- **Mode dev / bangun dari source:** Go 1.27+ (module `bs` ditulis dengan Go 1.27)
- Binary Freebuff di direktori config manicode (bisa dioverride lewat env
  `FREEBUFF_BIN`)
- File credentials `credentials.json` di direktori config manicode (bisa
  dioverride lewat flag `-creds`)
- Terminal interaktif (TUI memakai alt-screen)

---

## Penggunaan

| Tombol | Aksi |
|---|---|
| `↑` / `↓` atau `j` / `k` | pilih akun |
| `Enter` | aktifkan akun yang dipilih (switch `default`) |
| `a` | tambah akun: keluar dari TUI, jalankan `freebuff login` di terminal, lalu kembali |
| `r` | muat ulang daftar dari disk |
| `d` | hapus akun yang dipilih (konfirmasi dengan `y`) |
| `q` / `Ctrl+C` | keluar |

---

## Alur tambah akun (add account)

1. Tekan `a` di daftar akun.
2. `bs` memarkir akun aktif ke slot email-nya (data tidak hilang) dan
   menyimpan snapshot akun saat ini ke file state sementara.
3. TUI menutup dan `freebuff login` mengambil alih terminal. Output-nya —
   termasuk URL login — tampil langsung di sana. Salin URL, buka di
   browser, dan selesaikan login. Tidak ada yang dibuka otomatis.
4. Saat CLI selesai, `bs` membaca ulang `credentials.json`, lalu:
   - akun **baru** (email belum ada) → dijadikan `default`;
   - email **sama** dengan akun lama → sesi di-refresh (token baru ikut
     disalin ke slot);
   - tidak ada yang berubah (login dibatalkan) → `default` dibiarkan utuh.
5. Hasilnya dicetak satu baris (mis. `Akun baru aktif: ...`). Tekan
   **Enter** untuk membuka kembali TUI dengan daftar akun terbaru.

File state sementara dibersihkan otomatis. `bs` tidak pernah mencoba
membuka browser atau menangkap sendiri output login — `freebuff login`
memiliki terminalnya, cara paling andal untuk menjalankan login
interaktifnya.

---

## Format credentials.json

Contoh struktur setelah dikelola `bs` — **nilai di bawah hanya placeholder,
bukan data asli**:

```json
{
  "default": {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "Nama Akun Aktif",
    "email": "akun-utama@contoh.com",
    "authToken": "auth-token-rahasia",
    "fingerprintId": "enhanced-xxxxxxxx",
    "fingerprintHash": "hash-xxxxxxxx"
  },
  "akun-lain@contoh.com": {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "Nama Akun Lain",
    "email": "akun-lain@contoh.com",
    "authToken": "auth-token-rahasia",
    "fingerprintId": "enhanced-xxxxxxxx",
    "fingerprintHash": "hash-xxxxxxxx"
  }
}
```

Aturan yang dijaga `bs`:

- Key selain `default` selalu **email akun**.
- Key liar hasil login (mis. `defaults`) otomatis dirapikan saat `Save`.
- `default` selalu ada salinan parkirnya di slot email yang sama.
- File ditulis atomik (`.tmp` + rename) dengan mode file asli (default
  0600 kalau file belum ada).
- Isi field akun (`authToken`, `fingerprint*`) tidak pernah diubah isinya —
  hanya dipindah antar key.

---

## Pengembangan (dev)

> Mode dev menjalankan source Go secara langsung, bukan binary hasil
> instalasi:

```bash
cd ~/projects/buffswitch

go run .                                   # mode dev: jalankan TUI (butuh TTY)

gofmt -l .                                 # cek format (kosong = rapi)

"$(go env GOBIN)/golangci-lint" run ./...  # lint (standar: 0 issues)
```

**Lint adalah satu-satunya perintah tes/verifikasi yang dipakai project
ini** (golangci-lint v2, diinstal via
`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).
`go vet`/`go test` tidak dipakai dalam alur dev.

> Unit test logika store tersedia di `store_test.go` dan bisa dijalankan
> kapan pun dengan `go test ./...` — tetapi bukan bagian dari alur dev
> standar.

### Rilis binary (multi-OS)

Semua binary platform dibundel dalam satu package ini (folder `platform/`).
Saat install, `postinstall.mjs` memilih yang cocok dengan OS/arch pengguna
lalu menyalinnya ke `bin/bs.exe` (nama target statis — Linux/macOS
mengabaikan ekstensi `.exe`; yang penting bit eksekusi `0o755`), dan
memastikan ia bisa dijalankan. Hanya `buffsw-cli` yang di-publish —
tidak ada package platform terpisah.

Build semua platform ke `platform/` sebelum publish:

```bash
for t in "linux amd64 platform/buffsw-cli-linux-x64/bs" \
         "linux arm64 platform/buffsw-cli-linux-arm64/bs" \
         "linux arm platform/buffsw-cli-linux-arm/bs" \
         "linux 386 platform/buffsw-cli-linux-386/bs" \
         "darwin amd64 platform/buffsw-cli-darwin-x64/bs" \
         "darwin arm64 platform/buffsw-cli-darwin-arm64/bs" \
         "windows amd64 platform/buffsw-cli-win32-x64/bs.exe" \
         "windows arm64 platform/buffsw-cli-win32-arm64/bs.exe"; do
  set -- $t
  GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -trimpath -o "$3" .
done
```

`CGO_ENABLED=0` menghasilkan binary **statis**, jadi satu build Linux berjalan
baik di **glibc** maupun **musl**. Binary di-gitignore (`platform/*/bs*`)
tetapi masuk tarball lewat `files: ["platform"]`.

Kecilkan binary Linux dan Windows-x64 dengan UPX (darwin dan win-arm64
tidak didukung UPX):

```bash
upx --best --lzma platform/buffsw-cli-linux-*/*/bs platform/buffsw-cli-win32-x64/bs.exe
```

Peta file untuk pengembangan:

| File | Urusan |
|---|---|
| `main.go` | UI: model Bubble Tea, keymap, render daftar, loop login |
| `store.go` | Aturan data: `Load`, `Save`, `normalize`, `SwitchTo`, `ParkDefault`, `Delete`, `IngestAfterLogin` |
| `login.go` | Hand-off terminal: menjalankan `freebuff login`, ingest hasil, kelola file state login sementara |

---

## Pemecahan masalah

| Gejala | Solusi |
|---|---|
| `Gagal membaca ... : expected a JSON object` | `credentials.json` rusak/tidak valid — periksa JSON-nya |
| `Gagal menjalankan freebuff login` | cek `FREEBUFF_BIN` atau apakah binary `freebuff` ada |
| `State login tidak ditemukan` | file state sementara terhapus saat `bs` ditutup — tekan `r` setelah TUI terbuka untuk memuat ulang |
| Akun tidak muncul di daftar | tekan `r` untuk muat ulang dari disk |
| Login selesai tapi "Tidak ada akun baru" | kamu login dengan email yang sudah terdaftar — sesi di-refresh, bukan akun baru |
| Tampilan berantakan di terminal sempit | jendela otomatis menyusut; minimal lebar ~20 kolom |

---

## Lisensi

MIT.
