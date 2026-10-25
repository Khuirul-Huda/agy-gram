package bot

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	tele "gopkg.in/telebot.v3"
)

// HandleSessions lists recent conversation sessions for easy switching.
func (h *Handler) HandleSessions(c tele.Context) error {
	text, menu := h.buildSessionsMenu(0)
	return c.Send(text, menu, tele.ModeHTML)
}

// HandleSessionsCallback handles navigation and pagination of session history.
func (h *Handler) HandleSessionsCallback(c tele.Context) error {
	_ = c.Respond()
	text, menu := h.buildSessionsMenu(0)
	return safeEdit(c, text, menu, tele.ModeHTML)
}

// HandleSessionSwitchCallback activates an existing historical session.
func (h *Handler) HandleSessionSwitchCallback(c tele.Context) error {
	_ = c.Respond()
	cb := c.Callback()
	if cb == nil || cb.Data == "" {
		return nil
	}

	convID := cb.Data
	if err := h.session.SwitchSession(convID); err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal beralih ke sesi <code>%s</code>: %v", convID, err), tele.ModeHTML)
	}

	shortID := convID
	if len(shortID) > 10 {
		shortID = shortID[:10] + "..."
	}

	msg := fmt.Sprintf("✅ <b>Beralih ke sesi:</b> <code>%s</code>\nInstruksi berikutnya akan melanjutkan konteks sesi ini.", shortID)
	menu := &tele.ReplyMarkup{}
	btnMenu := menu.Data("🎮 Panel Kontrol", "main_menu")
	menu.Inline(menu.Row(btnMenu))

	return safeEdit(c, msg, menu, tele.ModeHTML)
}

func (h *Handler) buildSessionsMenu(page int) (string, *tele.ReplyMarkup) {
	menu := &tele.ReplyMarkup{}

	sessions, err := h.session.ListSessions(10)
	if err != nil || len(sessions) == 0 {
		btnNew := menu.Data("➕ Sesi Baru", "cmd_new")
		btnBack := menu.Data("⬅️ Kembali ke Menu", "main_menu")
		menu.Inline(menu.Row(btnNew), menu.Row(btnBack))
		return "📂 <b>Riwayat Sesi Percakapan</b>\n\nBelum ada riwayat sesi yang tersimpan.", menu
	}

	currentConvID := h.session.GetConversationID()
	var rows []tele.Row

	var sb strings.Builder
	sb.WriteString("📂 <b>Daftar Sesi Percakapan Sebelumnya:</b>\n\n")

	for idx, s := range sessions {
		shortID := s.ConversationID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		title := s.Title
		if title == "" {
			title = shortID
		}
		if len(title) > 20 {
			title = title[:17] + "..."
		}

		timeStr := s.UpdatedAt.Format("02 Jan 15:04")
		prefix := "▫️"
		if s.ConversationID == currentConvID {
			prefix = "👉 (Aktif)"
		}

		sb.WriteString(fmt.Sprintf("%s <b>%s</b> [<code>%s</code>] - <i>%s</i>\n", prefix, title, shortID, timeStr))

		btnLabel := fmt.Sprintf("▶️ %s (%s)", title, timeStr)
		if s.ConversationID == currentConvID {
			btnLabel = "✅ " + btnLabel
		}
		btn := menu.Data(btnLabel, "session_switch", s.ConversationID)
		rows = append(rows, menu.Row(btn))

		if idx >= 6 {
			break
		}
	}

	btnNew := menu.Data("➕ Buat Sesi Baru", "cmd_new")
	btnBack := menu.Data("⬅️ Kembali ke Menu", "main_menu")
	rows = append(rows, menu.Row(btnNew, btnBack))

	menu.Inline(rows...)
	return sb.String(), menu
}

// HandleWorkspace displays available workspaces or updates current workspace.
func (h *Handler) HandleWorkspace(c tele.Context) error {
	s := h.getSession(c)
	args := strings.TrimSpace(c.Message().Payload)
	if args != "" {
		s.SetWorkspace(args)
		return c.Send(fmt.Sprintf("📁 Workspace disetel ke: <code>%s</code>", args), tele.ModeHTML)
	}

	text, menu := h.buildWorkspaceMenuFor(s)
	return c.Send(text, menu, tele.ModeHTML)
}

// HandleWorkspaceCallback handles interactive workspace switching.
func (h *Handler) HandleWorkspaceCallback(c tele.Context) error {
	_ = c.Respond()
	s := h.getSession(c)
	cb := c.Callback()
	if cb != nil && cb.Data != "" {
		switch cb.Data {
		case "ws_home":
			s.SetWorkspace("/home/huda")
		case "ws_agy":
			s.SetWorkspace("/home/huda/repo/agy-gram")
		case "ws_mocci":
			s.SetWorkspace("/home/huda/repo/Mocci")
		case "ws_cctv":
			s.SetWorkspace("/home/huda/cctv_recordings")
		}
	}

	text, menu := h.buildWorkspaceMenuFor(s)
	return safeEdit(c, text, menu, tele.ModeHTML)
}

func (h *Handler) buildWorkspaceMenu() (string, *tele.ReplyMarkup) {
	return h.buildWorkspaceMenuFor(h.session)
}

func (h *Handler) buildWorkspaceMenuFor(s domain.SessionManager) (string, *tele.ReplyMarkup) {
	currentWs := s.GetWorkspace()
	if currentWs == "" {
		currentWs = "/home/huda"
	}

	text := fmt.Sprintf(`📁 <b>Pengaturan Target Workspace</b>

• <b>Workspace Aktif:</b> <code>%s</code>

Pilih direktori kerja yang ingin dikelola oleh Antigravity CLI:`, currentWs)

	menu := &tele.ReplyMarkup{}

	workspaces := []struct {
		id   string
		path string
		name string
	}{
		{"ws_home", "/home/huda", "🏠 Home (/home/huda)"},
		{"ws_agy", "/home/huda/repo/agy-gram", "🤖 agy-gram repo"},
		{"ws_mocci", "/home/huda/repo/Mocci", "🐱 Mocci Bot repo"},
		{"ws_cctv", "/home/huda/cctv_recordings", "📹 CCTV Recordings"},
	}

	var rows []tele.Row
	for _, ws := range workspaces {
		label := ws.name
		if filepath.Clean(currentWs) == filepath.Clean(ws.path) {
			label = "✅ " + label
		}
		btn := menu.Data(label, "ws_set", ws.id)
		rows = append(rows, menu.Row(btn))
	}

	btnBack := menu.Data("⬅️ Kembali ke Menu", "main_menu")
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)
	return text, menu
}
