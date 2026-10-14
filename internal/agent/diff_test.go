package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateGitDiff_NoIndex(t *testing.T) {
	before := "func hello() string {\n\treturn \"hi\"\n}\n"
	after := "func hello() string {\n\treturn \"hello world\"\n}\n"

	diff := generateGitDiff("main.go", before, after)
	if !strings.Contains(diff, "--- a/main.go") && !strings.Contains(diff, "-	return \"hi\"") {
		t.Errorf("expected git diff output, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+	return \"hello world\"") {
		t.Errorf("expected git diff addition, got:\n%s", diff)
	}
}

func TestGenerateGitDiff_InGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	_ = os.WriteFile(filePath, []byte("line1\nline2\n"), 0644)
	diff := generateGitDiff(filePath, "line1\n", "line1 modified\n")
	if diff == "" {
		t.Errorf("expected diff, got empty")
	}
}
