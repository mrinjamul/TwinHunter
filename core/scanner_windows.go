//go:build windows

package core

import (
	"io/fs"
)

func detectHardLink(info fs.FileInfo) bool {
	return false
}
