package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configurations.
type Config struct {
	TelegramBotToken string
	AllowedUserIDs   map[int64]struct{}
	AgyBinPath       string
	AgyWorkingDir    string
	AgyTimeout       time.Duration
	AgyDefaultMode   string
	AgyDefaultModel  string
}

// Load loads configuration from environment variables and optional .env file.
func Load() (*Config, error) {
	// Load .env if present; ignore error if file does not exist
	_ = godotenv.Load()

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	allowedIDsStr := os.Getenv("ALLOWED_USER_IDS")
	allowedUserIDs, err := parseAllowedUserIDs(allowedIDsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ALLOWED_USER_IDS: %w", err)
	}

	agyBinPath := getEnvOrDefault("AGY_BIN_PATH", "agy")
	agyWorkingDir := getEnvOrDefault("AGY_WORKING_DIR", os.Getenv("HOME"))
	if agyWorkingDir == "" {
		agyWorkingDir = "."
	}

	timeoutMinutes, err := strconv.Atoi(getEnvOrDefault("AGY_TIMEOUT_MINUTES", "5"))
	if err != nil || timeoutMinutes <= 0 {
		timeoutMinutes = 5
	}

	defaultMode := getEnvOrDefault("AGY_DEFAULT_MODE", "auto")
	defaultModel := strings.TrimSpace(os.Getenv("AGY_DEFAULT_MODEL"))

	return &Config{
		TelegramBotToken: token,
		AllowedUserIDs:   allowedUserIDs,
		AgyBinPath:       agyBinPath,
		AgyWorkingDir:    agyWorkingDir,
		AgyTimeout:       time.Duration(timeoutMinutes) * time.Minute,
		AgyDefaultMode:   defaultMode,
		AgyDefaultModel:  defaultModel,
	}, nil
}

// IsUserAllowed checks if a Telegram user ID is whitelisted.
func (c *Config) IsUserAllowed(userID int64) bool {
	if len(c.AllowedUserIDs) == 0 {
		return false
	}
	_, ok := c.AllowedUserIDs[userID]
	return ok
}

func parseAllowedUserIDs(raw string) (map[int64]struct{}, error) {
	result := make(map[int64]struct{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result, nil
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		id, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user id '%s': %w", trimmed, err)
		}
		result[id] = struct{}{}
	}

	return result, nil
}

func getEnvOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
