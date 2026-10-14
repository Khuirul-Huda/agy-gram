# AGY-Gram 🤖⚡

[English](README.md) | [Bahasa Indonesia](README.id.md)

**AGY-Gram** is a lightweight, high-performance bridge service written in **Golang** designed to control the **Google Antigravity CLI (`agy`)** directly from **Telegram**.

Architected around **Clean Architecture** and **SOLID** principles, AGY-Gram operates with an ultra-low memory footprint (~10-15 MB RAM). This makes it ideal for running on low-power home servers (such as Intel Celeron or single-board computers) without competing for system resources.

---

## 🏗️ Architecture & Clean Principles

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

- **Separation of Concerns & Hexagonal Architecture:** The core domain logic has zero dependencies on Telegram libraries, CLI frameworks, or external infrastructure.
- **Process Group Isolation:** Spawns child processes with `Setpgid: true` so cancellation (`/stop`) or timeouts terminate the entire process group cleanly without orphaned zombie processes.
- **Concurrency Safety:** Mutex synchronization prevents overlapping executions, protecting constrained multi-threaded CPUs from thrashing.
- **Smart Chunking:** Intelligently splits agent responses across Telegram's 4096-character limit without breaking code blocks or words awkwardly.
- **Typing Heartbeat & Fast ACK:** Sends an immediate placeholder reply and continuous typing indicator while the agent reasons and executes tasks to prevent timeouts.
- **Interactive Buttons (`[CONFIRM:]` & `[BUTTONS:]`):** When the AI outputs `[CONFIRM: Yes | No]` or `[BUTTONS: Opt1 | Opt2]`, AGY-Gram converts them into clickable Telegram Inline Keyboard buttons.
- **Native Command Menu:** Automatically registers commands on startup via `SetCommands` so Telegram displays the native "Menu" button.

---

## 📋 Telegram Commands

| Command | Description |
| :--- | :--- |
| **`/start`** | Welcome greeting and bot connectivity verification. |
| **`/help`** | Full command reference and usage guide. |
| **`/new`** | Reset session context (starts a fresh session without the `-c` continue flag). |
| **`/status`** | Real-time host hardware metrics (CPU, RAM, ZRAM Swap, SSD) and agent state. |
| **`/mode [auto\|plan\|accept-edits]`** | Inspect or switch Antigravity execution mode. |
| **`/model [model_name]`** | Interactive button menu or text override for AI models. |
| **`/effort [level]`** | Interactive button menu or text override for reasoning effort (`low`, `medium`, `high`, `xhigh`, `max`). |
| **`/thinking [on\|off]`** | Inspect or toggle verbose AI reasoning stream to Telegram. |
| **`/stop`** or **`/cancel`** | Terminate any active agent process group immediately. |
| *Plain Text Message* | Any text message is forwarded as an interactive prompt to `agy`. |

---

## ⚙️ Configuration (`.env`)

Copy the configuration template:

```bash
cp .env.example .env
nano .env
```

Configure the environment variables:

```ini
# Telegram bot token obtained from @BotFather
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ

# Authorized Telegram User IDs (comma-separated for multiple users)
# Requests from any ID not in this list are rejected.
ALLOWED_USER_IDS=123456789

# Path to the Antigravity binary (default: agy)
AGY_BIN_PATH=agy

# Working directory where agy commands will execute (e.g. /home/username/workspace or ./)
AGY_WORKING_DIR=./

# Execution timeout per prompt turn (in minutes, default: 5)
AGY_TIMEOUT_MINUTES=5

# Default execution mode (auto, plan, accept-edits)
AGY_DEFAULT_MODE=auto

# Default AI model override (optional, leave empty for CLI default)
AGY_DEFAULT_MODEL=

# Enable verbose thinking stream by default (true/false, default: false)
AGY_VERBOSE_THINKING=false
```

> [!IMPORTANT]
> **Finding your Telegram User ID:**  
> Message `@userinfobot` on Telegram to obtain your numeric account ID.

---

## 🚀 Building & Running (Production with PM2)

### 1. Build Binary

```bash
make build
```
Generates a standalone, static ELF binary in `bin/agy-gram` (~6 MB).

### 2. Run Daemon with PM2

```bash
# Start background daemon via PM2
make pm2-start
# or: pm2 start ecosystem.config.js

# Check status and memory consumption
pm2 status

# Tail real-time logs
make pm2-logs
# or: pm2 logs agy-gram

# Save PM2 process list across system reboots
pm2 save
```

### 3. Makefile Targets

- `make test` : Execute automated unit tests.
- `make clean` : Remove build binaries.
- `make pm2-restart` : Recompile binary and reload PM2 service.
- `make pm2-stop` : Stop the PM2 service.

---

## 🔒 Security Hardening

1. **User Whitelist**: Strictly enforces allowed Telegram User IDs at the middleware level.
2. **Outbound Long-Polling**: Connects to Telegram via HTTPS outbound long-polling only; **no open inbound ports or public IP exposure required**.
3. **Graceful Teardown**: Listens for `SIGINT` / `SIGTERM` signals to cleanly abort active child processes before shutting down.

---

## ⚠️ Disclaimer

This project is an independent open-source tool developed strictly for educational, research, and personal automation purposes. It is **not** officially affiliated with, endorsed by, sponsored by, or associated with Google, Antigravity, Telegram, or any of their affiliates or subsidiaries. All product names, logos, brands, and registered trademarks belong to their respective owners.
