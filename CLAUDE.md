# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Hless (executable: `hh`) is a terminal pager program designed for analyzing log files. It provides advanced features beyond traditional pagers like `less`, including intelligent line segmentation, multiple search patterns with color highlighting, and selective view capabilities.

## Build Commands

```bash
# Build native binary
make hh

# Build all targets (native + cross-compilation)
make all

# Cross-compile for Linux
make hh-amd64    # Linux x86_64
make hh-arm64    # Linux ARM64

# Run tests
make test
# or
go test ./...

# Generate test log files
make logs

# Clean build artifacts
make clean
```

## Architecture

### Core Design Principles
- Lines are broken into segments that fit the terminal width
- Cursor movement intelligently handles segments (can stay on same line while scrolling through segments)
- Efficient handling of large log files through streaming and buffering

### Key Components

1. **internal/lines/** - File content management
   - `buffer.go`: Manages file buffering and reading
   - `lines.go`: Line and section management, core data structures
   - `view.go`: Rendering logic for displaying content
   - `io.go`: I/O operations for file and stdin handling

2. **internal/model/** - Application state and logic
   - `model.go`: Main application model structure
   - `render.go`: Screen rendering logic
   - `search.go`: Search functionality implementation
   - `find.go`: Find operations
   - `update.go`: State update handling
   - `help.go`: Help system

3. **internal/tui/** - Terminal UI built with Bubble Tea
   - `tui.go`: UI component implementation
   - `style.go`: Lipgloss styling definitions

4. **internal/avltree/** - Custom AVL tree for efficient line indexing

5. **internal/jumpparser/** - Parser for jump commands (e.g., "100", "+10", "-5")

### Dependencies
- Go 1.21 or later
- charmbracelet/bubbletea - Terminal UI framework
- charmbracelet/lipgloss - Styling library
- fred1268/go-clap - Command-line parsing

## Development Workflow

1. **Running with Debug/Trace Logging**:
   ```bash
   ./hh --debug model,search myfile.log
   ./hh --trace all myfile.log
   ```
   Supported modules: `all`, `view`, `model`, `search`, `lines`, `io`

2. **CPU Profiling**:
   ```bash
   ./hh --profile myfile.log
   ```

3. **Testing Specific Packages**:
   ```bash
   go test ./internal/lines/...
   go test ./internal/model/...
   ```

4. **Generating Test Data**:
   ```bash
   make logs  # Creates various sized log files in logs/
   ```

## Key Implementation Details

- The `Lines` struct (internal/lines/lines.go) is the central data structure managing file content
- Line objects contain name, offset, and contents
- Section objects define named portions of files for generating line names
- View rendering handles terminal resize events and maintains scroll position
- Search functionality supports multiple patterns with different color assignments
- The model follows the Elm Architecture pattern through Bubble Tea

## Common Tasks

### Adding a New Feature
1. Update the model struct in `internal/model/model.go` if new state is needed
2. Add command handling in `internal/model/update.go`
3. Update rendering logic in `internal/model/render.go` if display changes
4. Add tests in the corresponding `*_test.go` file

### Debugging Issues
1. Enable debug logging for relevant modules
2. Check `internal/hhlog/` for logging utilities
3. Use CPU profiling for performance issues

### Working with the TUI
- The UI is built with Bubble Tea following the Elm Architecture
- Commands return `tea.Cmd` for async operations
- Updates modify the model and trigger re-renders
- Styling uses Lipgloss for terminal colors and formatting