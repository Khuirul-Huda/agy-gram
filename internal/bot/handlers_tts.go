package bot

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/tts"
	tele "gopkg.in/telebot.v3"
)

// HandleVoiceTTSCallback generates and delivers an audio voice note for the message.
func (h *Handler) HandleVoiceTTSCallback(c tele.Context) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "Menghasilkan suara Edge-TTS..."})
	if c.Callback() == nil || c.Callback().Message == nil {
		return nil
	}

	session := h.getSession(c)
	lang := session.GetLanguage()
	text := c.Callback().Message.Text

	ws := session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}
	outDir := filepath.Join(ws, "uploads")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	synth := tts.NewSynthesizer()
	if !synth.IsAvailable() {
		return c.Send("❌ Layanan Edge-TTS belum tersedia di server.")
	}

	voicePath, err := synth.GenerateVoiceNote(ctx, text, lang, outDir)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal membuat suara: %v", err))
	}

	voice := &tele.Voice{
		File:    tele.FromDisk(voicePath),
		Caption: "🔊 <b>Balasan Suara Edge-TTS</b>",
	}

	return c.Send(voice, tele.ModeHTML)
}

// attachTTSButton adds a [🔊 Dengar Suara] button to the reply markup.
func attachTTSButton(menu *tele.ReplyMarkup, lang string) *tele.ReplyMarkup {
	if menu == nil {
		menu = &tele.ReplyMarkup{}
	}

	btnLabel := "🔊 Dengar Suara"
	if lang == "en" {
		btnLabel = "🔊 Listen Voice"
	}

	btnTTS := menu.Data(btnLabel, "voice_tts")
	ttsRow := menu.Row(btnTTS)

	if len(menu.InlineKeyboard) > 0 {
		var iRow []tele.InlineButton
		for _, b := range ttsRow {
			if ib := b.Inline(); ib != nil {
				iRow = append(iRow, *ib)
			}
		}
		if len(iRow) > 0 {
			menu.InlineKeyboard = append(menu.InlineKeyboard, iRow)
			return menu
		}
	}

	menu.Inline(ttsRow)
	return menu
}
