package vfs

import (
	"io/fs"
	"os"
	"path"
	"time"
)

type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// NewFileInfo create file info which implements os.FileInfo
func NewFileInfo(name string, isDirectory bool, size int64, modTime time.Time) *FileInfo {
	mode := os.FileMode(0644)
	if isDirectory {
		mode = os.FileMode(0755) | os.ModeDir
	}
	return &FileInfo{name: path.Base(name), size: size, mode: mode, modTime: modTime}
}

var _ os.FileInfo = (*FileInfo)(nil)

func (f *FileInfo) Name() string {
	return f.name
}

func (f *FileInfo) Size() int64 {
	return f.size
}

func (f *FileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f *FileInfo) ModTime() time.Time {
	return f.modTime
}

func (f *FileInfo) IsDir() bool {
	return f.mode&os.ModeDir != 0
}

func (f *FileInfo) Sys() any {
	return nil
}
