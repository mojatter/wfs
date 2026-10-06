# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.0]

A behavior-convergence release for open files and file modes. osfs
`CreateFile` and `WriteFile` now apply `mode` to the file instead of to
its missing parents, so `fs.ModePerm` now yields an executable file.
memfs file handles behave as osfs ones do wherever the new
`wfstest.TestFileHandle` checks them; among other things, a deferred
`Close` after an explicit one now fails. `CopyFS` now copies trees with
directories into osfs, rejects non-regular entries with `fs.ErrInvalid`
and returns `Close` errors. One public addition and several visible
changes — read Changed before upgrading.

### Added

- `wfstest.TestFileHandle` checks open-file behavior shared by
  backends: `Write` on a file from `Open`, `CreateFile` truncation,
  writes across `Rename`, `RemoveFile` and `RemoveAll`, and use after
  `Close`. It describes POSIX behavior and needs `wfs.WriteFileFS`,
  `wfs.RemoveFileFS` and `wfs.RenameFS`. The `MemFile` docs now list
  memfs's intentional differences from osfs (#54).

### Changed

- osfs: `CreateFile` and `WriteFile` open the file with `mode` (before
  umask) instead of `os.Create`'s fixed 0666. Callers passing
  `fs.ModePerm` now get an executable file (0755 under umask 022) where
  they got 0644, `0o600` gives 0600, and `0` gives a 0000 file; on
  Windows a `mode` without `0o200` now creates a read-only file. An
  existing file keeps its mode. Callers that passed `fs.ModePerm` for a
  plain file should pass `0o644` (or `0o666`, the `os.Create` default)
  (#39).
- The `WriteFileFS` godoc now states this contract: `CreateFile` and
  `WriteFile` create the file with `mode` (before umask) and missing
  parents with `fs.ModePerm`, an existing file is truncated and keeps
  its mode, and backends without POSIX modes may ignore `mode` (#57).
- osfs, memfs: missing parent directories are created with
  `fs.ModePerm` (before umask on osfs) instead of the file's `mode`, so
  memfs reports `ModeDir|0o777` for them. `MkdirAll` itself is
  unchanged (#39).
- `wfstest.TestWriteFileFS` also writes a nested file with `0o600`, so
  a backend must create missing parents that a file mode without
  execute bits could not traverse. It does not check mode values (#39).
- `CopyFS` creates files with `0o666` and directories with
  `fs.ModePerm` (before umask) instead of the source entry's type bits,
  which had no permission bits. memfs now reports 0666 and 0777 where
  it reported 0000, or `ModeSymlink` with no permission bits for a
  file copied through a symlink; on osfs files keep getting 0644 under
  umask 022. Source modes are not preserved (#39, #56).
- `CopyFS` resolves symlinks with `fs.Stat` and fails with
  `fs.ErrInvalid` for entries that are not, or do not point to, regular
  files: symlinks to directories, named pipes, devices, sockets and
  irregular files. Symlinks to regular files are still copied as files.
  A symlink to a directory used to leave an empty file and fail with
  `is a directory`, a named pipe blocked forever, and a device was read.
  On a source without `fs.StatFS`, such as `fs.Sub` over `os.DirFS`
  (osfs and memfs supply their own `Sub`, which keeps `Stat`), resolving
  a symlink still opens it, so a symlink to a named pipe can still
  block (#61).
- memfs: after `Close`, `Read`, `Write`, `Stat`, `ReadDir`, `Sync` and
  a second `Close` return `fs.ErrClosed`, as on osfs except that osfs's
  `ReadDir` reports `use of closed file` instead. They used to keep
  working, and a second `Close` committed the buffer again over newer
  content. Code that defers `Close` after closing explicitly now gets
  an error from the deferred call (#45).
- memfs: `Close` commits into the entry `CreateFile` returned instead
  of looking the name up again, so a write follows a `Rename` and
  vanishes after a `RemoveFile` or `RemoveAll` made while the file is
  open, as on osfs. It used to land at the old name, resurrect a
  removed file, or fail with `EISDIR` if a directory had taken the name
  (#52).
- memfs: `CreateFile` on an existing file truncates it at once and sets
  its `ModTime`, as `O_TRUNC` does, so a later `Open` or `ReadFile`
  sees an empty file before `Close`. It used to keep the old content
  until a written file was closed, and kept it for good if nothing was
  written. `Close` now commits only after a `Write` of at least one
  byte; a zero-length `Write` used to count, so closing such a handle
  emptied content written to the name in the meantime (#50, #53).
- memfs: `Write` on a file from `Open` now fails with `EBADF`, as on
  osfs. It used to succeed: `Close` then committed the unread bytes
  plus the write, and a `Write` after a `Read` overwrote the stored
  bytes in place. On a directory it panicked (#43).

### Fixed

- osfs: `CreateFile` and `WriteFile` into a missing directory failed
  with `permission denied` for any mode lacking the owner's write or
  search (execute) bit, such as `0o600` or `0o644`, because the parents
  were created with that mode. The README's atomic-write example hit
  this (#39).
- memfs: `MemFile.Stat` on a file from `Open` looked the name up again,
  so after the entry was rewritten, renamed over or removed it
  described the new entry (or failed) while `Read` returned the opened
  bytes. It now returns the `FileInfo` taken at `Open` (#40).
- memfs: `Stat` on a file from `CreateFile` reported the committed
  entry, ignoring the bytes written so far. It now reports the entry as
  created, sized to the bytes written (#44).
- memfs: `Read` on a file from `CreateFile` consumed the bytes waiting
  for `Close`, so they were lost from the committed file. It now
  returns `io.EOF`, as on osfs (#47).
- `CopyFS` into osfs failed with `permission denied` for any tree with
  a directory, because the directories were created with no permission
  bits. `CopyFS` also leaked one descriptor per copied file; it now
  closes the source files (#56).
- osfs `WriteFile` and `CopyFS` dropped the `Close` error of the file
  they wrote and reported success, losing a deferred write-back failure
  such as `ENOSPC` on osfs, or the upload itself on backends that commit
  on `Close`. They now return it, joined with the `Write` error when
  both fail; a `Write` error alone is still returned unchanged (#58).

### Deprecated

- `osfs.NewOSFS` is now scheduled for removal in v1.0.0 instead of
  v0.8.0. Use `osfs.New` instead.

## [0.7.1]

A memfs-only bugfix release. No public API changes.

### Fixed

- memfs: `ModTime` was always the zero time. Entries now get the
  creation time, and `WriteFile` (and so `MemFile.Close`) refreshes it.
  `Rename` keeps it, matching osfs. A directory's `ModTime` does not
  change when entries are added to or removed from it (#36).
- memfs: `Stat` and `ReadDir` returned the stored entry itself, so a
  held `FileInfo` or `DirEntry` changed its `Name`, `Size` and
  `ModTime` when the file was later written or renamed, and reading it
  raced with the writer. They now return a copy, as osfs does (#36).

## [0.7.0]

A behavior-convergence release, continuing v0.6.0. Three memfs
operations (`RemoveFile`, `RemoveAll`, `Rename`) silently succeeded
where osfs reports an error; they now report it. One error kind
changes, and two removals that left the store inconsistent are fixed.
No public API changes — read Changed before upgrading.

### Changed

- memfs: `RemoveFile` now resolves the name before removing it. A
  missing name returns `fs.ErrNotExist` and a path below an existing
  file returns `ENOTDIR`; both were silently a success (#27).
- memfs: `RemoveFile` on a directory now removes it only when it is
  empty and returns `ENOTEMPTY` otherwise, matching `os.Remove`. It
  previously unlinked the directory and left the subtree behind (#32).
- memfs: `RemoveAll` now returns `ENOTDIR` for a path below an
  existing file. A missing name still returns `nil`, matching osfs and
  `os.RemoveAll` (#27).
- memfs: `Rename` no longer creates the missing parent directories of
  `newpath`. A missing parent returns `fs.ErrNotExist` and a parent
  that is a file returns `ENOTDIR`, matching `os.Rename`. A failed
  `Rename` is therefore a no-op again; the parents it created used to
  survive when a later check failed (#27).
- memfs: `Rename` onto an existing directory now returns `EEXIST`
  instead of `fs.ErrInvalid`, matching osfs on Unix, where `os.Rename`
  checks the destination itself before reaching `rename(2)` (#27).

### Fixed

- memfs: `RemoveAll(".")` left every descendant in the store. The
  child prefix was built as `//` at the root, which matches no key, so
  only the root marker was deleted. Reads of the orphans still
  succeeded while the root reported not-exist. The root itself is
  removed, as before and as on osfs; the next write recreates it (#26).
- memfs: a directory removed by `RemoveFile` left its children in the
  store, unreachable. They no longer appeared in `ReadDir` but were
  still readable, and `RemoveAll` could not reclaim them because it
  bails when the directory key is gone (#32).
- memfs: `value.name` held the full store key for files and the base
  segment for directories. `Name()` hid the difference on Unix, but on
  Windows it also split the name on `\`, which `fs.ValidPath` allows,
  so an entry written as `a\b.txt` was reported as `b.txt`. The field
  is now the base segment for both, and `Name()` returns it as written
  (#24).

### Deprecated

- `osfs.NewOSFS` is now scheduled for removal in v0.8.0. It was
  documented for removal in v0.6.0 but kept. Use `osfs.New` instead.

## [0.6.0]

A behavior-convergence release: memfs now matches osfs where the two
disagreed on `Sub`, and on which error a few operations return. No
public API changes, but several memfs error kinds change, and
`osfs.Sub` validates its argument for the first time — see Changed.

### Changed

- memfs: `Sub` is now lazy and no longer stats `dir`. A `Sub` into a
  missing directory succeeds, and a write through the returned FS
  creates the directory, matching `osfs` and the stdlib `fs.Sub`
  fallback so a memfs can stand in for an osfs. Callers that relied on
  `Sub` reporting `fs.ErrNotExist` (or rejecting a file) must check
  with `Stat` themselves.

  Caller-visible consequences on the osfs side of the same
  convergence: `Sub("")` now returns an error where it previously
  returned an FS rooted at `Dir`, non-clean paths such as `cache/`,
  `./cache` and `a//b` are rejected instead of being normalized, and on
  Windows `Sub` now rejects names containing `\` or `:`, like every
  other write path in the package. memfs and the stdlib `fs.Sub` reject
  the same non-clean paths, so this removes a divergence rather than
  adding one (#22).

- memfs: `ReadFile` on a directory now returns `EISDIR` instead of
  `fs.ErrInvalid`, matching osfs (#22).
- memfs: `CreateFile` and `WriteFile` on an existing directory now
  return `EISDIR` instead of `fs.ErrInvalid`, matching osfs (#28).
- memfs: `MkdirAll` now returns `ENOTDIR` instead of `fs.ErrInvalid`
  when a path component is an existing file, matching osfs. This also
  covers writes that create parents, so a write through a `Sub` rooted
  at a file reports the same error as osfs (#22).

### Fixed

- osfs: `Sub` now rejects paths that fail `fs.ValidPath`. Previously
  `Sub("../outside")` escaped the configured root, making files
  readable that `ReadFile` rejects on the same FS (#22).
- memfs: `mkdirAll` no longer applies the sub-FS root twice when
  called through a `Sub`. `root.Sub("a")` then `WriteFile("b/c.txt")`
  created `a/a` and `a/a/b` but never `a/b`, so `ReadDir("b")` failed
  while `ReadFile("b/c.txt")` succeeded. A directory created this way
  was also named `.`, which made the parent list an entry `.` and sent
  `fs.WalkDir` into infinite recursion (#22).
- memfs: `Open(".")`, `ReadFile(".")` and `Stat(".")` on an FS rooted at
  a file now return `ENOTDIR` instead of the file itself, matching osfs
  (#22).
- memfs: a path below an existing file now reports `ENOTDIR` instead of
  `fs.ErrNotExist`, matching osfs. This covers `Open`, `Stat`, `ReadDir`,
  `ReadFile` and `Rename`, including reads through a `Sub` rooted at a
  file (#22). `RemoveFile` and `RemoveAll` resolve no name at all and
  are left alone (#27).
- memfs: `Glob` through a `Sub` now returns names relative to the sub.
  It trimmed the sub's root without the separator, so it returned
  `/file01.txt`, which `Open` on the same FS then rejected as invalid
  (#22).

## [0.5.1]

A bug-fix release. No public API changes.

### Fixed

- memfs: directory enumeration (`ReadDir`, glob) and `RemoveAll` no
  longer mishandle prefix-named siblings. A sibling such as `dir0-tmp`
  shares the string prefix `dir0` and can sort between a directory key
  and its children (`-` (0x2d) < `/` (0x2f)), which previously cut the
  scan short or deleted the sibling along with `dir0/`. Enumeration and
  removal now match on the full path segment (`dir0/`) instead of the
  bare string prefix (#17).

## [0.5.0]

A documentation, tooling, and compatibility release. No public API
changes; the code-level features (`RenameFS`, `SyncWriterFile`, the
memfs `Sub` mutex fix, the memfs store performance work) all shipped
in v0.4.1 and are now properly documented.

### Added

- README sections covering capability layers, an `atomicWrite` example
  built on `RenameFS` + `SyncWriterFile`, and the memfs limitations
  callers should know about (Close-publishes-writes, Sync-is-a-noop,
  file-only Rename) (#12).
- `CHANGELOG.md`, starting with this entry (#13).
- CI: Go version matrix (`1.24`, `stable`) so regressions against the
  lowest supported toolchain are caught (#11).
- CI: `go test -race` is now run on every PR (#15).
- CI: aggregator job named `tests` so the branch protection required
  check stays stable across future matrix changes (#11 follow-up).

### Changed

- Minimum Go version is now 1.24 (was 1.26) so projects on older
  toolchains can consume wfs (#11).

### Deprecated

- `osfs.NewOSFS` is now documented as scheduled for removal in v0.6.0.
  Use `osfs.New` instead (#14).

## [0.4.1] and earlier

See the git log.

[Unreleased]: https://github.com/mojatter/wfs/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/mojatter/wfs/compare/v0.7.1...v0.8.0
[0.7.1]: https://github.com/mojatter/wfs/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/mojatter/wfs/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/mojatter/wfs/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/mojatter/wfs/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/mojatter/wfs/compare/v0.4.1...v0.5.0
