package main

import "github.com/charmbracelet/lipgloss"

// Color constants
const (
	ColorPrimary     = lipgloss.Color("205") // Pink/magenta
	ColorAhead       = lipgloss.Color("82")  // Green
	ColorBehind      = lipgloss.Color("196") // Red
	ColorGold        = lipgloss.Color("220") // Gold/yellow
	ColorMuted       = lipgloss.Color("244") // Gray
	ColorHighlightBg = lipgloss.Color("17")  // Dark blue
)

// Styles holds all UI styles
type Styles struct {
	title          lipgloss.Style
	segment        lipgloss.Style
	currentSegment lipgloss.Style
	ahead          lipgloss.Style
	behind         lipgloss.Style
	gold           lipgloss.Style
	pb             lipgloss.Style
	timer          lipgloss.Style
	controls       lipgloss.Style
}

func initializeStyles(width int) Styles {
	fullWidth := width
	if fullWidth < 40 {
		fullWidth = 40
	}

	return Styles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Align(lipgloss.Center).
			Width(fullWidth),
		segment: lipgloss.NewStyle().
			Width(fullWidth).
			Padding(0, 1),
		currentSegment: lipgloss.NewStyle().
			Width(fullWidth).
			Padding(0, 1).
			Background(ColorHighlightBg),
		ahead:  lipgloss.NewStyle().Foreground(ColorAhead),
		behind: lipgloss.NewStyle().Foreground(ColorBehind),
		gold:   lipgloss.NewStyle().Foreground(ColorGold),
		pb:     lipgloss.NewStyle().Foreground(ColorMuted),
		timer: lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center),
		controls: lipgloss.NewStyle().
			Width(fullWidth).
			Align(lipgloss.Center),
	}
}
