package vfs

import (
	"context"
	"github.com/spf13/afero"
	"io/fs"
	"net/http"
	"time"
)

// File represents a file in the filesystem.
type File = afero.File

type FS = afero.Fs

type Linker interface {
	PreSignedURL(ctx context.Context, name string, args ...LinkOptions) (*Link, error)
	PublicUrl(name string) (*Link, error)
	InternalUrl(name string, args ...LinkOptions) (*Link, error)
}

type Mover interface {
	// Move src target to dest
	Move(ctx context.Context, src, dest string) error
}

type Copier interface {
	// Copy src target to dest
	Copy(ctx context.Context, src, dest string) error
}

type Lister interface {
	ListPage(ctx context.Context, pageToken []byte, pageSize int, opts *ListOptions) (retval []*fs.FileInfo, nextPageToken []byte, err error)
}

type Initializer interface {
	Init(ctx context.Context) error
	Dispose(ctx context.Context) error
}

type Blob interface {
	FS
	Linker
	//TODO
	//Mover
	//Copier
	//Lister
}

type Link struct {
	URL        string         `json:"url"`
	Header     http.Header    `json:"header"` // needed header
	Status     int            // status maybe 200 or 206, etc
	Expiration *time.Duration // url expiration time
}

type LinkOptions struct {
	IP     string
	Header http.Header
	Type   string
	Expire *time.Duration
}

type ListOptions struct {
	// Prefix indicates that only blobs with a key starting with this prefix
	// should be returned.
	Prefix string
	// Delimiter sets the delimiter used to define a hierarchical namespace,
	// like a filesystem with "directories". It is highly recommended that you
	// use "" or "/" as the Delimiter. Other values should work through this API,
	// but service UIs generally assume "/".
	//
	// An empty delimiter means that the bucket is treated as a single flat
	// namespace.
	//
	// A non-empty delimiter means that any result with the delimiter in its key
	// after Prefix is stripped will be returned with ListObject.IsDir = true,
	// ListObject.Key truncated after the delimiter, and zero values for other
	// ListObject fields. These results represent "directories". Multiple results
	// in a "directory" are returned as a single result.
	Delimiter string
}

type fileWrapper struct {
	File
	closed func()
}

func newFileWrapper(f File, closed func()) *fileWrapper {
	return &fileWrapper{
		File:   f,
		closed: closed,
	}
}
func (f fileWrapper) Close() error {
	//TODO close fail?
	defer func() {
		if f.closed != nil {
			f.closed()
		}
		f.closed = nil
	}()
	return f.File.Close()
}
