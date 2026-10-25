# AGY-Gram 🤖⚡

<p align="center">
  <strong>The Ultra-Lightweight Telegram Bridge for Google Antigravity CLI (<code>agy</code>)</strong><br>
  Control your AI coding agent from your phone via text or voice, watch live tool progress in-place, and manage multiple projects seamlessly.
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License" /></a>
  <img src="https://img.shields.io/badge/RAM_Footprint-~10--15_MB-success?style=flat" alt="Low Memory Footprint" />
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey?style=flat" alt="Platforms" />
  <a href="https://core.telegram.org/bots"><img src="https://img.shields.io/badge/Telegram_Bot-Telebot.v3-2CA5E0?style=flat&logo=telegram" alt="Telegram Bot" /></a>
  <img src="https://img.shields.io/badge/Companion_for-Google_Antigravity-4285F4?style=flat&logo=google" alt="Google Antigravity" />
</p>

<p align="center">
  <a href="README.md"><strong>English</strong></a> · <a href="README.id.md"><strong>Bahasa Indonesia</strong></a>
</p>

---

## 🌟 Why AGY-Gram?

Most agent-to-chat bridges are built on heavy multi-runtime stacks (Python/Node) consuming 100–250+ MB of RAM just to idle. **AGY-Gram** is built strictly in **Golang** following **Clean Architecture** and **SOLID** principles, delivering an ultra-lean footprint of **only ~10–15 MB RAM**. 

It is purpose-built as a first-class companion for **Google Antigravity CLI (`agy`)**, optimized specifically for low-power home servers, Raspberry Pis, or cost-efficient VPS environments without competing for hardware resources.

---

## ⚡ Core Features

* 🎙️ **Direct Multimodal Audio & Voice Notes (Zero External Whisper):**
  * Send voice notes (`.oga`/`.ogg`) or audio recordings directly on Telegram.
  * Leverages Google Gemini's native audio multimodal understanding without needing local Whisper or external speech-to-text API latency.
* 📡 **Live In-Place Progress & Collapsible Thinking:**
  * Clean, single-message live updater that tracks agent activity in real time.
  * Real-time status for `read_file`, `edit_file`, `write_file`, and shell commands (`run_command`).
  * AI reasoning thoughts are rendered inside a sleek, expandable Telegram quote (`<blockquote expandable>`), keeping your chat clean and uncluttered.
  * Zero message spam — the live progress card seamlessly morphs into the final response upon completion.
* 👥 **Telegram Forum Topics Mode:**
  * Full support for Telegram Supergroup Forum Topics.
  * Multi-project isolation: each topic thread automatically retains its own conversation history, active workspace directory, and AI model settings.
* 🔗 **Smart Markdown & File Link Converter:**
  * Translates `file://` URIs and code symbols into clean monospaced Telegram elements (`<code>filename.go</code>`), eliminating raw, broken markdown syntax.
* 🔘 **Interactive Confirmation & Choice Buttons:**
  * Automatically turns agent prompts containing `[CONFIRM: Yes | No]` or `[BUTTONS: Option 1 | Option 2]` into clickable Telegram Inline Keyboard buttons.
* 📊 **Server & Telemetry Monitor (`/status`):**
  * Real-time hardware telemetry: CPU load, physical RAM, ZRAM swap, SSD disk usage, and agent daemon state.
* ⏳ **Token Quota & Rate Limit Inspection (`/usage`):**
  * Real-time inspection of your remaining Antigravity model quotas and weekly reset schedules.
* 🛡️ **Robust Process Group Isolation:**
  * Launches `agy` with POSIX `Setpgid: true` process group isolation. Cancelling via `/stop` or turn timeouts terminates all child processes cleanly with zero orphaned zombies.

---

## 🏗️ Architecture (Clean & Hexagonal)

```text
       Telegram Long-Polling
                 ↓
      ┌─────────────────────┐
      │   Telegram Bot UI   │  (internal/bot)
      │  (Auth & Rendering) │
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

- **Separation of Concerns:** Core domain contracts have zero external dependencies on Telegram, CLI, or storage libraries.
- **Embedded Persistence:** Session history, workspace mappings, and topic bindings persist across server reboots via embedded `bbolt` KV database.
- **Network Security:** Pure outbound HTTPS long-polling. Requires **no open inbound ports, no webhook certificates, and no public IP exposure**.

---

## 📋 Command Reference

| Command | Description |
| :--- | :--- |
| **`/start`** or **`/menu`** | Open the interactive graphical control panel dashboard. |
| **`/help`** | View complete user guide and syntax documentation. |
| **`/new`** | Reset active session memory and start a fresh conversation. |
| **`/status`** | Real-time host hardware metrics (CPU, RAM, ZRAM Swap, SSD) and runner state. |
| **`/usage`** | Check current Antigravity token quotas, model buckets, and reset windows. |
| **`/mode [auto\|plan\|accept-edits]`** | Switch execution mode (`auto`, `plan`, or `accept-edits`). |
| **`/model [name]`** | Interactive button selector or text override for AI models. |
| **`/effort [level]`** | Set reasoning effort (`low`, `medium`, `high`, `xhigh`, `max`). |
| **`/thinking [on\|off]`** | Inspect or toggle verbose AI reasoning stream. |
| **`/workspace [path]`** | Switch active working directory for `agy` execution. |
| **`/sessions`** | Browse and switch between previous conversation histories. |
| **`/restart`** | Perform a safe, graceful restart of the AGY-Gram daemon. |
| **`/stop`** or **`/cancel`** | Immediately kill active agent execution and child process tree. |
| *Plain Text Message* | Forwarded directly as a prompt turn to Antigravity CLI. |
| *Voice / Audio Message* | Forwarded directly to Antigravity CLI with native multimodal audio processing. |
| *Photo / Document* | Downloaded to `<workspace>/uploads` and analyzed visually by `agy`. |

---

## ⚙️ Configuration (`.env`)

1. Copy the environment configuration template:
   ```bash
   cp .env.example .env
   nano .env
   ```

2. Fill in your credentials:
   ```ini
   # Telegram bot token obtained from @BotFather
   TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ

   # Authorized Telegram User IDs (comma-separated for multiple users)
   # Requests from any ID not in this list are rejected.
   ALLOWED_USER_IDS=123456789

   # Path to the Antigravity binary (default: agy)
   AGY_BIN_PATH=agy

   # Working directory where agy commands will execute (e.g. /home/username or ./)
   AGY_WORKING_DIR=./

   # Execution timeout per prompt turn in minutes (0 for unlimited, default: 5)
   AGY_TIMEOUT_MINUTES=5

   # Default execution mode (auto, plan, accept-edits)
   AGY_DEFAULT_MODE=auto

   # Default AI model override (optional, leave empty for CLI default)
   AGY_DEFAULT_MODEL=

   # Enable verbose thinking stream by default (true/false, default: false)
   AGY_VERBOSE_THINKING=false

   # Optional outbound HTTP/HTTPS proxy for Telegram API (useful if Telegram is restricted)
   TELEGRAM_PROXY=
   ```

> [!TIP]
> **Find your Telegram User ID:** Message [`@userinfobot`](https://t.me/userinfobot) on Telegram to get your numeric ID.

---

## 🚀 Quick Start & Deployment

### Prerequisites
* **Go 1.22+** installed on host.
* **Google Antigravity CLI (`agy`)** installed and authenticated on the system `PATH`.

### 1. Build Standalone Binary

```bash
make build
```
This produces an optimized, static ELF binary in `bin/agy-gram` (~6–7 MB).

### 2. Run with PM2 (Recommended for 24/7 Home Servers)

```bash
# Start background daemon via PM2
make pm2-start

# View real-time status and memory consumption (~10 MB)
pm2 status

# Tail real-time service logs
make pm2-logs

# Persist daemon across host reboots
pm2 save
```

### 3. Makefile Utilities

* `make test` : Run unit tests across all packages.
* `make build` : Recompile standalone binary.
* `make pm2-restart` : Recompile binary and reload the PM2 service.
* `make pm2-stop` : Gracefully stop the PM2 process.
* `make clean` : Remove build artifacts.

---

## 🔒 Security Hardening

1. **User Whitelist Enforcement:** Only User IDs configured in `ALLOWED_USER_IDS` can interact with the bot. Unauthorized messages receive a polite rejection and are logged.
2. **Outbound-Only Polling:** Never opens inbound listening ports or exposes webhooks. Works cleanly behind NAT, dynamic IPs, and VPNs (Tailscale/WireGuard).
3. **No Proxy Contamination:** AGY-Gram guarantees that any local proxy used for Telegram communication is never inherited by child `agy` agent executions.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

## ⚠️ Disclaimer

This project is an independent open-source tool developed strictly for educational, research, and personal automation purposes. It is **not** officially affiliated with, endorsed by, sponsored by, or associated with Google, Antigravity, Telegram, or any of their affiliates or subsidiaries. All product names, logos, brands, and registered trademarks belong to their respective owners.
