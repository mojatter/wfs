# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/mojatter/wfs/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/mojatter/wfs/compare/v0.4.1...v0.5.0
