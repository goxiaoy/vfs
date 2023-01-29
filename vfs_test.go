package vfs

import (
	"embed"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"syscall"
	"testing"
)

//go:embed tests
var embedFs embed.FS

func TestMount(t *testing.T) {

	var err error

	vfs := New()

	//invalid args
	assert.Error(t, vfs.Mount("", afero.NewMemMapFs()))
	assert.Error(t, vfs.Mount("abc", afero.NewMemMapFs()))
	assert.Error(t, vfs.Mount("/", nil))
	assert.ErrorIs(t, vfs.Mount("/", vfs), ErrRecursive)

	memFsRoot := afero.NewMemMapFs()
	memFsA := afero.NewMemMapFs()

	// mount
	assert.NoError(t, vfs.Mount("/", memFsRoot))
	assert.NoError(t, vfs.Mount("/a", memFsA))
	assert.NoError(t, vfs.Mount("/a/b", afero.NewMemMapFs()))
	assert.NoError(t, vfs.Mount("/a/b/c", afero.FromIOFS{FS: embedFs}))
	assert.NoError(t, vfs.Mount("/b/c/d", afero.NewMemMapFs()))

	//test find mount
	{
		_, mfs, unroot := vfs.findMountPoint("/a.txt")
		assert.Equal(t, memFsRoot, mfs)
		assert.Equal(t, "a.txt", unroot)

		_, mfs, unroot = vfs.findMountPoint("/bc")
		assert.Equal(t, memFsRoot, mfs)
		assert.Equal(t, "bc", unroot)

		_, mfs, unroot = vfs.findMountPoint("/a/no")
		assert.Equal(t, memFsA, mfs)
		assert.Equal(t, "no", unroot)

		_, mfs, unroot = vfs.findMountPoint("/a/no/no")
		assert.Equal(t, memFsA, mfs)
		assert.Equal(t, "no/no", unroot)

		_, mfs, unroot = vfs.findMountPoint("/a")
		assert.Equal(t, memFsA, mfs)
		assert.Equal(t, "", unroot)

	}

	{
		exist, err := afero.Exists(vfs, "/a/b/c/tests/embed.txt")
		assert.NoError(t, err)
		assert.True(t, exist)
	}

	osFs := afero.NewBasePathFs(afero.NewOsFs(), "./temp")
	assert.NoError(t, osFs.MkdirAll(".", 0777))
	defer func() {
		assert.NoError(t, osFs.RemoveAll("."))
	}()

	assert.NoError(t, vfs.Mount("/e", osFs))

	_, err = vfs.Create("/a/b/c/1.txt")
	//read only
	assert.Error(t, err)

	f2, err := vfs.Create("/e/2.txt")
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, f2.Close())
	}()

	exist, err := afero.Exists(vfs, "/e/2.txt")
	assert.NoError(t, err)
	assert.True(t, exist)

	text := "hello world!"
	_, err = f2.WriteString(text)
	assert.NoError(t, err)

	err = vfs.MkdirAll("/f/g", 0777)
	assert.NoError(t, err)

	err = vfs.MkdirAll("/e/f/g", 0777)
	assert.NoError(t, err)

}

func TestUnmount(t *testing.T) {
	vfs := New()
	assert.NoError(t, vfs.Mount("/", afero.NewMemMapFs()))
	assert.NoError(t, vfs.Mount("/a", afero.NewMemMapFs()))

	assert.NoError(t, vfs.Unmount("/a", nil))
	fs := afero.NewMemMapFs()
	assert.NoError(t, vfs.Mount("/b", fs))
	assert.NoError(t, vfs.Unmount("", fs))

	assert.NoError(t, vfs.Mount("/c", afero.NewMemMapFs()))
	var err error
	_, err = vfs.Create("/c/1.txt")
	assert.NoError(t, err)
	mp, _, _ := vfs.findMountPoint("/c/1.txt")
	assert.Equal(t, int32(0), mp.openCount)
	_, err = vfs.Open("/c/1.txt")
	assert.NoError(t, err)
	mp, _, _ = vfs.findMountPoint("/c/1.txt")
	assert.Equal(t, int32(1), mp.openCount)
	assert.ErrorIs(t, vfs.Unmount("/c", nil), syscall.EBUSY)

	assert.ErrorIs(t, vfs.Unmount("/non-exist", nil), syscall.ENOENT)
}
