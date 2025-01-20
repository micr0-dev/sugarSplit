package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sugarSplit/pkg/sugarSplitCore"
)

type resetState int

const (
	noReset resetState = iota
	confirmingReset
)

type tickMsg time.Time

type model struct {
	run           *sugarSplitCore.Run
	width, height int
	resetState    resetState
	filename      string
}

func initialModel(filename string) model {
	state, err := sugarSplitCore.LoadRun(filename)
	if err != nil {
		fmt.Printf("Error loading run: %v\n", err)
		os.Exit(1)
	}

	run, err := sugarSplitCore.NewRun(state, "config.toml")
	if err != nil {
		fmt.Printf("Error creating run: %v\n", err)
		os.Exit(1)
	}

	return model{
		run:        run,
		resetState: noReset,
		filename:   filename,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), tea.EnterAltScreen)
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ### Update ###

func (m model) handleKey(key string) (tea.Model, tea.Cmd) {
	action, exists := m.run.GetAction(key)
	if !exists {
		return m, nil
	}

	switch action {
	case sugarSplitCore.ActionQuit:
		return m, tea.Quit

	case sugarSplitCore.ActionSplit:
		if !m.run.Started {
			m.run.Started = true
			m.run.StartTime = time.Now()
			m.run.CurrentSplit = 0
			m.run.UpdateHotkeyAvailability()
			return m, tick()
		} else if !m.run.Completed {
			if m.run.CurrentSplit < len(m.run.State.Segments.Segments) {
				m.run.Split(m.run.CurrentTime)
				m.run.UpdateHotkeyAvailability()
			}
		}

	case sugarSplitCore.ActionReset:
		if !m.run.ResettingState {
			m.run.ResettingState = true
			m.run.UpdateHotkeyAvailability()
		}

	case sugarSplitCore.ActionConfirm:
		if m.run.ResettingState {
			m.run.Reset()
			return m, nil
		}

	case sugarSplitCore.ActionSaveReset:
		if m.run.ResettingState {
			err := m.run.SaveRun(m.filename)
			if err != nil {
				fmt.Printf("Error saving run: %v\n", err)
			}
			m.run.Reset()
			return m, nil
		}

	case sugarSplitCore.ActionUndo:
		if m.run.Started && m.run.CurrentSplit > 0 && !m.run.Completed {
			m.run.UndoSplit()
			m.run.UpdateHotkeyAvailability()
		} else if m.run.Completed {
			m.run.UndoSplit()
			m.run.StartTime = time.Now().Add(-m.run.CurrentTime)
			m.run.UpdateHotkeyAvailability()
			return m, tick()
		}
	case sugarSplitCore.ActionSkip:
		if m.run.Started && !m.run.Completed {
			m.run.SkipSplit()
			m.run.UpdateHotkeyAvailability()
		}

	case sugarSplitCore.ActionCancel:
		if m.run.ResettingState {
			m.run.ResettingState = false
			m.run.UpdateHotkeyAvailability()
		}
	}

	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg.String())

	case tickMsg:
		if m.run.Started && !m.run.ResettingState {
			m.run.CurrentTime = time.Since(m.run.StartTime)
		}
		return m, tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// ### TUI ###

func (m model) View() string {
	var top, middle, bottom strings.Builder
	styles := initializeStyles(m.width)

	top.WriteString("\n")

	// Define which components go where
	topComponents := []sugarSplitCore.UIComponent{
		sugarSplitCore.UIHeader,
	}

	middleComponents := []sugarSplitCore.UIComponent{
		sugarSplitCore.UISplits,
	}

	bottomComponents := []sugarSplitCore.UIComponent{
		sugarSplitCore.UITimer,
		sugarSplitCore.UIPreviousSegment,
		sugarSplitCore.UIControls,
	}

	// Prepare all possible components
	components := map[sugarSplitCore.UIComponent]func() string{
		sugarSplitCore.UIHeader:          func() string { return m.renderHeader(styles) },
		sugarSplitCore.UISplits:          func() string { return m.renderSplits(styles) },
		sugarSplitCore.UITimer:           func() string { return m.renderTimer(styles) },
		sugarSplitCore.UIPreviousSegment: func() string { return m.renderPreviousSegment(styles) },
		sugarSplitCore.UIControls:        func() string { return m.renderControls(styles) },
	}

	// Helper function to render components in order
	renderComponents := func(componentList []sugarSplitCore.UIComponent, builder *strings.Builder) {
		for _, component := range m.run.UIConfig.Layout {
			if contains(componentList, component) {
				if renderFunc, exists := components[component]; exists {
					builder.WriteString(renderFunc())
				}
			}
		}
	}

	// Render each section
	renderComponents(topComponents, &top)
	renderComponents(middleComponents, &middle)
	renderComponents(bottomComponents, &bottom)

	// Calculate available space
	topHeight := strings.Count(top.String(), "\n")
	bottomHeight := strings.Count(bottom.String(), "\n")
	availableHeight := m.height - topHeight - bottomHeight

	// If we have middle content, make it fill available space
	middleContent := middle.String()
	if middleContent != "" {
		currentMiddleHeight := strings.Count(middleContent, "\n")
		if currentMiddleHeight < availableHeight {
			padding := availableHeight - currentMiddleHeight
			middle.WriteString(strings.Repeat("\n", padding))
		}
	} else {
		// If no middle content, add padding between top and bottom
		middle.WriteString(strings.Repeat("\n", availableHeight))
	}

	// Combine all sections
	return top.String() + middle.String() + bottom.String()
}

// Helper function to check if a slice contains an element
func contains(slice []sugarSplitCore.UIComponent, item sugarSplitCore.UIComponent) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (m model) renderHeader(styles Styles) string {
	var s strings.Builder
	headerSection := lipgloss.JoinVertical(lipgloss.Center,
		styles.title.Render(m.run.State.GameName),
		styles.title.Render(m.run.State.CategoryName),
	)

	if len(m.run.State.Segments.Segments) > 0 {
		sumOfBest := sugarSplitCore.GetSumOfBest(m.run.State.Segments.Segments)
		if sumOfBest > 0 {
			headerSection = lipgloss.JoinVertical(lipgloss.Center,
				headerSection,
				styles.title.Render(fmt.Sprintf("Sum of Best: %s", sugarSplitCore.FormatDuration(sumOfBest))),
			)
		}
	}

	s.WriteString(headerSection)
	s.WriteString("\n\n")
	return s.String()
}

func (m model) renderSplits(styles Styles) string {
	var s strings.Builder

	// Calculate space for splits section
	reservedSpace := 8 // Adjust based on your needs
	maxSplits := m.height - reservedSpace

	// Segments section (scrolling if needed)
	visibleSplits := len(m.run.State.Segments.Segments)
	if visibleSplits > maxSplits {
		// If we have more splits than space, show a window around the current split
		windowSize := maxSplits / 2
		startIdx := m.run.CurrentSplit - windowSize
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + maxSplits
		if endIdx > len(m.run.State.Segments.Segments) {
			endIdx = len(m.run.State.Segments.Segments)
			startIdx = endIdx - maxSplits
			if startIdx < 0 {
				startIdx = 0
			}
		}
		visibleSplits = endIdx - startIdx
	}

	// Splits rendering
	for i, segment := range m.run.State.Segments.Segments {
		var segmentText string
		pbTime := sugarSplitCore.ParseTime(segment.SplitTimes.SplitTime[0].RealTime)
		pbTimeStr := sugarSplitCore.FormatDuration(pbTime)

		// Left-aligned name, right-aligned times
		if i < m.run.CurrentSplit {
			var splitTime string
			if m.run.Splits[i] == 0 {
				splitTime = "-" // Show dash for skipped splits
			} else {
				splitTime = sugarSplitCore.FormatDuration(m.run.Splits[i])
			}

			var diffText string
			if m.run.Splits[i] == 0 {
				diffText = "-"
			} else {
				diff := m.run.Comparison[i]
				if diff < 0 {
					diffText = styles.ahead.Render(fmt.Sprintf("-%v", sugarSplitCore.FormatDuration(-diff)))
				} else {
					diffText = styles.behind.Render(fmt.Sprintf("+%v", sugarSplitCore.FormatDuration(diff)))
				}
			}

			nameWidth := m.width - 32 // Adjust based on your time format width
			segmentText = fmt.Sprintf("%-*s %15s %15s", nameWidth, segment.Name, splitTime, diffText)

			if m.run.IsGold[i] {
				segmentText = styles.gold.Render(segmentText)
			}
			s.WriteString(styles.segment.Render(segmentText))
		} else if i == m.run.CurrentSplit {
			nameWidth := m.width - 16
			segmentText = fmt.Sprintf("%-*s %15s", nameWidth, segment.Name, pbTimeStr)
			s.WriteString(styles.currentSegment.Render(segmentText))
		} else {
			nameWidth := m.width - 16
			segmentText = fmt.Sprintf("%-*s %15s", nameWidth, segment.Name, styles.pb.Render(pbTimeStr))
			s.WriteString(styles.segment.Render(segmentText))
		}
		s.WriteString("\n")
	}

	return s.String()
}

func (m model) renderTimer(styles Styles) string {
	var s strings.Builder

	timerStyle := styles.timer

	if m.run.Completed {
		timerStyle = timerStyle.Foreground(lipgloss.Color("205"))
	} else if m.run.CurrentSplit > 0 && m.run.Comparison[m.run.CurrentSplit-1] < 0 {
		timerStyle = timerStyle.Foreground(lipgloss.Color("82"))
	} else if m.run.CurrentSplit > 0 {
		timerStyle = timerStyle.Foreground(lipgloss.Color("196"))
	} else {
		timerStyle = timerStyle.Foreground(lipgloss.Color("205"))
	}

	bigTimer := getBigTimer(m.run.CurrentTime)
	for _, line := range bigTimer {
		s.WriteString(timerStyle.Render(line))
		s.WriteString("\n")
	}

	return s.String()
}

func (m model) renderPreviousSegment(styles Styles) string {
	var s strings.Builder

	if m.run.CurrentSplit > 0 {
		prevIndex := m.run.CurrentSplit - 1
		segmentTime := m.run.GetSegmentTime(prevIndex)
		pbSegmentTime := m.run.GetPBSegmentTime(prevIndex)

		if segmentTime > 0 && pbSegmentTime > 0 {
			diff := segmentTime - pbSegmentTime
			var diffText string

			if diff < 0 {
				diffText = styles.ahead.Render(fmt.Sprintf("-%v", sugarSplitCore.FormatDuration(-diff)))
			} else {
				diffText = styles.behind.Render(fmt.Sprintf("+%v", sugarSplitCore.FormatDuration(diff)))
			}

			s.WriteString(styles.segment.Render(fmt.Sprintf("Previous Segment: %s", diffText)))
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (m model) renderControls(styles Styles) string {
	var s strings.Builder

	// Controls
	s.WriteString(styles.controls.Render(m.run.GetAvailableHotkeys()))

	return s.String()
}

// Styles struct to keep all styles together
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
			Foreground(lipgloss.Color("205")).
			Align(lipgloss.Center).
			Width(fullWidth),
		segment: lipgloss.NewStyle().
			Width(fullWidth).
			Padding(0, 1),
		currentSegment: lipgloss.NewStyle().
			Width(fullWidth).
			Padding(0, 1).
			Background(lipgloss.Color("17")),
		ahead:  lipgloss.NewStyle().Foreground(lipgloss.Color("82")),
		behind: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		gold:   lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		pb:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		timer: lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center),
		controls: lipgloss.NewStyle().
			Width(fullWidth).
			Align(lipgloss.Center),
	}
}

// ### Utility ###

var bigNumbers = [][]string{
	{ // 0
		"█▀█",
		"█ █",
		"▀▀▀",
	},
	{ // 1
		"▄█ ",
		" █ ",
		"▀▀▀",
	},
	{ // 2
		"▀▀█",
		"█▀▀",
		"▀▀▀",
	},
	{ // 3
		"▀▀█",
		"▀▀█",
		"▀▀▀",
	},
	{ // 4
		"█ █",
		"▀▀█",
		"  ▀",
	},
	{ // 5
		"█▀▀",
		"▀▀█",
		"▀▀▀",
	},
	{ // 6
		"█▀▀",
		"█▀█",
		"▀▀▀",
	},
	{ // 7
		"█▀█",
		"  █",
		"  ▀",
	},
	{ // 8
		"█▀█",
		"█▀█",
		"▀▀▀",
	},
	{ // 9
		"█▀█",
		"▀▀█",
		"  ▀",
	},
}

func getBigNumber(n int) []string {
	if n < 0 || n > 9 {
		return []string{"   ", "   ", "   "}
	}
	return bigNumbers[n]
}

func getBigColon() []string {
	return []string{
		" ▀ ",
		"   ",
		" ▀ ",
	}
}

func getBigDot() []string {
	return []string{
		"   ",
		"   ",
		" ▀ ",
	}
}

func getBigTimer(d time.Duration) []string {
	d = d.Round(time.Millisecond)
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000 / 10 // Get centiseconds (2 digits)

	min1 := minutes / 10
	min2 := minutes % 10
	sec1 := seconds / 10
	sec2 := seconds % 10
	ms1 := milliseconds / 10
	ms2 := milliseconds % 10

	result := make([]string, 3)

	// Combine each row
	for i := 0; i < 3; i++ {
		result[i] = getBigNumber(min1)[i] + "  " + getBigNumber(min2)[i] +
			getBigColon()[i] +
			getBigNumber(sec1)[i] + "  " + getBigNumber(sec2)[i] +
			getBigDot()[i] +
			getBigNumber(ms1)[i] + "  " + getBigNumber(ms2)[i]
	}

	return result
}

// ### Main ###

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: livesplit <filename.lss>")
		os.Exit(1)
	}

	m := initialModel(os.Args[1])
	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
