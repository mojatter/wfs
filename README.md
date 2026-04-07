# wfs

[![PkgGoDev](https://pkg.go.dev/badge/github.com/mojatter/wfs)](https://pkg.go.dev/github.com/mojatter/wfs)
[![Report Card](https://goreportcard.com/badge/github.com/mojatter/wfs)](https://goreportcard.com/report/github.com/mojatter/wfs)

Package wfs provides writable [io/fs](https://pkg.go.dev/io/fs).FS interfaces.

```go
// WriterFile is a file that provides an implementation fs.File and io.Writer.
type WriterFile interface {
	fs.File
	io.Writer
}

// WriteFileFS is the interface implemented by a filesystem that provides an
// optimized implementation of MkdirAll, CreateFile, WriteFile.
type WriteFileFS interface {
	fs.FS
	MkdirAll(dir string, mode fs.FileMode) error
	CreateFile(name string, mode fs.FileMode) (WriterFile, error)
	WriteFile(name string, p []byte, mode fs.FileMode) (n int, err error)
}

// RemoveFileFS is the interface implemented by a filesystem that provides an
// implementation of RemoveFile.
type RemoveFileFS interface {
	fs.FS
	RemoveFile(name string) error
	RemoveAll(name string) error
}

// RenameFS is the interface implemented by a filesystem that supports
// renaming files. On POSIX-backed filesystems Rename is atomic when both
// paths are on the same filesystem, which is the primitive used to commit
// atomic writes.
type RenameFS interface {
	fs.FS
	Rename(oldpath, newpath string) error
}

// SyncWriterFile is a WriterFile that can flush its contents to stable
// storage. osfs implements Sync via (*os.File).Sync; memfs implements it
// as a no-op so the same caller code works on both backends.
type SyncWriterFile interface {
	WriterFile
	Sync() error
}
```

## Capability layers

wfs follows the same pattern as `io/fs`'s optional interfaces (`fs.GlobFS`,
`fs.StatFS`, ...): start from `fs.FS` and add capabilities by asserting to
optional interfaces. Each capability has a top-level helper that performs
the assertion and returns `ErrNotImplemented` (wrapped in `*fs.PathError`)
if the underlying filesystem does not support it.

| Capability | Interface | Helper |
| --- | --- | --- |
| Read | `fs.FS` | `fs.Open`, `fs.ReadFile`, ... |
| Write | `wfs.WriteFileFS` | `wfs.MkdirAll`, `wfs.CreateFile`, `wfs.WriteFile` |
| Remove | `wfs.RemoveFileFS` | `wfs.RemoveFile`, `wfs.RemoveAll` |
| Atomic rename | `wfs.RenameFS` | `wfs.Rename` |
| File-level fsync | `wfs.SyncWriterFile` | type-assert the `WriterFile` returned by `CreateFile` |

## Atomic writes

`RenameFS` and `SyncWriterFile` together let callers implement crash-safe
atomic writes (temp file + sync + rename) entirely through the wfs
abstraction. The pattern works unchanged across `osfs` (where Rename and
Sync delegate to the OS) and `memfs` (where Rename moves the entry under
the filesystem mutex and Sync is a no-op).

```go
func atomicWrite(fsys fs.FS, name string, src io.Reader) error {
	tmp := name + ".tmp-xxxxxx" // caller generates a unique suffix
	f, err := wfs.CreateFile(fsys, tmp, 0o644)
	if err != nil {
		return err
	}
	// Best-effort cleanup; no-op after successful rename.
	defer func() { _ = wfs.RemoveFile(fsys, tmp) }()

	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		return err
	}
	if sf, ok := f.(wfs.SyncWriterFile); ok {
		if err := sf.Sync(); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return wfs.Rename(fsys, tmp, name)
}
```

## memfs limitations

`memfs` is intended for tests and small in-process workflows. A few
behaviors differ from `osfs` and are worth knowing:

- **Writes are visible only after `Close`.** `MemFile` buffers writes
  locally; other readers do not see the new contents until `Close`
  returns successfully. `osfs` makes writes visible immediately.
- **`Sync` is a no-op.** It exists so that atomic-write helpers can share
  one code path across backends. On `memfs` it does *not* publish the
  buffered bytes — only `Close` does.
- **`Rename` supports files only.** Renaming a directory currently
  returns a `*fs.PathError`. `osfs.Rename` delegates to `os.Rename` and
  therefore handles directories on POSIX systems.

This is one of the solutions to an [issue](https://github.com/golang/go/issues/45757) of github.com/golango/go.

The following packages are an implementation of wfs.

- [osfs](https://pkg.go.dev/github.com/mojatter/wfs/osfs)
- [memfs](https://pkg.go.dev/github.com/mojatter/wfs/memfs)
- [s3fs](https://github.com/mojatter/s3fs)
- [gcsfs](https://github.com/mojatter/gcsfs)

## CopyFS

CopyFS walks the specified root directory on src and copies directories and files to dest filesystem.
The following code is an example.

```go
package main

import (
	"log"

	"github.com/mojatter/s3fs"
	"github.com/mojatter/wfs"
	"github.com/mojatter/wfs/osfs"
)

func main() {
	src := s3fs.New("your-bucket")
	dst := osfs.DirFS("local-dir")

	// NOTE: Copy files on s3://your-bucket to local-dir.
	if err := wfs.CopyFS(dst, src, "."); err != nil {
		log.Fatal(err)
	}
}
```