package gitops

import (
	"bytes"
	"strings"
	"unsafe"
)

func collectSyncablePaths(output []byte) ([]string, []string, bool, bool) {
	itemCount := estimatedNullSeparatedCount(output)
	needsNormalize := bytes.IndexByte(output, '\\') >= 0
	candidates := make([]string, 0, itemCount)
	deleted := make([]string, 0, max(1, itemCount/8))
	candidatesSorted := true
	deletedSorted := true
	lastCandidate := ""
	lastDeleted := ""
	for start := 0; start < len(output); {
		end := bytes.IndexByte(output[start:], 0)
		if end == -1 {
			end = len(output) - start
		}
		item := output[start : start+end]
		if len(item) > 0 {
			status, relPath := parseTaggedPathBytes(item, needsNormalize)
			if relPath != "" && !isProtectedPath(relPath) {
				if status == 'R' {
					if lastDeleted != "" && relPath < lastDeleted {
						deletedSorted = false
					}
					lastDeleted = relPath
					deleted = append(deleted, relPath)
				} else {
					if lastCandidate != "" && relPath < lastCandidate {
						candidatesSorted = false
					}
					lastCandidate = relPath
					candidates = append(candidates, relPath)
				}
			}
		}
		if start+end >= len(output) {
			break
		}
		start += end + 1
	}
	return candidates, deleted, candidatesSorted, deletedSorted
}

func estimatedNullSeparatedCount(output []byte) int {
	if len(output) == 0 {
		return 0
	}
	count := bytes.Count(output, []byte{0})
	if output[len(output)-1] != 0 {
		count++
	}
	return count
}

func isProtectedPath(relPath string) bool {
	if len(relPath) < 4 {
		return false
	}
	if !hasProtectedPathCandidate(relPath) {
		return false
	}
	segmentStart := 0
	for segmentStart <= len(relPath) {
		separatorOffset := stringsIndexPathSeparator(relPath, segmentStart)
		index := len(relPath)
		if separatorOffset >= 0 {
			index = segmentStart + separatorOffset
		}
		segmentLen := index - segmentStart
		if segmentLen == 4 && dotGitSegment(relPath, segmentStart) {
			return true
		}
		if index == len(relPath) && segmentLen == 10 && dotGitIgnoreSegment(relPath, segmentStart) {
			return true
		}
		if separatorOffset < 0 {
			break
		}
		segmentStart = index + 1
	}
	return false
}

func hasProtectedPathCandidate(relPath string) bool {
	for offset := 0; offset < len(relPath)-3; {
		index := strings.IndexByte(relPath[offset:], '.')
		if index < 0 {
			return false
		}
		index += offset
		offset = index + 1
		if index != 0 && !isPathSeparator(relPath[index-1]) {
			continue
		}
		if !dotGitSegment(relPath, index) {
			continue
		}
		return true
	}
	return false
}

func stringsIndexPathSeparator(path string, start int) int {
	for index := start; index < len(path); index++ {
		if isPathSeparator(path[index]) {
			return index - start
		}
	}
	return -1
}

func dotGitSegment(path string, start int) bool {
	return path[start] == '.' &&
		lowerASCIILetter(path[start+1]) == 'g' &&
		lowerASCIILetter(path[start+2]) == 'i' &&
		lowerASCIILetter(path[start+3]) == 't'
}

func dotGitIgnoreSegment(path string, start int) bool {
	return path[start] == '.' &&
		lowerASCIILetter(path[start+1]) == 'g' &&
		lowerASCIILetter(path[start+2]) == 'i' &&
		lowerASCIILetter(path[start+3]) == 't' &&
		lowerASCIILetter(path[start+4]) == 'i' &&
		lowerASCIILetter(path[start+5]) == 'g' &&
		lowerASCIILetter(path[start+6]) == 'n' &&
		lowerASCIILetter(path[start+7]) == 'o' &&
		lowerASCIILetter(path[start+8]) == 'r' &&
		lowerASCIILetter(path[start+9]) == 'e'
}

func lowerASCIILetter(char byte) byte {
	if char >= 'A' && char <= 'Z' {
		return char + ('a' - 'A')
	}
	return char
}

func parseTaggedPathBytes(item []byte, needsNormalize bool) (byte, string) {
	if len(item) < 3 || item[1] != ' ' {
		return 0, gitPathString(item, needsNormalize)
	}
	return item[0], gitPathString(item[2:], needsNormalize)
}

func gitPathString(raw []byte, needsNormalize bool) string {
	if needsNormalize {
		return normalizeSlashPathRaw(raw)
	}
	return bytesToStringView(raw)
}

func normalizeSlashPathRaw(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if bytes.IndexByte(raw, '\\') < 0 {
		return bytesToStringView(raw)
	}
	normalized := make([]byte, len(raw))
	for index, char := range raw {
		if char == '\\' {
			normalized[index] = '/'
			continue
		}
		normalized[index] = char
	}
	return string(normalized)
}

func normalizeSlashPath(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	return normalizeSlashPathRaw(trimmed)
}

func bytesToStringView(raw []byte) string {
	// Safe here because returned strings are immutable views over the runner output,
	// which remains heap-backed and alive as long as the strings are retained.
	return unsafe.String(unsafe.SliceData(raw), len(raw))
}

func isPathSeparator(char byte) bool {
	return char == '/' || char == '\\'
}
