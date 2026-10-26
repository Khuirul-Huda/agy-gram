package bot

import (
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	tele "gopkg.in/telebot.v3"
)

const (
	liveThrottleInterval   = 800 * time.Millisecond
	maxTelegramThinkingLen = 3500
)

// liveTracker manages in-place live progress, real-time delta streaming, and thinking updates on a single Telegram message.
type liveTracker struct {
	bot          *tele.Bot
	recipient    tele.Recipient
	msg          *tele.Message
	lang         string
	startTime    time.Time
	lastEditTime time.Time
	mu           sync.Mutex

	lastActivity string
	lastThinking string
	streamBuf    strings.Builder
}

func newLiveTracker(b *tele.Bot, recipient tele.Recipient, initialMsg *tele.Message, lang string) *liveTracker {
	return &liveTracker{
		bot:          b,
		recipient:    recipient,
		msg:          initialMsg,
		lang:         lang,
		startTime:    time.Now(),
		lastEditTime: time.Now(),
		lastActivity: defaultActivity(lang),
	}
}

func defaultActivity(lang string) string {
	if lang == "en" {
		return "Reasoning..."
	}
	return "Sedang menganalisis & merencanakan..."
}

// RecordTool updates the current tool action being performed without artificial truncation.
func (t *liveTracker) RecordTool(event domain.ToolEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	target := html.EscapeString(event.Target)

	switch event.Kind {
	case domain.ToolKindCommand:
		t.lastActivity = fmt.Sprintf("⚙️ <code>%s</code>", target)
	case domain.ToolKindRead:
		t.lastActivity = fmt.Sprintf("📖 <code>%s</code>", target)
	case domain.ToolKindEdit:
		t.lastActivity = fmt.Sprintf("📝 <code>%s</code>", target)
	case domain.ToolKindWrite:
		t.lastActivity = fmt.Sprintf("✍️ <code>%s</code>", target)
	case domain.ToolKindQuestion:
		t.lastActivity = fmt.Sprintf("❓ <b>%s</b>", target)
	default:
		t.lastActivity = fmt.Sprintf("🔧 <code>%s</code>", target)
	}

	t.maybeRenderLocked()
}

// RecordThinking updates the live thinking fulltext and streams immediately.
func (t *liveTracker) RecordThinking(thinking string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cleanThinking := strings.TrimSpace(thinking)
	if cleanThinking == "" {
		return
	}

	// Preserve fulltext thinking up to Telegram's maximum safe character limit
	if len(cleanThinking) > maxTelegramThinkingLen {
		cleanThinking = cleanThinking[:maxTelegramThinkingLen] + "\n...(dipotong karena batas pesan Telegram)..."
	}

	t.lastThinking = cleanThinking
	t.maybeRenderLocked()
}

// RecordDelta appends streamed response text and triggers real-time in-place editing.
func (t *liveTracker) RecordDelta(delta string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if delta == "" {
		return
	}
	t.streamBuf.WriteString(delta)
	t.maybeRenderLocked()
}

// maybeRenderLocked edits the message in place if throttle interval has elapsed.
func (t *liveTracker) maybeRenderLocked() {
	if t.msg == nil || t.bot == nil {
		return
	}

	if time.Since(t.lastEditTime) < liveThrottleInterval {
		return
	}

	text := t.buildProgressHTML()
	t.lastEditTime = time.Now()

	go func(m *tele.Message, content string) {
		_, _ = safeEditMessage(t.bot, m, content, tele.ModeHTML)
	}(t.msg, text)
}

func (t *liveTracker) buildProgressHTML() string {
	var sb strings.Builder

	// 1. Fulltext thinking expandable quote block at the top if present
	if t.lastThinking != "" {
		escaped := html.EscapeString(t.lastThinking)
		sb.WriteString(fmt.Sprintf("<blockquote expandable>🧠 <i>%s</i></blockquote>\n\n", escaped))
	}

	// 2. If the agent has started emitting response text deltas, render the response stream live below thinking!
	if t.streamBuf.Len() > 0 {
		currentStream := t.streamBuf.String()
		maxStreamLen := 3500 - sb.Len()
		if maxStreamLen < 500 {
			maxStreamLen = 500
		}
		if len(currentStream) > maxStreamLen {
			currentStream = currentStream[:maxStreamLen] + "\n...(streaming)..."
		}
		// Balance unclosed code blocks if currently mid-stream
		if strings.Count(currentStream, "```")%2 != 0 {
			currentStream += "\n```"
		}
		htmlText := MarkdownToTelegramHTML(currentStream)
		sb.WriteString(htmlText)
		sb.WriteString(" ▍")
		return sb.String()
	}

	elapsed := int(time.Since(t.startTime).Seconds())
	title := "Antigravity sedang bekerja"
	actLabel := "Aktivitas"
	if t.lang == "en" {
		title = "Antigravity is working"
		actLabel = "Activity"
	}

	sb.WriteString(fmt.Sprintf("⏳ <b>%s...</b> (%ds)\n\n", title, elapsed))
	sb.WriteString(fmt.Sprintf("▫️ <b>%s:</b> %s", actLabel, t.lastActivity))

	return sb.String()
}

// OverwriteFinal replaces the live progress message directly with the final agent response.
func (t *liveTracker) OverwriteFinal(c tele.Context, output string, buttons *tele.ReplyMarkup) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cleanOutput := FormatMarkdownTables(output)
	chunks := SplitMessage(cleanOutput)
	if len(chunks) == 0 {
		chunks = []string{"(No response returned by agent)"}
	}

	firstChunkHTML := MarkdownToTelegramHTML(chunks[0])

	// If single chunk and buttons exist, attach buttons directly to overwritten message
	if len(chunks) == 1 && buttons != nil && len(buttons.InlineKeyboard) > 0 {
		if t.msg != nil {
			_, err := safeEditMessage(t.bot, t.msg, firstChunkHTML, buttons, tele.ModeHTML)
			return err
		}
		return c.Send(firstChunkHTML, buttons, tele.ModeHTML)
	}

	// Overwrite first chunk
	if t.msg != nil {
		_, _ = safeEditMessage(t.bot, t.msg, firstChunkHTML, tele.ModeHTML)
	} else {
		_ = c.Send(firstChunkHTML, tele.ModeHTML)
	}

	// Send remaining chunks
	for i := 1; i < len(chunks); i++ {
		chunkHTML := MarkdownToTelegramHTML(chunks[i])
		isLast := i == len(chunks)-1

		if isLast && buttons != nil && len(buttons.InlineKeyboard) > 0 {
			_ = c.Send(chunkHTML, buttons, tele.ModeHTML)
		} else {
			_ = c.Send(chunkHTML, tele.ModeHTML)
		}
	}

	return nil
}

// OverwriteError replaces the live progress message with an error report.
func (t *liveTracker) OverwriteError(c tele.Context, errMsg string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	errHTML := fmt.Sprintf("❌ <b>Error:</b>\n<pre><code>%s</code></pre>", html.EscapeString(errMsg))
	if t.msg != nil {
		_, err := safeEditMessage(t.bot, t.msg, errHTML, tele.ModeHTML)
		return err
	}
	return c.Send(errHTML, tele.ModeHTML)
}

// safeEditMessage safely edits a Telegram message while ignoring harmless 'not modified' errors.
func safeEditMessage(b *tele.Bot, msg *tele.Message, what interface{}, opts ...interface{}) (*tele.Message, error) {
	if b == nil || msg == nil {
		return nil, nil
	}
	newMsg, err := b.Edit(msg, what, opts...)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return msg, nil
		}
		return nil, err
	}
	return newMsg, nil
}
