package testutil

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func InitRepo(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "repomirror@example.com")
	runGit(t, root, "config", "user.name", "RepoMirror Test")
}

func WriteFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func StageAll(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "add", "-A")
}

func CommitAll(t *testing.T, root string, message string) {
	t.Helper()
	StageAll(t, root)
	runGit(t, root, "commit", "-m", message)
}

func InitBareRepo(t *testing.T, root string) {
	t.Helper()
	runGitWithoutRoot(t, "init", "--bare", root)
}

func CurrentBranch(t *testing.T, root string) string {
	t.Helper()
	return strings.TrimSpace(RunGitOutput(t, root, "branch", "--show-current"))
}

func RunGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
	return string(bytes.TrimSpace(output))
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func runGitWithoutRoot(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}
