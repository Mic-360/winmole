package screens

import (
	"fmt"
	"strings"

	"github.com/mic-360/wimo/internal/state"
	"github.com/mic-360/wimo/internal/ui"
)

func Logs(theme ui.Theme, store state.Store, viewport string, width int) string {
	header := strings.Join([]string{
		theme.Header.Render("Runtime logs"),
		theme.MutedText.Render("Filter: " + blankIfEmpty(store.LogQuery, "all events")),
		theme.MutedText.Render(fmt.Sprintf("Tail: %v", !store.LogPaused)),
	}, "  ·  ")
	return ui.Card(theme, "Logs", header+"\n\n"+viewport, width)
}

func blankIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
