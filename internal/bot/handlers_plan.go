package bot

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

// isPlanResponse detects whether the agent output represents a plan.
func isPlanResponse(mode, output string) bool {
	if mode == "plan" {
		return true
	}
	lower := strings.ToLower(output)
	return strings.Contains(lower, "### plan") ||
		strings.Contains(lower, "## plan") ||
		strings.Contains(lower, "### rencana") ||
		strings.Contains(lower, "## rencana") ||
		strings.Contains(lower, "implementation plan") ||
		strings.Contains(lower, "action plan") ||
		strings.Contains(lower, "[plan:")
}

// attachPlanApprovalButtons appends plan approval buttons to reply markup.
func (h *Handler) attachPlanApprovalButtons(menu *tele.ReplyMarkup, lang string) *tele.ReplyMarkup {
	if menu == nil {
		menu = &tele.ReplyMarkup{}
	}
	btnApprove := menu.Data("🚀 Eksekusi Plan Ini", "plan_approve")
	btnRevise := menu.Data("✏️ Minta Revisi Plan", "plan_revise")
	if lang == "en" {
		btnApprove = menu.Data("🚀 Execute This Plan", "plan_approve")
		btnRevise = menu.Data("✏️ Request Plan Revision", "plan_revise")
	}

	planRow := menu.Row(btnApprove, btnRevise)
	if len(menu.InlineKeyboard) > 0 {
		var inlineRow []tele.InlineButton
		for _, b := range planRow {
			if ib := b.Inline(); ib != nil {
				inlineRow = append(inlineRow, *ib)
			}
		}
		if len(inlineRow) > 0 {
			menu.InlineKeyboard = append(menu.InlineKeyboard, inlineRow)
			return menu
		}
	}

	menu.Inline(planRow)
	return menu
}

// HandlePlanApproveCallback handles user clicking "🚀 Eksekusi Plan Ini".
func (h *Handler) HandlePlanApproveCallback(c tele.Context) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "Mengeksekusi plan..."})
	session := h.getSession(c)
	lang := session.GetLanguage()

	// Switch mode to auto to allow agent to execute changes
	session.SetMode("auto")

	notice := "🚀 <b>Plan Disetujui!</b>\nMode dialihkan ke <code>auto</code>. Memulai eksekusi langkah-langkah plan..."
	if lang == "en" {
		notice = "🚀 <b>Plan Approved!</b>\nSwitched execution mode to <code>auto</code>. Proceeding with execution..."
	}
	_ = c.Send(notice, tele.ModeHTML)

	prompt := "Lanjutkan dan terapkan seluruh langkah rencana/plan yang telah disetujui di atas sampai selesai tuntas."
	if lang == "en" {
		prompt = "Proceed and execute the entire approved plan above to completion."
	}
	return h.executePrompt(c, prompt)
}

// HandlePlanReviseCallback handles user clicking "✏️ Minta Revisi Plan".
func (h *Handler) HandlePlanReviseCallback(c tele.Context) error {
	_ = c.Respond()
	session := h.getSession(c)
	lang := session.GetLanguage()

	notice := "✏️ <b>Revisi Plan:</b>\nSilakan ketik poin-poin yang ingin diubah, ditambah, atau diperbaiki dari plan di atas:"
	if lang == "en" {
		notice = "✏️ <b>Revise Plan:</b>\nPlease type the points you want modified, added, or improved in the plan above:"
	}
	return c.Send(notice, tele.ModeHTML)
}
