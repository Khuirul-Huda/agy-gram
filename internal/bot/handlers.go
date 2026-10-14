package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	tele "gopkg.in/telebot.v3"
)

// Handler holds dependencies for processing Telegram bot commands and messages.
type Handler struct {
	runner      domain.AgentRunner
	session     domain.SessionManager
	monitor     domain.MetricsCollector
	choiceCache sync.Map
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
• /model [opsional: nama_model] - Menu interaktif tombol pilihan model AI
• /effort [opsional: low|med|high|xhigh|max] - Menu interaktif tingkat reasoning effort
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
   • <code>/model</code> : Tampilkan tombol interaktif untuk memilih model AI.
   • <code>/effort</code> : Tampilkan tombol interaktif untuk memilih reasoning effort.
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
	effort := h.session.GetEffort()
	if effort == "" {
		effort = "default"
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
• <b>Reasoning Effort:</b> %s
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
		html.EscapeString(effort),
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

// HandleModel displays interactive inline buttons or updates the agent model.
func (h *Handler) HandleModel(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		text, menu := h.buildModelMenu()
		return c.Send(text, menu, tele.ModeHTML)
	}

	if args == "reset" || args == "default" {
		h.session.SetModel("")
		return c.Send("✅ Model override direset ke konfigurasi default CLI.")
	}

	h.session.SetModel(args)
	return c.Send(fmt.Sprintf("✅ Model override disetel ke: <b>%s</b>", html.EscapeString(args)), tele.ModeHTML)
}

// HandleEffort displays interactive inline buttons or updates the reasoning effort.
func (h *Handler) HandleEffort(c tele.Context) error {
	args := strings.ToLower(strings.TrimSpace(c.Message().Payload))
	if args == "" {
		text, menu := h.buildEffortMenu()
		return c.Send(text, menu, tele.ModeHTML)
	}

	if args == "reset" || args == "default" {
		h.session.SetEffort("")
		return c.Send("✅ Reasoning effort direset ke konfigurasi default CLI.")
	}

	validEfforts := map[string]bool{"low": true, "medium": true, "high": true, "xhigh": true, "max": true}
	if !validEfforts[args] {
		return c.Send("❌ Tingkat effort tidak valid. Pilihan: <code>low</code>, <code>medium</code>, <code>high</code>, <code>xhigh</code>, <code>max</code>, atau <code>default</code>", tele.ModeHTML)
	}

	h.session.SetEffort(args)
	return c.Send(fmt.Sprintf("✅ Reasoning effort disetel ke: <b>%s</b>", html.EscapeString(args)), tele.ModeHTML)
}

// HandleSetModelCallback processes model selection inline button clicks.
func (h *Handler) HandleSetModelCallback(c tele.Context) error {
	_ = c.Respond()
	if cb := c.Callback(); cb != nil && cb.Data != "" {
		h.session.SetModel(cb.Data)
	}
	text, menu := h.buildModelMenu()
	return c.Edit(text, menu, tele.ModeHTML)
}

// HandleResetModelCallback resets model override back to CLI default.
func (h *Handler) HandleResetModelCallback(c tele.Context) error {
	_ = c.Respond()
	h.session.SetModel("")
	text, menu := h.buildModelMenu()
	return c.Edit(text, menu, tele.ModeHTML)
}

// HandleModelMenuCallback navigates back to the model selection menu.
func (h *Handler) HandleModelMenuCallback(c tele.Context) error {
	_ = c.Respond()
	text, menu := h.buildModelMenu()
	return c.Edit(text, menu, tele.ModeHTML)
}

// HandleEffortMenuCallback opens the reasoning effort selection menu.
func (h *Handler) HandleEffortMenuCallback(c tele.Context) error {
	_ = c.Respond()
	text, menu := h.buildEffortMenu()
	return c.Edit(text, menu, tele.ModeHTML)
}

// HandleSetEffortCallback processes reasoning effort inline button clicks.
func (h *Handler) HandleSetEffortCallback(c tele.Context) error {
	_ = c.Respond()
	if cb := c.Callback(); cb != nil {
		if cb.Data == "default" {
			h.session.SetEffort("")
		} else if cb.Data != "" {
			h.session.SetEffort(cb.Data)
		}
	}
	text, menu := h.buildEffortMenu()
	return c.Edit(text, menu, tele.ModeHTML)
}

func (h *Handler) buildModelMenu() (string, *tele.ReplyMarkup) {
	currentModel := h.session.GetModel()
	if currentModel == "" {
		currentModel = "default (CLI config)"
	}
	currentEffort := h.session.GetEffort()
	if currentEffort == "" {
		currentEffort = "default"
	}

	text := fmt.Sprintf(`🤖 <b>Pengaturan Model & Reasoning Effort</b>

• <b>Model Aktif:</b> <code>%s</code>
• <b>Reasoning Effort:</b> <code>%s</code>

Pilih model di bawah atau ubah reasoning effort:`,
		html.EscapeString(currentModel),
		html.EscapeString(currentEffort),
	)

	menu := &tele.ReplyMarkup{}

	type modelOption struct {
		ID   string
		Name string
	}
	models := []modelOption{
		{"gemini-3.8-flash-high", "Gemini 3.8 Flash (High)"},
		{"gemini-3.8-flash-medium", "Gemini 3.8 Flash (Med)"},
		{"gemini-3.8-flash-low", "Gemini 3.8 Flash (Low)"},
		{"claude-sonnet-4-6", "Claude Sonnet 4.6"},
		{"claude-opus-4-6-thinking", "Claude Opus 4.6"},
		{"gpt-oss-120b-medium", "GPT-OSS 120B"},
	}

	var rows []tele.Row
	for i := 0; i < len(models); i += 2 {
		var rowBtns []tele.Btn
		m1 := models[i]
		label1 := m1.Name
		if h.session.GetModel() == m1.ID {
			label1 = "✅ " + label1
		}
		btn1 := menu.Data(label1, "model_set", m1.ID)
		rowBtns = append(rowBtns, btn1)

		if i+1 < len(models) {
			m2 := models[i+1]
			label2 := m2.Name
			if h.session.GetModel() == m2.ID {
				label2 = "✅ " + label2
			}
			btn2 := menu.Data(label2, "model_set", m2.ID)
			rowBtns = append(rowBtns, btn2)
		}
		rows = append(rows, menu.Row(rowBtns...))
	}

	btnEffort := menu.Data(fmt.Sprintf("🎯 Atur Effort (%s)", currentEffort), "effort_menu")
	btnReset := menu.Data("🔄 Reset Default CLI", "model_reset")
	rows = append(rows, menu.Row(btnEffort), menu.Row(btnReset))

	menu.Inline(rows...)
	return text, menu
}

func (h *Handler) buildEffortMenu() (string, *tele.ReplyMarkup) {
	currentEffort := h.session.GetEffort()
	if currentEffort == "" {
		currentEffort = "default"
	}

	text := fmt.Sprintf(`🎯 <b>Pilih Reasoning Effort</b>

Tingkat penalaran saat ini: <code>%s</code>

Pilih kedalaman penalaran model AI yang Anda inginkan:`,
		html.EscapeString(currentEffort),
	)

	menu := &tele.ReplyMarkup{}

	efforts := []string{"low", "medium", "high", "xhigh", "max"}
	var row1 []tele.Btn
	for _, ef := range efforts[:3] {
		label := ef
		if h.session.GetEffort() == ef {
			label = "✅ " + label
		}
		row1 = append(row1, menu.Data(label, "effort_set", ef))
	}

	var row2 []tele.Btn
	for _, ef := range efforts[3:] {
		label := ef
		if h.session.GetEffort() == ef {
			label = "✅ " + label
		}
		row2 = append(row2, menu.Data(label, "effort_set", ef))
	}
	defaultLabel := "default"
	if h.session.GetEffort() == "" {
		defaultLabel = "✅ default"
	}
	row2 = append(row2, menu.Data(defaultLabel, "effort_set", "default"))

	btnBack := menu.Data("⬅️ Kembali ke Menu Model", "model_menu")

	menu.Inline(
		menu.Row(row1...),
		menu.Row(row2...),
		menu.Row(btnBack),
	)
	return text, menu
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
	return h.executePrompt(c, prompt)
}

// HandleUserChoiceCallback handles clicks on dynamically generated confirmation/choice buttons.
func (h *Handler) HandleUserChoiceCallback(c tele.Context) error {
	_ = c.Respond()
	if cb := c.Callback(); cb != nil && cb.Data != "" {
		choiceID := cb.Data
		val, ok := h.choiceCache.Load(choiceID)
		if !ok {
			return c.Send("⚠️ Pilihan telah kadaluarsa atau tidak valid.")
		}
		choiceText := val.(string)

		// Visually update the message: show chosen option and remove buttons
		if cb.Message != nil {
			origText := cb.Message.Text
			updatedHTML := MarkdownToTelegramHTML(origText) + fmt.Sprintf("\n\n👉 <b>Pilihan Anda:</b> <code>%s</code>", html.EscapeString(choiceText))
			_ = c.Edit(updatedHTML, tele.ModeHTML)
		}

		return h.executePrompt(c, choiceText)
	}
	return nil
}

func (h *Handler) executePrompt(c tele.Context, prompt string) error {
	if h.runner.IsRunning() {
		return c.Send("⚠️ Agent sedang memproses permintaan sebelumnya. Gunakan /stop jika ingin membatalkannya.")
	}

	// 1. Fast ACK: Send immediate reply to acknowledge receipt and prevent client timeout
	ackMsg, ackErr := c.Bot().Send(c.Recipient(), "⏳ <i>Sedang memproses instruksi Anda...</i>", tele.ModeHTML)

	// 2. Start continuous periodic typing indicator ticker
	stopTyping := make(chan struct{})
	go h.startTypingTicker(c, stopTyping)
	defer close(stopTyping)

	opts := domain.PromptOptions{
		Prompt:          prompt,
		ContinueSession: h.session.ShouldContinue(),
		Mode:            h.session.GetMode(),
		Model:           h.session.GetModel(),
		Effort:          h.session.GetEffort(),
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

	// Clean up initial acknowledgment placeholder message
	if ackErr == nil && ackMsg != nil {
		_ = c.Bot().Delete(ackMsg)
	}

	if err != nil {
		if errors.Is(err, domain.ErrExecutionCancelled) {
			return c.Send("🛑 Permintaan dibatalkan.")
		}
		return c.Send(fmt.Sprintf("❌ <b>Error saat menjalankan agent:</b>\n<pre><code>%s</code></pre>", html.EscapeString(err.Error())), tele.ModeHTML)
	}

	// Mark session as continuous after successful turn
	h.session.SetContinue(true)

	// Extract interactive confirmation/choice buttons from output
	cleanOutput, buttons := ExtractInteractiveButtons(output)

	// Send output with chunks rendered in Telegram HTML
	chunks := SplitMessage(cleanOutput)
	for i, chunk := range chunks {
		htmlContent := MarkdownToTelegramHTML(chunk)
		if i == len(chunks)-1 {
			// Append elapsed time footer on the last chunk AFTER html conversion (prevents <i> escaping bug)
			htmlContent = fmt.Sprintf("%s\n\n⏱️ <i>(Selesai dalam %v)</i>", htmlContent, elapsed)

			if len(buttons) > 0 {
				menu := &tele.ReplyMarkup{}
				var rows []tele.Row
				var rowBtns []tele.Btn
				for idx, btnText := range buttons {
					choiceID := fmt.Sprintf("ch_%d_%d", time.Now().UnixNano()%10000000, idx)
					h.choiceCache.Store(choiceID, btnText)
					btn := menu.Data(btnText, "user_choice", choiceID)
					rowBtns = append(rowBtns, btn)
					if len(rowBtns) == 2 {
						rows = append(rows, menu.Row(rowBtns...))
						rowBtns = nil
					}
				}
				if len(rowBtns) > 0 {
					rows = append(rows, menu.Row(rowBtns...))
				}
				menu.Inline(rows...)

				if err := c.Send(htmlContent, menu, tele.ModeHTML); err != nil {
					_ = c.Send(chunk, menu, tele.NoPreview)
				}
				continue
			}
		}

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
