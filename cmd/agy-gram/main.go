package main

import (
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Khuirul-Huda/agy-gram/internal/agent"
	"github.com/Khuirul-Huda/agy-gram/internal/bot"
	"github.com/Khuirul-Huda/agy-gram/internal/config"
	"github.com/Khuirul-Huda/agy-gram/internal/storage"
	"github.com/Khuirul-Huda/agy-gram/internal/system"
)

func main() {
	setupLogger()

	slog.Info("Initializing AGY-Gram bridge service...")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Open lightning-fast embedded bbolt DB for persistent sessions and configs
	dbPath := filepath.Join("data", "agy_gram.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		slog.Error("Failed to open bbolt database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize core components following Clean Architecture
	runner := agent.NewRunner(cfg.AgyBinPath, cfg.AgyWorkingDir, cfg.AgyTimeout)
	session := agent.NewSession(cfg.AgyDefaultMode, cfg.AgyDefaultModel, cfg.VerboseThinking, store)
	monitor := system.NewMonitor(runner)
	handler := bot.NewHandler(runner, session, monitor)

	telegramBot, err := bot.NewBot(cfg, handler, session)
	if err != nil {
		slog.Error("Failed to initialize Telegram bot", "error", err)
		os.Exit(1)
	}

	// Handle graceful shutdown signals (SIGINT, SIGTERM)
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdownChan
		slog.Info("Shutdown signal received", "signal", sig.String())

		// Terminate any active CLI child process
		if runner.IsRunning() {
			slog.Info("Terminating active agent process...")
			runner.CancelCurrent()
		}

		telegramBot.Stop()
	}()

	slog.Info("AGY-Gram is ready and listening for Telegram updates",
		"working_dir", cfg.AgyWorkingDir,
		"allowed_users_count", len(cfg.AllowedUserIDs),
		"timeout", cfg.AgyTimeout.String(),
	)

	// Start bot polling loop (blocks until stopped)
	telegramBot.Start()

	slog.Info("AGY-Gram service exited cleanly.")
}

func setupLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
