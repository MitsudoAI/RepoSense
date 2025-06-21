# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RepoSense is a Git repository batch management tool written in Go, designed to efficiently manage large numbers (250+) of locally cloned repositories through parallel operations.

**Current Status**: Design phase complete, implementation needed.

## Tech Stack & Dependencies

- **Language**: Go 1.21+
- **Planned CLI Framework**: cobra or urfave/cli
- **Key Dependencies**: progressbar, viper (config), logrus/zap (logging)
- **Architecture**: Worker pool pattern with goroutines for concurrent Git operations

## Project Structure (Target)

```
├── cmd/           # CLI entry points
├── pkg/           # Public API packages  
├── internal/      # Private implementation
│   ├── scanner/   # Repository discovery
│   ├── updater/   # Git operations with worker pool
│   └── reporter/  # Progress and results
└── DESIGN.md      # Comprehensive design document
```

## Development Commands

Project is fully implemented! Standard Go commands:
```bash
go run cmd/reposense/main.go    # Run development version
go build -o reposense ./cmd/reposense  # Build binary
go test ./...                   # Run tests
go fmt ./...                    # Format code
go mod tidy                     # Manage dependencies
```

Usage examples:
```bash
./reposense --help              # Show help
./reposense scan /path/to/repos # Scan repositories
./reposense update /path/to/repos --workers 20  # Batch update
./reposense status /path/to/repos --format table # Show status
```

## Core Architecture

### Main Components
- **Scanner**: Auto-discovers Git repositories in specified directories
- **Updater**: Manages worker pool (default 10 goroutines) for parallel `git pull` operations  
- **Reporter**: Real-time progress display and final statistics

### Key Patterns
- Worker pool for concurrent Git operations
- Producer-consumer task queue
- Repository abstraction for Git commands

## CLI Interface (Planned)

```bash
reposense update [options] <directory>
```

Options include parallel worker count, filtering, dry-run mode, and reporting formats.

## Important Design Considerations

- **Concurrency**: Default 10 workers, configurable via CLI
- **Error Handling**: Continue processing other repos when individual operations fail
- **Performance**: Designed for 250+ repositories with progress tracking
- **Extensibility**: Architecture supports future AI-enhanced features (RAG-based code search)

## Implementation Status

✅ **COMPLETED** - All core functionality implemented:
- ✅ Repository Scanner - Auto-discover Git repositories
- ✅ Batch Updater - Parallel git pull operations with worker pool
- ✅ Status Collector - Detailed repository status information
- ✅ Progress Reporter - Real-time progress and multiple output formats
- ✅ CLI Interface - Full command-line interface with cobra
- ✅ Configuration Management - Flexible configuration options
- ✅ Metadata Analysis System - Comprehensive repository analysis with quality scoring
- ✅ Language Detection - Programming language identification and statistics
- ✅ Framework Detection - Development framework identification
- ✅ License Detection - Open source license identification
- ✅ Quality Scoring - 0-10 scale quality assessment based on best practices
- ✅ Metadata Search - Advanced search capabilities by language, type, quality, etc.
- ✅ LLM Integration - AI-enhanced project descriptions with caching
- ✅ Pure Go SQLite - Cross-platform compatibility without CGO dependencies

## Development Notes

- Read DESIGN.md for comprehensive requirements and implementation details
- Focus on Day 1 core functionality first: scanning, parallel updates, basic reporting
- Use structured logging (logrus/zap) from the start
- Implement proper error handling for Git operations that may fail