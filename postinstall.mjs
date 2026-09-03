#!/usr/bin/env node
// postinstall buffsw-cli.
//
// Semua binary platform dikirim dalam satu package (folder `platform/`).
// Skrip ini memilih binary yang cocok dengan OS/arch pengguna dan
// menyalinnya ke `bin/bs.exe` (target statis yang dipakai `bin.bs`),
// lalu memastikan ia bisa dijalankan.
//
// `bin/bs.exe` cuma nama target yang disamaratakan — untuk Linux/macOS
// ekstensi `.exe` tidak berarti apa-apa; yang penting bit eksekusi (0o755).

import { spawnSync } from "node:child_process";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chmodSync, copyFileSync, existsSync, mkdirSync, unlinkSync } from "node:fs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Target tempat binary diletakkan; nama ini sama untuk semua sistem operasi.
const targetBinary = path.join(__dirname, "bin", "bs.exe");

// `process.platform:process.arch` -> nama folder platform di `platform/`.
const PLATFORM_FOLDERS = {
  "linux:x64": "buffsw-cli-linux-x64",
  "linux:arm64": "buffsw-cli-linux-arm64",
  "linux:arm": "buffsw-cli-linux-arm",
  "linux:ia32": "buffsw-cli-linux-386",
  "darwin:x64": "buffsw-cli-darwin-x64",
  "darwin:arm64": "buffsw-cli-darwin-arm64",
  "win32:x64": "buffsw-cli-win32-x64",
  "win32:arm64": "buffsw-cli-win32-arm64",
};

const KEY = `${os.platform()}:${os.arch()}`;
const folder = PLATFORM_FOLDERS[KEY];

if (!folder) {
  console.error(`buffsw-cli: platform/arch tidak didukung (${KEY})`);
  process.exit(1);
}

// Nama file binary di dalam folder platform.
const binName = folder.includes("win32") ? "bs.exe" : "bs";
const source = path.join(__dirname, "platform", folder, binName);

if (!existsSync(source)) {
  console.error(`buffsw-cli: binary untuk ${KEY} tidak ditemukan: ${source}`);
  process.exit(1);
}

mkdirSync(path.dirname(targetBinary), { recursive: true });
if (existsSync(targetBinary)) unlinkSync(targetBinary);
copyFileSync(source, targetBinary);
chmodSync(targetBinary, 0o755);

// Jalankan `--help`; berhasil = exit 0. (TUI butuh TTY, jadi --version tidak dipakai.)
const r = spawnSync(targetBinary, ["--help"], { stdio: "ignore", windowsHide: true });
if (r.status !== 0) {
  console.error(`buffsw-cli: binary ${targetBinary} tidak bisa dijalankan`);
  process.exit(1);
}

console.log(`buffsw-cli: terpasang → ${targetBinary}`);