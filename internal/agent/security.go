package agent

import (
	"regexp"
	"strings"
)

// DangerousPatterns lists regex patterns for high-risk system commands.
var DangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+-[rfR]{1,3}\b`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=`),
	regexp.MustCompile(`(?i)\b(:(){ :\|:& };:)\b`), // forkbomb
	regexp.MustCompile(`(?i)\bshutdown\b`),
	regexp.MustCompile(`(?i)\breboot\b`),
	regexp.MustCompile(`(?i)\bpoweroff\b`),
	regexp.MustCompile(`(?i)\binit\s+0\b`),
	regexp.MustCompile(`(?i)\bufw\s+(disable|reset)\b`),
	regexp.MustCompile(`(?i)\biptables\s+-F\b`),
	regexp.MustCompile(`(?i)\b(tune2fs|fdisk|parted)\b`),
	regexp.MustCompile(`(?i)\bchmod\s+(-R\s+)?777\s+/`),
	regexp.MustCompile(`(?i)\bchown\s+-R\b`),
	regexp.MustCompile(`(?i)/home/huda/victims`),
	regexp.MustCompile(`(?i)/home/huda/endstone/worlds_saved`),
}

// IsCommandRisky checks if a bash command matches known high-risk destructive operations.
func IsCommandRisky(cmd string) (bool, string) {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return false, ""
	}

	for _, p := range DangerousPatterns {
		if p.MatchString(trimmed) {
			return true, p.String()
		}
	}

	return false, ""
}
