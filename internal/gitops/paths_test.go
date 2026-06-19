package gitops

import (
	"reflect"
	"sort"
	"testing"
)

func TestCollectSyncablePaths(t *testing.T) {
	paths, deleted, candidatesSorted, deletedSorted := collectSyncablePaths([]byte("? new.txt\x00H keep.txt\x00R gone.txt\x00H keep.txt\x00"))
	expected := []string{"keep.txt", "keep.txt", "new.txt"}
	if !reflect.DeepEqual(sortedStrings(paths), expected) {
		t.Fatalf("unexpected syncable paths: got %v want %v", paths, expected)
	}
	if !reflect.DeepEqual(deleted, []string{"gone.txt"}) {
		t.Fatalf("expected deleted path to be tracked, got %v", deleted)
	}
	if candidatesSorted {
		t.Fatalf("expected candidate paths to be marked unsorted")
	}
	if !deletedSorted {
		t.Fatalf("expected deleted paths to stay sorted")
	}
}

func TestIsProtectedPath(t *testing.T) {
	testCases := []struct {
		path string
		want bool
	}{
		{path: ".gitignore", want: true},
		{path: "nested/.gitignore", want: true},
		{path: ".git/config", want: true},
		{path: "nested\\.git\\config", want: true},
		{path: "docs/readme.md", want: false},
	}

	for _, testCase := range testCases {
		if got := isProtectedPath(testCase.path); got != testCase.want {
			t.Fatalf("isProtectedPath(%q) = %v, want %v", testCase.path, got, testCase.want)
		}
	}
}

func TestCompactSortedPaths(t *testing.T) {
	paths := compactSortedPaths(
		[]string{"a.txt", "a.txt", "b.txt", "c.txt", "c.txt"},
		[]string{"b.txt"},
	)
	expected := []string{"a.txt", "c.txt"}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected compacted paths: got %v want %v", paths, expected)
	}
}

func TestParseIgnoredPaths(t *testing.T) {
	ignored := parseIgnoredPaths([]byte(".gitignore:1:ignored/\tignored/a.txt\n.env:2:*.env\tprod.env\n"))
	expected := map[string]string{
		"ignored/a.txt": "ignore-protected",
		"prod.env":      "env-protected",
	}
	if !reflect.DeepEqual(ignored, expected) {
		t.Fatalf("unexpected ignored paths: got %v want %v", ignored, expected)
	}
}

func TestParseIgnoredPathSet(t *testing.T) {
	ignored := parseIgnoredPathSet([]byte("ignored/a.txt\nprod.env\n"))
	expected := map[string]struct{}{
		"ignored/a.txt": {},
		"prod.env":      {},
	}
	if !reflect.DeepEqual(ignored, expected) {
		t.Fatalf("unexpected ignored path set: got %v want %v", ignored, expected)
	}
}

func TestParseIgnoredPathSetPreservesSpacesAndCRLF(t *testing.T) {
	ignored := parseIgnoredPathSet([]byte(" spaced name.txt \r\nnested\\path.txt\r\n"))
	expected := map[string]struct{}{
		" spaced name.txt ": {},
		"nested/path.txt":   {},
	}
	if !reflect.DeepEqual(ignored, expected) {
		t.Fatalf("unexpected ignored path set: got %v want %v", ignored, expected)
	}
}

func TestIgnoredRuleLabelMatchesMixedCasePatterns(t *testing.T) {
	testCases := []struct {
		pattern string
		want    string
	}{
		{pattern: "  *.ENV  ", want: "env-protected"},
		{pattern: "Config/*.YAML", want: "cfg-protected"},
		{pattern: "Secrets/KEY.txt", want: "secret-protected"},
		{pattern: "ignored/", want: "ignore-protected"},
	}

	for _, testCase := range testCases {
		if got := ignoredRuleLabel(testCase.pattern); got != testCase.want {
			t.Fatalf("ignoredRuleLabel(%q) = %q, want %q", testCase.pattern, got, testCase.want)
		}
	}
}

func sortedStrings(values []string) []string {
	cloned := append([]string(nil), values...)
	sort.Strings(cloned)
	return cloned
}
