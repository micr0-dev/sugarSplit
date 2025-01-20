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

func (m model) View() string {
	var s strings.Builder

	// Use full terminal width for styles
	fullWidth := m.width
	if fullWidth < 40 { // minimum width
		fullWidth = 40
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Align(lipgloss.Center).
		Width(fullWidth)

	segmentStyle := lipgloss.NewStyle().
		Width(fullWidth).
		Padding(0, 1)

	currentSegmentStyle := lipgloss.NewStyle().
		Width(fullWidth).
		Padding(0, 1).
		Background(lipgloss.Color("17"))

	// Color styles remain the same
	aheadStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	behindStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	goldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	pbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Header section (top of screen)
	headerSection := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(m.run.State.GameName),
		titleStyle.Render(m.run.State.CategoryName),
	)

	if len(m.run.State.Segments.Segments) > 0 {
		sumOfBest := sugarSplitCore.GetSumOfBest(m.run.State.Segments.Segments)
		if sumOfBest > 0 {
			headerSection = lipgloss.JoinVertical(lipgloss.Center,
				headerSection,
				titleStyle.Render(fmt.Sprintf("Sum of Best: %s", sugarSplitCore.FormatDuration(sumOfBest))),
			)
		}
	}

	s.WriteString(headerSection)
	s.WriteString("\n\n")

	// Calculate space for splits section
	reservedSpace := 8
	maxSplits := m.height - reservedSpace

	// Segments section (scrolling if needed)
	visibleSplits := len(m.run.State.Segments.Segments)
	if visibleSplits > maxSplits {
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
			splitTime := sugarSplitCore.FormatDuration(m.run.Splits[i])
			diff := m.run.Comparison[i]

			diffText := ""
			if diff < 0 {
				diffText = aheadStyle.Render(fmt.Sprintf("-%v", sugarSplitCore.FormatDuration(-diff)))
			} else {
				diffText = behindStyle.Render(fmt.Sprintf("+%v", sugarSplitCore.FormatDuration(diff)))
			}

			nameWidth := fullWidth - 32
			segmentText = fmt.Sprintf("%-*s %15s %15s", nameWidth, segment.Name, splitTime, diffText)

			if m.run.IsGold[i] {
				segmentText = goldStyle.Render(segmentText)
			}
			s.WriteString(segmentStyle.Render(segmentText))
		} else if i == m.run.CurrentSplit {
			nameWidth := fullWidth - 16
			segmentText = fmt.Sprintf("%-*s %15s", nameWidth, segment.Name, pbTimeStr)
			s.WriteString(currentSegmentStyle.Render(segmentText))
		} else {
			nameWidth := fullWidth - 16
			segmentText = fmt.Sprintf("%-*s %15s", nameWidth, segment.Name, pbStyle.Render(pbTimeStr))
			s.WriteString(segmentStyle.Render(segmentText))
		}
		s.WriteString("\n")
	}

	// Calculate remaining space
	remainingSpace := m.height - visibleSplits - reservedSpace
	if remainingSpace > 0 {
		s.WriteString(strings.Repeat("\n", remainingSpace))
	}

	// Timer section (centered)
	timerStyle := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center)

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

	// Previous segment (left-aligned)
	if m.run.CurrentSplit > 0 {
		prevSegStyle := lipgloss.NewStyle().
			Width(fullWidth).
			Padding(0, 1)

		prevIndex := m.run.CurrentSplit - 1
		segmentTime := m.run.GetSegmentTime(prevIndex)
		pbSegmentTime := m.run.GetPBSegmentTime(prevIndex)

		if segmentTime > 0 && pbSegmentTime > 0 {
			diff := segmentTime - pbSegmentTime
			var diffText string

			if diff < 0 {
				diffText = aheadStyle.Render(fmt.Sprintf("-%v", sugarSplitCore.FormatDuration(-diff)))
			} else {
				diffText = behindStyle.Render(fmt.Sprintf("+%v", sugarSplitCore.FormatDuration(diff)))
			}

			s.WriteString(prevSegStyle.Render(fmt.Sprintf("Previous Segment: %s", diffText)))
			s.WriteString("\n")
		}
	}

	// Reset confirmation (centered)
	if m.resetState != noReset {
		confirmStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("236")).
			Padding(1).
			Width(fullWidth).
			Align(lipgloss.Center)

		if m.resetState == confirmingReset {
			s.WriteString(confirmStyle.Render("Reset run? (Y)es, (S)ave and reset, (N)o"))
		}
	}

	// Controls (bottom-aligned, centered)
	controlStyle := lipgloss.NewStyle().
		Width(fullWidth).
		Align(lipgloss.Center)

	controls := m.run.GetAvailableHotkeys()

	s.WriteString(controlStyle.Render(controls))

	return s.String()
}

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
