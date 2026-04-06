// Package osfs provides a filesystem for the OS.
package osfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mojatter/wfs"
)

// DirFS returns a filesystem for the tree of files rooted at the directory dir.
// The filesystem can write using wfs.WriteFile(fsys fs.FS, name string, p []byte).
func DirFS(dir string) fs.FS {
	return New(dir)
}

// containsDenyWin reports whether any path characters of windows are within s.
func containsDenyWin(s string) bool {
	return strings.ContainsAny(s, `\:`)
}

// isInvalidPath reports whether the given path name is valid for use in a call to Create and Write.
func isInvalidPath(name string) bool {
	return !fs.ValidPath(name) || runtime.GOOS == "windows" && containsDenyWin(name)
}

// Indirected through package vars so tests can swap them out. osfs is a
// filesystem abstraction whose purpose is to operate on caller-supplied
// paths joined under the configured root Dir, after isInvalidPath rejects
// traversal-unsafe names — so the G304-style concern around os.Create is
// intentional here.
var (
	osCreateFunc    = os.Create
	osMkdirAllFunc  = os.MkdirAll
	osRemoveFunc    = os.Remove
	osRemoveAllFunc = os.RemoveAll
	osRenameFunc    = os.Rename
)

// OSFS represents a filesystem for the OS.
type OSFS struct {
	Dir  string
	osFS *wfs.FSDelegator
}

var (
	_ fs.FS            = (*OSFS)(nil)
	_ fs.GlobFS        = (*OSFS)(nil)
	_ fs.ReadDirFS     = (*OSFS)(nil)
	_ fs.ReadFileFS    = (*OSFS)(nil)
	_ fs.StatFS        = (*OSFS)(nil)
	_ fs.SubFS         = (*OSFS)(nil)
	_ wfs.WriteFileFS  = (*OSFS)(nil)
	_ wfs.RemoveFileFS = (*OSFS)(nil)
	_ wfs.RenameFS     = (*OSFS)(nil)
)

// NewOSFS returns a filesystem for the tree of files rooted at the directory dir.
// Deprecated: Use New.
func NewOSFS(dir string) *OSFS {
	return New(dir)
}

// New returns a filesystem for the tree of files rooted at the directory dir.
func New(dir string) *OSFS {
	return &OSFS{
		Dir:  dir,
		osFS: wfs.DelegateFS(os.DirFS(dir)),
	}
}

// Open opens the named file.
func (fsys *OSFS) Open(name string) (fs.File, error) {
	return fsys.osFS.Open(name)
}

// Glob returns the names of all files matching pattern, providing an implementation
// of the top-level Glob function.
func (fsys *OSFS) Glob(pattern string) ([]string, error) {
	return fsys.osFS.Glob(pattern)
}

// ReadDir reads the named directory and returns a list of directory entries sorted
// by filename.
func (fsys *OSFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	return fsys.osFS.ReadDir(dir)
}

// ReadFile reads the named file and returns its contents.
func (fsys *OSFS) ReadFile(name string) ([]byte, error) {
	return fsys.osFS.ReadFile(name)
}

// Stat returns a FileInfo describing the file. If there is an error, it should be
// of type *PathError.
func (fsys *OSFS) Stat(name string) (fs.FileInfo, error) {
	return fsys.osFS.Stat(name)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (fsys *OSFS) Sub(dir string) (fs.FS, error) {
	return NewOSFS(filepath.Join(fsys.Dir, dir)), nil
}

// MkdirAll creates the named directory.
func (fsys *OSFS) MkdirAll(dir string, mode fs.FileMode) error {
	if isInvalidPath(dir) {
		return &fs.PathError{Op: "MkdirAll", Path: dir, Err: fs.ErrInvalid}
	}
	return osMkdirAllFunc(filepath.Join(fsys.Dir, dir), mode)
}

// CreateFile creates the named file.
func (fsys *OSFS) CreateFile(name string, mode fs.FileMode) (wfs.WriterFile, error) {
	if isInvalidPath(name) {
		return nil, &fs.PathError{Op: "Create", Path: name, Err: fs.ErrInvalid}
	}
	path := filepath.Join(fsys.Dir, name)
	err := osMkdirAllFunc(filepath.Dir(path), mode)
	if err != nil {
		return nil, err
	}
	return osCreateFunc(path)
}

// WriteFile writes the specified bytes to the named file.
func (fsys *OSFS) WriteFile(name string, p []byte, mode fs.FileMode) (int, error) {
	f, err := fsys.CreateFile(name, mode)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return f.Write(p)
}

// RemoveFile removes the specified named file.
func (fsys *OSFS) RemoveFile(name string) error {
	if isInvalidPath(name) {
		return &fs.PathError{Op: "Remove", Path: name, Err: fs.ErrInvalid}
	}
	return osRemoveFunc(filepath.Join(fsys.Dir, name))
}

// Rename renames oldpath to newpath using os.Rename. On POSIX-backed
// filesystems this is atomic when both paths are on the same filesystem.
func (fsys *OSFS) Rename(oldpath, newpath string) error {
	if isInvalidPath(oldpath) {
		return &fs.PathError{Op: "Rename", Path: oldpath, Err: fs.ErrInvalid}
	}
	if isInvalidPath(newpath) {
		return &fs.PathError{Op: "Rename", Path: newpath, Err: fs.ErrInvalid}
	}
	return osRenameFunc(filepath.Join(fsys.Dir, oldpath), filepath.Join(fsys.Dir, newpath))
}

// RemoveAll removes path and any children it contains.
func (fsys *OSFS) RemoveAll(path string) error {
	if isInvalidPath(path) {
		return &fs.PathError{Op: "RemoveAll", Path: path, Err: fs.ErrInvalid}
	}
	return osRemoveAllFunc(filepath.Join(fsys.Dir, path))
}
