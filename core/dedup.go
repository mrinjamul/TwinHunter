package core

import (
	"sort"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

// GroupBySize groups files by their size, returning only groups with 2+ files.
func GroupBySize(files []models.FileInfo) map[int64][]models.FileInfo {
	sizeMap := make(map[int64][]models.FileInfo)
	for _, f := range files {
		sizeMap[f.Size] = append(sizeMap[f.Size], f)
	}

	result := make(map[int64][]models.FileInfo)
	for size, group := range sizeMap {
		if len(group) >= 2 {
			result[size] = group
		}
	}
	return result
}

// GroupByHash groups annotated files by their hash, returning only groups with 2+ files.
func GroupByHash(annotated []AnnotatedFile) map[string][]models.FileInfo {
	hashMap := make(map[string][]models.FileInfo)
	for _, af := range annotated {
		if af.Hash == "" {
			continue
		}
		hashMap[af.Hash] = append(hashMap[af.Hash], af.File)
	}

	result := make(map[string][]models.FileInfo)
	for hash, group := range hashMap {
		if len(group) >= 2 {
			result[hash] = group
		}
	}
	return result
}

// FindDuplicates runs the full two-stage dedup pipeline:
// size grouping → Blake3 → SHA256 verification → duplicate groups.
func FindDuplicates(files []models.FileInfo, workers int, onProgress func(HashProgress)) []models.DuplicateGroup {
	sizeGroups := GroupBySize(files)

	var candidates []models.FileInfo
	for _, group := range sizeGroups {
		candidates = append(candidates, group...)
	}

	if len(candidates) == 0 {
		return nil
	}

	blake3Results := AnnotateFiles(candidates, "blake3", workers, onProgress)
	blake3Matches := GroupByHash(blake3Results)

	var verifiedGroups []models.DuplicateGroup

	for _, group := range blake3Matches {
		shaResults := AnnotateFiles(group, "sha256", workers, onProgress)
		shaMap := make(map[string][]models.FileInfo)
		for _, sr := range shaResults {
			if sr.Hash == "" {
				continue
			}
			shaMap[sr.Hash] = append(shaMap[sr.Hash], sr.File)
		}

		for shaHash, shaGroup := range shaMap {
			if len(shaGroup) >= 2 {
				verifiedGroups = append(verifiedGroups, models.DuplicateGroup{
					Hash:  shaHash,
					Size:  shaGroup[0].Size,
					Files: shaGroup,
				})
			}
		}
	}

	return verifiedGroups
}

// SortGroupsBySize sorts groups by file size descending (default).
func SortGroupsBySize(groups []models.DuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Size > groups[j].Size
	})
}

// SortGroupsByCount sorts groups by number of copies descending.
func SortGroupsByCount(groups []models.DuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Files) > len(groups[j].Files)
	})
}

// SortGroupsByPath sorts groups alphabetically by first file path.
func SortGroupsByPath(groups []models.DuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Files) == 0 || len(groups[j].Files) == 0 {
			return false
		}
		return groups[i].Files[0].Path < groups[j].Files[0].Path
	})
}

// ApplyKeepStrategy selects which file to keep and which to remove.
func ApplyKeepStrategy(group models.DuplicateGroup, strategy string) (keep models.FileInfo, toRemove []models.FileInfo) {
	if len(group.Files) == 0 {
		return models.FileInfo{}, nil
	}
	if len(group.Files) == 1 {
		return group.Files[0], nil
	}

	sorted := make([]models.FileInfo, len(group.Files))
	copy(sorted, group.Files)

	switch strategy {
	case "oldest":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].ModTime.Equal(sorted[j].ModTime) {
				return len(sorted[i].Path) < len(sorted[j].Path)
			}
			return sorted[i].ModTime.Before(sorted[j].ModTime)
		})
	case "newest":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].ModTime.Equal(sorted[j].ModTime) {
				return len(sorted[i].Path) < len(sorted[j].Path)
			}
			return sorted[i].ModTime.After(sorted[j].ModTime)
		})
	case "shortest":
		sort.Slice(sorted, func(i, j int) bool {
			if len(sorted[i].Path) == len(sorted[j].Path) {
				return sorted[i].ModTime.Before(sorted[j].ModTime)
			}
			return len(sorted[i].Path) < len(sorted[j].Path)
		})
	default:
	}

	keep = sorted[0]
	toRemove = sorted[1:]
	return
}

// BuildReport creates a Report from the scan results.
func BuildReport(allFiles []models.FileInfo, groups []models.DuplicateGroup, scanPath string) models.Report {
	var totalSize int64
	var dupFiles int
	var wastedSpace int64

	for _, f := range allFiles {
		totalSize += f.Size
	}

	for _, g := range groups {
		dupFiles += len(g.Files) - 1
		wastedSpace += g.Size * int64(len(g.Files)-1)
	}

	return models.Report{
		ScanPath:    scanPath,
		TotalFiles:  len(allFiles),
		TotalSize:   totalSize,
		UniqueFiles: len(allFiles) - dupFiles,
		DupGroups:   groups,
		DupFiles:    dupFiles,
		WastedSpace: wastedSpace,
		ScannedAt:   time.Now(),
	}
}
