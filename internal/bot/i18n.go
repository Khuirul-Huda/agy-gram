package bot

import (
	"fmt"
	"html"
	"time"
)

// Language helper functions returning localized strings based on "id" or "en".

func msgFastAck(lang string) string {
	if lang == "en" {
		return "⏳ <i>Processing your instruction...</i>"
	}
	return "⏳ <i>Sedang memproses instruksi Anda...</i>"
}

func msgElapsedFooter(lang string, elapsed time.Duration) string {
	if lang == "en" {
		return fmt.Sprintf("⏱️ <i>(Finished in %v)</i>", elapsed)
	}
	return fmt.Sprintf("⏱️ <i>(Selesai dalam %v)</i>", elapsed)
}

func msgCommandHeader(lang, cmd string) string {
	escapedCmd := html.EscapeString(cmd)
	if lang == "en" {
		return fmt.Sprintf("⚙️ <b>Executing Command:</b>\n<pre><code class=\"language-bash\">%s</code></pre>", escapedCmd)
	}
	return fmt.Sprintf("⚙️ <b>Menjalankan Perintah:</b>\n<pre><code class=\"language-bash\">%s</code></pre>", escapedCmd)
}

func msgFileEditHeader(lang, toolName, targetFile, diff string) string {
	escapedFile := html.EscapeString(targetFile)
	escapedDiff := html.EscapeString(diff)

	title := "Mengedit File:"
	if toolName == "write_to_file" {
		title = "Menulis File:"
	}
	if lang == "en" {
		title = "Editing File:"
		if toolName == "write_to_file" {
			title = "Writing File:"
		}
	}

	if escapedDiff != "" {
		return fmt.Sprintf("📝 <b>%s</b> <code>%s</code>\n<pre><code class=\"language-diff\">%s</code></pre>", title, escapedFile, escapedDiff)
	}
	return fmt.Sprintf("📝 <b>%s</b> <code>%s</code>", title, escapedFile)
}

func msgThinkingHeader(lang, thinking string) string {
	escapedThinking := html.EscapeString(thinking)
	title := "Proses Berpikir:"
	if lang == "en" {
		title = "Thinking Process:"
	}
	return fmt.Sprintf("🧠 <b>%s</b>\n<blockquote>%s</blockquote>", title, escapedThinking)
}

func msgChoiceSelected(lang, choice string) string {
	if lang == "en" {
		return fmt.Sprintf("👉 <b>Your Selection:</b> <code>%s</code>", html.EscapeString(choice))
	}
	return fmt.Sprintf("👉 <b>Pilihan Anda:</b> <code>%s</code>", html.EscapeString(choice))
}

func msgStart(lang string) string {
	if lang == "en" {
		return `🚀 <b>Welcome to AGY-Gram!</b>

Telegram control bridge for Antigravity CLI on your local server.

<b>Available Commands:</b>
• /help - Full command reference
• /new - Start a fresh conversation session
• /status - Check host system metrics (RAM, CPU, disk)
• /mode [auto|plan|accept-edits] - Switch agent mode
• /model - Interactive AI model selection menu
• /effort - Interactive reasoning effort selection
• /lang - Switch language (Bahasa Indonesia / English)
• /thinking [on|off] - Toggle verbose thinking stream
• /stop - Force abort currently running agent task

Send any plain text message to interact directly with Antigravity CLI.`
	}

	return `🚀 <b>Selamat Datang di AGY-Gram!</b>

Jembatan kontrol Telegram untuk Antigravity CLI di server lokal Anda.

<b>Perintah yang tersedia:</b>
• /help - Panduan lengkap perintah
• /new - Mulai percakapan sesi baru
• /status - Cek penggunaan resource server (RAM, CPU, disk)
• /mode [auto|plan|accept-edits] - Cek atau ubah mode eksekusi
• /model - Menu interaktif tombol pilihan model AI
• /effort - Menu interaktif tingkat reasoning effort
• /lang - Ganti bahasa (Bahasa Indonesia / English)
• /thinking [on|off] - Toggle verbose thinking stream
• /stop - Hentikan paksa proses agent yang sedang berjalan

Kirimkan pesan teks biasa untuk langsung berinteraksi dengan Antigravity CLI.`
}

func msgHelp(lang string) string {
	if lang == "en" {
		return `📖 <b>AGY-Gram User Guide</b>

1. <b>Send Any Message:</b>
   Any regular message is directly executed by Antigravity CLI.
   Chat sessions automatically continue (<code>-c</code>) until reset.

2. <b>Real-Time Streaming:</b>
   • Terminal commands (CMD) and file modifications (Diffs) executed by the agent are sent live.
   • If <code>/thinking on</code> is enabled, AI reasoning traces are sent live.

3. <b>Interactive Buttons:</b>
   • If AI asks for confirmation (e.g. <code>[CONFIRM: Yes | No]</code>), clickable buttons appear automatically!

4. <b>Runtime Settings:</b>
   • <code>/model</code> : Choose AI model.
   • <code>/effort</code> : Adjust reasoning effort.
   • <code>/usage</code> : Check quota and rate limits.
   • <code>/lang</code> : Toggle English / Indonesian.
   • <code>/status</code> : System resources and agent metrics.`
	}

	return `📖 <b>Panduan Penggunaan AGY-Gram</b>

1. <b>Kirim Pesan Bebas:</b>
   Setiap pesan teks yang dikirimkan langsung dieksekusi oleh Antigravity CLI.
   Sesi chat otomatis bersambung (<code>-c</code>) hingga Anda meresetnya.

2. <b>Notifikasi Real-Time:</b>
   • Perintah terminal (CMD) dan perubahan file (Diff) yang dijalankan agent langsung dikirim secara real-time.
   • Jika <code>/thinking on</code> diaktifkan, proses penalaran AI dikirim saat itu juga.

3. <b>Tombol Interaktif Otomatis:</b>
   • Jika AI meminta konfirmasi (misal: <code>[CONFIRM: Ya | Tidak]</code>), tombol otomatis muncul di Telegram!

4. <b>Konfigurasi Runtime:</b>
   • <code>/model</code> : Pilih model AI.
   • <code>/effort</code> : Atur reasoning effort.
   • <code>/usage</code> : Cek kuota token & limit model AI.
   • <code>/lang</code> : Ganti Bahasa Indonesia / English.
   • <code>/status</code> : Cek RAM, ZRAM, SSD, dan status server.`
}

func msgNewSession(lang string) string {
	if lang == "en" {
		return "🔄 <b>Conversation session has been reset.</b> Your next message will start a fresh session."
	}
	return "🔄 <b>Sesi percakapan telah direset.</b> Pesan berikutnya akan memulai sesi baru tanpa konteks lama."
}
