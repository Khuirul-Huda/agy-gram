package bot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	defaultAudioExt      = ".mp3"
	defaultVoiceExt      = ".oga"
	defaultUploadsDir    = "/home/huda"
	uploadDirectoryPerms = 0755
)

// HandleVoice handles incoming voice notes from Telegram.
func (h *Handler) HandleVoice(c tele.Context) error {
	msg := c.Message()
	if msg == nil || msg.Voice == nil {
		return nil
	}

	uploadDir, err := h.ensureUploadsDir(c)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal membuat direktori upload: %v", err))
	}

	fileName := generateVoiceFileName(time.Now().Unix())
	destPath := filepath.Join(uploadDir, fileName)

	if err := h.downloadTelegramFile(c, &msg.Voice.File, destPath); err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal mengunduh voice note dari Telegram: %v", err))
	}

	caption := extractCaption(msg.Caption, msg.Voice.Caption)
	prompt := formatAudioPrompt(caption, destPath)

	_ = c.Send(fmt.Sprintf("🎙️ Voice note diterima & disimpan di <code>%s</code>. Memproses...", destPath), tele.ModeHTML)
	return h.executePrompt(c, prompt)
}

// HandleAudio handles incoming audio files from Telegram.
func (h *Handler) HandleAudio(c tele.Context) error {
	msg := c.Message()
	if msg == nil || msg.Audio == nil {
		return nil
	}

	uploadDir, err := h.ensureUploadsDir(c)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal membuat direktori upload: %v", err))
	}

	audio := msg.Audio
	ext := determineAudioExtension(audio.FileName, audio.MIME)
	fileName := generateAudioFileName(time.Now().Unix(), ext)
	destPath := filepath.Join(uploadDir, fileName)

	if err := h.downloadTelegramFile(c, &audio.File, destPath); err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal mengunduh file audio dari Telegram: %v", err))
	}

	caption := extractCaption(msg.Caption, audio.Caption)
	prompt := formatAudioPrompt(caption, destPath)

	_ = c.Send(fmt.Sprintf("🎵 Audio diterima & disimpan di <code>%s</code>. Memproses...", destPath), tele.ModeHTML)
	return h.executePrompt(c, prompt)
}

// ensureUploadsDir creates and returns the uploads directory in the current active workspace.
func (h *Handler) ensureUploadsDir(c tele.Context) (string, error) {
	workspace := defaultUploadsDir
	session := h.getSession(c)
	if session != nil {
		if current := session.GetWorkspace(); current != "" {
			workspace = current
		}
	}

	uploadDir := filepath.Join(workspace, "uploads")
	if err := os.MkdirAll(uploadDir, uploadDirectoryPerms); err != nil {
		return "", err
	}
	return uploadDir, nil
}

// downloadTelegramFile downloads a Telegram file stream and writes it to destPath.
func (h *Handler) downloadTelegramFile(c tele.Context, file *tele.File, destPath string) error {
	rc, err := c.Bot().File(file)
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return err
	}
	return nil
}

// generateVoiceFileName formats the standard file name for voice notes.
func generateVoiceFileName(timestamp int64) string {
	return fmt.Sprintf("voice_%d%s", timestamp, defaultVoiceExt)
}

// generateAudioFileName formats the standard file name for audio tracks.
func generateAudioFileName(timestamp int64, ext string) string {
	cleanExt := strings.TrimSpace(ext)
	if cleanExt == "" {
		cleanExt = defaultAudioExt
	}
	if !strings.HasPrefix(cleanExt, ".") {
		cleanExt = "." + cleanExt
	}
	return fmt.Sprintf("audio_%d%s", timestamp, cleanExt)
}

// determineAudioExtension returns a file extension deduced from file name or mime type.
func determineAudioExtension(fileName, mimeType string) string {
	ext := filepath.Ext(strings.TrimSpace(fileName))
	if ext != "" {
		return strings.ToLower(ext)
	}

	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "audio/ogg", "application/ogg":
		return ".oga"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/m4a", "audio/x-m4a", "audio/mp4":
		return ".m4a"
	case "audio/flac", "audio/x-flac":
		return ".flac"
	case "audio/aac":
		return ".aac"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	default:
		return defaultAudioExt
	}
}

// extractCaption returns the first available non-empty caption.
func extractCaption(messageCaption, mediaCaption string) string {
	if trimmed := strings.TrimSpace(messageCaption); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(mediaCaption)
}

// formatAudioPrompt builds the LLM prompt incorporating optional caption and the audio file path.
func formatAudioPrompt(caption, destPath string) string {
	trimmed := strings.TrimSpace(caption)
	if trimmed == "" {
		return fmt.Sprintf("[Audio File]: %s", destPath)
	}
	return fmt.Sprintf("%s\n\n[Audio File]: %s", trimmed, destPath)
}
