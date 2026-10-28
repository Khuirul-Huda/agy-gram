package bot

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGetTTSLabels(t *testing.T) {
	idLabels := getTTSLabels("id")
	if !strings.Contains(idLabels.rendering, "Merender") {
		t.Errorf("expected Indonesian rendering label, got %q", idLabels.rendering)
	}
	if !strings.Contains(idLabels.uploading, "Mengirim") {
		t.Errorf("expected Indonesian uploading label, got %q", idLabels.uploading)
	}
	if !strings.Contains(idLabels.replay, "Dengar Ulang") {
		t.Errorf("expected Indonesian replay label, got %q", idLabels.replay)
	}

	enLabels := getTTSLabels("en")
	if !strings.Contains(enLabels.rendering, "Rendering") {
		t.Errorf("expected English rendering label, got %q", enLabels.rendering)
	}
	if !strings.Contains(enLabels.uploading, "Sending") {
		t.Errorf("expected English uploading label, got %q", enLabels.uploading)
	}
	if !strings.Contains(enLabels.replay, "Listen Again") {
		t.Errorf("expected English replay label, got %q", enLabels.replay)
	}
}

func TestIsTTSButton(t *testing.T) {
	ttsCases := []struct {
		name     string
		btn      tele.InlineButton
		expected bool
	}{
		{
			name:     "Unique voice_tts",
			btn:      tele.InlineButton{Unique: "voice_tts"},
			expected: true,
		},
		{
			name:     "Unique voice_loading",
			btn:      tele.InlineButton{Unique: "voice_loading"},
			expected: true,
		},
		{
			name:     "Data with voice_tts",
			btn:      tele.InlineButton{Data: "\fvoice_tts"},
			expected: true,
		},
		{
			name:     "Data with voice_loading",
			btn:      tele.InlineButton{Data: "\fvoice_loading"},
			expected: true,
		},
		{
			name:     "Text with Dengar",
			btn:      tele.InlineButton{Text: "🔊 Dengar Suara"},
			expected: true,
		},
		{
			name:     "Text with Listen",
			btn:      tele.InlineButton{Text: "🔊 Listen Again"},
			expected: true,
		},
		{
			name:     "Git diff button should be false",
			btn:      tele.InlineButton{Unique: "git_diff", Data: "\fgit_diff", Text: "📝 Git Diff"},
			expected: false,
		},
		{
			name:     "Unrelated button should be false",
			btn:      tele.InlineButton{Unique: "model_set", Data: "\fmodel_set|flash", Text: "⚡ Flash"},
			expected: false,
		},
	}

	for _, tc := range ttsCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := isTTSButton(tc.btn)
			if actual != tc.expected {
				t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, actual)
			}
		})
	}
}

func TestAttachTTSButton(t *testing.T) {
	// 1. Attach to empty menu
	menu := attachTTSButton(nil, "id")
	if len(menu.InlineKeyboard) != 1 {
		t.Fatalf("expected 1 row, got %d", len(menu.InlineKeyboard))
	}
	if !isTTSButton(menu.InlineKeyboard[0][0]) {
		t.Errorf("expected attached button to be TTS button")
	}

	// 2. Attach to existing menu with other buttons
	existingMenu := &tele.ReplyMarkup{}
	existingBtn := existingMenu.Data("Git Diff", "git_diff")
	existingMenu.Inline(existingMenu.Row(existingBtn))

	updatedMenu := attachTTSButton(existingMenu, "en")
	if len(updatedMenu.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(updatedMenu.InlineKeyboard))
	}
	if isTTSButton(updatedMenu.InlineKeyboard[0][0]) {
		t.Errorf("first row should remain Git Diff")
	}
	if !isTTSButton(updatedMenu.InlineKeyboard[1][0]) {
		t.Errorf("second row should be TTS button")
	}
}
