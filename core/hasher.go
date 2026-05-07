package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/mrinjamul/twinhunter/models"
	"github.com/zeebo/blake3"
)

// HashBlake3 computes the Blake3 hash of a file.
func HashBlake3(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashSHA256 computes the SHA256 hash of a file.
func HashSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashResult is a single file hash result.
type HashResult struct {
	Path string
	Hash string
	Err  error
}

// HashPipeline hashes a list of files concurrently using the given algorithm.
// algorithm: "blake3" or "sha256"
func HashPipeline(paths []string, algorithm string, workers int) []HashResult {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(paths) {
		workers = len(paths)
	}

	results := make([]HashResult, len(paths))
	sem := make(chan struct{}, workers)
	done := make(chan struct{})

	type work struct {
		idx  int
		path string
	}
	ch := make(chan work, len(paths))

	for i, p := range paths {
		ch <- work{i, p}
	}
	close(ch)

	for i := 0; i < workers; i++ {
		go func() {
			for w := range ch {
				sem <- struct{}{}
				var hash string
				var err error
				switch algorithm {
				case "sha256":
					hash, err = HashSHA256(w.path)
				default:
					hash, err = HashBlake3(w.path)
				}
				results[w.idx] = HashResult{Path: w.path, Hash: hash, Err: err}
				<-sem
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}

	return results
}

// AnnotatedFile pairs a models.FileInfo with a hash result.
type AnnotatedFile struct {
	File models.FileInfo
	Hash string
}

// AnnotateFiles adds hashes to file infos concurrently.
func AnnotateFiles(files []models.FileInfo, algorithm string, workers int) []AnnotatedFile {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}

	hashResults := HashPipeline(paths, algorithm, workers)

	annotated := make([]AnnotatedFile, len(files))
	for i, hr := range hashResults {
		annotated[i] = AnnotatedFile{
			File: files[i],
			Hash: hr.Hash,
		}
	}

	return annotated
}

// FormatSize returns a human-readable file size string.
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
