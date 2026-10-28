package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	edgeTTSBin = "/home/huda/.local/share/edge-tts-venv/bin/edge-tts"
	stripRegex = regexp.MustCompile("(?s)```.*?```|`.*?`|<.*?>|[*_#~>]")
)

// Synthesizer generates speech audio files using Edge-TTS.
type Synthesizer struct {
	binPath string
}

// NewSynthesizer initializes an Edge-TTS synthesizer instance.
func NewSynthesizer() *Synthesizer {
	bin := edgeTTSBin
	if _, err := os.Stat(bin); err != nil {
		if path, err := exec.LookPath("edge-tts"); err == nil {
			bin = path
		}
	}
	return &Synthesizer{binPath: bin}
}

// IsAvailable reports whether the Edge-TTS CLI is installed and ready.
func (s *Synthesizer) IsAvailable() bool {
	if s.binPath == "" {
		return false
	}
	_, err := os.Stat(s.binPath)
	return err == nil
}

// GenerateVoiceNote synthesizes text into a Telegram-compatible Opus .ogg voice note.
func (s *Synthesizer) GenerateVoiceNote(ctx context.Context, text, lang, outDir string) (string, error) {
	if !s.IsAvailable() {
		return "", fmt.Errorf("edge-tts binary not found at %s", s.binPath)
	}

	cleanText := cleanTextForSpeech(text)
	if cleanText == "" {
		return "", fmt.Errorf("empty text after cleaning")
	}

	voice := "id-ID-ArdiNeural"
	if lang == "en" {
		voice = "en-US-ChristopherNeural"
	}

	_ = os.MkdirAll(outDir, 0755)
	tmpMP3 := filepath.Join(outDir, fmt.Sprintf("tts_raw_%d.mp3", time.Now().UnixNano()))
	outOGG := filepath.Join(outDir, fmt.Sprintf("voice_reply_%d.ogg", time.Now().UnixNano()))

	defer os.Remove(tmpMP3)

	// 1. Synthesize MP3 via edge-tts CLI
	cmdTTS := exec.CommandContext(ctx, s.binPath, "--voice", voice, "--text", cleanText, "--write-media", tmpMP3)
	if err := cmdTTS.Run(); err != nil {
		return "", fmt.Errorf("edge-tts failed: %w", err)
	}

	// 2. Convert MP3 to Opus OGG for native Telegram voice note waveform
	cmdFF := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", tmpMP3, "-c:a", "libopus", "-b:a", "32k", outOGG)
	if err := cmdFF.Run(); err != nil {
		// Fallback to mp3 if ffmpeg fails
		return tmpMP3, nil
	}

	return outOGG, nil
}

// cleanTextForSpeech removes markdown symbols, code blocks, thinking blocks, and tags so TTS sounds natural.
func cleanTextForSpeech(s string) string {
	// 1. Remove thinking quotes if present (e.g. 🧠 ... or blockquotes)
	if idx := strings.Index(s, "🧠"); idx != -1 {
		afterThinking := s[idx:]
		if endIdx := strings.Index(afterThinking, "\n\n"); endIdx != -1 {
			s = afterThinking[endIdx+2:]
		}
	}

	cleaned := stripRegex.ReplaceAllString(s, " ")
	lines := strings.Split(cleaned, "\n")
	var goodLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasPrefix(trimmed, "⏱️") && !strings.HasPrefix(trimmed, "⏳") {
			goodLines = append(goodLines, trimmed)
		}
	}
	res := strings.Join(goodLines, ". ")

	// Limit to ~450 characters for fast ~8s synthesis on low-power CPUs
	const maxSpeechLen = 450
	if len(res) > maxSpeechLen {
		cut := maxSpeechLen
		if lastDot := strings.LastIndex(res[:cut], ". "); lastDot > 200 {
			cut = lastDot + 1
		}
		res = res[:cut] + "..."
	}
	return strings.TrimSpace(res)
}
