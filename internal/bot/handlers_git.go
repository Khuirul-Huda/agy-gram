package bot

import (
	"bytes"
	"fmt"
	"html"
	"os/exec"
	"strings"

	tele "gopkg.in/telebot.v3"
)

// HandleDiff displays uncommitted changes in the current workspace.
func (h *Handler) HandleDiff(c tele.Context) error {
	session := h.getSession(c)
	ws := session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}

	diffOutput, hasChanges, err := getGitDiff(ws)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal memeriksa git: <code>%s</code>", html.EscapeString(err.Error())), tele.ModeHTML)
	}

	if !hasChanges {
		return c.Send("✅ Tidak ada perubahan uncommitted di workspace ini.")
	}

	if len(diffOutput) > 3500 {
		diffOutput = diffOutput[:3500] + "\n...(diff dipotong karena panjang)..."
	}

	msg := fmt.Sprintf("📝 <b>Git Diff (Workspace: <code>%s</code>):</b>\n<pre><code class=\"language-diff\">%s</code></pre>",
		html.EscapeString(ws), html.EscapeString(diffOutput))

	menu := &tele.ReplyMarkup{}
	btnCommit := menu.Data("💾 Commit & Push", "git_commit")
	btnUndo := menu.Data("⏪ Undo Perubahan", "git_undo")
	menu.Inline(menu.Row(btnCommit, btnUndo))

	return c.Send(msg, menu, tele.ModeHTML)
}

// HandleCommit commits changes with an optional message or auto-generated message via agy.
func (h *Handler) HandleCommit(c tele.Context) error {
	msg := strings.TrimSpace(c.Message().Payload)
	if msg == "" {
		return h.executePrompt(c, "Tolong analisis uncommitted git diff saat ini, buat commit message yang deskriptif dan profesional, lalu lakukan git commit dan git push.")
	}

	session := h.getSession(c)
	ws := session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}

	addCmd := exec.Command("git", "-C", ws, "add", "-A")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal git add: %s", html.EscapeString(string(out))), tele.ModeHTML)
	}

	commitCmd := exec.Command("git", "-C", ws, "commit", "-m", msg)
	commitOut, err := commitCmd.CombinedOutput()
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal git commit: %s", html.EscapeString(string(commitOut))), tele.ModeHTML)
	}

	res := fmt.Sprintf("✅ <b>Git Commit Berhasil:</b>\n<pre><code>%s</code></pre>", html.EscapeString(string(commitOut)))

	// Try push
	pushCmd := exec.Command("git", "-C", ws, "push")
	if pushOut, err := pushCmd.CombinedOutput(); err == nil {
		res += fmt.Sprintf("\n🚀 <b>Git Push Berhasil:</b>\n<pre><code>%s</code></pre>", html.EscapeString(string(pushOut)))
	}

	return c.Send(res, tele.ModeHTML)
}

// HandleUndo discards all uncommitted modifications in the workspace.
func (h *Handler) HandleUndo(c tele.Context) error {
	session := h.getSession(c)
	ws := session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}

	restoreCmd := exec.Command("git", "-C", ws, "restore", ".")
	_ = restoreCmd.Run()

	cleanCmd := exec.Command("git", "-C", ws, "clean", "-fd")
	cleanOut, err := cleanCmd.CombinedOutput()
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal revert: %s", html.EscapeString(string(cleanOut))), tele.ModeHTML)
	}

	return c.Send("⏪ <b>Semua perubahan uncommitted berhasil dibatalkan.</b>", tele.ModeHTML)
}

// HandleGitDiffCallback triggers HandleDiff from inline buttons.
func (h *Handler) HandleGitDiffCallback(c tele.Context) error {
	_ = c.Respond()
	return h.HandleDiff(c)
}

// HandleGitCommitCallback prompts agy to auto-commit uncommitted changes.
func (h *Handler) HandleGitCommitCallback(c tele.Context) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "Memproses commit..."})
	return h.executePrompt(c, "Tolong analisis uncommitted git diff saat ini, buat commit message yang deskriptif dan profesional, lalu lakukan git commit dan git push.")
}

// HandleGitUndoCallback handles the undo inline button.
func (h *Handler) HandleGitUndoCallback(c tele.Context) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "Membatalkan perubahan..."})
	return h.HandleUndo(c)
}

// attachGitOpsButtons appends GitOps action buttons if the workspace has uncommitted changes.
func (h *Handler) attachGitOpsButtons(menu *tele.ReplyMarkup, ws string, lang string) *tele.ReplyMarkup {
	_, hasChanges, err := getGitDiff(ws)
	if err != nil || !hasChanges {
		return menu
	}

	if menu == nil {
		menu = &tele.ReplyMarkup{}
	}

	diffLabel := "🔍 Lihat Git Diff"
	commitLabel := "💾 Auto Commit & Push"
	undoLabel := "⏪ Undo Perubahan"
	if lang == "en" {
		diffLabel = "🔍 View Git Diff"
		commitLabel = "💾 Auto Commit & Push"
		undoLabel = "⏪ Undo Changes"
	}

	btnDiff := menu.Data(diffLabel, "git_diff")
	btnCommit := menu.Data(commitLabel, "git_commit")
	btnUndo := menu.Data(undoLabel, "git_undo")

	row1 := menu.Row(btnDiff, btnCommit)
	row2 := menu.Row(btnUndo)

	if len(menu.InlineKeyboard) > 0 {
		var iRow1, iRow2 []tele.InlineButton
		for _, b := range row1 {
			if ib := b.Inline(); ib != nil {
				iRow1 = append(iRow1, *ib)
			}
		}
		for _, b := range row2 {
			if ib := b.Inline(); ib != nil {
				iRow2 = append(iRow2, *ib)
			}
		}
		if len(iRow1) > 0 {
			menu.InlineKeyboard = append(menu.InlineKeyboard, iRow1)
		}
		if len(iRow2) > 0 {
			menu.InlineKeyboard = append(menu.InlineKeyboard, iRow2)
		}
		return menu
	}

	menu.Inline(row1, row2)
	return menu
}

func getGitDiff(ws string) (string, bool, error) {
	cmdRoot := exec.Command("git", "-C", ws, "rev-parse", "--is-inside-work-tree")
	if err := cmdRoot.Run(); err != nil {
		return "", false, nil // Not a git repo
	}

	cmdStatus := exec.Command("git", "-C", ws, "status", "--porcelain")
	statusOut, err := cmdStatus.Output()
	if err != nil || len(bytes.TrimSpace(statusOut)) == 0 {
		return "", false, nil // Clean working tree
	}

	cmdDiff := exec.Command("git", "-C", ws, "diff", "--stat")
	statOut, _ := cmdDiff.Output()

	cmdFullDiff := exec.Command("git", "-C", ws, "diff")
	fullDiff, _ := cmdFullDiff.Output()

	combined := fmt.Sprintf("%s\n\n%s", strings.TrimSpace(string(statOut)), strings.TrimSpace(string(fullDiff)))
	return strings.TrimSpace(combined), true, nil
}
