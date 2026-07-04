# mdv — Terminal Markdown Viewer: Design Spec

**Date:** 2026-07-03
**Status:** Approved

---

## Overview

`mdv` is a terminal-based markdown viewer built in Go using the Charm ecosystem. It renders markdown documents beautifully in the terminal — styled like GitHub's preview — with a built-in file browser for navigating repos. Editing delegates to `$EDITOR` (nvim).

### Goals

- Render markdown as a styled document (headings, syntax-highlighted code, tables, blockquotes) rather than syntax-highlighted source text
- Navigate a directory's markdown files via a toggleable sidebar
- Edit files in `$EDITOR`, re-render automatically on return
- Single self-contained binary, no config required

### Non-Goals (deferred)

- Live reload while editing (fsnotify-based, planned for v2)
- Configurable keybindings or themes (planned for v3)
- Support for non-markdown file types

---

## Usage

```bash
mdv              # open file browser at current directory
mdv ./path/repo  # open file browser at specified path
```

---

## UI Layout

```
┌─────────────────────────────────────────────────────────────┐
│  mdv — README.md                                            │
├──────────────┬──────────────────────────────────────────────┤
│ BROWSER      │                                              │
│              │  # My Project                                │
│  > README.md │                                              │
│    CONTRIB.. │  A short description of what this does.     │
│    docs/     │                                              │
│      api.md  │  ## Installation                            │
│      guide.. │                                              │
│              │  ```bash                                     │
│              │  go install ...                              │
│              │  ```                                         │
│              │                                              │
├──────────────┴──────────────────────────────────────────────┤
│  [b] browser  [i] edit  [q] quit  [↑↓/jk] scroll           │
└─────────────────────────────────────────────────────────────┘
```

### Sidebar behaviour

- Default width: 20% of terminal width (min 18 cols, max 30 cols)
- Automatically collapses when terminal width < 80 columns
- Toggled with `b` — viewer expands to fill full width when hidden

### Keybindings

| Key | Action |
|-----|--------|
| `b` | Toggle sidebar |
| `i` | Open current file in `$EDITOR`; re-render on return. No-op if no file is open. |
| `j` / `↓` | Scroll down (viewer) or move cursor down (browser) |
| `k` / `↑` | Scroll up (viewer) or move cursor up (browser) |
| `Enter` | Open selected file from browser |
| `q` / `Ctrl+C` | Quit |

---

## Architecture

### Technology Stack

- **Language:** Go
- **TUI framework:** `github.com/charmbracelet/bubbletea`
- **Markdown renderer:** `github.com/charmbracelet/glamour`
- **Layout & styling:** `github.com/charmbracelet/lipgloss`
- **Primitives:** `github.com/charmbracelet/bubbles` (viewport, list)

### Model Structure

Three bubbletea models composed into a single root `App`:

#### `Browser`
- Owns: directory path, file tree (`.md`/`.mdx` only), cursor position
- On init: walks the filesystem recursively, filters to markdown files
- On `Enter`: emits `FileSelectedMsg{Path: string}` consumed by `App`; focus shifts to Viewer automatically
- Handles `j`/`k`/`↑`/`↓` cursor movement when focused

#### `Viewer`
- Owns: rendered glamour output, scroll offset, current file path
- Listens for `FileSelectedMsg`: reads file from disk, passes content through `glamour.Render()`, updates viewport
- Handles `j`/`k`/`↑`/`↓` scrolling when focused
- On glamour render failure: displays raw markdown text (content is never lost)

#### `App` (root)
- Owns: layout dimensions, sidebar visibility bool, focus state (browser vs viewer)
- Handles global keybindings: `b` (toggle sidebar), `i` (edit), `q` (quit)
- On `i`: issues `tea.ExecProcess` to suspend bubbletea, forks `$EDITOR <path>`, waits for exit, sends `FileSelectedMsg` to trigger Viewer re-render
- Routes terminal resize events to all child models

### Data Flow

```
filesystem
    └─> Browser (walks & filters on init)
            └─> FileSelectedMsg (on Enter)
                    └─> Viewer (reads file, glamour.Render())
                                └─> rendered output (viewport)

[i] keypress
    └─> App suspends bubbletea
            └─> $EDITOR forks (tea.ExecProcess)
                    └─> editor exits
                            └─> App resumes, sends FileSelectedMsg
                                        └─> Viewer re-renders
```

---

## Project Structure

```
terminal-markdown/
├── main.go              # entry point, arg parsing, tea.NewProgram
├── app/
│   ├── app.go           # root App model, layout, global keybindings
│   ├── browser.go       # Browser model, filesystem walk, file list
│   └── viewer.go        # Viewer model, glamour rendering, scroll
├── go.mod
└── go.sum
```

---

## Error Handling

| Scenario | Behaviour |
|----------|-----------|
| No `.md` files in directory | Friendly empty state message in sidebar |
| `$EDITOR` not set | Inline error in status bar; no crash |
| File unreadable | Error message displayed in viewer pane |
| `glamour.Render()` fails | Falls back to raw markdown text |
| Terminal width < 80 cols | Sidebar auto-collapses; viewer goes full-width |

---

## Build & Distribution

```bash
go build -o mdv .
```

Single binary. No runtime dependencies, no config files required.

---

## Future Work (out of scope for v1)

- **v2:** Live reload via `fsnotify` — re-render whenever the open file changes on disk while `$EDITOR` has it open
- **v3:** TOML config file for custom keybindings and glamour theme overrides
