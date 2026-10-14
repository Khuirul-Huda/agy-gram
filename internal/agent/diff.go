package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// generateGitDiff uses git diff to compute clean unified diffs for file edits.
func generateGitDiff(targetFile, targetContent, replacementContent string) string {
	if strings.TrimSpace(targetFile) == "" {
		return formatDiff(targetContent, replacementContent)
	}

	dir := filepath.Dir(targetFile)
	if dir == "" {
		dir = "."
	}

	// 1. Check if targetFile resides inside a git repository
	cmdRoot := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	outRoot, err := cmdRoot.Output()
	if err == nil {
		repoRoot := strings.TrimSpace(string(outRoot))
		relPath, relErr := filepath.Rel(repoRoot, targetFile)
		if relErr == nil && repoRoot != "" {
			// Try running git diff against current worktree
			cmdDiff := exec.Command("git", "-C", repoRoot, "diff", "--no-color", "--", relPath)
			var stdoutBuf bytes.Buffer
			cmdDiff.Stdout = &stdoutBuf
			if cmdDiff.Run() == nil {
				gitDiffOut := strings.TrimSpace(stdoutBuf.String())
				if gitDiffOut != "" {
					return truncateDiffOutput(gitDiffOut, 35)
				}
			}
		}
	}

	// 2. If not a git repo or file not committed yet, use git diff --no-index between before and after
	if targetContent != "" || replacementContent != "" {
		noIndexDiff := runGitNoIndexDiff(targetFile, targetContent, replacementContent)
		if strings.TrimSpace(noIndexDiff) != "" {
			return truncateDiffOutput(noIndexDiff, 35)
		}
	}

	// 3. Fallback to built-in clean line diff
	return formatDiff(targetContent, replacementContent)
}

func runGitNoIndexDiff(targetFile, beforeContent, afterContent string) string {
	tmpDir, err := os.MkdirTemp("", "agy_diff_*")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(tmpDir)

	baseName := filepath.Base(targetFile)
	beforePath := filepath.Join(tmpDir, "a_"+baseName)
	afterPath := filepath.Join(tmpDir, "b_"+baseName)

	if err := os.WriteFile(beforePath, []byte(beforeContent), 0600); err != nil {
		return ""
	}
	if err := os.WriteFile(afterPath, []byte(afterContent), 0600); err != nil {
		return ""
	}

	// git diff --no-index returns exit status 1 when differences are found!
	cmd := exec.Command("git", "diff", "--no-index", "--no-color", beforePath, afterPath)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	_ = cmd.Run()

	out := outBuf.String()
	if strings.TrimSpace(out) == "" {
		return ""
	}

	// Replace temporary paths with friendly file labels
	out = strings.ReplaceAll(out, beforePath, "a/"+baseName)
	out = strings.ReplaceAll(out, afterPath, "b/"+baseName)

	return strings.TrimSpace(out)
}

func truncateDiffOutput(diffStr string, maxLines int) string {
	lines := strings.Split(diffStr, "\n")
	if len(lines) <= maxLines {
		return diffStr
	}

	head := lines[:20]
	tail := lines[len(lines)-10:]

	var sb strings.Builder
	for _, l := range head {
		sb.WriteString(l + "\n")
	}
	sb.WriteString(fmt.Sprintf("... (%d baris diff dipotong) ...\n", len(lines)-30))
	for _, l := range tail {
		sb.WriteString(l + "\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}
