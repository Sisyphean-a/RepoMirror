package syncer

import "RepoMirror/internal/platform"

type cleanupPath struct {
	directory string
	relPath   string
}

type cleanupGroup struct {
	inline  [cleanupInlineEntryLimit]cleanupPath
	entries []cleanupPath
	seen    map[string]struct{}
}

const cleanupInlineEntryLimit = 4
const cleanupLinearScanLimit = 16

func removeUniqueCleanupPaths(fs platform.FileSystem, root string, cleanupGroups []cleanupGroup) error {
	total := cleanupGroupEntryCount(cleanupGroups)
	if total <= cleanupLinearScanLimit*4 {
		return removeUniqueCleanupPathsLinear(fs, root, cleanupGroups, total)
	}
	seen := make(map[string]struct{}, total)
	for _, group := range cleanupGroups {
		for _, candidate := range group.entries {
			if _, exists := seen[candidate.directory]; exists {
				continue
			}
			seen[candidate.directory] = struct{}{}
			if err := fs.RemoveEmptyParentsFromRoot(root, candidate.relPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeUniqueCleanupPathsLinear(
	fs platform.FileSystem,
	root string,
	cleanupGroups []cleanupGroup,
	total int,
) error {
	var inline [cleanupLinearScanLimit * 4]string
	seen := inline[:0]
	if total > len(inline) {
		seen = make([]string, 0, total)
	}
	for _, group := range cleanupGroups {
		for _, candidate := range group.entries {
			if containsCleanupDirectory(seen, candidate.directory) {
				continue
			}
			seen = append(seen, candidate.directory)
			if err := fs.RemoveEmptyParentsFromRoot(root, candidate.relPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanupGroupEntryCount(cleanupGroups []cleanupGroup) int {
	total := 0
	for _, group := range cleanupGroups {
		total += len(group.entries)
	}
	return total
}

func hasCleanupDirectory(paths []cleanupPath, directory string) bool {
	for index := range paths {
		if paths[index].directory == directory {
			return true
		}
	}
	return false
}

func containsCleanupDirectory(paths []string, directory string) bool {
	for index := range paths {
		if paths[index] == directory {
			return true
		}
	}
	return false
}

func cleanupSeenSet(paths []cleanupPath) map[string]struct{} {
	seen := make(map[string]struct{}, len(paths))
	for index := range paths {
		seen[paths[index].directory] = struct{}{}
	}
	return seen
}
