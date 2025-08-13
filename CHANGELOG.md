# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.9.0] - 2025-01-13

### Added
- Fast, concurrent file download capability with configurable worker pools
- Resume support for interrupted downloads with automatic chunk management
- Progress tracking with visual indicators and ETA calculation
- Configurable retry mechanism with exponential backoff
- Cross-platform support (Linux, macOS, Windows)
- Comprehensive test coverage including unit and integration tests
- CI/CD pipeline setup with GitHub Actions
- Basic error handling and recovery mechanisms
- Command-line interface with intuitive flags and options
- Timeout configuration for download operations

### Changed
- Initial release with core functionality

### Fixed
- macOS timeout command compatibility in test scripts
- CI job naming conflicts
- Docker environment test issues

[Unreleased]: https://github.com/forest6511/godl/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/forest6511/godl/releases/tag/v0.9.0