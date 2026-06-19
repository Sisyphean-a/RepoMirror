package gitops

import (
	"bytes"
	"path/filepath"
	"strings"
)

func parseIgnoredPaths(output []byte) map[string]string {
	ignored := make(map[string]string, estimatedIgnoredPathCount(output))
	ruleLabels := make(map[string]string, 4)
	for len(output) > 0 {
		line, rest, found := bytes.Cut(output, []byte{'\n'})
		output = rest
		if !found {
			output = nil
		}
		relPath, rule := parseIgnoredPathRuleBytes(line, ruleLabels)
		if relPath == "" {
			continue
		}
		ignored[relPath] = rule
	}
	return ignored
}

func parseIgnoredPathSet(output []byte) map[string]struct{} {
	ignored := make(map[string]struct{}, estimatedIgnoredPathCount(output))
	for start := 0; start < len(output); {
		end := bytes.IndexByte(output[start:], '\n')
		if end == -1 {
			end = len(output) - start
		}
		relPath := parseIgnoredPathLine(output[start : start+end])
		if relPath != "" {
			ignored[relPath] = struct{}{}
		}
		if start+end >= len(output) {
			break
		}
		start += end + 1
	}
	return ignored
}

func parseIgnoredPathLine(line []byte) string {
	line = trimTrailingCarriageReturn(line)
	if len(line) == 0 {
		return ""
	}
	return normalizeSlashPathRaw(line)
}

func trimTrailingCarriageReturn(line []byte) []byte {
	if len(line) != 0 && line[len(line)-1] == '\r' {
		return line[:len(line)-1]
	}
	return line
}

func estimatedIgnoredPathCount(output []byte) int {
	if len(output) == 0 {
		return 0
	}
	count := bytes.Count(output, []byte{'\n'})
	if output[len(output)-1] != '\n' {
		count++
	}
	return count
}

func parseIgnoredPathRule(line string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
	if len(parts) != 2 {
		return "", ""
	}
	meta := parts[0]
	path := filepath.ToSlash(strings.TrimSpace(parts[1]))
	patternStart := strings.LastIndex(meta, ":")
	if patternStart == -1 || patternStart == len(meta)-1 {
		return path, "ignore-protected"
	}
	return path, ignoredRuleLabel(meta[patternStart+1:])
}

func parseIgnoredPathRuleBytes(line []byte, ruleLabels map[string]string) (string, string) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return "", ""
	}
	meta, rawPath, found := bytes.Cut(trimmed, []byte{'\t'})
	if !found {
		return "", ""
	}
	path := normalizeSlashPath(trimTrailingCarriageReturn(rawPath))
	patternStart := bytes.LastIndexByte(meta, ':')
	if patternStart == -1 || patternStart == len(meta)-1 {
		return path, "ignore-protected"
	}
	return path, cachedIgnoredRuleLabel(meta[patternStart+1:], ruleLabels)
}

func cachedIgnoredRuleLabel(pattern []byte, ruleLabels map[string]string) string {
	key := strings.TrimSpace(bytesToStringView(pattern))
	if rule, ok := ruleLabels[key]; ok {
		return rule
	}
	rule := ignoredRuleLabel(key)
	ruleLabels[key] = rule
	return rule
}

func ignoredRuleLabel(pattern string) string {
	lower := strings.ToLower(strings.TrimSpace(pattern))
	switch {
	case strings.Contains(lower, ".env"):
		return "env-protected"
	case strings.Contains(lower, ".yaml"), strings.Contains(lower, ".yml"), strings.Contains(lower, "config"):
		return "cfg-protected"
	case strings.Contains(lower, "secret"), strings.Contains(lower, "key"):
		return "secret-protected"
	default:
		return "ignore-protected"
	}
}
