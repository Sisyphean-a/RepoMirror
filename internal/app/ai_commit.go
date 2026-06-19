package app

import (
	"fmt"
	"os"
	"strings"

	"RepoMirror/internal/model"
)

type CommitMessageGenerator interface {
	Generate(apiKey string, changes string) (string, error)
}

func (s *Service) SetCommitGenerator(generator CommitMessageGenerator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commitGenerator = generator
}

func (s *Service) GenerateCommitMessage() (string, error) {
	cfg := s.currentConfig()
	apiKey := resolveAICommitAPIKey(cfg)
	if apiKey == "" {
		return "", fmt.Errorf("DeepSeek API Key is not configured")
	}
	generator := s.currentCommitGenerator()
	if generator == nil {
		return "", fmt.Errorf("AI commit generator is not configured")
	}
	targetRoot, err := s.resolveTargetRoot(cfg)
	if err != nil {
		return "", err
	}
	changes, err := s.inspector.DescribeWorkingTree(targetRoot)
	if err != nil {
		return "", err
	}
	return generator.Generate(apiKey, changes)
}

func (s *Service) SetAICommitAPIKey(apiKey string) (model.DashboardState, error) {
	cfg, err := s.updateConfig(func(next *model.AppConfig) {
		next.AICommitAPIKey = strings.TrimSpace(apiKey)
	})
	if err != nil {
		return model.DashboardState{}, err
	}
	return s.buildState(cfg)
}

func (s *Service) currentCommitGenerator() CommitMessageGenerator {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commitGenerator
}

func hasAICommitAPIKey(cfg model.AppConfig) bool {
	return resolveAICommitAPIKey(cfg) != ""
}

func resolveAICommitAPIKey(cfg model.AppConfig) string {
	if apiKey := strings.TrimSpace(cfg.AICommitAPIKey); apiKey != "" {
		return apiKey
	}
	return strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
}
