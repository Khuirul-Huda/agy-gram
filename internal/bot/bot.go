package bot

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/config"
	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	"github.com/Khuirul-Huda/agy-gram/internal/storage"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

// Bot wraps telebot.Bot with routing and middlewares.
type Bot struct {
	teleBot *tele.Bot
	cfg     *config.Config
	handler *Handler
	session domain.SessionManager
	store   *storage.BoltStore
}

// NewBot initializes a new Telegram bot instance.
func NewBot(cfg *config.Config, handler *Handler, session domain.SessionManager, store ...*storage.BoltStore) (*Bot, error) {
	client := &http.Client{Timeout: time.Minute}
	if cfg.TelegramProxy != "" {
		if proxyURL, err := url.Parse(cfg.TelegramProxy); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			slog.Info("Using configured outbound proxy for Telegram Bot API", "proxy", cfg.TelegramProxy)
		} else {
			slog.Warn("Failed to parse Telegram proxy URL, falling back to direct connection", "error", err)
		}
	}

	pref := tele.Settings{
		Token:  cfg.TelegramBotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		Client: client,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	var sStore *storage.BoltStore
	if len(store) > 0 {
		sStore = store[0]
	}

	bot := &Bot{
		teleBot: b,
		cfg:     cfg,
		handler: handler,
		session: session,
		store:   sStore,
	}

	bot.registerMiddlewares()
	bot.registerRoutes()
	bot.registerBotCommands()

	return bot, nil
}

func (b *Bot) registerBotCommands() {
	cmds := []tele.Command{
		{Text: "menu", Description: "Buka panel kontrol utama interaktif"},
		{Text: "start", Description: "Mulai dan info bot"},
		{Text: "help", Description: "Panduan lengkap penggunaan"},
		{Text: "new", Description: "Mulai sesi percakapan baru"},
		{Text: "model", Description: "Pilih model AI (Gemini, Claude, GPT)"},
		{Text: "effort", Description: "Atur tingkat reasoning effort"},
		{Text: "mode", Description: "Atur mode (auto, plan, accept-edits)"},
		{Text: "thinking", Description: "Toggle verbose thinking stream"},
		{Text: "sessions", Description: "Pilih / beralih riwayat sesi percakapan"},
		{Text: "workspace", Description: "Ganti target direktori kerja AGY"},
		{Text: "diff", Description: "Lihat uncommitted git diff di workspace"},
		{Text: "commit", Description: "Commit perubahan git (atau biarkan agy analisa)"},
		{Text: "undo", Description: "Batalkan perubahan uncommitted git"},
		{Text: "status", Description: "Cek hardware & status server"},
		{Text: "usage", Description: "Cek kuota token & limit model AI"},
		{Text: "restart", Description: "Restart aman service AGY-Gram"},
		{Text: "stop", Description: "Hentikan proses yang sedang berjalan"},
	}

	// 1. Register default scope
	if err := b.teleBot.SetCommands(cmds); err != nil {
		slog.Warn("Gagal mendaftarkan menu perintah default ke Telegram", "err", err)
	}

	// 2. Register for private chats explicitly
	if err := b.teleBot.SetCommands(cmds, tele.CommandScope{Type: tele.CommandScopeAllPrivateChats}); err != nil {
		slog.Warn("Gagal mendaftarkan menu perintah all_private_chats ke Telegram", "err", err)
	} else {
		slog.Info("Menu perintah bot (all_private_chats) berhasil didaftarkan ke Telegram")
	}

	// 3. Set chat menu button to show commands menu
	if _, err := b.teleBot.Raw("setChatMenuButton", map[string]interface{}{
		"menu_button": map[string]string{"type": "commands"},
	}); err != nil {
		slog.Warn("Gagal mengatur menu button ke commands", "err", err)
	}
}

// Start begins listening for updates (blocking).
func (b *Bot) Start() {
	slog.Info("Starting Telegram bot polling", "bot_user", b.teleBot.Me.Username)

	// Check if this startup was triggered by a restart request and recover active or completed turns
	CheckAndNotifyRestart(b.teleBot, b.session, b.cfg, b.store)

	b.teleBot.Start()
}

// Stop gracefully stops the Telegram bot poller.
func (b *Bot) Stop() {
	slog.Info("Stopping Telegram bot...")
	b.teleBot.Stop()
}

func (b *Bot) registerMiddlewares() {
	// Auto recover from panics in handlers
	b.teleBot.Use(middleware.Recover())

	// Whitelist authentication middleware
	b.teleBot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender == nil {
				return nil
			}

			if !b.cfg.IsUserAllowed(sender.ID) {
				slog.Warn("Unauthorized access attempt",
					"user_id", sender.ID,
					"username", sender.Username,
					"first_name", sender.FirstName,
				)
				return c.Send(fmt.Sprintf("⛔ *Akses Ditolak*\nUser ID Telegram Anda: `%d`\n\nSilakan daftarkan ID ini ke file `.env` di variabel `ALLOWED_USER_IDS`.", sender.ID), tele.ModeMarkdown)
			}

			return next(c)
		}
	})
}

func (b *Bot) registerRoutes() {
	b.teleBot.Handle("/start", b.handler.HandleStart)
	b.teleBot.Handle("/menu", b.handler.HandleMenu)
	b.teleBot.Handle("/help", b.handler.HandleHelp)
	b.teleBot.Handle("/new", b.handler.HandleNew)
	b.teleBot.Handle("/status", b.handler.HandleStatus)
	b.teleBot.Handle("/usage", b.handler.HandleUsage)
	b.teleBot.Handle("/mode", b.handler.HandleMode)
	b.teleBot.Handle("/model", b.handler.HandleModel)
	b.teleBot.Handle("/effort", b.handler.HandleEffort)
	b.teleBot.Handle("/lang", b.handler.HandleLang)
	b.teleBot.Handle("/thinking", b.handler.HandleThinking)
	b.teleBot.Handle("/sessions", b.handler.HandleSessions)
	b.teleBot.Handle("/workspace", b.handler.HandleWorkspace)
	b.teleBot.Handle("/restart", b.handler.HandleRestart)
	b.teleBot.Handle("/stop", b.handler.HandleCancel)
	b.teleBot.Handle("/cancel", b.handler.HandleCancel)
	b.teleBot.Handle("/diff", b.handler.HandleDiff)
	b.teleBot.Handle("/commit", b.handler.HandleCommit)
	b.teleBot.Handle("/undo", b.handler.HandleUndo)

	// Inline keyboard callback handlers
	b.teleBot.Handle(&tele.Btn{Unique: "main_menu"}, b.handler.HandleMenuCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "model_menu"}, b.handler.HandleModelMenuCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "model_set"}, b.handler.HandleSetModelCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "model_reset"}, b.handler.HandleResetModelCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "effort_menu"}, b.handler.HandleEffortMenuCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "effort_set"}, b.handler.HandleSetEffortCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "mode_menu"}, func(c tele.Context) error {
		_ = c.Respond()
		text, menu := b.handler.buildModeMenu()
		return safeEdit(c, text, menu, tele.ModeHTML)
	})
	b.teleBot.Handle(&tele.Btn{Unique: "mode_set"}, b.handler.HandleSetModeCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "thinking_menu"}, func(c tele.Context) error {
		_ = c.Respond()
		text, menu := b.handler.buildThinkingMenu()
		return safeEdit(c, text, menu, tele.ModeHTML)
	})
	b.teleBot.Handle(&tele.Btn{Unique: "thinking_set"}, b.handler.HandleSetThinkingCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "lang_menu"}, b.handler.HandleLang)
	b.teleBot.Handle(&tele.Btn{Unique: "lang_set"}, b.handler.HandleSetLangCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_status"}, b.handler.HandleStatusCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_usage"}, b.handler.HandleUsageCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_new"}, b.handler.HandleNewCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_restart"}, b.handler.HandleRestart)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_stop"}, b.handler.HandleStopCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_sessions"}, b.handler.HandleSessionsCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "cmd_workspace"}, b.handler.HandleWorkspace)
	b.teleBot.Handle(&tele.Btn{Unique: "session_switch"}, b.handler.HandleSessionSwitchCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "ws_set"}, b.handler.HandleWorkspaceCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "file_download"}, b.handler.HandleFileDownloadCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "user_choice"}, b.handler.HandleUserChoiceCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "plan_approve"}, b.handler.HandlePlanApproveCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "plan_revise"}, b.handler.HandlePlanReviseCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "turn_resume"}, b.handler.HandleTurnResumeCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "git_diff"}, b.handler.HandleGitDiffCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "git_commit"}, b.handler.HandleGitCommitCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "git_undo"}, b.handler.HandleGitUndoCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "voice_tts"}, b.handler.HandleVoiceTTSCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "voice_loading"}, func(c tele.Context) error {
		session := b.handler.getSession(c)
		toast := "⏳ Audio sedang diproses, mohon tunggu sebentar..."
		if session.GetLanguage() == "en" {
			toast = "⏳ Audio is being processed, please wait..."
		}
		return c.Respond(&tele.CallbackResponse{Text: toast})
	})

	// Plain text messages routed as agent prompts
	b.teleBot.Handle(tele.OnText, b.handler.HandlePrompt)

	// Media uploads (multimodal support)
	b.teleBot.Handle(tele.OnPhoto, b.handler.HandlePhoto)
	b.teleBot.Handle(tele.OnDocument, b.handler.HandleDocument)
	b.teleBot.Handle(tele.OnVoice, b.handler.HandleVoice)
	b.teleBot.Handle(tele.OnAudio, b.handler.HandleAudio)
}
