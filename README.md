# AGY-Gram 🤖⚡

**AGY-Gram** adalah jembatan (*bridge service*) ringan dan efisien berbasis **Golang** untuk mengontrol **Google Antigravity CLI (`agy`)** langsung dari aplikasi **Telegram**.

Dirancang dengan prinsip arsitektur bersih (*Clean Principles* / *Clean Architecture* & *SOLID*), AGY-Gram memiliki jejak memori sangat rendah (~10-15 MB RAM), menjadikannya solusi ideal untuk server berdaya rendah seperti Intel Celeron / single-board computer tanpa membebani resource sistem.

---

## 🏗️ Arsitektur & Clean Principles

```text
       Telegram Long-Polling
                 ↓
      ┌─────────────────────┐
      │   Telegram Bot UI   │  (internal/bot)
      │  (Auth & Formatting)│
      └──────────┬──────────┘
                 │ calls domain interfaces
                 ▼
      ┌─────────────────────┐
      │     Domain Core     │  (internal/domain)
      │ (Models & Contracts)│  [Zero external dependencies]
      └──────────┬──────────┘
                 │ implemented by adapters
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

- **Separation of Concerns & Hexagonal Architecture:** Inti domain tidak memiliki dependensi terhadap framework Telegram maupun library eksternal.
- **Process Group Isolation:** Menjalankan child process dengan `Setpgid: true` sehingga pembatalan tugas (`/stop`) atau timeout membunuh seluruh grup proses tanpa meninggalkan *zombie process*.
- **Concurrency Safety:** Dilengkapi mutex lock agar eksekusi CLI tidak saling tumpang tindih pada CPU multi-thread terbatas.
- **Auto-Chunking:** Memecah teks output secara cerdas sesuai batas 4096 karakter Telegram tanpa memotong blok kode secara sembarangan.
- **Typing Heartbeat:** Mengirim indikator status `"typing..."` secara berkala ke Telegram selama agent memproses instruksi.

---

## 📋 Daftar Perintah Telegram

| Perintah | Deskripsi |
| :--- | :--- |
| **`/start`** | Pesan sambutan dan status koneksi bot. |
| **`/help`** | Panduan lengkap penggunaan dan daftar perintah. |
| **`/new`** | Reset konteks sesi perbincangan (memulai percakapan baru tanpa flag `-c`). |
| **`/status`** | Cek metrik hardware server (CPU, RAM, ZRAM, SSD) dan status agent. |
| **`/mode [auto\|plan\|accept-edits]`** | Cek atau ubah mode eksekusi Antigravity CLI. |
| **`/model [nama_model]`** | Cek atau atur model AI override (gunakan `/model reset` untuk kembali ke default). |
| **`/stop`** atau **`/cancel`** | Hentikan paksa tugas agent yang sedang berpikir atau berjalan. |
| *Pesan Teks Biasa* | Setiap pesan non-perintah otomatis dieksekusi sebagai prompt interaktif ke `agy`. |

---

## ⚙️ Konfigurasi (`.env`)

Salin template konfigurasi `.env.example` ke `.env`:

```bash
cp .env.example .env
nano .env
```

Isi variabel berikut:

```ini
# Token bot Telegram dari @BotFather
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ

# ID Telegram Anda (pisahkan dengan koma jika lebih dari satu)
# Pesan dari user selain ID ini otomatis ditolak demi keamanan server.
ALLOWED_USER_IDS=123456789

# Path binary Antigravity CLI (default: agy)
AGY_BIN_PATH=agy

# Working directory saat menjalankan agy
AGY_WORKING_DIR=/home/huda

# Batas waktu eksekusi per prompt (dalam menit, default: 5)
AGY_TIMEOUT_MINUTES=5

# Mode default (auto, plan, accept-edits)
AGY_DEFAULT_MODE=auto

# Model default (opsional, kosongkan untuk menggunakan default CLI)
AGY_DEFAULT_MODEL=
```

> [!IMPORTANT]
> **Cara Mengetahui Telegram User ID Anda:**  
> Kirim pesan ke bot `@userinfobot` di Telegram untuk melihat User ID numerik Anda.

---

## 🚀 Kompilasi & Menjalankan (Production dengan PM2)

### 1. Kompilasi Binary

```bash
make build
```
File binary ELF standalone akan dibuat di `bin/agy-gram` (ukuran ~6 MB).

### 2. Jalankan di Background via PM2

```bash
# Menjalankan daemon via PM2
make pm2-start
# atau: pm2 start ecosystem.config.js

# Cek status dan penggunaan RAM
pm2 status

# Melihat log streaming
make pm2-logs
# atau: pm2 logs agy-gram

# Simpan konfigurasi agar auto-start saat reboot
pm2 save
```

### 3. Perintah Tambahan (Makefile)

- `make test` : Menjalankan automated unit tests.
- `make clean` : Menghapus artefak build.
- `make pm2-restart` : Kompilasi ulang dan restart service PM2.
- `make pm2-stop` : Menghentikan service di PM2.

---

## 🔒 Keamanan (Security Hardening)

1. **Whitelist Middleware**: Bot tidak akan memproses atau merespons instruksi dari User ID yang tidak terdaftar di `ALLOWED_USER_IDS`.
2. **Localhost Only**: Menggunakan Telegram Long-Polling via HTTPS keluar (`outbound-only`), sehingga **tidak perlu membuka port router/firewall UFW** ke internet publik.
3. **Graceful Shutdown**: Menangani sinyal `SIGINT` dan `SIGTERM` secara bersih, memastikan semua sub-proses dan koneksi ditutup dengan tertib.
