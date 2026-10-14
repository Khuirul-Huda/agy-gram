package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	tele "gopkg.in/telebot.v3"
)

// Handler holds dependencies for processing Telegram bot commands and messages.
type Handler struct {
	runner  domain.AgentRunner
	session domain.SessionManager
	monitor domain.MetricsCollector
}

// NewHandler creates a new Handler instance.
func NewHandler(runner domain.AgentRunner, session domain.SessionManager, monitor domain.MetricsCollector) *Handler {
	return &Handler{
		runner:  runner,
		session: session,
		monitor: monitor,
	}
}

// HandleStart sends the welcome message.
func (h *Handler) HandleStart(c tele.Context) error {
	msg := `🚀 <b>Selamat Datang di AGY-Gram!</b>

Jembatan kontrol Telegram untuk Antigravity CLI di server lokal Anda.

<b>Perintah yang tersedia:</b>
• /help - Panduan lengkap perintah
• /new - Mulai percakapan sesi baru
• /status - Cek penggunaan resource server (RAM, CPU, disk)
• /mode [auto|plan|accept-edits] - Cek atau ubah mode eksekusi
• /model [nama_model] - Cek atau ubah model AI
• /thinking [on|off] - Toggle verbose thinking stream
• /stop - Hentikan paksa proses agent yang sedang berjalan

Kirimkan pesan teks biasa untuk langsung berinteraksi dengan Antigravity CLI.`

	return c.Send(msg, tele.ModeHTML)
}

// HandleHelp displays available commands and usage guide.
func (h *Handler) HandleHelp(c tele.Context) error {
	msg := `📖 <b>Panduan Penggunaan AGY-Gram</b>

1. <b>Kirim Pesan Bebas:</b>
   Setiap pesan teks yang dikirimkan langsung dieksekusi oleh Antigravity CLI.
   Sesi chat otomatis bersambung (<code>-c</code>) hingga Anda meresetnya.

2. <b>Notifikasi Real-Time:</b>
   • Perintah terminal (CMD) yang dijalankan agent akan langsung dikirim ke chat secara real-time.
   • Jika <code>/thinking on</code> diaktifkan, proses penalaran AI juga akan dikirim sebelum jawaban akhir.

3. <b>Manajemen Sesi:</b>
   • <code>/new</code> : Mulai topik / sesi percakapan baru dari awal.
   • <code>/stop</code> : Batalkan tugas agent yang sedang berjalan.

4. <b>Konfigurasi Runtime:</b>
   • <code>/status</code> : Cek RAM, ZRAM, SSD, dan status server.
   • <code>/mode [opsi]</code> : Ganti mode agent (misal: <code>/mode plan</code>).
   • <code>/model [opsi]</code> : Ganti model (misal: <code>/model flash</code>).
   • <code>/thinking [on|off]</code> : Aktifkan / matikan log proses berpikir AI.`

	return c.Send(msg, tele.ModeHTML)
}

// HandleNew resets the agent conversation context.
func (h *Handler) HandleNew(c tele.Context) error {
	h.session.Reset()
	return c.Send("🔄 <b>Sesi percakapan telah direset.</b> Pesan berikutnya akan memulai sesi baru tanpa konteks lama.", tele.ModeHTML)
}

// HandleStatus reports host server hardware resources and agent status.
func (h *Handler) HandleStatus(c tele.Context) error {
	status, err := h.monitor.Collect()
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal membaca status sistem: %v", err))
	}

	mode := h.session.GetMode()
	if mode == "" {
		mode = "default"
	}
	model := h.session.GetModel()
	if model == "" {
		model = "default (CLI config)"
	}

	sessionStatus := "Baru (Next: Fresh)"
	if h.session.ShouldContinue() {
		sessionStatus = "Aktif (Next: Continue)"
	}

	thinkingStatus := "OFF"
	if h.session.IsThinkingEnabled() {
		thinkingStatus = "ON"
	}

	msg := fmt.Sprintf(`🖥️ <b>Server & Agent Status</b>

<b>Hardware & Host:</b>
• <b>CPU:</b> %s
• <b>Uptime:</b> %s
• <b>RAM:</b> %s
• <b>ZRAM Swap:</b> %s
• <b>Disk (/):</b> %s

<b>Antigravity Config:</b>
• <b>Agent State:</b> %s
• <b>Session:</b> %s
• <b>Mode:</b> %s
• <b>Model:</b> %s
• <b>Verbose Thinking:</b> %s`,
		html.EscapeString(status.CPUModel),
		html.EscapeString(status.Uptime),
		html.EscapeString(status.MemoryUsage),
		html.EscapeString(status.SwapUsage),
		html.EscapeString(status.DiskUsage),
		html.EscapeString(status.AgentStatus),
		html.EscapeString(sessionStatus),
		html.EscapeString(mode),
		html.EscapeString(model),
		thinkingStatus,
	)

	return c.Send(msg, tele.ModeHTML)
}

// HandleMode displays or updates the agent execution mode.
func (h *Handler) HandleMode(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		return c.Send(fmt.Sprintf("ℹ️ Mode saat ini: <b>%s</b>\n\nUbah dengan: <code>/mode [auto|plan|accept-edits]</code>", html.EscapeString(h.session.GetMode())), tele.ModeHTML)
	}

	validModes := map[string]bool{"auto": true, "plan": true, "accept-edits": true}
	if !validModes[args] {
		return c.Send("❌ Mode tidak valid. Pilihan: <code>auto</code>, <code>plan</code>, <code>accept-edits</code>", tele.ModeHTML)
	}

	h.session.SetMode(args)
	return c.Send(fmt.Sprintf("✅ Mode eksekusi diubah menjadi: <b>%s</b>", html.EscapeString(args)), tele.ModeHTML)
}

// HandleModel displays or updates the agent model.
func (h *Handler) HandleModel(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		current := h.session.GetModel()
		if current == "" {
			current = "default (CLI config)"
		}
		return c.Send(fmt.Sprintf("ℹ️ Model override saat ini: <b>%s</b>\n\nUbah dengan: <code>/model [nama_model]</code> (atau <code>/model reset</code>)", html.EscapeString(current)), tele.ModeHTML)
	}

	if args == "reset" || args == "default" {
		h.session.SetModel("")
		return c.Send("✅ Model override direset ke konfigurasi default CLI.")
	}

	h.session.SetModel(args)
	return c.Send(fmt.Sprintf("✅ Model override disetel ke: <b>%s</b>", html.EscapeString(args)), tele.ModeHTML)
}

// HandleThinking displays or updates the verbose thinking setting.
func (h *Handler) HandleThinking(c tele.Context) error {
	args := strings.ToLower(strings.TrimSpace(c.Message().Payload))
	if args == "" {
		status := "OFF"
		if h.session.IsThinkingEnabled() {
			status = "ON"
		}
		return c.Send(fmt.Sprintf("ℹ️ Verbose thinking saat ini: <b>%s</b>\n\nUbah dengan: <code>/thinking on</code> atau <code>/thinking off</code>", status), tele.ModeHTML)
	}

	if args == "on" || args == "1" || args == "enable" || args == "true" {
		h.session.SetThinkingEnabled(true)
		return c.Send("✅ <b>Verbose thinking diaktifkan.</b> Log proses berpikir AI akan dikirim ke Telegram.", tele.ModeHTML)
	}

	if args == "off" || args == "0" || args == "disable" || args == "false" {
		h.session.SetThinkingEnabled(false)
		return c.Send("✅ <b>Verbose thinking dinonaktifkan.</b>", tele.ModeHTML)
	}

	return c.Send("❌ Pilihan tidak valid. Gunakan <code>/thinking on</code> atau <code>/thinking off</code>.", tele.ModeHTML)
}

// HandleCancel terminates the active agent process.
func (h *Handler) HandleCancel(c tele.Context) error {
	if !h.runner.IsRunning() {
		return c.Send("ℹ️ Tidak ada proses agent yang sedang berjalan.")
	}

	if h.runner.CancelCurrent() {
		return c.Send("🛑 <b>Proses agent berhasil dihentikan paksa.</b>", tele.ModeHTML)
	}

	return c.Send("❌ Gagal menghentikan proses agent.")
}

// HandlePrompt processes a regular text message through Antigravity CLI.
func (h *Handler) HandlePrompt(c tele.Context) error {
	prompt := strings.TrimSpace(c.Message().Text)
	if prompt == "" {
		return nil
	}

	if h.runner.IsRunning() {
		return c.Send("⚠️ Agent sedang memproses permintaan sebelumnya. Gunakan /stop jika ingin membatalkannya.")
	}

	// Start periodic typing indicator ticker
	stopTyping := make(chan struct{})
	go h.startTypingTicker(c, stopTyping)
	defer close(stopTyping)

	opts := domain.PromptOptions{
		Prompt:          prompt,
		ContinueSession: h.session.ShouldContinue(),
		Mode:            h.session.GetMode(),
		Model:           h.session.GetModel(),
		VerboseThinking: h.session.IsThinkingEnabled(),
		OnCommand: func(cmd string) {
			escapedCmd := html.EscapeString(cmd)
			msg := fmt.Sprintf("⚙️ <b>Menjalankan Perintah:</b>\n<pre><code class=\"language-bash\">%s</code></pre>", escapedCmd)
			if err := c.Send(msg, tele.ModeHTML); err != nil {
				_ = c.Send(fmt.Sprintf("⚙️ Menjalankan Perintah:\n%s", cmd))
			}
		},
		OnThinking: func(thinking string) {
			chunks := SplitMessage(thinking)
			for _, chunk := range chunks {
				escapedThinking := html.EscapeString(chunk)
				msg := fmt.Sprintf("🧠 <b>Thinking Process:</b>\n<blockquote>%s</blockquote>", escapedThinking)
				if err := c.Send(msg, tele.ModeHTML); err != nil {
					_ = c.Send("🧠 Thinking Process:\n" + chunk)
				}
			}
		},
	}

	startTime := time.Now()
	output, err := h.runner.Execute(context.Background(), opts)
	elapsed := time.Since(startTime).Round(time.Millisecond)

	if err != nil {
		if errors.Is(err, domain.ErrExecutionCancelled) {
			return c.Send("🛑 Permintaan dibatalkan.")
		}
		return c.Send(fmt.Sprintf("❌ <b>Error saat menjalankan agent:</b>\n<pre><code>%s</code></pre>", html.EscapeString(err.Error())), tele.ModeHTML)
	}

	// Mark session as continuous after successful turn
	h.session.SetContinue(true)

	// Send output with chunks rendered in Telegram HTML
	chunks := SplitMessage(output)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			// Append elapsed time footer on the last chunk
			chunk = fmt.Sprintf("%s\n\n⏱️ <i>(Selesai dalam %v)</i>", chunk, elapsed)
		}

		htmlContent := MarkdownToTelegramHTML(chunk)
		if err := c.Send(htmlContent, tele.ModeHTML); err != nil {
			// Fallback to legacy markdown if HTML parser encounters an issue
			if errMd := c.Send(chunk, tele.ModeMarkdown); errMd != nil {
				// Fallback to plain text
				_ = c.Send(chunk, tele.NoPreview)
			}
		}
	}

	return nil
}

func (h *Handler) startTypingTicker(c tele.Context, stopChan <-chan struct{}) {
	_ = c.Notify(tele.Typing)
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = c.Notify(tele.Typing)
		case <-stopChan:
			return
		}
	}
}
