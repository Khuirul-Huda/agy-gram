package bot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/tts"
	tele "gopkg.in/telebot.v3"
)

// HandleVoiceTTSCallback generates and delivers an audio voice note for the message asynchronously.
func (h *Handler) HandleVoiceTTSCallback(c tele.Context) error {
	_ = c.Respond(&tele.CallbackResponse{Text: "🎙️ Sedang membuat audio suara..."})
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
	bot := c.Bot()
	recipient := c.Recipient()

	// Execute synthesis asynchronously so it never blocks the bot or active agent turns
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		synth := tts.NewSynthesizer()
		if !synth.IsAvailable() {
			_, _ = bot.Send(recipient, "❌ Layanan Edge-TTS belum tersedia di server.")
			return
		}

		voicePath, err := synth.GenerateVoiceNote(ctx, text, lang, outDir)
		if err != nil {
			_, _ = bot.Send(recipient, fmt.Sprintf("❌ Gagal membuat suara: %v", err))
			return
		}
		defer os.Remove(voicePath)

		voice := &tele.Voice{
			File:    tele.FromDisk(voicePath),
			Caption: "🔊 <b>Balasan Suara Edge-TTS</b>",
		}

		_, _ = bot.Send(recipient, voice, tele.ModeHTML)
	}()

	return nil
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
