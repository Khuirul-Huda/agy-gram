package bot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetGitDiff_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "not_git_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	diff, hasChanges, err := getGitDiff(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasChanges || diff != "" {
		t.Errorf("expected no changes for non-git directory")
	}
}

func TestGetGitDiff_CleanAndDirty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = exec.Command("git", "-C", tmpDir, "init").Run()
	_ = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", tmpDir, "config", "user.name", "Tester").Run()

	filePath := filepath.Join(tmpDir, "hello.txt")
	_ = os.WriteFile(filePath, []byte("line 1\n"), 0644)
	_ = exec.Command("git", "-C", tmpDir, "add", ".").Run()
	_ = exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run()

	// Clean worktree
	_, hasChanges, _ := getGitDiff(tmpDir)
	if hasChanges {
		t.Errorf("expected no changes after commit")
	}

	// Dirty worktree
	_ = os.WriteFile(filePath, []byte("line 1\nline 2\n"), 0644)
	diff, hasChanges, _ := getGitDiff(tmpDir)
	if !hasChanges {
		t.Errorf("expected hasChanges = true after file edit")
	}
	if len(diff) == 0 {
		t.Errorf("expected non-empty diff")
	}
}
