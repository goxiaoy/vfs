package vfs

import (
	"context"
	"io/fs"
	"os"
	"path"
	"sync/atomic"
	"syscall"
	"time"
)

func (v *Vfs) Create(name string) (File, error) {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return nil, syscall.ENOENT
	}
	return fsys.Create(unrooted)
}

func (v *Vfs) Mkdir(name string, perm os.FileMode) (err error) {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		err = syscall.ENOENT
		goto error
	}
	if err = fsys.Mkdir(unrooted, perm); err != nil {
		goto error
	}
	return nil
	err = syscall.ENOTSUP
error:
	return &fs.PathError{Op: "mkdir", Path: name, Err: err}
}

func (v *Vfs) MkdirAll(p string, perm os.FileMode) (err error) {
	//TODO optimize for same FS
	p = path.Clean(p)

	// Fast path: if we can tell whether path is a directory or file, stop with success or error.
	dir, err := v.Stat(p)
	if err == nil {
		if dir.IsDir() {
			return nil
		}
		return &os.PathError{Op: "mkdir", Path: p, Err: syscall.ENOTDIR}
	}

	// Slow path: make sure parent exists and then call Mkdir for path.
	i := len(p)
	for i > 0 && os.IsPathSeparator(p[i-1]) { // Skip trailing path separator.
		i--
	}
	j := i
	for j > 0 && !os.IsPathSeparator(p[j-1]) { // Scan backward over element.
		j--
	}
	if j > 1 {
		// Recursively Create parent
		err = v.MkdirAll(p[0:j-1], perm)
		if err != nil {
			return err
		}
	}
	// Parent now exists; invoke Mkdir and use its result.
	return v.Mkdir(p, perm)
}

func (v *Vfs) Open(name string) (f File, err error) {
	v.mtab.mu.RLock()
	mp, fsys, unrooted := v.findMountPoint(name)
	if mp != nil {
		if atomic.AddInt32(&mp.openCount, 1) < 0 {
			atomic.AddInt32(&mp.openCount, -1)
			err = syscall.EMFILE
		}
	}
	v.mtab.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if mp != nil {
		f, err = fsys.Open(unrooted)
		if err != nil {
			return nil, err
		}
		return newFileWrapper(f, mp.closed), nil
	}
	return nil, syscall.ENOENT
}

func (v *Vfs) OpenFile(name string, flag int, perm os.FileMode) (f File, err error) {
	v.mtab.mu.RLock()
	mp, fsys, unrooted := v.findMountPoint(name)
	if mp != nil {
		if atomic.AddInt32(&mp.openCount, 1) < 0 {
			atomic.AddInt32(&mp.openCount, -1)
			err = syscall.EMFILE
		}
	}
	v.mtab.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if mp != nil {
		f, err = fsys.OpenFile(unrooted, flag, perm)
		if err != nil {
			return nil, err
		}
		return newFileWrapper(f, mp.closed), nil
	}
	return nil, syscall.ENOENT
}

func (v *Vfs) Remove(name string) error {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return syscall.ENOENT
	}
	return fsys.Remove(unrooted)
}

func (v *Vfs) RemoveAll(path string) error {
	//TODO different FS under path
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(path)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return syscall.ENOENT
	}
	return fsys.RemoveAll(unrooted)
}

func (v *Vfs) Rename(oldname, newname string) error {
	v.mtab.mu.RLock()
	_, oldfs, oldunrooted := v.findMountPoint(oldname)
	_, newfs, newunrooted := v.findMountPoint(newname)
	v.mtab.mu.RUnlock()
	if oldfs == nil || newfs == nil {
		return syscall.ENOENT
	}
	if oldfs == newfs {
		return oldfs.Rename(oldunrooted, newunrooted)
	}
	// unsupported operation
	// TODO should we add option for copy then delete
	return syscall.ENOTSUP
}

func (v *Vfs) Stat(name string) (os.FileInfo, error) {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return nil, syscall.ENOENT
	}
	return fsys.Stat(unrooted)
}

func (v *Vfs) Name() string {
	return Name
}

func (v *Vfs) Chmod(name string, mode os.FileMode) error {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return syscall.ENOENT
	}
	return fsys.Chmod(unrooted, mode)
}

func (v *Vfs) Chown(name string, uid, gid int) error {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return syscall.ENOENT
	}
	return fsys.Chown(unrooted, uid, gid)
}

func (v *Vfs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return syscall.ENOENT
	}
	return fsys.Chtimes(unrooted, atime, mtime)
}

func (v *Vfs) PreSignedURL(ctx context.Context, name string, args ...LinkOptions) (*Link, error) {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return nil, syscall.ENOENT
	}
	if fsys, ok := fsys.(Linker); !ok {
		return nil, ErrNotSupported
	} else {
		return fsys.PreSignedURL(ctx, unrooted, args...)
	}
}

func (v *Vfs) PublicUrl(ctx context.Context, name string) (*Link, error) {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return nil, syscall.ENOENT
	}
	if fsys, ok := fsys.(Linker); !ok {
		return nil, ErrNotSupported
	} else {
		return fsys.PublicUrl(ctx, unrooted)
	}
}

func (v *Vfs) InternalUrl(ctx context.Context, name string, args ...LinkOptions) (*Link, error) {
	v.mtab.mu.RLock()
	_, fsys, unrooted := v.findMountPoint(name)
	v.mtab.mu.RUnlock()
	if fsys == nil {
		return nil, syscall.ENOENT
	}
	if fsys, ok := fsys.(Linker); !ok {
		return nil, ErrNotSupported
	} else {
		return fsys.InternalUrl(ctx, unrooted, args...)
	}
}
