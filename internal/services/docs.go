package services

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/glamour"

	"github.com/mic-360/wimo/internal/state"
)

func LoadHelpDocs() ([]state.HelpDoc, error) {
	paths := []string{
		filepath.Join("docs", "KEYBINDINGS.md"),
		filepath.Join("docs", "TUI_ARCHITECTURE.md"),
		filepath.Join("docs", "UI_COMPONENTS.md"),
		filepath.Join("docs", "DEVELOPER_GUIDE.md"),
	}
	renderer, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(82))
	docs := make([]state.HelpDoc, 0, len(paths))
	for _, path := range paths {
		payload, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		payload = stripUTF8BOM(payload)
		rendered, _ := renderer.Render(string(payload))
		docs = append(docs, state.HelpDoc{ID: filepath.Base(path), Title: trimExt(filepath.Base(path)), Path: path, Content: rendered})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Title < docs[j].Title })
	return docs, nil
}

func trimExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}
