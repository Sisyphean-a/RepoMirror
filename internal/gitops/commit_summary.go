package gitops

import (
	"fmt"
	"strings"
)

func (s *Service) DescribeWorkingTree(repoPath string) (string, error) {
	root, err := s.ResolveRepositoryRoot(repoPath)
	if err != nil {
		return "", err
	}
	return s.describeWorkingTreeFromRoot(root)
}

func (s *Service) describeWorkingTreeFromRoot(root string) (string, error) {
	sections, err := s.collectWorkingTreeSections(root)
	if err != nil {
		return "", err
	}
	if len(sections) == 0 {
		return "", fmt.Errorf("target repository has no changes to describe")
	}
	return strings.Join(sections, "\n\n"), nil
}

func (s *Service) collectWorkingTreeSections(root string) ([]string, error) {
	definitions := []struct {
		title string
		args  []string
	}{
		{title: "Git status", args: []string{"status", "--short"}},
		{title: "Unstaged diff stat", args: []string{"diff", "--stat", "--find-renames"}},
		{title: "Staged diff stat", args: []string{"diff", "--cached", "--stat", "--find-renames", "--root"}},
		{title: "Unstaged diff", args: []string{"diff", "--unified=0", "--find-renames"}},
		{title: "Staged diff", args: []string{"diff", "--cached", "--unified=0", "--find-renames", "--root"}},
	}
	sections := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		section, err := s.readWorkingTreeSection(root, definition.title, definition.args...)
		if err != nil {
			return nil, err
		}
		if section != "" {
			sections = append(sections, section)
		}
	}
	return sections, nil
}

func (s *Service) readWorkingTreeSection(root string, title string, args ...string) (string, error) {
	output, err := s.runner.Run(root, nil, args...)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", strings.ToLower(title), err)
	}
	content := strings.TrimSpace(string(output))
	if content == "" {
		return "", nil
	}
	return title + ":\n" + content, nil
}
