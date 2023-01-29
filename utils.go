package vfs

import (
	"io"
	"path/filepath"
)

// CopyFile slow copy file across different FS
func CopyFile(srcFs FS, srcFilePath string, destFs FS, destFilePath string) error {
	// Some code from https://www.socketloop.com/tutorials/golang-copy-directory-including-sub-directories-files
	srcFile, err := srcFs.Open(srcFilePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := destFs.Create(destFilePath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	if err != nil {
		err = destFs.Chmod(destFilePath, srcInfo.Mode())
	}

	return nil
}

// CopyDir slow copy dir across different FS
func CopyDir(srcFs FS, srcDirPath string, destFs FS, destDirPath string) error {
	// Some code from https://www.socketloop.com/tutorials/golang-copy-directory-including-sub-directories-files

	// get properties of source dir
	srcInfo, err := srcFs.Stat(srcDirPath)
	if err != nil {
		return err
	}

	// create dest dir
	if err = destFs.MkdirAll(destDirPath, srcInfo.Mode()); err != nil {
		return err
	}

	directory, err := srcFs.Open(srcDirPath)
	if err != nil {
		return err
	}
	defer directory.Close()

	entries, err := directory.Readdir(-1)

	for _, e := range entries {
		srcFullPath := filepath.Join(srcDirPath, e.Name())
		destFullPath := filepath.Join(destDirPath, e.Name())

		if e.IsDir() {
			// create sub-directories - recursively
			if err = CopyDir(srcFs, srcFullPath, destFs, destFullPath); err != nil {
				return err
			}
		} else {
			// perform copy
			if err = CopyFile(srcFs, srcFullPath, destFs, destFullPath); err != nil {
				return err
			}
		}
	}

	return nil
}
