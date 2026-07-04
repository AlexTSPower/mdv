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
