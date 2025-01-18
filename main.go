package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type resetState int

const (
	noReset resetState = iota
	confirmingReset
)

// XML structures remain the same
type LiveSplitState struct {
	XMLName              xml.Name       `xml:"Run"`
	GameName             string         `xml:"GameName"`
	CategoryName         string         `xml:"CategoryName"`
	Metadata             Metadata       `xml:"Metadata"`
	Offset               string         `xml:"Offset"`
	AttemptCount         int            `xml:"AttemptCount"`
	AttemptHistory       AttemptHistory `xml:"AttemptHistory"`
	Segments             Segments       `xml:"Segments"`
	AutoSplitterSettings string         `xml:"AutoSplitterSettings,omitempty"`
}

type Metadata struct {
	Run      MetadataRun `xml:"Run"`
	Platform string      `xml:"Platform,omitempty"`
}

type MetadataRun struct {
	Version string `xml:"version,attr"`
}

type AttemptHistory struct {
	Attempt []Attempt `xml:"Attempt"`
}

type Attempt struct {
	ID              string `xml:"id,attr"`
	Started         string `xml:"started,attr"`
	IsStartedSynced string `xml:"isStartedSynced,attr"`
	Ended           string `xml:"ended,attr"`
	IsEndedSynced   string `xml:"isEndedSynced,attr"`
}

type Segments struct {
	Segments []Segment `xml:"Segment"`
}

type Segment struct {
	Name            string          `xml:"Name"`
	Icon            string          `xml:"Icon"`
	SplitTimes      SplitTimes      `xml:"SplitTimes"`
	BestSegmentTime BestSegmentTime `xml:"BestSegmentTime"`
	SegmentHistory  SegmentHistory  `xml:"SegmentHistory"`
}

type SplitTimes struct {
	SplitTime []SplitTime `xml:"SplitTime"`
}

type SplitTime struct {
	Name     string `xml:"name,attr"`
	RealTime string `xml:"RealTime"`
}

type BestSegmentTime struct {
	RealTime string `xml:"RealTime"`
}

type SegmentHistory struct {
	Time []Time `xml:"Time"`
}

type Time struct {
	ID       string `xml:"id,attr"`
	RealTime string `xml:"RealTime"`
}

type tickMsg time.Time

type model struct {
	run           LiveSplitState
	currentSplit  int
	width, height int
	started       bool
	startTime     time.Time
	currentTime   time.Duration
	splits        []time.Duration
	comparison    []time.Duration // Store time differences
	isGold        []bool          // Track if split was gold
	resetState    resetState
	filename      string
	completed     bool
}

func initialModel(filename string) model {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var run LiveSplitState
	err = xml.Unmarshal(data, &run)
	if err != nil {
		fmt.Printf("Error parsing LSS file: %v\n", err)
		os.Exit(1)
	}

	return model{
		run:          run,
		currentSplit: -1,
		started:      false,
		splits:       make([]time.Duration, len(run.Segments.Segments)),
		comparison:   make([]time.Duration, len(run.Segments.Segments)),
		isGold:       make([]bool, len(run.Segments.Segments)),
		resetState:   noReset,
		filename:     filename,
		completed:    false,
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

func calculateDifference(current, pb time.Duration) string {
	diff := current - pb
	if diff < 0 {
		return fmt.Sprintf("-" + formatDuration(diff*-1))
	}
	return fmt.Sprintf("+" + formatDuration(diff))
}

func (m *model) isPB() bool {
	if m.currentSplit != len(m.run.Segments.Segments) {
		return false
	}

	lastSplitTime := m.splits[len(m.splits)-1]
	currentPB := parseTime(m.run.Segments.Segments[len(m.splits)-1].SplitTimes.SplitTime[0].RealTime)

	return lastSplitTime < currentPB || currentPB == 0
}

func (m *model) saveRun() error {
	// Create new attempt
	newAttemptID := len(m.run.AttemptHistory.Attempt) + 1
	now := time.Now()

	attempt := Attempt{
		ID:              fmt.Sprintf("%d", newAttemptID),
		Started:         now.Format("01/02/2006 15:04:05"),
		IsStartedSynced: "True",
		Ended:           now.Format("01/02/2006 15:04:05"),
		IsEndedSynced:   "True",
	}

	m.run.AttemptHistory.Attempt = append(m.run.AttemptHistory.Attempt, attempt)
	m.run.AttemptCount++

	isPB := m.isPB()

	// Update segments
	for i, split := range m.splits {
		if split > 0 { // Only save completed splits
			// Add to segment history
			newTime := Time{
				ID:       fmt.Sprintf("%d", newAttemptID),
				RealTime: formatDurationLSS(split),
			}
			m.run.Segments.Segments[i].SegmentHistory.Time = append(
				m.run.Segments.Segments[i].SegmentHistory.Time,
				newTime,
			)

			// Update best segment time if this was a gold split
			if m.isGold[i] {
				var splitTime time.Duration
				if i == 0 {
					splitTime = split
				} else {
					splitTime = split - m.splits[i-1]
				}
				m.run.Segments.Segments[i].BestSegmentTime.RealTime = formatDurationLSS(splitTime)
			}

			// Update PB split time if this is a PB run
			if isPB {
				// Ensure there's at least one SplitTime
				if len(m.run.Segments.Segments[i].SplitTimes.SplitTime) == 0 {
					m.run.Segments.Segments[i].SplitTimes.SplitTime = append(
						m.run.Segments.Segments[i].SplitTimes.SplitTime,
						SplitTime{Name: "Personal Best"},
					)
				}
				m.run.Segments.Segments[i].SplitTimes.SplitTime[0].RealTime = formatDurationLSS(split)
			}
		}
	}

	// Read existing file content
	existingData, err := ioutil.ReadFile(m.filename)
	if err != nil {
		return err
	}

	// Create temporary struct to preserve any unknown XML elements
	var temp interface{}
	if err := xml.Unmarshal(existingData, &temp); err != nil {
		return err
	}

	// Marshal the updated run
	data, err := xml.MarshalIndent(m.run, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return ioutil.WriteFile(m.filename, data, 0644)
}

// Add function to format duration in LiveSplit format
func formatDurationLSS(d time.Duration) string {
	d = d.Round(time.Millisecond)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	ms := d.Milliseconds()

	return fmt.Sprintf("%02d:%02d:%02d.%07d", h, m, s, ms*10000)
}

func (m model) getSegmentTime(splitIndex int) time.Duration {
	if splitIndex < 0 || splitIndex >= len(m.splits) {
		return 0
	}

	if m.splits[splitIndex] == 0 {
		return 0
	}

	if splitIndex == 0 {
		return m.splits[0]
	}

	return m.splits[splitIndex] - m.splits[splitIndex-1]
}

func (m model) getPBSegmentTime(splitIndex int) time.Duration {
	segment := m.run.Segments.Segments[splitIndex]
	pbTime := parseTime(segment.SplitTimes.SplitTime[0].RealTime)

	if splitIndex == 0 {
		return pbTime
	}

	prevPBTime := parseTime(m.run.Segments.Segments[splitIndex-1].SplitTimes.SplitTime[0].RealTime)
	return pbTime - prevPBTime
}

func (m model) getSumOfBest() time.Duration {
	var sum time.Duration
	for _, segment := range m.run.Segments.Segments {
		if segment.BestSegmentTime.RealTime != "" {
			bestSegTime := parseTime(segment.BestSegmentTime.RealTime)
			sum += bestSegTime
		}
	}
	return sum
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ": // Space key
			if m.completed {
				// Do nothing if run is completed
				return m, nil
			}

			if !m.started {
				m.started = true
				m.startTime = time.Now()
				m.currentSplit = 0
				return m, tick()
			} else {
				if m.currentSplit < len(m.run.Segments.Segments) {
					m.splits[m.currentSplit] = m.currentTime

					// Compare with PB and check for gold
					pbTime := parseTime(m.run.Segments.Segments[m.currentSplit].SplitTimes.SplitTime[0].RealTime)
					goldTime := parseTime(m.run.Segments.Segments[m.currentSplit].BestSegmentTime.RealTime)

					// Calculate split time (time since last split)
					var splitTime time.Duration
					if m.currentSplit == 0 {
						splitTime = m.currentTime
					} else {
						splitTime = m.currentTime - m.splits[m.currentSplit-1]
					}

					// Check if this is a gold split
					m.isGold[m.currentSplit] = splitTime < goldTime || goldTime == 0

					// Calculate comparison to PB
					m.comparison[m.currentSplit] = m.currentTime - pbTime

					m.currentSplit++
					if m.currentSplit >= len(m.run.Segments.Segments) {
						m.started = false
						m.completed = true
					}
				}
			}
		case "r":
			if (m.started || m.completed) && m.resetState == noReset {
				m.resetState = confirmingReset
				return m, nil
			}
		case "y":
			if m.resetState == confirmingReset {
				// Reset without saving
				m.started = false
				m.completed = false
				m.currentSplit = -1
				m.currentTime = 0
				m.splits = make([]time.Duration, len(m.run.Segments.Segments))
				m.comparison = make([]time.Duration, len(m.run.Segments.Segments))
				m.isGold = make([]bool, len(m.run.Segments.Segments))
				m.resetState = noReset
			}
		case "s":
			if m.resetState == confirmingReset {
				// Save and reset
				err := m.saveRun()
				if err != nil {
					// Handle error (you might want to show this in the UI)
					fmt.Printf("Error saving run: %v\n", err)
				}
				m.started = false
				m.completed = false
				m.currentSplit = -1
				m.currentTime = 0
				m.splits = make([]time.Duration, len(m.run.Segments.Segments))
				m.comparison = make([]time.Duration, len(m.run.Segments.Segments))
				m.isGold = make([]bool, len(m.run.Segments.Segments))
				m.resetState = noReset
			}
		case "z":
			if m.started && m.currentSplit > 0 && !m.completed {
				m.currentSplit--
				// Clear the split time and comparison for the undone split
				m.splits[m.currentSplit] = 0
				m.comparison[m.currentSplit] = 0
				m.isGold[m.currentSplit] = false
			} else if m.completed {
				// If the run was completed, undo the last split and resume the run
				m.currentSplit--
				m.splits[m.currentSplit] = 0
				m.comparison[m.currentSplit] = 0
				m.isGold[m.currentSplit] = false
				m.completed = false
				m.started = true
				m.startTime = time.Now().Add(-m.currentTime)
				return m, tick()
			}

		case "n", "esc":
			if m.resetState != noReset {
				m.resetState = noReset
			}
		}
	case tickMsg:
		if m.started {
			m.currentTime = time.Since(m.startTime)
		}
		return m, tick()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	ms := d / time.Millisecond

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
	}
	return fmt.Sprintf("%02d:%02d.%03d", m, s, ms)
}

func parseTime(timeStr string) time.Duration {
	if timeStr == "" {
		return 0
	}

	// Remove potential "00:" prefix
	timeStr = strings.TrimPrefix(timeStr, "00:")

	// Parse hours, minutes, seconds, and fractional seconds
	var hours, minutes, seconds int
	var fraction float64

	parts := strings.Split(timeStr, ":")
	if len(parts) == 3 {
		fmt.Sscanf(parts[0], "%d", &hours)
		fmt.Sscanf(parts[1], "%d", &minutes)
		fmt.Sscanf(parts[2], "%f", &fraction)
		seconds = int(fraction)
		fraction = fraction - float64(seconds)
	} else if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &minutes)
		fmt.Sscanf(parts[1], "%f", &fraction)
		seconds = int(fraction)
		fraction = fraction - float64(seconds)
	}

	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(fraction*float64(time.Second))

	return duration
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
		titleStyle.Render(m.run.GameName),
		titleStyle.Render(m.run.CategoryName),
	)

	if len(m.run.Segments.Segments) > 0 {
		sumOfBest := m.getSumOfBest()
		if sumOfBest > 0 {
			headerSection = lipgloss.JoinVertical(lipgloss.Center,
				headerSection,
				titleStyle.Render(fmt.Sprintf("Sum of Best: %s", formatDuration(sumOfBest))),
			)
		}
	}

	s.WriteString(headerSection)
	s.WriteString("\n\n")

	// Calculate space for splits section
	// Reserve space for header, timer, previous segment, and controls
	reservedSpace := 8 // Adjust based on your needs
	maxSplits := m.height - reservedSpace

	// Segments section (scrolling if needed)
	visibleSplits := len(m.run.Segments.Segments)
	if visibleSplits > maxSplits {
		// If we have more splits than space, show a window around the current split
		windowSize := maxSplits / 2
		startIdx := m.currentSplit - windowSize
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + maxSplits
		if endIdx > len(m.run.Segments.Segments) {
			endIdx = len(m.run.Segments.Segments)
			startIdx = endIdx - maxSplits
			if startIdx < 0 {
				startIdx = 0
			}
		}
		visibleSplits = endIdx - startIdx
	}

	// Splits rendering
	for i, segment := range m.run.Segments.Segments {
		var segmentText string
		pbTime := parseTime(segment.SplitTimes.SplitTime[0].RealTime)
		pbTimeStr := formatDuration(pbTime)

		// Left-aligned name, right-aligned times
		if i < m.currentSplit {
			splitTime := formatDuration(m.splits[i])
			diff := m.comparison[i]

			diffText := ""
			if diff < 0 {
				diffText = aheadStyle.Render(fmt.Sprintf("-%v", formatDuration(-diff)))
			} else {
				diffText = behindStyle.Render(fmt.Sprintf("+%v", formatDuration(diff)))
			}

			nameWidth := fullWidth - 32 // Adjust based on your time format width
			segmentText = fmt.Sprintf("%-*s %15s %15s", nameWidth, segment.Name, splitTime, diffText)

			if m.isGold[i] {
				segmentText = goldStyle.Render(segmentText)
			}
			s.WriteString(segmentStyle.Render(segmentText))
		} else if i == m.currentSplit {
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

	if m.completed {
		timerStyle = timerStyle.Foreground(lipgloss.Color("205"))
	} else if m.currentSplit > 0 && m.comparison[m.currentSplit-1] < 0 {
		timerStyle = timerStyle.Foreground(lipgloss.Color("82"))
	} else if m.currentSplit > 0 {
		timerStyle = timerStyle.Foreground(lipgloss.Color("196"))
	} else {
		timerStyle = timerStyle.Foreground(lipgloss.Color("205"))
	}

	bigTimer := getBigTimer(m.currentTime)
	for _, line := range bigTimer {
		s.WriteString(timerStyle.Render(line))
		s.WriteString("\n")
	}

	// Previous segment (left-aligned)
	if m.currentSplit > 0 {
		prevSegStyle := lipgloss.NewStyle().
			Width(fullWidth).
			Padding(0, 1)

		prevIndex := m.currentSplit - 1
		segmentTime := m.getSegmentTime(prevIndex)
		pbSegmentTime := m.getPBSegmentTime(prevIndex)

		if segmentTime > 0 && pbSegmentTime > 0 {
			diff := segmentTime - pbSegmentTime
			var diffText string

			if diff < 0 {
				diffText = aheadStyle.Render(fmt.Sprintf("-%v", formatDuration(-diff)))
			} else {
				diffText = behindStyle.Render(fmt.Sprintf("+%v", formatDuration(diff)))
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

	controls := "Space: Start/Split  R: Reset  Q: Quit"
	if m.started {
		controls = "Space: Split  Z: Undo  R: Reset  Q: Quit"
	}
	if m.completed {
		controls = "R: Reset  Q: Quit"
	}
	if m.resetState != noReset {
		controls = "Y: Confirm  N/ESC: Cancel  S: Save and Reset"
	}

	s.WriteString(controlStyle.Render(controls))

	return s.String()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: livesplit <filename.lss>")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(os.Args[1]))
	if err := p.Start(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
