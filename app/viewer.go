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
