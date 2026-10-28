package bot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/tts"
	tele "gopkg.in/telebot.v3"
)

const (
	ttsSynthesisTimeout = 60 * time.Second
	ttsActionInterval   = 3500 * time.Millisecond
)

type ttsLabels struct {
	rendering string
	uploading string
	replay    string
	initial   string
	toast     string
}

func getTTSLabels(lang string) ttsLabels {
	if lang == "en" {
		return ttsLabels{
			rendering: "⏳ Rendering Audio...",
			uploading: "⏳ Sending Audio...",
			replay:    "🔊 Listen Again",
			initial:   "🔊 Listen Voice",
			toast:     "🎙️ Rendering voice note...",
		}
	}
	return ttsLabels{
		rendering: "⏳ Sedang Merender Audio...",
		uploading: "⏳ Sedang Mengirim Audio...",
		replay:    "🔊 Dengar Ulang Suara",
		initial:   "🔊 Dengar Suara",
		toast:     "🎙️ Sedang merender audio...",
	}
}

// HandleVoiceTTSCallback generates and delivers an audio voice note for the message asynchronously.
func (h *Handler) HandleVoiceTTSCallback(c tele.Context) error {
	if c.Callback() == nil || c.Callback().Message == nil {
		return nil
	}

	session := h.getSession(c)
	lang := session.GetLanguage()
	labels := getTTSLabels(lang)

	_ = c.Respond(&tele.CallbackResponse{Text: labels.toast})

	msg := c.Callback().Message
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	if text == "" {
		return nil
	}

	bot := c.Bot()
	recipient := c.Recipient()

	// Immediate visual feedback on the button to prevent duplicate clicks
	updateTTSButtonState(bot, msg, labels.rendering, "voice_loading")

	ws := session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}
	outDir := filepath.Join(ws, "uploads")

	// Execute synthesis asynchronously so it never blocks the bot or active agent turns
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), ttsSynthesisTimeout)
		defer cancel()

		synth := tts.NewSynthesizer()
		if !synth.IsAvailable() {
			updateTTSButtonState(bot, msg, labels.initial, "voice_tts")
			_, _ = bot.Send(recipient, "❌ Layanan Edge-TTS belum tersedia di server.")
			return
		}

		stopTicker := startActionTicker(bot, recipient, tele.RecordingAudio, ttsActionInterval)
		voicePath, err := synth.GenerateVoiceNote(ctx, text, lang, outDir)
		stopTicker()

		if err != nil {
			updateTTSButtonState(bot, msg, labels.initial, "voice_tts")
			_, _ = bot.Send(recipient, fmt.Sprintf("❌ Gagal membuat suara: %v", err))
			return
		}
		defer os.Remove(voicePath)

		// Indicate upload status on button and chat header
		updateTTSButtonState(bot, msg, labels.uploading, "voice_loading")
		_ = bot.Notify(recipient, tele.UploadingAudio)

		caption := "🔊 <b>Balasan Suara Edge-TTS</b>"
		if lang == "en" {
			caption = "🔊 <b>Edge-TTS Voice Reply</b>"
		}

		voice := &tele.Voice{
			File:    tele.FromDisk(voicePath),
			Caption: caption,
		}

		_, err = bot.Send(recipient, voice, tele.ModeHTML)
		if err != nil {
			updateTTSButtonState(bot, msg, labels.initial, "voice_tts")
			_, _ = bot.Send(recipient, fmt.Sprintf("❌ Gagal mengirim audio: %v", err))
			return
		}

		// Restore button to replay state
		updateTTSButtonState(bot, msg, labels.replay, "voice_tts")
	}()

	return nil
}

// startActionTicker sends recurring chat action notifications until the returned stop function is called.
func startActionTicker(bot *tele.Bot, recipient tele.Recipient, action tele.ChatAction, interval time.Duration) func() {
	stopChan := make(chan struct{})
	_ = bot.Notify(recipient, action)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = bot.Notify(recipient, action)
			case <-stopChan:
				return
			}
		}
	}()

	return func() {
		select {
		case <-stopChan:
		default:
			close(stopChan)
		}
	}
}

// isTTSButton identifies whether an inline button is associated with TTS actions.
func isTTSButton(btn tele.InlineButton) bool {
	if btn.Unique == "voice_tts" || btn.Unique == "voice_loading" {
		return true
	}
	if strings.Contains(btn.Data, "voice_tts") || strings.Contains(btn.Data, "voice_loading") {
		return true
	}
	if strings.Contains(btn.Text, "Dengar") || strings.Contains(btn.Text, "Listen") {
		return true
	}
	return false
}

// updateTTSButtonState updates the label and callback data of the TTS button in the message.
func updateTTSButtonState(bot *tele.Bot, msg *tele.Message, newLabel, newUnique string) {
	if bot == nil || msg == nil {
		return
	}

	markup := &tele.ReplyMarkup{}
	found := false

	if msg.ReplyMarkup != nil && len(msg.ReplyMarkup.InlineKeyboard) > 0 {
		var newKeyboard [][]tele.InlineButton
		for _, row := range msg.ReplyMarkup.InlineKeyboard {
			var newRow []tele.InlineButton
			for _, btn := range row {
				if isTTSButton(btn) {
					btn.Text = newLabel
					btn.Unique = newUnique
					btn.Data = ""
					found = true
				}
				newRow = append(newRow, btn)
			}
			newKeyboard = append(newKeyboard, newRow)
		}
		markup.InlineKeyboard = newKeyboard
	}

	if !found {
		btn := markup.Data(newLabel, newUnique)
		if ib := btn.Inline(); ib != nil {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []tele.InlineButton{*ib})
		}
	}

	if editedMsg, err := bot.EditReplyMarkup(msg, markup); err == nil && editedMsg != nil {
		msg.ReplyMarkup = editedMsg.ReplyMarkup
	} else {
		msg.ReplyMarkup = markup
	}
}

// attachTTSButton adds a [🔊 Dengar Suara] button to the reply markup.
func attachTTSButton(menu *tele.ReplyMarkup, lang string) *tele.ReplyMarkup {
	if menu == nil {
		menu = &tele.ReplyMarkup{}
	}

	labels := getTTSLabels(lang)
	btnTTS := menu.Data(labels.initial, "voice_tts")
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
