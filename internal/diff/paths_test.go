package diff

import "testing"

func TestIsProtected(t *testing.T) {
	testCases := []struct {
		path string
		want bool
	}{
		{path: ".gitignore", want: true},
		{path: "nested/.gitignore", want: true},
		{path: ".git/config", want: true},
		{path: "nested/.git/config", want: true},
		{path: "nested\\.git\\config", want: true},
		{path: "docs/git-guide.md", want: false},
	}

	for _, testCase := range testCases {
		if got := isProtected(testCase.path); got != testCase.want {
			t.Fatalf("isProtected(%q) = %v, want %v", testCase.path, got, testCase.want)
		}
	}
}

func TestFullPath(t *testing.T) {
	testCases := []struct {
		root    string
		relPath string
		want    string
	}{
		{root: "C:\\repo", relPath: "dir/file.txt", want: "C:\\repo\\dir\\file.txt"},
		{root: "C:\\repo\\", relPath: "dir/file.txt", want: "C:\\repo\\dir\\file.txt"},
		{root: "", relPath: "dir/file.txt", want: "dir\\file.txt"},
		{root: "", relPath: "dir\\file.txt", want: "dir\\file.txt"},
	}

	for _, testCase := range testCases {
		if got := fullPath(testCase.root, testCase.relPath); got != testCase.want {
			t.Fatalf("fullPath(%q, %q) = %q, want %q", testCase.root, testCase.relPath, got, testCase.want)
		}
	}
}

func BenchmarkFullPath(b *testing.B) {
	root := "C:\\repo"
	relPath := "dir/file-1234.txt"

	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		path := fullPath(root, relPath)
		if path == "" {
			b.Fatal("expected full path")
		}
	}
}

func TestMergedPathCount(t *testing.T) {
	testCases := []struct {
		source []string
		target []string
		want   int
	}{
		{source: nil, target: nil, want: 0},
		{source: []string{"a.txt"}, target: nil, want: 1},
		{source: nil, target: []string{"a.txt"}, want: 1},
		{source: []string{"a.txt", "b.txt"}, target: []string{"b.txt", "c.txt"}, want: 3},
		{source: []string{"a.txt", "c.txt"}, target: []string{"b.txt", "d.txt"}, want: 4},
	}

	for _, testCase := range testCases {
		if got := mergedPathCount(testCase.source, testCase.target); got != testCase.want {
			t.Fatalf("mergedPathCount(%v, %v) = %d, want %d", testCase.source, testCase.target, got, testCase.want)
		}
	}
}
