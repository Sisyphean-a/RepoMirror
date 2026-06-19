package diff

import "strings"

func isProtected(relPath string) bool {
	if lastSegmentEqualFoldASCII(relPath, ".gitignore") {
		return true
	}
	segmentStart := 0
	for index := 0; index <= len(relPath); index++ {
		if index < len(relPath) && !isPathSeparator(relPath[index]) {
			continue
		}
		if equalFoldASCII(relPath[segmentStart:index], ".git") {
			return true
		}
		segmentStart = index + 1
	}
	return false
}

func fullPath(root string, relPath string) string {
	if root == "" {
		return nativeRelativePath(relPath)
	}
	buffer := make([]byte, 0, len(root)+len(relPath)+1)
	buffer = append(buffer, root...)
	if !isPathSeparator(root[len(root)-1]) {
		buffer = append(buffer, '\\')
	}
	buffer = appendNativeRelativePath(buffer, relPath)
	return string(buffer)
}

func nativeRelativePath(relPath string) string {
	if strings.IndexByte(relPath, '/') < 0 {
		return relPath
	}
	buffer := make([]byte, 0, len(relPath))
	buffer = appendNativeRelativePath(buffer, relPath)
	return string(buffer)
}

func appendNativeRelativePath(buffer []byte, relPath string) []byte {
	for index := 0; index < len(relPath); index++ {
		if relPath[index] == '/' {
			buffer = append(buffer, '\\')
			continue
		}
		buffer = append(buffer, relPath[index])
	}
	return buffer
}

func equalFoldASCII(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := 0; index < len(left); index++ {
		if lowerASCIILetter(left[index]) != lowerASCIILetter(right[index]) {
			return false
		}
	}
	return true
}

func lastSegmentEqualFoldASCII(relPath string, target string) bool {
	end := len(relPath)
	for end > 0 && isPathSeparator(relPath[end-1]) {
		end--
	}
	start := end
	for start > 0 && !isPathSeparator(relPath[start-1]) {
		start--
	}
	return equalFoldASCII(relPath[start:end], target)
}

func lowerASCIILetter(char byte) byte {
	if char >= 'A' && char <= 'Z' {
		return char + ('a' - 'A')
	}
	return char
}

func isPathSeparator(char byte) bool {
	return char == '/' || char == '\\'
}
