package vfs

import (
	"errors"
	"github.com/goxiaoy/vfs/pkg/trie"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	Name = "vfs"
)

var (
	ErrRecursive    = errors.New("recursive mount may cause dead lock")
	ErrNotSupported = errors.New("not supported")
)

type Vfs struct {
	mtab mountTable
}

func New() *Vfs {
	return &Vfs{
		mtab: mountTable{
			mounts: trie.NewPathTrieWithConfig[*MountPoint](&trie.PathTrieConfig{Segmenter: trie.PathSegmenter2}),
		},
	}
}

// Mount mounts a filesystem with a provided prefix. Prefix can be any
// slash-separated path and does not have to represent an existing directory
// (in this respect it is similar to URL path). Mounted filesystem becomes
// available to os package.
func (v *Vfs) Mount(prefix string, fsys FS) error {
	if prefix == "" || prefix[0] != '/' || fsys == nil {
		return &fs.PathError{Op: "mount", Path: prefix, Err: syscall.EINVAL}
	}
	if v == fsys {
		return &fs.PathError{Op: "mount", Path: prefix, Err: ErrRecursive}
	}
	prefix = path.Clean(prefix)
	v.mtab.mu.Lock()
	v.mtab.mounts.Put(prefix, &MountPoint{prefix, fsys, 0})
	v.mtab.mu.Unlock()
	return nil
}

// Unmount unmounts the last mounted filesystem that match fsys and prefix.
// At least one parameter must be specified (not empty or nil).
func (v *Vfs) Unmount(prefix string, fsys FS) error {
	if prefix == "" && fsys == nil {
		return &fs.PathError{Op: "unmount", Path: prefix, Err: syscall.ENOENT}
	}
	prefix = path.Clean(prefix)
	v.mtab.mu.Lock()
	remove := ""
	v.mtab.mounts.Walk(func(key string, value *MountPoint) error {
		mp := value
		if (prefix == "." || mp.prefix == prefix) && (fsys == nil || mp.fS == fsys) {
			//found
			remove = key
			fsys = mp.fS
			return errors.New("")
		}
		return nil
	})

	var err error
	var mp *MountPoint
	if len(remove) == 0 {
		err = syscall.ENOENT
		goto skip
	}
	mp, _ = v.mtab.mounts.Get(remove)
	if atomic.LoadInt32(&mp.openCount) != 0 {
		err = syscall.EBUSY
		goto skip
	}
	v.mtab.mounts.Delete(remove)
skip:
	v.mtab.mu.Unlock()
	if err != nil {
		return &fs.PathError{Op: "unmount", Path: prefix, Err: err}
	}

	// close the fsys if it has no another mount point

	v.mtab.mu.RLock()

	v.mtab.mounts.Walk(func(key string, value *MountPoint) error {
		if value.fS == fsys {
			fsys = nil // fsys is still mounted with another prefix
			return errors.New("")
		}
		return nil
	})

	v.mtab.mu.RUnlock()
	if fsys == nil {
		return nil
	}
	if fsys, ok := fsys.(interface{ Sync() error }); ok {
		if err = fsys.Sync(); err != nil {
			return &fs.PathError{Op: "unmount", Path: prefix, Err: err}
		}
	}
	return nil
}

func (v *Vfs) Mounts() []*MountPoint {
	v.mtab.mu.RLock()
	var list []*MountPoint
	v.mtab.mounts.Walk(func(key string, value *MountPoint) error {
		list = append(list, value)
		return nil
	})
	v.mtab.mu.RUnlock()
	return list
}

// findMountPoints find matched mount point according to name
func (v *Vfs) findMountPoint(name string) (mp *MountPoint, fsys FS, unrooted string) {
	name = path.Clean(name)
	//only support slash
	name = filepath.ToSlash(name)

	mp, unrooted = v.mtab.mounts.Get(name)
	if mp == nil {
		mpRoot, _ := v.mtab.mounts.Get("/")
		if mpRoot != nil {
			return mpRoot, mpRoot.fS, strings.TrimPrefix(name, "/")
		}
		return nil, nil, unrooted
	}
	return mp, mp.fS, unrooted
}

// A MountPoint represents a mounted file system.
type MountPoint struct {
	prefix    string // path to FS
	fS        FS     // mounted file system
	openCount int32  // number of open files
}

func (mp *MountPoint) closed() {
	if atomic.AddInt32(&mp.openCount, -1) < 0 {
		panic("open count < 0")
	}
}

func (mp *MountPoint) GetPrefix() string {
	return mp.prefix
}

func (mp *MountPoint) GetFS() FS {
	return mp.fS
}
func (mp *MountPoint) GetOpenCount() int32 {
	return mp.openCount
}

type mountTable struct {
	mu     sync.RWMutex
	mounts *trie.PathTrie[*MountPoint]
}

var _ Blob = (*Vfs)(nil)
