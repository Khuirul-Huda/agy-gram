package bot

import (
	"fmt"
	"html"

	tele "gopkg.in/telebot.v3"
)

// notifyRiskyCommand sends an immediate interactive security alert to the user when a high-risk operation is detected.
func (h *Handler) notifyRiskyCommand(c tele.Context, cmd, pattern, lang string) {
	title := "⚠️ PERINGATAN OPERASI BERISIKO TINGGI"
	desc := "Perintah terdeteksi berpotensi destruktif terhadap sistem atau data:"
	stopLabel := "🛑 Batalkan Eksekusi"
	statusLabel := "📋 Cek Status Sistem"

	if lang == "en" {
		title = "⚠️ HIGH-RISK OPERATION WARNING"
		desc = "Detected a command with potentially destructive system or data impact:"
		stopLabel = "🛑 Abort Execution"
		statusLabel = "📋 Check System Status"
	}

	msg := fmt.Sprintf("<b>%s</b>\n%s\nPattern: <code>%s</code>\n<pre><code class=\"language-bash\">%s</code></pre>",
		title, desc, html.EscapeString(pattern), html.EscapeString(cmd))

	menu := &tele.ReplyMarkup{}
	btnStop := menu.Data(stopLabel, "cmd_stop")
	btnStatus := menu.Data(statusLabel, "cmd_status")
	menu.Inline(menu.Row(btnStop, btnStatus))

	_ = c.Send(msg, menu, tele.ModeHTML)
}
