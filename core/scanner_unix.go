//go:build !windows

package core

import (
	"io/fs"
	"syscall"
)

func detectHardLink(info fs.FileInfo) bool {
	switch st := info.Sys().(type) {
	case *syscall.Stat_t:
		return st.Nlink > 1
	}
	return false
}
