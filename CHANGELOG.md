# Changelog

Format mengikuti [Keep a Changelog](https://keepachangelog.com/); versi
[SemVer](https://semver.org/).

## [0.1.4] — 2026-09-25

### Added
- **Langsung jalankan `freebuff`**: `Enter` mengaktifkan akun, lalu `bs`
  keluar dan langsung menjalankan CLI `freebuff` dengan akun tersebut;
  `bs` selesai saat `freebuff` selesai.
- **Statistik pemakaian** per akun (`buffswitch-stats.json` di samping
  `credentials.json`): jumlah pemakaian, total durasi sesi (Enter →
  `freebuff` exit), dan terakhir dipakai; ditampilkan per baris akun.
- **Urutan least-used**: daftar akun diurutkan paling jarang dipakai dulu.
- Flag `-creds` kini juga memindahkan lokasi `buffswitch-stats.json` dan
  file state login — sandbox testing tidak menyentuh config asli.

## [0.1.3] — 2026-09-05

- Rilis stabil sebelumnya (lihat README).
