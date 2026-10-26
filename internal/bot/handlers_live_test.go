package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

func TestLiveTracker_BuildProgressHTML(t *testing.T) {
	tracker := &liveTracker{
		lang:         "id",
		startTime:    time.Now().Add(-5 * time.Second),
		lastActivity: "Sedang menganalisis...",
		lastThinking: "Memeriksa arsitektur sistem",
	}

	html := tracker.buildProgressHTML()

	if !strings.Contains(html, "Antigravity sedang bekerja") {
		t.Errorf("expected html to contain 'Antigravity sedang bekerja', got: %s", html)
	}
	if !strings.Contains(html, "Sedang menganalisis...") {
		t.Errorf("expected html to contain activity, got: %s", html)
	}
	if !strings.Contains(html, "<blockquote expandable>") {
		t.Errorf("expected html to contain expandable blockquote, got: %s", html)
	}
	if !strings.Contains(html, "Memeriksa arsitektur sistem") {
		t.Errorf("expected html to contain thinking, got: %s", html)
	}
}

func TestLiveTracker_RecordTool(t *testing.T) {
	tracker := &liveTracker{
		lang:         "en",
		startTime:    time.Now(),
		lastEditTime: time.Now(),
	}

	tests := []struct {
		event domain.ToolEvent
		want  string
	}{
		{
			event: domain.ToolEvent{Kind: domain.ToolKindCommand, Target: "git status"},
			want:  "⚙️ <code>git status</code>",
		},
		{
			event: domain.ToolEvent{Kind: domain.ToolKindRead, Target: "/home/huda/test.go"},
			want:  "📖 <code>/home/huda/test.go</code>",
		},
		{
			event: domain.ToolEvent{Kind: domain.ToolKindEdit, Target: "/home/huda/main.go"},
			want:  "📝 <code>/home/huda/main.go</code>",
		},
		{
			event: domain.ToolEvent{Kind: domain.ToolKindWrite, Target: "/home/huda/new.txt"},
			want:  "✍️ <code>/home/huda/new.txt</code>",
		},
	}

	for _, tt := range tests {
		tracker.RecordTool(tt.event)
		if tracker.lastActivity != tt.want {
			t.Errorf("expected lastActivity '%s', got '%s'", tt.want, tracker.lastActivity)
		}
	}
}

func TestLiveTracker_RecordThinking_FullText(t *testing.T) {
	tracker := &liveTracker{
		lang:         "id",
		startTime:    time.Now(),
		lastEditTime: time.Now(),
	}

	// 500 characters should be preserved in full without artificial truncation
	longThinking := strings.Repeat("A", 500)
	tracker.RecordThinking(longThinking)

	if len(tracker.lastThinking) != 500 {
		t.Errorf("expected fulltext thinking len 500, got %d", len(tracker.lastThinking))
	}
	if strings.HasPrefix(tracker.lastThinking, "...") {
		t.Errorf("expected thinking not to have '...' prefix")
	}

	// Beyond maxTelegramThinkingLen (3500) should be safely capped to prevent Telegram API errors
	hugeThinking := strings.Repeat("B", 4000)
	tracker.RecordThinking(hugeThinking)
	if len(tracker.lastThinking) <= 3500 {
		t.Errorf("expected thinking to be capped around maxTelegramThinkingLen")
	}
}
