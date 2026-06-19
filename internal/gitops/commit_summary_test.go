package gitops

import (
	"strings"
	"testing"
)

func TestDescribeWorkingTreeIncludesStatusAndDiffSections(t *testing.T) {
	runner := &runnerStub{
		responses: map[string][]byte{
			"rev-parse --show-toplevel":                       []byte("C:/repo\n"),
			"status --short":                                  []byte(" M tracked.txt\n?? added.txt\n"),
			"diff --stat --find-renames":                      []byte(" tracked.txt | 2 +-\n"),
			"diff --cached --stat --find-renames --root":      []byte(""),
			"diff --unified=0 --find-renames":                 []byte("@@ -1 +1 @@\n-old\n+new\n"),
			"diff --cached --unified=0 --find-renames --root": []byte(""),
		},
	}

	summary, err := NewService(runner).DescribeWorkingTree("repo")
	if err != nil {
		t.Fatalf("DescribeWorkingTree failed: %v", err)
	}
	if !strings.Contains(summary, "Git status:\nM tracked.txt") && !strings.Contains(summary, "Git status:\n M tracked.txt") {
		t.Fatalf("summary should include git status, got %q", summary)
	}
	if !strings.Contains(summary, "Unstaged diff stat:\ntracked.txt | 2 +-") {
		t.Fatalf("summary should include diff stat, got %q", summary)
	}
	if !strings.Contains(summary, "Unstaged diff:\n@@ -1 +1 @@") {
		t.Fatalf("summary should include diff patch, got %q", summary)
	}
}
