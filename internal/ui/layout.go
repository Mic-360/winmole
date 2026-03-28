package ui

type Layout struct {
	Width        int
	Height       int
	SidebarWidth int
	ContentWidth int
	BodyHeight   int
}

func ComputeLayout(width, height int) Layout {
	sidebar := 28
	if width < 110 {
		sidebar = 24
	}
	if width < 88 {
		sidebar = 20
	}
	if sidebar > width/2 {
		sidebar = width / 2
	}
	if sidebar < 18 {
		sidebar = 18
	}
	if sidebar >= width {
		sidebar = width - 1
	}
	contentWidth := width - sidebar
	if contentWidth < 0 {
		contentWidth = 0
	}
	bodyHeight := height - 3
	if bodyHeight < 10 {
		bodyHeight = 10
	}
	return Layout{Width: width, Height: height, SidebarWidth: sidebar, ContentWidth: contentWidth, BodyHeight: bodyHeight}
}
