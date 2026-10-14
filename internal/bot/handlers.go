package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/agent"
	"github.com/Khuirul-Huda/agy-gram/internal/config"
	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	tele "gopkg.in/telebot.v3"
)

const restartMarkerFile = "/tmp/agy_gram_restarting"

func markRestarting(userID int64) {
	_ = os.WriteFile(restartMarkerFile, []byte(strconv.FormatInt(userID, 10)), 0644)
}

// CheckAndNotifyRestart checks if agy-gram was restarted and sends a recovery confirmation.
func CheckAndNotifyRestart(b *tele.Bot, session domain.SessionManager, cfg *config.Config) {
	data, err := os.ReadFile(restartMarkerFile)
	if err != nil {
		return
	}
	_ = os.Remove(restartMarkerFile)

	var targetUserID int64
	userIDStr := strings.TrimSpace(string(data))
	if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil && id > 0 {
		targetUserID = id
	} else if session != nil && session.GetLastUserID() > 0 {
		targetUserID = session.GetLastUserID()
	} else if cfg != nil && len(cfg.AllowedUserIDs) > 0 {
		for uid := range cfg.AllowedUserIDs {
			targetUserID = uid
			break
		}
	}

	if targetUserID == 0 {
		return
	}

	user := &tele.User{ID: targetUserID}
	msg := "🚀 <b>AGY-Gram berhasil direstart dan sudah kembali online!</b>\nSilakan lanjutkan instruksi Anda."
	if session != nil {
		if convID := session.GetConversationID(); convID != "" {
			msg += fmt.Sprintf("\n\n📌 <b>Sesi Aktif:</b> <code>%s</code> (tersambung kembali)", convID)
		}
	}

	menu := &tele.ReplyMarkup{}
	btnMenu := menu.Data("🎮 Panel Kontrol", "main_menu")
	menu.Inline(menu.Row(btnMenu))

	_, err = b.Send(user, msg, menu, tele.ModeHTML)
	if err != nil {
		slog.Warn("Failed to send post-restart notification", "error", err, "user_id", targetUserID)
	} else {
		slog.Info("Sent post-restart notification", "user_id", targetUserID)
	}
}

func safeEdit(c tele.Context, what interface{}, opts ...interface{}) error {
	err := c.Edit(what, opts...)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
	}
	return err
}

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

// HandleStart sends the welcome message and interactive menu.
func (h *Handler) HandleStart(c tele.Context) error {
	return h.HandleMenu(c)
}

// HandleMenu renders the central interactive control panel dashboard.
func (h *Handler) HandleMenu(c tele.Context) error {
	text, menu := h.buildMainMenu()
	return c.Send(text, menu, tele.ModeHTML)
}

// HandleMenuCallback refreshes or navigates back to the main menu.
func (h *Handler) HandleMenuCallback(c tele.Context) error {
	_ = c.Respond()
	text, menu := h.buildMainMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

func (h *Handler) buildMainMenu() (string, *tele.ReplyMarkup) {
	currentMode := h.session.GetMode()
	if currentMode == "" {
		currentMode = "auto"
	}
	currentModel := h.session.GetModel()
	if currentModel == "" {
		currentModel = "default"
	}
	currentEffort := h.session.GetEffort()
	if currentEffort == "" {
		currentEffort = "default"
	}
	thinkingStatus := "OFF"
	if h.session.IsThinkingEnabled() {
		thinkingStatus = "ON"
	}

	sessionStatus := "Baru (Fresh)"
	if h.session.ShouldContinue() {
		convID := h.session.GetConversationID()
		if convID != "" {
			shortID := convID
			if len(shortID) > 8 {
				shortID = shortID[:8] + "..."
			}
			sessionStatus = fmt.Sprintf("Aktif (<code>%s</code>)", shortID)
		} else {
			sessionStatus = "Aktif (Continue)"
		}
	}

	currentWs := h.session.GetWorkspace()
	if currentWs == "" {
		currentWs = "/home/huda"
	}

	text := fmt.Sprintf(`🎮 <b>AGY-Gram Control Panel</b>

• <b>Workspace:</b> <code>%s</code>
• <b>Model:</b> <code>%s</code>
• <b>Effort:</b> <code>%s</code>
• <b>Mode:</b> <code>%s</code>
• <b>Thinking:</b> <code>%s</code>
• <b>Session:</b> %s

<i>Gunakan tombol di bawah untuk mengubah pengaturan secara instan:</i>`,
		html.EscapeString(currentWs),
		html.EscapeString(currentModel),
		html.EscapeString(currentEffort),
		html.EscapeString(currentMode),
		thinkingStatus,
		sessionStatus,
	)

	menu := &tele.ReplyMarkup{}

	btnModel := menu.Data(fmt.Sprintf("🤖 Model: %s", currentModel), "model_menu")
	btnEffort := menu.Data(fmt.Sprintf("🎯 Effort: %s", currentEffort), "effort_menu")
	btnMode := menu.Data(fmt.Sprintf("🔄 Mode: %s", currentMode), "mode_menu")
	btnThinking := menu.Data(fmt.Sprintf("🧠 Thinking: %s", thinkingStatus), "thinking_menu")
	btnSessions := menu.Data("📂 Riwayat Sesi", "cmd_sessions")
	btnWorkspace := menu.Data("📁 Workspace", "cmd_workspace")
	btnStatus := menu.Data("📊 Status Server", "cmd_status")
	btnNewSession := menu.Data("🧹 Reset Sesi", "cmd_new")
	btnRestart := menu.Data("♻️ Restart Bot", "cmd_restart")
	btnStop := menu.Data("🛑 Stop Agent", "cmd_stop")

	menu.Inline(
		menu.Row(btnModel, btnEffort),
		menu.Row(btnMode, btnThinking),
		menu.Row(btnSessions, btnWorkspace),
		menu.Row(btnStatus, btnNewSession),
		menu.Row(btnRestart, btnStop),
	)

	return text, menu
}

// HandleRestart gracefully restarts agy-gram via deferred detached shell command.
func (h *Handler) HandleRestart(c tele.Context) error {
	sender := c.Sender()
	if sender != nil {
		markRestarting(sender.ID)
		h.session.SetLastUserID(sender.ID)
	}

	_ = c.Send("♻️ <b>Memulai restart AGY-Gram...</b>\nBot akan offline beberapa detik dan segera memberi tahu Anda setelah kembali aktif.", tele.ModeHTML)

	// Execute detached restart command after 1.5 second delay
	go func() {
		time.Sleep(1500 * time.Millisecond)
		cmd := exec.Command("sh", "-c", "sleep 1 && pm2 restart agy-gram")
		_ = cmd.Start()
	}()

	return nil
}

// HandleNewCallback resets the conversation session and refreshes menu.
func (h *Handler) HandleNewCallback(c tele.Context) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "Sesi percakapan berhasil direset."})
	h.session.Reset()
	text, menu := h.buildMainMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

// HandleStatusCallback renders status into the message.
func (h *Handler) HandleStatusCallback(c tele.Context) error {
	_ = c.Respond()
	status, err := h.monitor.Collect()
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal membaca status: %v", err))
	}

	mode := h.session.GetMode()
	if mode == "" {
		mode = "auto"
	}
	model := h.session.GetModel()
	if model == "" {
		model = "default (CLI config)"
	}
	effort := h.session.GetEffort()
	if effort == "" {
		effort = "default"
	}

	sessionStatus := "Baru (Fresh)"
	if h.session.ShouldContinue() {
		sessionStatus = "Aktif (Continue)"
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

	menu := &tele.ReplyMarkup{}
	btnRefresh := menu.Data("🔄 Refresh", "cmd_status")
	btnBack := menu.Data("⬅️ Kembali ke Menu", "main_menu")
	menu.Inline(menu.Row(btnRefresh, btnBack))

	return safeEdit(c, msg, menu, tele.ModeHTML)
}

// HandleStopCallback terminates active agent.
func (h *Handler) HandleStopCallback(c tele.Context) error {
	if !h.runner.IsRunning() {
		_ = c.Respond(&tele.CallbackResponse{Text: "Tidak ada proses agent yang sedang berjalan."})
		return nil
	}

	if h.runner.CancelCurrent() {
		_ = c.Respond(&tele.CallbackResponse{Text: "Proses agent berhasil dihentikan."})
		return c.Send("🛑 <b>Proses agent berhasil dihentikan paksa.</b>", tele.ModeHTML)
	}

	_ = c.Respond(&tele.CallbackResponse{Text: "Gagal menghentikan agent."})
	return nil
}

// HandleHelp displays available commands and usage guide with quick action buttons.
func (h *Handler) HandleHelp(c tele.Context) error {
	lang := h.session.GetLanguage()
	menu := &tele.ReplyMarkup{}
	btnMenu := menu.Data("🎮 Panel Kontrol", "main_menu")
	btnStatus := menu.Data("📊 Status Server", "cmd_status")
	menu.Inline(menu.Row(btnMenu, btnStatus))
	return c.Send(msgHelp(lang), menu, tele.ModeHTML)
}

// HandleNew resets the agent conversation context.
func (h *Handler) HandleNew(c tele.Context) error {
	h.session.Reset()
	lang := h.session.GetLanguage()
	return c.Send(msgNewSession(lang), tele.ModeHTML)
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
		convID := h.session.GetConversationID()
		if convID != "" {
			shortID := convID
			if len(shortID) > 8 {
				shortID = shortID[:8] + "..."
			}
			sessionStatus = fmt.Sprintf("Aktif (<code>%s</code>)", shortID)
		} else {
			sessionStatus = "Aktif (Next: Continue)"
		}
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

	menu := &tele.ReplyMarkup{}
	btnRefresh := menu.Data("🔄 Refresh", "cmd_status")
	btnMenu := menu.Data("🎮 Panel Kontrol", "main_menu")
	menu.Inline(menu.Row(btnRefresh, btnMenu))

	return c.Send(msg, menu, tele.ModeHTML)
}

// HandleMode displays interactive mode buttons or updates the agent execution mode.
func (h *Handler) HandleMode(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		text, menu := h.buildModeMenu()
		return c.Send(text, menu, tele.ModeHTML)
	}

	validModes := map[string]bool{"auto": true, "plan": true, "accept-edits": true}
	if !validModes[args] {
		return c.Send("❌ Mode tidak valid. Pilihan: <code>auto</code>, <code>plan</code>, <code>accept-edits</code>", tele.ModeHTML)
	}

	h.session.SetMode(args)
	return c.Send(fmt.Sprintf("✅ Mode eksekusi diubah menjadi: <b>%s</b>", html.EscapeString(args)), tele.ModeHTML)
}

// HandleSetModeCallback processes mode selection inline button clicks.
func (h *Handler) HandleSetModeCallback(c tele.Context) error {
	_ = c.Respond()
	if cb := c.Callback(); cb != nil && cb.Data != "" {
		h.session.SetMode(cb.Data)
	}
	text, menu := h.buildModeMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

func (h *Handler) buildModeMenu() (string, *tele.ReplyMarkup) {
	currentMode := h.session.GetMode()
	if currentMode == "" {
		currentMode = "auto"
	}

	text := fmt.Sprintf(`🔄 <b>Pilih Mode Eksekusi Agent</b>

• <b>Mode Aktif:</b> <code>%s</code>

Pilih mode di bawah:
• <b>auto</b>: Agent mengeksekusi aksi otomatis secara independen.
• <b>plan</b>: Agent hanya menyusun rencana tindakan tanpa eksekusi.
• <b>accept-edits</b>: Agent menjalankan proses review & eksekusi edit.`, html.EscapeString(currentMode))

	menu := &tele.ReplyMarkup{}
	modes := []struct {
		id   string
		name string
	}{
		{"auto", "⚡ Auto"},
		{"plan", "📋 Plan"},
		{"accept-edits", "✏️ Accept-Edits"},
	}

	var rowBtns []tele.Btn
	for _, m := range modes {
		label := m.name
		if currentMode == m.id {
			label = "✅ " + label
		}
		rowBtns = append(rowBtns, menu.Data(label, "mode_set", m.id))
	}

	btnBack := menu.Data("⬅️ Kembali ke Menu", "main_menu")
	menu.Inline(menu.Row(rowBtns...), menu.Row(btnBack))
	return text, menu
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
	return safeEdit(c, text, menu, tele.ModeHTML)
}

// HandleResetModelCallback resets model override back to CLI default.
func (h *Handler) HandleResetModelCallback(c tele.Context) error {
	_ = c.Respond()
	h.session.SetModel("")
	text, menu := h.buildModelMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

// HandleModelMenuCallback navigates back to the model selection menu.
func (h *Handler) HandleModelMenuCallback(c tele.Context) error {
	_ = c.Respond()
	text, menu := h.buildModelMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

// HandleEffortMenuCallback opens the reasoning effort selection menu.
func (h *Handler) HandleEffortMenuCallback(c tele.Context) error {
	_ = c.Respond()
	text, menu := h.buildEffortMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
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
	return safeEdit(c, text, menu, tele.ModeHTML)
}

// HandleLang displays interactive language buttons or updates the interface language.
func (h *Handler) HandleLang(c tele.Context) error {
	args := strings.ToLower(strings.TrimSpace(c.Message().Payload))
	if args == "id" || args == "indonesia" {
		h.session.SetLanguage("id")
		return c.Send("✅ Bahasa antarmuka diubah ke <b>Bahasa Indonesia</b> 🇮🇩.", tele.ModeHTML)
	}
	if args == "en" || args == "english" {
		h.session.SetLanguage("en")
		return c.Send("✅ Interface language switched to <b>English</b> 🇬🇧.", tele.ModeHTML)
	}

	text, menu := h.buildLangMenu()
	return c.Send(text, menu, tele.ModeHTML)
}

// HandleSetLangCallback processes language selection button clicks.
func (h *Handler) HandleSetLangCallback(c tele.Context) error {
	_ = c.Respond()
	if cb := c.Callback(); cb != nil && cb.Data != "" {
		h.session.SetLanguage(cb.Data)
	}
	text, menu := h.buildLangMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

func (h *Handler) buildLangMenu() (string, *tele.ReplyMarkup) {
	currentLang := h.session.GetLanguage()
	text := `🌐 <b>Pilih Bahasa / Select Language</b>

Silakan pilih bahasa antarmuka bot / Please select the bot interface language:`

	menu := &tele.ReplyMarkup{}

	btnIDText := "🇮🇩 Bahasa Indonesia"
	if currentLang == "id" {
		btnIDText = "✅ " + btnIDText
	}
	btnENText := "🇬🇧 English"
	if currentLang == "en" {
		btnENText = "✅ " + btnENText
	}

	btnID := menu.Data(btnIDText, "lang_set", "id")
	btnEN := menu.Data(btnENText, "lang_set", "en")
	btnBack := menu.Data("⬅️ Kembali ke Menu Utama", "main_menu")

	menu.Inline(menu.Row(btnID, btnEN), menu.Row(btnBack))
	return text, menu
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
	btnBack := menu.Data("⬅️ Kembali ke Menu Utama", "main_menu")
	rows = append(rows, menu.Row(btnEffort, btnReset), menu.Row(btnBack))

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

// HandleThinking displays interactive buttons or updates the verbose thinking setting.
func (h *Handler) HandleThinking(c tele.Context) error {
	args := strings.ToLower(strings.TrimSpace(c.Message().Payload))
	if args == "" {
		text, menu := h.buildThinkingMenu()
		return c.Send(text, menu, tele.ModeHTML)
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

// HandleSetThinkingCallback processes verbose thinking toggle button clicks.
func (h *Handler) HandleSetThinkingCallback(c tele.Context) error {
	_ = c.Respond()
	if cb := c.Callback(); cb != nil && cb.Data != "" {
		enabled := cb.Data == "on"
		h.session.SetThinkingEnabled(enabled)
	}
	text, menu := h.buildThinkingMenu()
	return safeEdit(c, text, menu, tele.ModeHTML)
}

func (h *Handler) buildThinkingMenu() (string, *tele.ReplyMarkup) {
	status := "OFF"
	if h.session.IsThinkingEnabled() {
		status = "ON"
	}

	text := fmt.Sprintf(`🧠 <b>Pengaturan Verbose Thinking</b>

• <b>Status Saat Ini:</b> <b>%s</b>

Jika diaktifkan (ON), log penalaran (thinking block) model AI akan dikirim secara live ke Telegram selama proses respon berlangsung.`, status)

	menu := &tele.ReplyMarkup{}
	btnONText := "🟢 ON (Aktif)"
	if h.session.IsThinkingEnabled() {
		btnONText = "✅ " + btnONText
	}
	btnOFFText := "🔴 OFF (Nonaktif)"
	if !h.session.IsThinkingEnabled() {
		btnOFFText = "✅ " + btnOFFText
	}

	btnON := menu.Data(btnONText, "thinking_set", "on")
	btnOFF := menu.Data(btnOFFText, "thinking_set", "off")
	btnBack := menu.Data("⬅️ Kembali ke Menu", "main_menu")

	menu.Inline(menu.Row(btnON, btnOFF), menu.Row(btnBack))
	return text, menu
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
		lang := h.session.GetLanguage()

		// Visually update the message: show chosen option and remove buttons
		if cb.Message != nil {
			origText := cb.Message.Text
			updatedHTML := MarkdownToTelegramHTML(origText) + "\n\n" + msgChoiceSelected(lang, choiceText)
			_ = safeEdit(c, updatedHTML, tele.ModeHTML)
		}

		return h.executePrompt(c, choiceText)
	}
	return nil
}

func (h *Handler) executePrompt(c tele.Context, prompt string) error {
	lang := h.session.GetLanguage()
	sender := c.Sender()
	if sender != nil {
		h.session.SetLastUserID(sender.ID)
	}

	if h.runner.IsRunning() {
		if lang == "en" {
			return c.Send("⚠️ Agent is already processing a request. Use /stop to cancel it.")
		}
		return c.Send("⚠️ Agent sedang memproses permintaan sebelumnya. Gunakan /stop jika ingin membatalkannya.")
	}

	// 1. Fast ACK: Send immediate reply to acknowledge receipt and prevent client timeout
	ackMsg, ackErr := c.Bot().Send(c.Recipient(), msgFastAck(lang), tele.ModeHTML)

	// 2. Start continuous periodic typing indicator ticker
	stopTyping := make(chan struct{})
	go h.startTypingTicker(c, stopTyping)
	defer close(stopTyping)

	var (
		streamBuf    strings.Builder
		streamMu     sync.Mutex
		lastEditTime time.Time
	)

	opts := domain.PromptOptions{
		Prompt:          prompt,
		ContinueSession: h.session.ShouldContinue(),
		ConversationID:  h.session.GetConversationID(),
		Workspace:       h.session.GetWorkspace(),
		Mode:            h.session.GetMode(),
		Model:           h.session.GetModel(),
		Effort:          h.session.GetEffort(),
		VerboseThinking: h.session.IsThinkingEnabled(),
		OnConversation: func(convID string) {
			h.session.SetConversationID(convID)
		},
		OnCommand: func(cmd string) {
			isRisky, pattern := agent.IsCommandRisky(cmd)
			if isRisky {
				warnMsg := fmt.Sprintf("⚠️ <b>PERINGATAN OPERASI BERISIKO TINGGI:</b>\nPerintah terdeteksi berbahaya (pattern: <code>%s</code>):\n<pre><code class=\"language-bash\">%s</code></pre>", html.EscapeString(pattern), html.EscapeString(cmd))
				_ = c.Send(warnMsg, tele.ModeHTML)
			}
			msg := msgCommandHeader(lang, cmd)
			if err := c.Send(msg, tele.ModeHTML); err != nil {
				_ = c.Send(fmt.Sprintf("⚙️ Menjalankan Perintah:\n%s", cmd))
			}
		},
		OnFileEdit: func(toolName, targetFile, diff string) {
			msg := msgFileEditHeader(lang, toolName, targetFile, diff)
			menu := &tele.ReplyMarkup{}
			btnDownload := menu.Data("📥 Unduh File", "file_download", targetFile)
			menu.Inline(menu.Row(btnDownload))

			if err := c.Send(msg, menu, tele.ModeHTML); err != nil {
				_ = c.Send(fmt.Sprintf("📝 %s: %s\n%s", toolName, targetFile, diff), menu)
			}
		},
		OnThinking: func(thinking string) {
			chunks := SplitMessage(thinking)
			for _, chunk := range chunks {
				msg := msgThinkingHeader(lang, chunk)
				if err := c.Send(msg, tele.ModeHTML); err != nil {
					_ = c.Send("🧠 Thinking Process:\n" + chunk)
				}
			}
		},
		OnTextDelta: func(delta string) {
			// Throttle live message editing to avoid Telegram rate limits
			streamMu.Lock()
			streamBuf.WriteString(delta)
			now := time.Now()
			if ackMsg != nil && now.Sub(lastEditTime) >= 1500*time.Millisecond && streamBuf.Len() > 0 {
				lastEditTime = now
				currentText := streamBuf.String()
				streamMu.Unlock()
				if len(currentText) > 3500 {
					currentText = currentText[:3500] + "\n...(streaming)..."
				}
				html := MarkdownToTelegramHTML(currentText)
				_ = safeEdit(c, html, tele.ModeHTML)
				return
			}
			streamMu.Unlock()
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
			if lang == "en" {
				return c.Send("🛑 Request cancelled.")
			}
			return c.Send("🛑 Permintaan dibatalkan.")
		}
		errTitle := "Error saat menjalankan agent:"
		if lang == "en" {
			errTitle = "Error executing agent:"
		}
		return c.Send(fmt.Sprintf("❌ <b>%s</b>\n<pre><code>%s</code></pre>", errTitle, html.EscapeString(err.Error())), tele.ModeHTML)
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
			htmlContent = fmt.Sprintf("%s\n\n%s", htmlContent, msgElapsedFooter(lang, elapsed))

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
