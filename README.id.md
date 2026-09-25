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
npm i -g buffswitch
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
↑/↓ atau j/k pilih · Enter aktifkan & jalankan freebuff · a tambah · d hapus · r reload · q keluar
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
  yang tadinya aktif diparkir ke slot key **email**-nya. Setelah itu `bs`
  **keluar dan langsung menjalankan CLI `freebuff`** dengan akun tersebut;
  saat `freebuff` selesai, `bs` pun selesai.
- **Statistik pemakaian** — tiap baris menampilkan seberapa sering & lama
  akun dipakai (`12x · 3j · 2j lalu`), dicatat di file terpisah
  `buffswitch-stats.json`; daftar diurutkan **paling jarang dipakai dulu**.
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
buffswitch/            repo; nama package npm-nya `buffswitch`
├── go.mod            module bs (bubbletea + lipgloss)
├── main.go           TUI Bubble Tea: daftar akun, keymap, loop login
├── store.go          logika credentials.json: load/save/normalisasi/
│                     switch/parkir/ingest setelah login
├── login.go          hand-off terminal: menjalankan `freebuff login`,
│                     ingest hasil, kelola file state login sementara
├── store_test.go     unit test logika store
├── stats.go          statistik pemakaian per akun (sesi, jumlah, terakhir
│                     dipakai) → buffswitch-stats.json di samping credentials.json
├── stats_test.go     unit test logika stats
├── package.json      wrapper npm: `bin.bs` → `bin/bs.exe` (tarball tidak
│                     membawa binary — cuma stub + postinstall)
├── postinstall.mjs   unduh binary yang cocok dengan OS/arch pengguna dari
│                     GitHub Release → `bin/bs.exe`
├── bin/
│   └── bs.exe        placeholder teks biasa; diganti postinstall
└── dist/              binary hasil build, satu per OS/arch (di-gitignore)
                      — upload ke GitHub Release dengan `gh`
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

- **Untuk instalasi & pakai:** Node.js/npm (untuk `npm i -g buffswitch`)
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
| `Enter` | aktifkan akun yang dipilih, keluar dari `bs`, lalu jalankan `freebuff` |
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

Package npm **tidak membawa binary sama sekali**. Binary tiap platform
hidup sebagai **asset GitHub Release** (repo `ozan-fn/buffswitch`, tag
`v<versi>`). Saat install, `postinstall.mjs` mengunduh **hanya satu yang
cocok dengan OS/arch pengguna** dan menyimpannya sebagai `bin/bs.exe` —
jadi pengguna mengunduh beberapa MB, tidak pernah kedelapan (~48 MB).
`bin/bs.exe` di tarball berupa stub teks biasa (tanpa shebang) yang hanya
mencetak error kalau postinstall tidak pernah berjalan — ia file, bukan
skrip shell, jadi Windows tidak akan pernah mencoba menjalankan `sh`.

Build semua platform ke `dist/` sebelum rilis:

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

`CGO_ENABLED=0` menghasilkan binary **statis**, jadi satu build Linux
berjalan baik di **glibc** maupun **musl**. Binary di-gitignore (`dist/`).

Opsional: kecilkan dengan UPX sebelum upload (Linux dan Windows-x64 saja;
darwin dan win-arm64 tidak didukung):

```bash
upx --best --lzma dist/buffsw-cli-linux-* dist/buffsw-cli-win32-x64.exe
```

Publish rilis (hanya satu package npm):

1. Pastikan tag release sama dengan versi package — naikkan `"version"`
   di `package.json`, lalu `git tag v0.1.1` (buat tag sesuai versi yang
   dirilis).
2. Upload binary hasil build ke release GitHub:
   `gh release upload v0.1.1 dist/*`
   (buat release dulu dengan `gh release create v0.1.1` kalau perlu).
3. Publish package npm: `npm publish`.

`postinstall.mjs` menyusun URL unduhan dari versi package
(`https://github.com/ozan-fn/buffswitch/releases/download/v0.1.1/buffsw-cli-linux-x64`),
jadi tag release dan versi package harus selalu cocok. URL bisa dioverride
per instalasi lewat env `BUFFSW_CLI_BINARY_URL` (mis. mirror).

Peta file untuk pengembangan:

| File | Urusan |
|---|---|
| `main.go` | UI: model Bubble Tea, keymap, render daftar, loop login |
| `store.go` | Aturan data: `Load`, `Save`, `normalize`, `SwitchTo`, `ParkDefault`, `Delete`, `IngestAfterLogin` |
| `stats.go` | Statistik pemakaian per akun: pelacakan sesi, jumlah, terakhir dipakai → `buffswitch-stats.json` |
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
| `Error: bs is not installed correctly.` / di Windows `The system cannot find the path specified.` | postinstall tidak pernah berjalan atau unduhan gagal — pasang ulang tanpa `--ignore-scripts` (mis. `npm i -g --force buffswitch`), cek asset release sudah di-upload (lihat Rilis binary), atau jalankan `node postinstall.mjs` di dalam folder package yang terpasang |
| Instal dari clone git/CI gagal | `dist/` di-gitignore dan asset release mungkin belum di-upload — build dulu binary untuk OS-mu ke `dist/` lalu upload ke release (lihat Rilis binary) |

---

## Lisensi

MIT.
