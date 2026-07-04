package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestScanMarkdown_FiltersCorrectly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(dir, "guide.mdx"), []byte("# Guide"), 0644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "docs", "api.md"), []byte("# API"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "hidden.md"), []byte("# Hidden"), 0644)

	mdFiles, mdDirs, err := scanMarkdown(dir)
	if err != nil {
		t.Fatal(err)
	}

	total := 0
	for _, files := range mdFiles {
		total += len(files)
	}
	// README.md, guide.mdx, docs/api.md — not notes.txt, not .git/hidden.md
	if total != 3 {
		t.Errorf("got %d files, want 3", total)
	}
	if !mdDirs[filepath.Join(dir, "docs")] {
		t.Error("docs/ should be in mdDirs")
	}
	if mdDirs[filepath.Join(dir, ".git")] {
		t.Error(".git/ should not be in mdDirs")
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

func TestBrowser_EnterOnDirNavigatesInto(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("# Guide"), 0644)

	b, err := NewBrowser(dir, 30, 20)
	if err != nil {
		t.Fatal(err)
	}

	// First item should be the "docs/" directory entry (no files at root level)
	item, ok := b.list.SelectedItem().(browserItem)
	if !ok || item.kind != kindDir {
		t.Fatalf("expected first item to be a directory, got kind=%v display=%q", item.kind, item.display)
	}

	// Enter navigates into docs/ — no FileSelectedMsg, just navigation
	b, cmd := b.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd when navigating into a directory")
	}
	if b.currentDir != filepath.Join(dir, "docs") {
		t.Errorf("currentDir = %q, want %q", b.currentDir, filepath.Join(dir, "docs"))
	}

	// First item in docs/ should be ".." (kindParent)
	item, ok = b.list.SelectedItem().(browserItem)
	if !ok || item.kind != kindParent {
		t.Fatalf("expected first item to be parent (..), got kind=%v display=%q", item.kind, item.display)
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
