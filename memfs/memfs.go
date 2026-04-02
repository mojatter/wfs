// Package memfs provides an in-memory filesystem.
package memfs

import (
	"bytes"
	"io"
	"io/fs"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/mojatter/wfs"
)

// MemFS represents an in-memory filesystem.
// MemFS keeps fs.FileMode but that permission is not checked.
type MemFS struct {
	mutex sync.Mutex
	dir   string
	store *store
}

var (
	_ fs.FS            = (*MemFS)(nil)
	_ fs.GlobFS        = (*MemFS)(nil)
	_ fs.ReadDirFS     = (*MemFS)(nil)
	_ fs.ReadFileFS    = (*MemFS)(nil)
	_ fs.StatFS        = (*MemFS)(nil)
	_ fs.SubFS         = (*MemFS)(nil)
	_ wfs.WriteFileFS  = (*MemFS)(nil)
	_ wfs.RemoveFileFS = (*MemFS)(nil)
)

// New returns a new MemFS.
func New() *MemFS {
	return &MemFS{
		dir:   "/",
		store: newStore(),
	}
}

func (fsys *MemFS) key(name string) string {
	return path.Clean(path.Join(fsys.dir, name))
}

func (fsys *MemFS) rel(name string) string {
	return strings.TrimPrefix(name, fsys.dir)
}

func (fsys *MemFS) open(name string) (*value, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "Open", Path: name, Err: fs.ErrInvalid}
	}
	v := fsys.store.get(fsys.key(name))
	if v == nil {
		return nil, &fs.PathError{Op: "Open", Path: name, Err: fs.ErrNotExist}
	}
	return v, nil
}

func (fsys *MemFS) mkdirAll(dir string, mode fs.FileMode) error {
	if !fs.ValidPath(dir) {
		return &fs.PathError{Op: "MkdirAll", Path: dir, Err: fs.ErrInvalid}
	}
	keys := strings.Split(fsys.key(dir), "/")
	for i, k := range keys {
		key := fsys.key(path.Join(keys[0 : i+1]...))
		if v := fsys.store.get(key); v != nil {
			if !v.isDir {
				return &fs.PathError{Op: "MkdirAll", Path: dir, Err: fs.ErrInvalid}
			}
			continue
		}
		if k == "" {
			k = "."
		}
		v := &value{name: k, mode: mode | fs.ModeDir, isDir: true}
		fsys.store.put(key, v)
	}
	return nil
}

func (fsys *MemFS) create(name string, mode fs.FileMode) (*value, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "Create", Path: name, Err: fs.ErrInvalid}
	}
	err := fsys.mkdirAll(path.Dir(name), mode)
	if err != nil {
		return nil, err
	}
	key := fsys.key(name)
	v := fsys.store.get(key)
	if v == nil {
		v = &value{name: key, mode: mode}
		fsys.store.put(key, v)
	} else if v.isDir {
		return nil, &fs.PathError{Op: "Create", Path: name, Err: fs.ErrInvalid}
	}
	return v, nil
}

// Open opens the named file.
func (fsys *MemFS) Open(name string) (fs.File, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	v, err := fsys.open(name)
	if err != nil {
		return nil, err
	}

	f := &MemFile{
		fsys: fsys,
		name: name,
		mode: v.mode,
	}
	if !v.isDir {
		f.buf = bytes.NewBuffer(v.data)
	}
	return f, nil
}

// Glob returns the names of all files matching pattern, providing an implementation
// of the top-level Glob function.
func (fsys *MemFS) Glob(pattern string) ([]string, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	keys, err := fsys.store.prefixGlobKeys(fsys.dir, pattern)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, key := range keys {
		names = append(names, fsys.rel(key))
	}
	return names, nil
}

// ReadDir reads the named directory and returns a list of directory entries sorted
// by filename.
func (fsys *MemFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	v, err := fsys.open(dir)
	if err != nil {
		return nil, err
	}
	if !v.isDir {
		return nil, &fs.PathError{Op: "ReadDir", Path: dir, Err: syscall.ENOTDIR}
	}

	prefix := fsys.key(dir)
	keys := fsys.store.prefixKeys(prefix)
	var dirEntries []fs.DirEntry
	for _, key := range keys {
		dirEntries = append(dirEntries, fsys.store.get(key))
	}
	return dirEntries, nil
}

// ReadFile reads the named file and returns its contents.
func (fsys *MemFS) ReadFile(name string) ([]byte, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	v, err := fsys.open(name)
	if err != nil {
		return nil, err
	}
	if v.isDir {
		return nil, &fs.PathError{Op: "ReadFile", Path: name, Err: fs.ErrInvalid}
	}
	dest := make([]byte, len(v.data))
	copy(dest, v.data)
	return dest, nil
}

// Stat returns a FileInfo describing the file. If there is an error, it should be
// of type *PathError.
func (fsys *MemFS) Stat(name string) (fs.FileInfo, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	return fsys.open(name)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (fsys *MemFS) Sub(dir string) (fs.FS, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "Sub", Path: dir, Err: fs.ErrInvalid}
	}
	info, err := fsys.open(dir)
	if err != nil {
		return nil, err
	}
	if !info.isDir {
		return nil, &fs.PathError{Op: "Sub", Path: dir, Err: fs.ErrInvalid}
	}
	return &MemFS{
		dir:   path.Join(fsys.dir, dir),
		store: fsys.store,
	}, nil
}

// MkdirAll creates the named directory.
func (fsys *MemFS) MkdirAll(dir string, mode fs.FileMode) error {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	return fsys.mkdirAll(dir, mode)
}

// CreateFile creates the named file.
func (fsys *MemFS) CreateFile(name string, mode fs.FileMode) (wfs.WriterFile, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	if _, err := fsys.create(name, mode); err != nil {
		return nil, err
	}
	return &MemFile{
		fsys: fsys,
		name: name,
		buf:  new(bytes.Buffer),
		mode: mode,
	}, nil
}

// WriteFile writes the specified bytes to the named file.
func (fsys *MemFS) WriteFile(name string, p []byte, mode fs.FileMode) (int, error) {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	v, err := fsys.create(name, mode)
	if err != nil {
		return 0, err
	}
	v.data = make([]byte, len(p))
	return copy(v.data, p), nil
}

// RemoveFile removes the specified named file.
func (fsys *MemFS) RemoveFile(name string) error {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "RemoveFile", Path: name, Err: fs.ErrInvalid}
	}

	fsys.store.remove(fsys.key(name))
	return nil
}

// RemoveAll removes path and any children it contains.
func (fsys *MemFS) RemoveAll(path string) error {
	fsys.mutex.Lock()
	defer fsys.mutex.Unlock()

	if !fs.ValidPath(path) {
		return &fs.PathError{Op: "RemoveAll", Path: path, Err: fs.ErrInvalid}
	}

	fsys.store.removeAll(fsys.key(path))
	return nil
}

// MemFile represents an in-memory file.
// MemFile implements fs.File, fs.ReadDirFile and wfs.WriterFile.
type MemFile struct {
	fsys       *MemFS
	name       string
	buf        *bytes.Buffer
	mode       fs.FileMode
	dirRead    bool
	dirEntries []fs.DirEntry
	dirIndex   int
	wrote      bool
}

var (
	_ fs.File        = (*MemFile)(nil)
	_ fs.ReadDirFile = (*MemFile)(nil)
	_ wfs.WriterFile = (*MemFile)(nil)
)

// Read reads bytes from this file.
func (f *MemFile) Read(p []byte) (int, error) {
	if f.buf == nil {
		return 0, &fs.PathError{Op: "Read", Path: f.name, Err: syscall.EISDIR}
	}
	return f.buf.Read(p)
}

// Stat returns the fs.FileInfo of this file.
func (f *MemFile) Stat() (fs.FileInfo, error) {
	return f.fsys.Stat(f.name)
}

// Close closes streams.
func (f *MemFile) Close() error {
	if f.wrote {
		var err error
		_, err = f.fsys.WriteFile(f.name, f.buf.Bytes(), f.mode)
		return err
	}
	f.dirEntries = nil
	return nil
}

// ReadDir reads sub directories.
func (f *MemFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.dirRead {
		f.dirRead = true
		var err error
		f.dirEntries, err = f.fsys.ReadDir(f.name)
		if err != nil {
			return nil, err
		}
	}
	max := len(f.dirEntries)
	if f.dirIndex >= max {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		n = max - f.dirIndex
	}
	end := f.dirIndex + n
	if end > max {
		end = max
	}
	defer func() { f.dirIndex = end }()

	return f.dirEntries[f.dirIndex:end], nil
}

// Write writes the specified bytes to this file.
func (f *MemFile) Write(p []byte) (int, error) {
	f.wrote = true
	return f.buf.Write(p)
}
