# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/mojatter/wfs/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/mojatter/wfs/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/mojatter/wfs/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/mojatter/wfs/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/mojatter/wfs/compare/v0.4.1...v0.5.0
