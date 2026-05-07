package models

import "time"

// FileInfo represents metadata about a scanned file.
type FileInfo struct {
	Path       string
	Name       string
	Size       int64
	ModTime    time.Time
	Blake3     string
	SHA256     string
	IsHardLink bool
}

// DuplicateGroup is a set of files that share the same content hash.
type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []FileInfo
}

// Report holds the full result of a duplicate scan.
type Report struct {
	ScanPath    string            `json:"scan_path"`
	TotalFiles  int               `json:"total_files"`
	TotalSize   int64             `json:"total_size"`
	UniqueFiles int               `json:"unique_files"`
	DupGroups   []DuplicateGroup  `json:"duplicate_groups"`
	DupFiles    int               `json:"duplicate_files"`
	WastedSpace int64             `json:"wasted_space"`
	ScannedAt   time.Time         `json:"scanned_at"`
}

// ScanStats tracks live progress during a scan.
type ScanStats struct {
	FilesScanned int
	TotalFiles   int
	CurrentPath  string
}
