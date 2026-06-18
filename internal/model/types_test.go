package model

import "testing"

func TestRepositoryNameEmptyPath(t *testing.T) {
	if name := RepositoryName(""); name != "" {
		t.Fatalf("expected empty repository name, got %q", name)
	}
}
