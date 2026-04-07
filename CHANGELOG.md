# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0]

### Added

- `RenameFS` interface and top-level `wfs.Rename` helper for atomic
  cross-filesystem rename support (#6).
- `SyncWriterFile` capability interface so callers can fsync a
  `WriterFile` before closing it. `osfs` wraps `(*os.File).Sync` and
  `memfs` implements it as a no-op so the same caller code works on
  both backends (#6).
- `wfstest.TestRenameFS` reusable conformance test for `RenameFS`
  implementations (#6).
- CI: `staticcheck` and `gosec` are now run on every push and pull
  request (#7).
- CI: Go version matrix (`1.24`, `1.25`, `stable`) so regressions
  against the lowest supported toolchain are caught.

### Changed

- Minimum Go version is now 1.24 (was 1.26) so projects on older
  toolchains can consume wfs.
- `memfs` now documents in the `MemFile` godoc that writes are buffered
  locally and only published on `Close`, that `Sync` is a no-op, and
  that the same applies when used with the new `SyncWriterFile`
  capability (#10).

### Fixed

- `memfs.MemFS.Sub` previously returned a new `MemFS` with a fresh
  `sync.Mutex` while sharing the underlying store. Concurrent access
  through a parent and any of its sub-filesystems therefore raced.
  `Sub` now shares the parent's mutex pointer so all views of the same
  store serialize through a single lock (#8).

### Performance

- `memfs/store.put` no longer calls `sort.Strings` on every insert.
  Bulk loads are now O(n) per put instead of O(n log n), giving roughly
  a 30x speedup on the included `BenchmarkStore_put` (#9).
- `memfs/store.removeAll` uses binary search to find the end of the
  prefix range instead of a linear scan (#9).

## [0.4.1] and earlier

See the git log.

[Unreleased]: https://github.com/mojatter/wfs/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/mojatter/wfs/compare/v0.4.1...v0.5.0
