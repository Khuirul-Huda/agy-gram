# AGY-Gram 🤖⚡

<p align="center">
  <strong>Bridge Telegram Ultra-Ringan & Berkinerja Tinggi Khusus Google Antigravity CLI (<code>agy</code>)</strong><br>
  Kendalikan AI coding agent langsung dari ponsel via teks atau pesan suara, pantau progres eksekusi tool secara in-place, dan kelola multi-project dengan rapi.
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Versi Go" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/Lisensi-MIT-blue.svg" alt="Lisensi" /></a>
  <img src="https://img.shields.io/badge/RAM_Footprint-~10--15_MB-success?style=flat" alt="Konsumsi RAM Ringan" />
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey?style=flat" alt="Platform" />
  <a href="https://core.telegram.org/bots"><img src="https://img.shields.io/badge/Telegram_Bot-Telebot.v3-2CA5E0?style=flat&logo=telegram" alt="Bot Telegram" /></a>
  <img src="https://img.shields.io/badge/Companion_for-Google_Antigravity-4285F4?style=flat&logo=google" alt="Google Antigravity" />
</p>

<p align="center">
  <a href="README.md"><strong>English</strong></a> · <a href="README.id.md"><strong>Bahasa Indonesia</strong></a>
</p>

---

## 🌟 Mengapa Memilih AGY-Gram?

Sebagian besar bridge coding agent lain dibangun menggunakan stack runtime yang berat (Python atau Node.js) dan memakan memori 100–250+ MB RAM hanya untuk kondisi *idle*. **AGY-Gram** ditulis murni menggunakan **Golang** berlandaskan prinsip **Clean Architecture** dan **SOLID**, dengan penggunaan memori yang luar biasa hemat: **hanya ~10–15 MB RAM**.

Proyek ini dirancang secara spesifik sebagai jembatan terbaik bagi **Google Antigravity CLI (`agy`)**, sangat cocok dijalankan di home server hemat daya (seperti prosesor Intel Celeron atau single-board computer) serta VPS murah tanpa membebani resource perangkat keras Anda.

---

## ⚡ Fitur Utama

* 🎙️ **Audio Multimodal & Voice Note Native (Tanpa Whisper Eksternal):**
  * Kirim instruksi rekaman suara (`.oga`/`.ogg`) atau file audio langsung di Telegram.
  * Memanfaatkan pemahaman audio multimodal bawaan Google Gemini tanpa perlu latensi transkripsi STT pihak ketiga atau model Whisper lokal yang membebani CPU.
* 📡 **Live In-Place Progress & Inline Thinking:**
  * Pembaruan progres cerdas dalam satu pesan tanpa membanjiri riwayat obrolan (*zero spam*).
  * Status real-time saat agent melakukan `read_file`, `edit_file`, `write_file`, maupun perintah shell (`run_command`).
  * Proses penalaran (*thinking*) AI ditampilkan di dalam kutipan Telegram yang rapi dan bisa diperluas (`<blockquote expandable>`).
  * Begitu selesai, kartu progres langsung bertransformasi menjadi balasan akhir.
* 👥 **Dukungan Telegram Forum Topics:**
  * Mendukung fitur Topic di Telegram Supergroup.
  * Isolasi percakapan otomatis: setiap topik memiliki memori percakapan, direktori workspace, dan pengaturan model AI tersendiri yang terpisah secara independen.
* 🔗 **Konverter Cerdas Tautan File Markdown:**
  * Mengonversi skema tautan `file://` dan simbol kode menjadi format monospaced Telegram (`<code>nama_file.go</code>`), menghapus tampilan markdown mentah yang rusak.
* 🔘 **Tombol Konfirmasi & Pilihan Interaktif:**
  * Otomatis menerjemahkan pola respons AI seperti `[CONFIRM: Ya | Tidak]` atau `[BUTTONS: Opsi 1 | Opsi 2]` menjadi tombol Telegram Inline Keyboard yang dapat diklik.
* 📊 **Pemantau Telemetri Server (`/status`):**
  * Metrik hardware real-time: beban CPU, penggunaan memori RAM, ZRAM swap, kapasitas SSD, dan status proses agent.
* ⏳ **Pemeriksaan Kuota Token & Rate Limit (`/usage`):**
  * Pantau sisa kuota model Antigravity dan jadwal reset limit mingguan secara langsung.
* 🛡️ **Isolasi Process Group Aman:**
  * Menjalankan `agy` dengan isolasi `Setpgid: true`. Pembatalan via `/stop` atau batas waktu timeout akan mematikan seluruh proses turunan tanpa meninggalkan *zombie process*.

---

## 🏗️ Arsitektur Bersih (Hexagonal)

```text
       Telegram Long-Polling
                 ↓
      ┌─────────────────────┐
      │   Telegram Bot UI   │  (internal/bot)
      │  (Auth & Rendering) │
      └──────────┬──────────┘
                 │ memanggil interface domain
                 ▼
      ┌─────────────────────┐
      │     Domain Core     │  (internal/domain)
      │ (Models & Contracts)│  [Nol dependensi eksternal]
      └──────────┬──────────┘
                 │ diimplementasikan oleh adapter
       ┌─────────┴─────────┐
       ▼                   ▼
┌──────────────┐   ┌──────────────┐
│ Agent Runner │   │System Monitor│
│(internal/agt)│   │(internal/sys)│
└──────┬───────┘   └──────────────┘
       ▼
Antigravity CLI
(`agy -p ...`)
```

- **Pemisahan Tanggung Jawab:** Logika inti domain tidak bergantung pada library Telegram, CLI, maupun database tertentu.
- **Penyimpanan Tersemat:** Riwayat sesi dan pengaturan workspace tersimpan aman menggunakan database embedded key-value `bbolt`.
- **Keamanan Jaringan:** Menggunakan koneksi outbound HTTPS long-polling. **Tidak memerlukan port publik terbuka, sertifikat webhook, maupun IP publik**.

---

## 📋 Daftar Perintah Telegram

| Perintah | Deskripsi |
| :--- | :--- |
| **`/start`** / **`/menu`** | Membuka panel kontrol utama interaktif. |
| **`/help`** | Menampilkan panduan lengkap penggunaan bot. |
| **`/new`** | Mereset memori sesi aktif dan memulai percakapan baru. |
| **`/status`** | Cek kesehatan hardware host (CPU, RAM, ZRAM, SSD) dan status daemon. |
| **`/usage`** | Cek kuota token Antigravity dan jadwal reset model. |
| **`/mode [auto\|plan\|accept-edits]`** | Beralih mode kerja (`auto`, `plan`, atau `accept-edits`). |
| **`/model [nama]`** | Memilih model AI via tombol interaktif atau teks langsung. |
| **`/effort [level]`** | Mengatur tingkat reasoning effort (`low`, `medium`, `high`, `xhigh`, `max`). |
| **`/thinking [on\|off]`** | Menyalakan atau mematikan stream penalaran verbose AI. |
| **`/workspace [path]`** | Mengganti target folder kerja tempat `agy` beroperasi. |
| **`/sessions`** | Melihat riwayat dan beralih ke sesi percakapan terdahulu. |
| **`/restart`** | Menjadwalkan restart aman untuk service background AGY-Gram. |
| **`/stop`** / **`/cancel`** | Menghentikan paksa proses agent yang sedang berjalan. |
| *Pesan Teks Biasa* | Diteruskan langsung sebagai instruksi prompt ke Antigravity CLI. |
| *Pesan Suara / Audio* | Diteruskan langsung ke Antigravity CLI via kapabilitas audio multimodal. |
| *Foto / Dokumen* | Diunduh ke `<workspace>/uploads` dan dianalisis secara visual oleh `agy`. |

---

## ⚙️ Konfigurasi (`.env`)

1. Salin template file konfigurasi:
   ```bash
   cp .env.example .env
   nano .env
   ```

2. Sesuaikan variabel lingkungan:
   ```ini
   # Token bot Telegram yang diperoleh dari @BotFather
   TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ

   # ID Pengguna Telegram yang diberi izin akses (pisahkan dengan koma)
   ALLOWED_USER_IDS=123456789

   # Path binary Antigravity (default: agy)
   AGY_BIN_PATH=agy

   # Direktori kerja default saat agy mengeksekusi instruksi
   AGY_WORKING_DIR=./

   # Batas waktu eksekusi per instruksi dalam menit (0 untuk tanpa batas, default: 5)
   AGY_TIMEOUT_MINUTES=5

   # Mode eksekusi default (auto, plan, accept-edits)
   AGY_DEFAULT_MODE=auto

   # Model AI default (opsional, kosongkan untuk memakai default CLI)
   AGY_DEFAULT_MODEL=

   # Tampilkan proses berpikir AI secara default (true/false, default: false)
   AGY_VERBOSE_THINKING=false

   # Proxy outbound opsional untuk Telegram API
   TELEGRAM_PROXY=
   ```

> [!TIP]
> **Cara Menemukan User ID Telegram:** Kirim pesan ke bot [`@userinfobot`](https://t.me/userinfobot) di Telegram untuk melihat ID numerik akun Anda.

---

## 🚀 Panduan Build & Menjalankan (Production)

### Prasyarat
* **Go 1.22+** terpasang di sistem.
* **Google Antigravity CLI (`agy`)** sudah terinstal dan terautentikasi pada `PATH`.

### 1. Kompilasi Binary

```bash
make build
```
Menghasilkan file binary statis di `bin/agy-gram` (~6–7 MB).

### 2. Jalankan sebagai Daemon Menggunakan PM2

```bash
# Menjalankan daemon di background via PM2
make pm2-start

# Memeriksa status dan pemakaian RAM (~10 MB)
pm2 status

# Melihat log aktivitas secara real-time
make pm2-logs

# Menyimpan konfigurasi proses agar otomatis aktif setelah server reboot
pm2 save
```

### 3. Perintah Makefile

* `make test` : Menjalankan automated unit test untuk seluruh paket.
* `make build` : Mengompilasi ulang binary.
* `make pm2-restart` : Kompilasi ulang dan me-reload service PM2.
* `make pm2-stop` : Menghentikan service PM2.
* `make clean` : Membersihkan file build.

---

## 🔒 Pengerasan Keamanan (Security)

1. **Whitelist ID Pengguna:** Hanya akun yang terdaftar pada `ALLOWED_USER_IDS` yang dapat mengirim perintah ke bot.
2. **Koneksi Outbound Saja:** Tidak membuka port incoming sama sekali, aman di balik NAT dan firewall rumah.
3. **Isolasi Proxy:** Memastikan variabel proxy lokal tidak mengontaminasi eksekusi internal CLI Antigravity.

---

## 📄 Lisensi

Proyek ini dirilis di bawah lisensi terbuka [MIT License](LICENSE).

---

## ⚠️ Pernyataan Sangkalan (Disclaimer)

Proyek ini adalah alat sumber terbuka independen yang dikembangkan untuk tujuan riset, edukasi, dan otomasi pribadi. Proyek ini **tidak** berafiliasi resmi, didukung, disponsori, atau terkait dengan Google, Antigravity, Telegram, maupun anak perusahaan mereka.
