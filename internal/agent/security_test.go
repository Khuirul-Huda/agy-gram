package agent

import "testing"

func TestIsCommandRisky(t *testing.T) {
	riskyCases := []string{
		"rm -rf /tmp/test",
		"rm -r node_modules",
		"sudo mkfs.ext4 /dev/sdb",
		"shutdown now",
		"reboot",
		"ufw disable",
		"cat /home/huda/victims/data.txt",
		"rm -f /home/huda/endstone/worlds_saved/level.dat",
	}

	for _, cmd := range riskyCases {
		isRisky, pattern := IsCommandRisky(cmd)
		if !isRisky {
			t.Errorf("expected command '%s' to be risky, but was not detected", cmd)
		}
		if pattern == "" {
			t.Errorf("expected pattern for '%s', got empty", cmd)
		}
	}

	safeCases := []string{
		"git status",
		"ls -la",
		"go test ./...",
		"cat README.md",
		"pm2 status",
		"uptime",
	}

	for _, cmd := range safeCases {
		isRisky, _ := IsCommandRisky(cmd)
		if isRisky {
			t.Errorf("expected command '%s' to be safe, but flagged as risky", cmd)
		}
	}
}
