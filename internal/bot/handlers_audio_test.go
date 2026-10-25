package bot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	tele "gopkg.in/telebot.v3"
)

type stubSessionManager struct {
	domain.SessionManager
	workspace string
}

func (s *stubSessionManager) GetWorkspace() string {
	return s.workspace
}

type stubContext struct {
	tele.Context
	msg *tele.Message
}

func (s *stubContext) Message() *tele.Message {
	return s.msg
}

func TestDetermineAudioExtension(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mimeType string
		expected string
	}{
		{
			name:     "Preserve standard mp3 extension",
			fileName: "sample.mp3",
			mimeType: "audio/mpeg",
			expected: ".mp3",
		},
		{
			name:     "Preserve and lowercase WAV extension",
			fileName: "RECORDING.WAV",
			mimeType: "",
			expected: ".wav",
		},
		{
			name:     "Preserve m4a extension",
			fileName: "audio_track.m4a",
			mimeType: "",
			expected: ".m4a",
		},
		{
			name:     "MIME audio/ogg fallback",
			fileName: "voice_note",
			mimeType: "audio/ogg",
			expected: ".oga",
		},
		{
			name:     "MIME application/ogg fallback",
			fileName: "",
			mimeType: "application/ogg",
			expected: ".oga",
		},
		{
			name:     "MIME audio/wav fallback",
			fileName: "",
			mimeType: "audio/wav",
			expected: ".wav",
		},
		{
			name:     "MIME audio/mp4 fallback",
			fileName: "",
			mimeType: "audio/mp4",
			expected: ".m4a",
		},
		{
			name:     "MIME audio/flac fallback",
			fileName: "",
			mimeType: "audio/flac",
			expected: ".flac",
		},
		{
			name:     "MIME audio/aac fallback",
			fileName: "",
			mimeType: "audio/aac",
			expected: ".aac",
		},
		{
			name:     "Default to mp3 for unknown MIME",
			fileName: "unknown_audio",
			mimeType: "application/octet-stream",
			expected: ".mp3",
		},
		{
			name:     "Default to mp3 when both empty",
			fileName: "",
			mimeType: "",
			expected: ".mp3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := determineAudioExtension(tc.fileName, tc.mimeType)
			if actual != tc.expected {
				t.Errorf("determineAudioExtension(%q, %q) = %q; want %q", tc.fileName, tc.mimeType, actual, tc.expected)
			}
		})
	}
}

func TestGenerateVoiceFileName(t *testing.T) {
	timestamp := int64(1729000000)
	expected := "voice_1729000000.oga"
	actual := generateVoiceFileName(timestamp)

	if actual != expected {
		t.Errorf("generateVoiceFileName(%d) = %q; want %q", timestamp, actual, expected)
	}
}

func TestGenerateAudioFileName(t *testing.T) {
	timestamp := int64(1729000000)

	tests := []struct {
		name     string
		ext      string
		expected string
	}{
		{
			name:     "With dot prefix",
			ext:      ".wav",
			expected: "audio_1729000000.wav",
		},
		{
			name:     "Without dot prefix",
			ext:      "flac",
			expected: "audio_1729000000.flac",
		},
		{
			name:     "Empty extension fallback to default mp3",
			ext:      "",
			expected: "audio_1729000000.mp3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := generateAudioFileName(timestamp, tc.ext)
			if actual != tc.expected {
				t.Errorf("generateAudioFileName(%d, %q) = %q; want %q", timestamp, tc.ext, actual, tc.expected)
			}
		})
	}
}

func TestExtractCaption(t *testing.T) {
	tests := []struct {
		name         string
		msgCaption   string
		mediaCaption string
		expected     string
	}{
		{
			name:         "Message caption present",
			msgCaption:   "  Help me summarize this  ",
			mediaCaption: "",
			expected:     "Help me summarize this",
		},
		{
			name:         "Media caption fallback",
			msgCaption:   "   ",
			mediaCaption: "Track caption",
			expected:     "Track caption",
		},
		{
			name:         "Message caption takes precedence",
			msgCaption:   "Priority caption",
			mediaCaption: "Ignored caption",
			expected:     "Priority caption",
		},
		{
			name:         "Both empty",
			msgCaption:   "  ",
			mediaCaption: "",
			expected:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := extractCaption(tc.msgCaption, tc.mediaCaption)
			if actual != tc.expected {
				t.Errorf("extractCaption(%q, %q) = %q; want %q", tc.msgCaption, tc.mediaCaption, actual, tc.expected)
			}
		})
	}
}

func TestFormatAudioPrompt(t *testing.T) {
	filePath := "/home/huda/uploads/voice_1729000000.oga"

	tests := []struct {
		name     string
		caption  string
		expected string
	}{
		{
			name:     "Without caption",
			caption:  "",
			expected: "[Audio File]: /home/huda/uploads/voice_1729000000.oga",
		},
		{
			name:     "Whitespace only caption",
			caption:  "   \n  ",
			expected: "[Audio File]: /home/huda/uploads/voice_1729000000.oga",
		},
		{
			name:     "With custom caption",
			caption:  "Tolong transkrip audio ini",
			expected: "Tolong transkrip audio ini\n\n[Audio File]: /home/huda/uploads/voice_1729000000.oga",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatAudioPrompt(tc.caption, filePath)
			if actual != tc.expected {
				t.Errorf("formatAudioPrompt(%q, %q) = %q; want %q", tc.caption, filePath, actual, tc.expected)
			}
		})
	}
}

func TestEnsureUploadsDir(t *testing.T) {
	tempWorkspace := t.TempDir()
	handler := &Handler{
		session: &stubSessionManager{
			workspace: tempWorkspace,
		},
	}

	uploadsDir, err := handler.ensureUploadsDir(nil)
	if err != nil {
		t.Fatalf("ensureUploadsDir() returned error: %v", err)
	}

	expectedDir := filepath.Join(tempWorkspace, "uploads")
	if uploadsDir != expectedDir {
		t.Errorf("ensureUploadsDir() = %q; want %q", uploadsDir, expectedDir)
	}

	fi, err := os.Stat(uploadsDir)
	if err != nil {
		t.Fatalf("failed to stat uploads dir: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("expected %q to be directory", uploadsDir)
	}
}

func TestHandleVoice_GuardClauses(t *testing.T) {
	handler := &Handler{}

	// Test nil message
	if err := handler.HandleVoice(&stubContext{msg: nil}); err != nil {
		t.Errorf("expected nil error on nil message, got %v", err)
	}

	// Test message without voice
	if err := handler.HandleVoice(&stubContext{msg: &tele.Message{}}); err != nil {
		t.Errorf("expected nil error on message without voice, got %v", err)
	}
}

func TestHandleAudio_GuardClauses(t *testing.T) {
	handler := &Handler{}

	// Test nil message
	if err := handler.HandleAudio(&stubContext{msg: nil}); err != nil {
		t.Errorf("expected nil error on nil message, got %v", err)
	}

	// Test message without audio
	if err := handler.HandleAudio(&stubContext{msg: &tele.Message{}}); err != nil {
		t.Errorf("expected nil error on message without audio, got %v", err)
	}
}
