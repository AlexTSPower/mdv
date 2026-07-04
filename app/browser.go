package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type itemKind int

const (
	kindFile   itemKind = iota
	kindDir
	kindParent
)

// browserItem is one entry in the browser list: a file, a directory, or "..".
type browserItem struct {
	kind    itemKind
	absPath string
	display string
}

func (b browserItem) FilterValue() string { return b.display }
func (b browserItem) Title() string       { return b.display }
func (b browserItem) Description() string { return "" }

// Browser is the sidebar file-browser model.
type Browser struct {
	root       string
	currentDir string
	mdFiles    map[string][]string // dir -> []absolute file paths in that dir
	mdDirs     map[string]bool     // dirs that contain markdown anywhere in subtree
	list       list.Model
}

// NewBrowser constructs a Browser rooted at root with the given dimensions.
func NewBrowser(root string, width, height int) (Browser, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Browser{}, err
	}

	mdFiles, mdDirs, err := scanMarkdown(absRoot)
	if err != nil {
		return Browser{}, err
	}

	b := Browser{
		root:       absRoot,
		currentDir: absRoot,
		mdFiles:    mdFiles,
		mdDirs:     mdDirs,
	}

	items := b.buildItems(absRoot)

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(items, delegate, width, height)
	l.Title = "BROWSER"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap = browserKeyMap()

	b.list = l
	return b, nil
}

// buildItems returns the list items for a given directory: ".." (if not root),
// subdirectories that contain markdown, then markdown files in that dir.
func (b Browser) buildItems(dir string) []list.Item {
	var items []list.Item

	if dir != b.root {
		items = append(items, browserItem{
			kind:    kindParent,
			absPath: filepath.Dir(dir),
			display: "..",
		})
	}

	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		childPath := filepath.Join(dir, entry.Name())
		if b.mdDirs[childPath] {
			items = append(items, browserItem{
				kind:    kindDir,
				absPath: childPath,
				display: "▸ " + entry.Name() + "/",
			})
		}
	}

	for _, f := range b.mdFiles[dir] {
		items = append(items, browserItem{
			kind:    kindFile,
			absPath: f,
			display: filepath.Base(f),
		})
	}

	return items
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
			item, ok := b.list.SelectedItem().(browserItem)
			if !ok {
				return b, nil
			}
			switch item.kind {
			case kindFile:
				path := item.absPath
				return b, func() tea.Msg { return FileSelectedMsg{Path: path} }
			case kindDir, kindParent:
				b.currentDir = item.absPath
				b.list.SetItems(b.buildItems(item.absPath))
				return b, nil
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
	if len(b.mdFiles) == 0 {
		return "\n  No markdown files found."
	}
	return b.list.View()
}

// scanMarkdown walks root once and returns:
// - mdFiles: map from directory to the markdown files directly in that directory
// - mdDirs: set of directories that contain markdown anywhere in their subtree
func scanMarkdown(root string) (map[string][]string, map[string]bool, error) {
	mdFiles := make(map[string][]string)
	mdDirs := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != root && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && isMarkdown(path) {
			dir := filepath.Dir(path)
			mdFiles[dir] = append(mdFiles[dir], path)
			// Mark all ancestor directories up to root as containing markdown.
			for p := dir; ; p = filepath.Dir(p) {
				if mdDirs[p] {
					break // ancestors already marked from a previous file
				}
				mdDirs[p] = true
				if p == root {
					break
				}
			}
		}
		return nil
	})
	return mdFiles, mdDirs, err
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".mdx"
}
