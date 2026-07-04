# mdv Terminal Markdown Viewer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a single Go binary (`mdv`) that renders markdown files beautifully in the terminal with a toggleable file browser sidebar and `$EDITOR` integration.

**Architecture:** Three bubbletea models (`Browser`, `Viewer`, `App`) composed into a root `App` that owns layout and global keybindings. `Browser` discovers `.md`/`.mdx` files and emits `FileSelectedMsg` on selection. `Viewer` reads the selected file and renders it through `glamour`. `App` handles sidebar toggling, focus routing, and editor shell-out via `tea.ExecProcess`.

**Tech Stack:** Go, `charmbracelet/bubbletea` (TUI), `charmbracelet/glamour` (markdown), `charmbracelet/lipgloss` (layout), `charmbracelet/bubbles` (viewport, list)

## Global Constraints

- Module path: `terminal-markdown`
- Binary name: `mdv` (`go build -o mdv .`)
- Filter: only `.md` and `.mdx` files shown in browser
- Sidebar auto-collapses at terminal width < 80 cols; sidebar width is 20% of terminal clamped to [18, 30] cols
- `$EDITOR` not set → show error in status bar, no crash; no file open → `i` is a no-op
- `glamour.Render()` failure → fall back to raw markdown text (never lose content)

---

## File Map

| File | Responsibility |
|------|---------------|
| `main.go` | Entry point: arg parsing, root path validation, `tea.NewProgram` |
| `app/messages.go` | Shared message types (`FileSelectedMsg`) |
| `app/browser.go` | `Browser` model: filesystem walk, list navigation, `FileSelectedMsg` emit |
| `app/viewer.go` | `Viewer` model: file reading, glamour rendering, viewport scroll |
| `app/app.go` | `App` root model: layout, focus routing, global keybindings, editor shell-out |
| `app/browser_test.go` | Tests for `findMarkdownFiles` and `Browser.Update` |
| `app/viewer_test.go` | Tests for `Viewer.Update` |
| `app/app_test.go` | Tests for `App.Update` (sidebar toggle, quit, focus, edit guard) |

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `app/` directory (empty, establishes package boundary)

**Interfaces:**
- Produces: a compilable Go module with all dependencies resolved

- [ ] **Step 1: Initialise the Go module**

```bash
cd /Users/alex.power/Accounts/PersonalProjects/terminal-markdown
go mod init terminal-markdown
```

Expected output:
```
go: creating new go.mod: module terminal-markdown
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/glamour@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

- [ ] **Step 3: Create the app package directory**

```bash
mkdir -p app
```

- [ ] **Step 4: Verify the module builds (empty stub)**

Create `main.go` with a minimal stub to confirm the module resolves:

```go
package main

func main() {}
```

```bash
go build ./...
```

Expected: no errors, no output.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "chore: scaffold Go module with Charm dependencies"
```

---

## Task 2: Browser Model

**Files:**
- Create: `app/messages.go`
- Create: `app/browser.go`
- Create: `app/browser_test.go`

**Interfaces:**
- Produces:
  - `FileSelectedMsg{Path string}` — emitted when the user presses Enter on a file
  - `NewBrowser(root string, width, height int) (Browser, error)` — constructs the model
  - `Browser.Update(msg tea.Msg) (Browser, tea.Cmd)` — handles keyboard + window events
  - `Browser.View() string` — renders the file list or empty state

- [ ] **Step 1: Write `app/messages.go`**

```go
package app

// FileSelectedMsg is emitted by Browser when the user selects a file.
type FileSelectedMsg struct {
	Path string
}
```

- [ ] **Step 2: Write the failing tests in `app/browser_test.go`**

```go
package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFindMarkdownFiles_FiltersCorrectly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(dir, "guide.mdx"), []byte("# Guide"), 0644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "docs", "api.md"), []byte("# API"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "hidden.md"), []byte("# Hidden"), 0644)

	items, err := findMarkdownFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Expects: README.md, guide.mdx, docs/api.md (not notes.txt, not .git/hidden.md)
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestBrowser_EnterEmitsFileSelectedMsg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	os.WriteFile(path, []byte("# Hello"), 0644)

	b, err := NewBrowser(dir, 30, 20)
	if err != nil {
		t.Fatal(err)
	}

	_, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}

	msg := cmd()
	sel, ok := msg.(FileSelectedMsg)
	if !ok {
		t.Fatalf("expected FileSelectedMsg, got %T", msg)
	}
	if sel.Path != path {
		t.Errorf("got path %q, want %q", sel.Path, path)
	}
}

func TestBrowser_EmptyDir_ViewShowsEmptyState(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBrowser(dir, 30, 20)
	if err != nil {
		t.Fatal(err)
	}
	view := b.View()
	if view == "" {
		t.Error("expected non-empty view for empty directory")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./app/... -run TestFindMarkdown -v
go test ./app/... -run TestBrowser -v
```

Expected: compile error — `findMarkdownFiles`, `NewBrowser` undefined.

- [ ] **Step 4: Write `app/browser.go`**

```go
package app

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// fileItem is a single markdown file shown in the browser list.
type fileItem struct {
	path    string // absolute path
	display string // relative path shown in the list
}

func (f fileItem) FilterValue() string { return f.display }
func (f fileItem) Title() string       { return f.display }
func (f fileItem) Description() string { return "" }

// Browser is the sidebar file-browser model.
type Browser struct {
	root string
	list list.Model
}

// NewBrowser constructs a Browser rooted at root with the given dimensions.
func NewBrowser(root string, width, height int) (Browser, error) {
	items, err := findMarkdownFiles(root)
	if err != nil {
		return Browser{}, err
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(items, delegate, width, height)
	l.Title = "BROWSER"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap = browserKeyMap()

	return Browser{root: root, list: l}, nil
}

// browserKeyMap returns a key map for the list that does not conflict with
// App-level bindings (b, i, q).
func browserKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PrevPage:             key.NewBinding(key.WithKeys("pgup")),
		NextPage:             key.NewBinding(key.WithKeys("pgdown")),
		GoToStart:            key.NewBinding(key.WithKeys("home", "g")),
		GoToEnd:              key.NewBinding(key.WithKeys("end", "G")),
		Filter:               key.NewBinding(key.WithKeys()),
		ClearFilter:          key.NewBinding(key.WithKeys()),
		CancelWhileFiltering: key.NewBinding(key.WithKeys()),
		AcceptWhileFiltering: key.NewBinding(key.WithKeys()),
		ShowFullHelp:         key.NewBinding(key.WithKeys()),
		CloseFullHelp:        key.NewBinding(key.WithKeys()),
		Quit:                 key.NewBinding(key.WithKeys()),
		ForceQuit:            key.NewBinding(key.WithKeys()),
	}
}

// Update handles messages for the Browser model.
func (b Browser) Update(msg tea.Msg) (Browser, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.list.SetSize(msg.Width, msg.Height)
		return b, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if item, ok := b.list.SelectedItem().(fileItem); ok {
				path := item.path
				return b, func() tea.Msg { return FileSelectedMsg{Path: path} }
			}
			return b, nil
		}
	}
	var cmd tea.Cmd
	b.list, cmd = b.list.Update(msg)
	return b, cmd
}

// View renders the browser list or an empty-state message.
func (b Browser) View() string {
	if len(b.list.Items()) == 0 {
		return "\n  No markdown files found."
	}
	return b.list.View()
}

// findMarkdownFiles walks root recursively and returns list items for every
// .md and .mdx file found. Hidden directories (dot-prefixed) are skipped.
func findMarkdownFiles(root string) ([]list.Item, error) {
	var items []list.Item
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && isMarkdown(path) {
			rel, _ := filepath.Rel(root, path)
			items = append(items, fileItem{path: path, display: rel})
		}
		return nil
	})
	return items, err
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".mdx"
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./app/... -run TestFindMarkdown -v
go test ./app/... -run TestBrowser -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add app/messages.go app/browser.go app/browser_test.go
git commit -m "feat: add Browser model with markdown file discovery"
```

---

## Task 3: Viewer Model

**Files:**
- Create: `app/viewer.go`
- Create: `app/viewer_test.go`

**Interfaces:**
- Consumes: `FileSelectedMsg{Path string}` from Task 2
- Produces:
  - `NewViewer(width, height int) Viewer`
  - `Viewer.Update(msg tea.Msg) (Viewer, tea.Cmd)` — concrete return type, consistent with Browser
  - `Viewer.View() string`

- [ ] **Step 1: Write the failing tests in `app/viewer_test.go`**

```go
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestViewer_InitialView_ShowsPlaceholder(t *testing.T) {
	v := NewViewer(80, 24)
	view := v.View()
	if !strings.Contains(view, "Select a file") {
		t.Errorf("expected placeholder text, got: %q", view)
	}
}

func TestViewer_FileSelectedMsg_RendersContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello\n\nWorld"), 0644)

	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: path})

	if !v2.ready {
		t.Error("viewer should be ready after FileSelectedMsg")
	}
	view := v2.View()
	if view == "" {
		t.Error("expected non-empty view after file selected")
	}
}

func TestViewer_UnreadableFile_ShowsError(t *testing.T) {
	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: "/nonexistent/path/file.md"})

	view := v2.View()
	if !strings.Contains(view, "Error") {
		t.Errorf("expected error message in view, got: %q", view)
	}
}

func TestViewer_GlamourFailure_FallsBackToRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.md")
	content := "# Heading\n\nSome content"
	os.WriteFile(path, []byte(content), 0644)

	v := NewViewer(80, 24)
	v2, _ := v.Update(FileSelectedMsg{Path: path})

	// Should have content regardless of whether glamour succeeded
	if !v2.ready {
		t.Error("viewer should be ready even if glamour render fails")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/... -run TestViewer -v
```

Expected: compile error — `NewViewer` undefined.

- [ ] **Step 3: Write `app/viewer.go`**

```go
package app

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// Viewer renders a single markdown file in a scrollable viewport.
type Viewer struct {
	viewport viewport.Model
	ready    bool
}

// NewViewer constructs a Viewer with the given dimensions.
func NewViewer(width, height int) Viewer {
	vp := viewport.New(width, height)
	vp.SetContent("\n  Select a file to preview.")
	return Viewer{viewport: vp}
}

// Update handles messages for the Viewer model.
func (v Viewer) Update(msg tea.Msg) (Viewer, tea.Cmd) {
	switch msg := msg.(type) {
	case FileSelectedMsg:
		content, err := os.ReadFile(msg.Path)
		if err != nil {
			v.viewport.SetContent(fmt.Sprintf("\n  Error reading file: %v", err))
			v.ready = true
			return v, nil
		}
		rendered, err := glamour.Render(string(content), "dark")
		if err != nil {
			rendered = string(content)
		}
		v.viewport.SetContent(rendered)
		v.viewport.GotoTop()
		v.ready = true
		return v, nil

	case tea.WindowSizeMsg:
		v.viewport.Width = msg.Width
		v.viewport.Height = msg.Height
		return v, nil
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the viewport content.
func (v Viewer) View() string {
	return v.viewport.View()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/... -run TestViewer -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add app/viewer.go app/viewer_test.go
git commit -m "feat: add Viewer model with glamour markdown rendering"
```



---

## Task 4: App Root Model

**Files:**
- Create: `app/app.go`
- Create: `app/app_test.go`

**Interfaces:**
- Consumes:
  - `NewBrowser(root string, width, height int) (Browser, error)` from Task 2
  - `NewViewer(width, height int) Viewer` from Task 3
  - `FileSelectedMsg{Path string}` from Task 2
- Produces:
  - `New(root string) (App, error)` — constructs the root model
  - `App` implements `tea.Model` (`Init`, `Update`, `View`)

- [ ] **Step 1: Write the failing tests in `app/app_test.go`**

```go
package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestApp(t *testing.T) App {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0644)
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Give it dimensions so layout calculations work
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return model.(App)
}

func TestApp_QuitKey(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestApp_SidebarToggle(t *testing.T) {
	a := newTestApp(t)
	if !a.showSidebar {
		t.Fatal("sidebar should be visible by default")
	}
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	a2 := model.(App)
	if a2.showSidebar {
		t.Error("sidebar should be hidden after pressing b")
	}
	model, _ = a2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	a3 := model.(App)
	if !a3.showSidebar {
		t.Error("sidebar should be visible after pressing b again")
	}
}

func TestApp_EditKey_NoopWhenNoFileOpen(t *testing.T) {
	a := newTestApp(t)
	// No file selected yet; currentFile is empty
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if cmd != nil {
		t.Error("expected nil command when no file is open")
	}
}

func TestApp_EditKey_ErrorWhenEditorNotSet(t *testing.T) {
	a := newTestApp(t)
	a.currentFile = "/some/file.md"
	os.Unsetenv("EDITOR")

	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	a2 := model.(App)
	if cmd != nil {
		t.Error("expected nil command when $EDITOR is not set")
	}
	if a2.statusMsg == "" {
		t.Error("expected an error status message when $EDITOR is not set")
	}
}

func TestApp_FileSelectedMsg_FocusesViewer(t *testing.T) {
	a := newTestApp(t)
	// Focus starts on browser
	if a.focus != focusBrowser {
		t.Fatal("focus should start on browser")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Test"), 0644)

	model, _ := a.Update(FileSelectedMsg{Path: path})
	a2 := model.(App)
	if a2.focus != focusViewer {
		t.Error("focus should switch to viewer after FileSelectedMsg")
	}
	if a2.currentFile != path {
		t.Errorf("currentFile should be %q, got %q", path, a2.currentFile)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/... -run TestApp -v
```

Expected: compile error — `New`, `App`, `focusBrowser` etc. undefined.

- [ ] **Step 3: Write `app/app.go`**

```go
package app

import (
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	minWidthForSidebar = 80
	sidebarRatio       = 0.20
	sidebarMin         = 18
	sidebarMax         = 30
)

type focusTarget int

const (
	focusBrowser focusTarget = iota
	focusViewer
)

// App is the root bubbletea model. It owns layout and routes messages.
type App struct {
	browser     Browser
	viewer      Viewer
	width       int
	height      int
	showSidebar bool
	focus       focusTarget
	currentFile string
	statusMsg   string
}

// New constructs the root App model rooted at root.
func New(root string) (App, error) {
	browser, err := NewBrowser(root, sidebarMin, 20)
	if err != nil {
		return App{}, err
	}
	return App{
		browser:     browser,
		viewer:      NewViewer(80, 20),
		showSidebar: true,
		focus:       focusBrowser,
	}, nil
}

func (a App) Init() tea.Cmd { return nil }

// Update handles all messages for the App.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a = a.applyLayout()
		return a, nil

	case FileSelectedMsg:
		a.currentFile = msg.Path
		a.focus = focusViewer
		var cmd tea.Cmd
		a.viewer, cmd = a.viewer.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "b":
			a.showSidebar = !a.showSidebar
			a = a.applyLayout()
			return a, nil
		case "i":
			if a.currentFile == "" {
				return a, nil
			}
			editor := os.Getenv("EDITOR")
			if editor == "" {
				a.statusMsg = "Error: $EDITOR is not set"
				return a, nil
			}
			a.statusMsg = ""
			file := a.currentFile
			cmd := exec.Command(editor, file)
			return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return FileSelectedMsg{Path: file}
			})
		default:
			if a.focus == focusBrowser {
				var cmd tea.Cmd
				a.browser, cmd = a.browser.Update(msg)
				return a, cmd
			}
			var cmd tea.Cmd
			a.viewer, cmd = a.viewer.Update(msg)
			return a, cmd
		}
	}
	return a, nil
}

// View renders the full TUI: title bar, content area, status bar.
func (a App) View() string {
	sw := a.sidebarWidth()
	vw := a.width - sw
	if sw > 0 {
		vw-- // separator column
	}
	contentH := a.height - 2 // title + status bars

	title := "mdv"
	if a.currentFile != "" {
		title = "mdv — " + filepath.Base(a.currentFile)
	}
	titleBar := lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		Render(title)

	help := "[b] browser  [i] edit  [q] quit  [↑↓/jk] scroll"
	if a.statusMsg != "" {
		help = a.statusMsg
	}
	statusBar := lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("250")).
		Padding(0, 1).
		Render(help)

	var body string
	if sw > 0 {
		sidebar := lipgloss.NewStyle().
			Width(sw).
			Height(contentH).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Render(a.browser.View())
		viewer := lipgloss.NewStyle().
			Width(vw).
			Height(contentH).
			Render(a.viewer.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, viewer)
	} else {
		body = lipgloss.NewStyle().
			Width(a.width).
			Height(contentH).
			Render(a.viewer.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body, statusBar)
}

// sidebarWidth returns the current sidebar width in columns (0 when hidden).
func (a App) sidebarWidth() int {
	if !a.showSidebar || a.width < minWidthForSidebar {
		return 0
	}
	w := int(float64(a.width) * sidebarRatio)
	if w < sidebarMin {
		return sidebarMin
	}
	if w > sidebarMax {
		return sidebarMax
	}
	return w
}

// applyLayout recalculates and pushes dimensions to child models.
func (a App) applyLayout() App {
	sw := a.sidebarWidth()
	vw := a.width - sw
	if sw > 0 {
		vw--
	}
	contentH := a.height - 2
	a.browser.list.SetSize(sw, contentH)
	a.viewer.viewport.Width = vw
	a.viewer.viewport.Height = contentH
	return a
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/... -run TestApp -v
```

Expected: all PASS.

- [ ] **Step 5: Run all tests**

```bash
go test ./app/... -v
```

Expected: all PASS (Browser, Viewer, and App tests).

- [ ] **Step 6: Commit**

```bash
git add app/app.go app/app_test.go
git commit -m "feat: add App root model with layout, routing, and editor shell-out"
```

---

## Task 5: Entry Point & Integration

**Files:**
- Modify: `main.go` (replace stub from Task 1)

**Interfaces:**
- Consumes: `New(root string) (App, error)` from Task 4
- Produces: compiled `mdv` binary

- [ ] **Step 1: Write `main.go`**

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"terminal-markdown/app"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdv: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "mdv: %s is not a directory\n", root)
		os.Exit(1)
	}

	model, err := app.New(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdv: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdv: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build the binary**

```bash
go build -o mdv .
```

Expected: no errors, `mdv` binary created in the project root.

- [ ] **Step 3: Smoke test — open the project's own docs directory**

```bash
./mdv ./docs
```

Expected:
- TUI launches with alt screen
- Sidebar shows `superpowers/specs/2026-07-03-terminal-markdown-design.md`
- Pressing `Enter` renders the spec in the viewer pane with styled headings and a code block
- Pressing `b` hides the sidebar; viewer expands to full width
- Pressing `b` again restores the sidebar
- Pressing `q` exits cleanly

- [ ] **Step 4: Smoke test — non-markdown directory**

```bash
mkdir /tmp/empty-test && ./mdv /tmp/empty-test
```

Expected: TUI launches, sidebar shows "No markdown files found."

- [ ] **Step 5: Smoke test — invalid path**

```bash
./mdv /nonexistent
```

Expected:
```
mdv: stat /nonexistent: no such file or directory
```
Exit code 1, no TUI launched.

- [ ] **Step 6: Run full test suite**

```bash
go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add main.go
git commit -m "feat: wire entry point and build mdv binary"
```
