#!/usr/bin/env node
// postinstall buffsw-cli.
//
// Package npm ini TIDAK membawa binary apa pun. Binary tiap platform
// di-upload ke GitHub Release (tag `v<versi package>`, repo
// ozan-fn/buffswitch). Skrip ini mengunduh SATU binary yang cocok dengan
// OS/arch pengguna dan menyimpannya sebagai `bin/bs.exe` (target statis
// yang dirujuk `bin.bs`), lalu memastikan ia bisa dijalankan.
//
// `bin/bs.exe` cuma nama target yang disamaratakan — untuk Linux/macOS
// ekstensi `.exe` tidak berarti apa-apa; yang penting bit eksekusi (0o755).

import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Repo GitHub tempat binary di-upload (lewati env BUFFSW_CLI_BINARY_URL
// untuk memakai URL lain, mis. mirror).
const REPO = "ozan-fn/buffswitch";

// `process.platform:process.arch` -> nama asset binary di GitHub Release.
// Asset dibuat oleh perintah build di README (folder `dist/`) lalu
// di-upload dengan `gh release upload v<versi> dist/*`.
const ASSETS = {
  "linux:x64": "buffsw-cli-linux-x64",
  "linux:arm64": "buffsw-cli-linux-arm64",
  "linux:arm": "buffsw-cli-linux-arm",
  "linux:ia32": "buffsw-cli-linux-386",
  "darwin:x64": "buffsw-cli-darwin-x64",
  "darwin:arm64": "buffsw-cli-darwin-arm64",
  "win32:x64": "buffsw-cli-win32-x64.exe",
  "win32:arm64": "buffsw-cli-win32-arm64.exe",
};

const packageJson = JSON.parse(fs.readFileSync(path.join(__dirname, "package.json"), "utf8"));

const targetBinary = path.join(__dirname, "bin", "bs.exe");

const key = `${os.platform()}:${os.arch()}`;
const asset = ASSETS[key];
if (!asset) {
  console.error(`buffsw-cli: platform/arch tidak didukung (${key})`);
  process.exit(1);
}

const version = packageJson.version;
const downloadUrl =
  process.env.BUFFSW_CLI_BINARY_URL ??
  `https://github.com/${REPO}/releases/download/v${version}/${asset}`;

async function main() {
  fs.mkdirSync(path.dirname(targetBinary), { recursive: true });

  // Unduh ke file sementara dulu, baru pindahkan — kalau gagal, stub
  // `bin/bs.exe` yang lama tidak ikut terhapus/rusak.
  const tmp = path.join(path.dirname(targetBinary), `.bs.exe-${process.pid}.tmp`);
  try {
    const res = await fetch(downloadUrl);
    if (!res.ok) {
      throw new Error(`HTTP ${res.status} ${res.statusText}`);
    }
    const buf = Buffer.from(await res.arrayBuffer());
    fs.writeFileSync(tmp, buf);
    fs.chmodSync(tmp, 0o755);
  } catch (err) {
    fs.rmSync(tmp, { force: true });
    console.error(
      `buffsw-cli: gagal mengunduh binary dari:\n  ${downloadUrl}\n` +
        `(${err.message})\n` +
        `Pastikan versi package (${version}) sama dengan tag release yang sudah di-upload\n` +
        `assetnya: gh release upload v${version} dist/*\n` +
        `atau jalankan manual: node postinstall.mjs`,
    );
    process.exit(1);
  }

  fs.renameSync(tmp, targetBinary);

  // `--help` dipakai karena TUI butuh TTY (flag Go keluar 0 untuk --help).
  const { spawnSync } = await import("node:child_process");
  const r = spawnSync(targetBinary, ["--help"], { stdio: "ignore", windowsHide: true });
  if (r.status !== 0) {
    console.error(`buffsw-cli: binary ${targetBinary} tidak bisa dijalankan`);
    process.exit(1);
  }

  console.log(`buffsw-cli: terpasang → ${targetBinary}`);
}

main().catch((err) => {
  console.error(`buffsw-cli: ${err.message}`);
  process.exit(1);
});
