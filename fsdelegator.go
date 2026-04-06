package wfs

import (
	"io/fs"
	"time"
)

// OpenFSDelegator implements fs.FS interface.
type OpenFSDelegator struct {
	OpenFunc func(name string) (fs.File, error)
}

// Open calls OpenFunc(name).
func (d *OpenFSDelegator) Open(name string) (fs.File, error) {
	if d.OpenFunc == nil {
		return nil, &fs.PathError{Op: "Open", Path: name, Err: ErrNotImplemented}
	}
	return d.OpenFunc(name)
}

// DelegateOpenFS returns a OpenFSDelegator delegates fsys.Open.
func DelegateOpenFS(fsys fs.FS) *OpenFSDelegator {
	return &OpenFSDelegator{OpenFunc: fsys.Open}
}

// FSDelegator implements all filesystem interfaces in io/fs and WriteFileFS.
type FSDelegator struct {
	OpenFunc       func(name string) (fs.File, error)
	ReadDirFunc    func(name string) ([]fs.DirEntry, error)
	ReadFileFunc   func(name string) ([]byte, error)
	GlobFunc       func(pattern string) ([]string, error)
	StatFunc       func(name string) (fs.FileInfo, error)
	SubFunc        func(dir string) (fs.FS, error)
	MkdirAllFunc   func(dir string, mode fs.FileMode) error
	CreateFileFunc func(name string, mode fs.FileMode) (WriterFile, error)
	WriteFileFunc  func(name string, p []byte, mode fs.FileMode) (int, error)
	RemoveFileFunc func(name string) error
	RemoveAllFunc  func(path string) error
	RenameFunc     func(oldpath, newpath string) error
}

var (
	_ fs.FS         = (*FSDelegator)(nil)
	_ fs.GlobFS     = (*FSDelegator)(nil)
	_ fs.ReadDirFS  = (*FSDelegator)(nil)
	_ fs.ReadFileFS = (*FSDelegator)(nil)
	_ fs.StatFS     = (*FSDelegator)(nil)
	_ fs.SubFS      = (*FSDelegator)(nil)
	_ WriteFileFS   = (*FSDelegator)(nil)
	_ RemoveFileFS  = (*FSDelegator)(nil)
	_ RenameFS      = (*FSDelegator)(nil)
)

// Open calls OpenFunc(name).
func (d *FSDelegator) Open(name string) (fs.File, error) {
	if d.OpenFunc == nil {
		return nil, &fs.PathError{Op: "Open", Path: name, Err: ErrNotImplemented}
	}
	return d.OpenFunc(name)
}

// ReadDir calls ReadDirFunc(name).
func (d *FSDelegator) ReadDir(name string) ([]fs.DirEntry, error) {
	if d.ReadDirFunc == nil {
		return nil, &fs.PathError{Op: "ReadDir", Path: name, Err: ErrNotImplemented}
	}
	return d.ReadDirFunc(name)
}

// ReadFile calls ReadFileFunc(name).
func (d *FSDelegator) ReadFile(name string) ([]byte, error) {
	if d.ReadFileFunc == nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: name, Err: ErrNotImplemented}
	}
	return d.ReadFileFunc(name)
}

// Glob calls GlobFunc(name).
func (d *FSDelegator) Glob(pattern string) ([]string, error) {
	if d.GlobFunc == nil {
		return nil, &fs.PathError{Op: "Glob", Path: pattern, Err: ErrNotImplemented}
	}
	return d.GlobFunc(pattern)
}

// Stat calls StatFunc(name).
func (d *FSDelegator) Stat(name string) (fs.FileInfo, error) {
	if d.StatFunc == nil {
		return nil, &fs.PathError{Op: "Stat", Path: name, Err: ErrNotImplemented}
	}
	return d.StatFunc(name)
}

// Sub calls SubFunc(name).
func (d *FSDelegator) Sub(name string) (fs.FS, error) {
	if d.SubFunc == nil {
		return nil, &fs.PathError{Op: "Sub", Path: name, Err: ErrNotImplemented}
	}
	return d.SubFunc(name)
}

// MkdirAll calls MkdirAllFunc(dir).
func (d *FSDelegator) MkdirAll(dir string, mode fs.FileMode) error {
	if d.MkdirAllFunc == nil {
		// NOTE: return no error.
		return nil
	}
	return d.MkdirAllFunc(dir, mode)
}

// CreateFile calls CreateFileFunc(name).
func (d *FSDelegator) CreateFile(name string, mode fs.FileMode) (WriterFile, error) {
	if d.CreateFileFunc == nil {
		return nil, &fs.PathError{Op: "CreateFile", Path: name, Err: ErrNotImplemented}
	}
	return d.CreateFileFunc(name, mode)
}

// WriteFile calls WriteFileFunc(name).
func (d *FSDelegator) WriteFile(name string, p []byte, mode fs.FileMode) (int, error) {
	if d.WriteFileFunc == nil {
		return 0, &fs.PathError{Op: "WriteFile", Path: name, Err: ErrNotImplemented}
	}
	return d.WriteFileFunc(name, p, mode)
}

// RemoveFile calls RemoveFileFunc(name).
func (d *FSDelegator) RemoveFile(name string) error {
	if d.RemoveFileFunc == nil {
		return &fs.PathError{Op: "RemoveFile", Path: name, Err: ErrNotImplemented}
	}
	return d.RemoveFileFunc(name)
}

// Rename calls RenameFunc(oldpath, newpath).
func (d *FSDelegator) Rename(oldpath, newpath string) error {
	if d.RenameFunc == nil {
		return &fs.PathError{Op: "Rename", Path: oldpath, Err: ErrNotImplemented}
	}
	return d.RenameFunc(oldpath, newpath)
}

// RemoveAll calls RemoveAllFunc(name).
func (d *FSDelegator) RemoveAll(path string) error {
	if d.RemoveAllFunc == nil {
		return &fs.PathError{Op: "RemoveAll", Path: path, Err: ErrNotImplemented}
	}
	return d.RemoveAllFunc(path)
}

// DelegateFS returns a FSDelegator delegates the functions of the specified filesystem.
// If you want to delegate an open only filesystem like os.DirFS(dir string) use DelegateOpenFS instead.
func DelegateFS(fsys fs.FS) *FSDelegator {
	d := &FSDelegator{
		OpenFunc: fsys.Open,
	}
	if casted, ok := fsys.(fs.ReadDirFS); ok {
		d.ReadDirFunc = casted.ReadDir
	} else {
		d.ReadDirFunc = func(name string) ([]fs.DirEntry, error) {
			return fs.ReadDir(fsys, name)
		}
	}
	if casted, ok := fsys.(fs.ReadFileFS); ok {
		d.ReadFileFunc = casted.ReadFile
	} else {
		d.ReadFileFunc = func(name string) ([]byte, error) {
			return fs.ReadFile(fsys, name)
		}
	}
	if casted, ok := fsys.(fs.GlobFS); ok {
		d.GlobFunc = casted.Glob
	} else {
		d.GlobFunc = func(pattern string) ([]string, error) {
			return fs.Glob(fsys, pattern)
		}
	}
	if casted, ok := fsys.(fs.StatFS); ok {
		d.StatFunc = casted.Stat
	} else {
		d.StatFunc = func(name string) (fs.FileInfo, error) {
			return fs.Stat(fsys, name)
		}
	}
	if casted, ok := fsys.(fs.SubFS); ok {
		d.SubFunc = casted.Sub
	} else {
		d.SubFunc = func(dir string) (fs.FS, error) {
			return fs.Sub(fsys, dir)
		}
	}
	if casted, ok := fsys.(WriteFileFS); ok {
		d.CreateFileFunc = casted.CreateFile
		d.WriteFileFunc = casted.WriteFile
	}
	if casted, ok := fsys.(RemoveFileFS); ok {
		d.RemoveFileFunc = casted.RemoveFile
		d.RemoveAllFunc = casted.RemoveAll
	}
	if casted, ok := fsys.(RenameFS); ok {
		d.RenameFunc = casted.Rename
	}
	return d
}

// FileDelegator implements fs.File, fs.ReadDirFile and WriterFile interface.
type FileDelegator struct {
	StatFunc    func() (fs.FileInfo, error)
	ReadFunc    func(p []byte) (int, error)
	CloseFunc   func() error
	ReadDirFunc func(n int) ([]fs.DirEntry, error)
	WriteFunc   func(p []byte) (int, error)
	SyncFunc    func() error
}

var (
	_ fs.File        = (*FileDelegator)(nil)
	_ fs.ReadDirFile = (*FileDelegator)(nil)
	_ WriterFile     = (*FileDelegator)(nil)
	_ SyncWriterFile = (*FileDelegator)(nil)
)

// Stat calls StatFunc().
func (f *FileDelegator) Stat() (fs.FileInfo, error) {
	if f.StatFunc == nil {
		return nil, ErrNotImplemented
	}
	return f.StatFunc()
}

// Read calls ReadFunc(p).
func (f *FileDelegator) Read(p []byte) (int, error) {
	if f.ReadFunc == nil {
		return 0, ErrNotImplemented
	}
	return f.ReadFunc(p)
}

// Close calls CloseFunc().
func (f *FileDelegator) Close() error {
	if f.CloseFunc == nil {
		// NOTE: return no error.
		return nil
	}
	return f.CloseFunc()
}

// ReadDir calls ReadDirFunc(n).
func (f *FileDelegator) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.ReadDirFunc == nil {
		return nil, ErrNotImplemented
	}
	return f.ReadDirFunc(n)
}

// Write calls WriteFunc(n).
func (f *FileDelegator) Write(p []byte) (int, error) {
	if f.WriteFunc == nil {
		return 0, ErrNotImplemented
	}
	return f.WriteFunc(p)
}

// Sync calls SyncFunc().
func (f *FileDelegator) Sync() error {
	if f.SyncFunc == nil {
		// NOTE: return no error so non-syncing files behave as a no-op.
		return nil
	}
	return f.SyncFunc()
}

// DelegateFile returns a FileDelegator delegates the functions of the specified file.
func DelegateFile(f fs.File) *FileDelegator {
	d := &FileDelegator{
		StatFunc:  f.Stat,
		ReadFunc:  f.Read,
		CloseFunc: f.Close,
	}
	if f, ok := f.(fs.ReadDirFile); ok {
		d.ReadDirFunc = f.ReadDir
	}
	if f, ok := f.(WriterFile); ok {
		d.WriteFunc = f.Write
	}
	if f, ok := f.(SyncWriterFile); ok {
		d.SyncFunc = f.Sync
	}
	return d
}

// DirEntryValues holds values for fs.DirEntry.
type DirEntryValues struct {
	Name  string
	IsDir bool
	Type  fs.FileMode
	Info  fs.FileInfo
}

// DirEntryDelegator implements fs.DirEntry.
type DirEntryDelegator struct {
	Values   DirEntryValues
	InfoFunc func() (fs.FileInfo, error)
}

var _ (fs.DirEntry) = (*DirEntryDelegator)(nil)

// Name returns d.Values.Name.
func (d *DirEntryDelegator) Name() string {
	return d.Values.Name
}

// IsDir returns d.Values.IsDir.
func (d *DirEntryDelegator) IsDir() bool {
	return d.Values.IsDir
}

// Type returns d.Values.Type.
func (d *DirEntryDelegator) Type() fs.FileMode {
	return d.Values.Type
}

// Info calls d.InfoFunc if the function is set otherwise returns d.Values.Info.
func (d *DirEntryDelegator) Info() (fs.FileInfo, error) {
	if d.InfoFunc != nil {
		return d.InfoFunc()
	}
	return d.Values.Info, nil
}

// DelegateDirEntry returns a DirEntryDelegator delegates the functions of the specified DirEntry.
func DelegateDirEntry(d fs.DirEntry) *DirEntryDelegator {
	return &DirEntryDelegator{
		Values: DirEntryValues{
			Name:  d.Name(),
			IsDir: d.IsDir(),
			Type:  d.Type(),
		},
		InfoFunc: d.Info,
	}
}

// FileInfoValues holds values for fs.FileInfo.
type FileInfoValues struct {
	Name    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
	Sys     interface{}
}

// FileInfoDelegator implements fs.FileInfo.
type FileInfoDelegator struct {
	Values FileInfoValues
}

var _ (fs.FileInfo) = (*FileInfoDelegator)(nil)

// Name returns d.Values.Name.
func (d *FileInfoDelegator) Name() string {
	return d.Values.Name
}

// Size returns d.Values.Size.
func (d *FileInfoDelegator) Size() int64 {
	return d.Values.Size
}

// Mode returns d.Values.Mode.
func (d *FileInfoDelegator) Mode() fs.FileMode {
	return d.Values.Mode
}

// ModTime returns d.Values.ModTime.
func (d *FileInfoDelegator) ModTime() time.Time {
	return d.Values.ModTime
}

// IsDir returns d.Values.IsDir.
func (d *FileInfoDelegator) IsDir() bool {
	return d.Values.IsDir
}

// Sys returns d.Values.Sys.
func (d *FileInfoDelegator) Sys() interface{} {
	return d.Values.Sys
}

// DelegateFileInfo returns a FileInfoDelegator delegates the functions of the specified FileInfo.
func DelegateFileInfo(info fs.FileInfo) *FileInfoDelegator {
	return &FileInfoDelegator{
		Values: FileInfoValues{
			Name:    info.Name(),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
			Sys:     info.Sys(),
		},
	}
}
