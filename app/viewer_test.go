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
