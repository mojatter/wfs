# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- memfs: `Sub` is now lazy and no longer stats `dir`. A `Sub` into a
  missing directory succeeds, and a write through the returned FS
  creates the directory, matching `osfs` and the stdlib `fs.Sub`
  fallback so a memfs can stand in for an osfs. Callers that relied on
  `Sub` reporting `fs.ErrNotExist` (or rejecting a file) must check
  with `Stat` themselves.

  Two caller-visible consequences on the osfs side of the same
  convergence: `Sub("")` now returns an error where it previously
  returned an FS rooted at `Dir`, and on Windows `Sub` now rejects
  names containing `\` or `:`, like every other write path in the
  package.

### Fixed

- osfs: `Sub` now rejects paths that fail `fs.ValidPath`. Previously
  `Sub("../outside")` escaped the configured root, making files
  readable that `ReadFile` rejects on the same FS.
- memfs: `mkdirAll` no longer applies the sub-FS root twice when
  called through a `Sub`. `root.Sub("a")` then `WriteFile("b/c.txt")`
  created `a/a` and `a/a/b` but never `a/b`, so `ReadDir("b")` failed
  while `ReadFile("b/c.txt")` succeeded. A directory created this way
  was also named `.`, which made the parent list an entry `.` and sent
  `fs.WalkDir` into infinite recursion.

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

[Unreleased]: https://github.com/mojatter/wfs/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/mojatter/wfs/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/mojatter/wfs/compare/v0.4.1...v0.5.0
