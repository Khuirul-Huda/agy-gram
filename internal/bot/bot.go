package bot

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/config"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

// Bot wraps telebot.Bot with routing and middlewares.
type Bot struct {
	teleBot *tele.Bot
	cfg     *config.Config
	handler *Handler
}

// NewBot initializes a new Telegram bot instance.
func NewBot(cfg *config.Config, handler *Handler) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.TelegramBotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		teleBot: b,
		cfg:     cfg,
		handler: handler,
	}

	bot.registerMiddlewares()
	bot.registerRoutes()

	return bot, nil
}

// Start begins listening for updates (blocking).
func (b *Bot) Start() {
	slog.Info("Starting Telegram bot polling", "bot_user", b.teleBot.Me.Username)
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
	b.teleBot.Handle("/help", b.handler.HandleHelp)
	b.teleBot.Handle("/new", b.handler.HandleNew)
	b.teleBot.Handle("/status", b.handler.HandleStatus)
	b.teleBot.Handle("/mode", b.handler.HandleMode)
	b.teleBot.Handle("/model", b.handler.HandleModel)
	b.teleBot.Handle("/effort", b.handler.HandleEffort)
	b.teleBot.Handle("/thinking", b.handler.HandleThinking)
	b.teleBot.Handle("/stop", b.handler.HandleCancel)
	b.teleBot.Handle("/cancel", b.handler.HandleCancel)

	// Inline keyboard callback handlers
	b.teleBot.Handle(&tele.Btn{Unique: "model_set"}, b.handler.HandleSetModelCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "model_reset"}, b.handler.HandleResetModelCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "model_menu"}, b.handler.HandleModelMenuCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "effort_menu"}, b.handler.HandleEffortMenuCallback)
	b.teleBot.Handle(&tele.Btn{Unique: "effort_set"}, b.handler.HandleSetEffortCallback)

	// Plain text messages routed as agent prompts
	b.teleBot.Handle(tele.OnText, b.handler.HandlePrompt)
}
