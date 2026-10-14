package bot

import (
	"context"
	"errors"
	"fmt"
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
	msg := `🚀 *Selamat Datang di AGY-Gram!*

Jembatan kontrol Telegram untuk Antigravity CLI di server lokal Anda.

*Perintah yang tersedia:*
• /help - Panduan lengkap perintah
• /new - Mulai percakapan sesi baru
• /status - Cek penggunaan resource server (RAM, CPU, disk)
• /mode [auto|plan|accept-edits] - Cek atau ubah mode eksekusi
• /model [nama_model] - Cek atau ubah model AI
• /stop - Hentikan paksa proses agent yang sedang berjalan

Kirimkan pesan teks biasa untuk langsung berinteraksi dengan Antigravity CLI.`

	return c.Send(msg, tele.ModeMarkdown)
}

// HandleHelp displays available commands and usage guide.
func (h *Handler) HandleHelp(c tele.Context) error {
	msg := `📖 *Panduan Penggunaan AGY-Gram*

1. *Kirim Pesan Bebas:*
   Setiap pesan teks yang dikirimkan langsung dieksekusi oleh Antigravity CLI.
   Sesi chat otomatis bersambung (` + "`-c`" + `) hingga Anda meresetnya.

2. *Manajemen Sesi:*
   • ` + "`/new`" + ` : Mulai topik / sesi percakapan baru dari awal.
   • ` + "`/stop`" + ` : Batalkan tugas agent yang sedang berjalan.

3. *Konfigurasi Runtime:*
   • ` + "`/status`" + ` : Cek RAM, ZRAM, SSD, dan status server.
   • ` + "`/mode [opsi]`" + ` : Ganti mode agent (misal: ` + "`/mode plan`" + `).
   • ` + "`/model [opsi]`" + ` : Ganti model (misal: ` + "`/model flash`" + `).`

	return c.Send(msg, tele.ModeMarkdown)
}

// HandleNew resets the agent conversation context.
func (h *Handler) HandleNew(c tele.Context) error {
	h.session.Reset()
	return c.Send("🔄 *Sesi percakapan telah direset.* Pesan berikutnya akan memulai sesi baru tanpa konteks lama.", tele.ModeMarkdown)
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

	msg := fmt.Sprintf(`🖥️ *Server & Agent Status*

*Hardware & Host:*
• *CPU:* %s
• *Uptime:* %s
• *RAM:* %s
• *ZRAM Swap:* %s
• *Disk (/):* %s

*Antigravity Config:*
• *Agent State:* %s
• *Session:* %s
• *Mode:* %s
• *Model:* %s`,
		status.CPUModel,
		status.Uptime,
		status.MemoryUsage,
		status.SwapUsage,
		status.DiskUsage,
		status.AgentStatus,
		sessionStatus,
		mode,
		model,
	)

	return c.Send(msg, tele.ModeMarkdown)
}

// HandleMode displays or updates the agent execution mode.
func (h *Handler) HandleMode(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		return c.Send(fmt.Sprintf("ℹ️ Mode saat ini: *%s*\n\nUbah dengan: `/mode [auto|plan|accept-edits]`", h.session.GetMode()), tele.ModeMarkdown)
	}

	validModes := map[string]bool{"auto": true, "plan": true, "accept-edits": true}
	if !validModes[args] {
		return c.Send("❌ Mode tidak valid. Pilihan: `auto`, `plan`, `accept-edits`", tele.ModeMarkdown)
	}

	h.session.SetMode(args)
	return c.Send(fmt.Sprintf("✅ Mode eksekusi diubah menjadi: *%s*", args), tele.ModeMarkdown)
}

// HandleModel displays or updates the agent model.
func (h *Handler) HandleModel(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		current := h.session.GetModel()
		if current == "" {
			current = "default (CLI config)"
		}
		return c.Send(fmt.Sprintf("ℹ️ Model override saat ini: *%s*\n\nUbah dengan: `/model [nama_model]` (atau `/model reset` untuk kembali ke default)", current), tele.ModeMarkdown)
	}

	if args == "reset" || args == "default" {
		h.session.SetModel("")
		return c.Send("✅ Model override direset ke konfigurasi default CLI.")
	}

	h.session.SetModel(args)
	return c.Send(fmt.Sprintf("✅ Model override disetel ke: *%s*", args))
}

// HandleCancel terminates the active agent process.
func (h *Handler) HandleCancel(c tele.Context) error {
	if !h.runner.IsRunning() {
		return c.Send("ℹ️ Tidak ada proses agent yang sedang berjalan.")
	}

	if h.runner.CancelCurrent() {
		return c.Send("🛑 *Proses agent berhasil dihentikan paksa.*", tele.ModeMarkdown)
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
	}

	startTime := time.Now()
	output, err := h.runner.Execute(context.Background(), opts)
	elapsed := time.Since(startTime).Round(time.Millisecond)

	if err != nil {
		if errors.Is(err, domain.ErrExecutionCancelled) {
			return c.Send("🛑 Permintaan dibatalkan.")
		}
		return c.Send(fmt.Sprintf("❌ *Error saat menjalankan agent:*\n```\n%s\n```", err.Error()), tele.ModeMarkdown)
	}

	// Mark session as continuous after successful turn
	h.session.SetContinue(true)

	// Send output with chunks if it exceeds Telegram's limit
	chunks := SplitMessage(output)
	for i, chunk := range chunks {
		if i == len(chunks)-1 && len(chunks) > 1 {
			// Append elapsed time footer on the last chunk
			chunk = fmt.Sprintf("%s\n\n⏱️ _(Selesai dalam %v)_", chunk, elapsed)
		}
		if err := c.Send(chunk); err != nil {
			// Fallback without formatting if Telegram fails markdown parsing
			_ = c.Send(chunk, tele.NoPreview)
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
