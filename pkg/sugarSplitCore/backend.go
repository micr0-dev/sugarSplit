package sugarSplitCore

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

// XML structures
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

// Run represents the current state of a run
type Run struct {
	State          *LiveSplitState
	CurrentSplit   int
	Splits         []time.Duration
	IsGold         []bool
	Comparison     []time.Duration
	StartTime      time.Time
	CurrentTime    time.Duration
	Started        bool
	Completed      bool
	ResettingState bool
	Hotkeys        []Hotkey
	UIConfig       *UIConfig
}

// ### Core Splitter functions ###

// NewRun creates a new Run instance from a LiveSplitState
func NewRun(state *LiveSplitState, configPath string) (*Run, error) {
	hotkeys, err := LoadHotkeys(configPath)
	if err != nil {
		return nil, err
	}

	uiConfig, err := LoadUIConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading UI config: %v", err)
	}

	run := &Run{
		State:        state,
		CurrentSplit: -1,
		Splits:       make([]time.Duration, len(state.Segments.Segments)),
		IsGold:       make([]bool, len(state.Segments.Segments)),
		Comparison:   make([]time.Duration, len(state.Segments.Segments)),
		Started:      false,
		Completed:    false,
		Hotkeys:      hotkeys,
		UIConfig:     uiConfig,
	}

	run.UpdateHotkeyAvailability()
	return run, nil
}

// SaveRun saves the current run state to file
func (r *Run) SaveRun(filename string) error {
	// Create new attempt
	newAttemptID := len(r.State.AttemptHistory.Attempt) + 1
	now := time.Now()

	attempt := Attempt{
		ID:              fmt.Sprintf("%d", newAttemptID),
		Started:         now.Format("01/02/2006 15:04:05"),
		IsStartedSynced: "True",
		Ended:           now.Format("01/02/2006 15:04:05"),
		IsEndedSynced:   "True",
	}

	r.State.AttemptHistory.Attempt = append(r.State.AttemptHistory.Attempt, attempt)
	r.State.AttemptCount++

	isPB := r.IsPB()

	// Update segments
	for i, split := range r.Splits {
		if split > 0 {
			// Add to segment history
			newTime := Time{
				ID:       fmt.Sprintf("%d", newAttemptID),
				RealTime: formatDurationLSS(split),
			}
			r.State.Segments.Segments[i].SegmentHistory.Time = append(
				r.State.Segments.Segments[i].SegmentHistory.Time,
				newTime,
			)

			// Update best segment time if this was a gold split
			if r.IsGold[i] {
				var splitTime time.Duration
				if i == 0 {
					splitTime = split
				} else {
					splitTime = split - r.Splits[i-1]
				}
				r.State.Segments.Segments[i].BestSegmentTime.RealTime = formatDurationLSS(splitTime)
			}

			// Update PB split time if this is a PB run
			if isPB {
				if len(r.State.Segments.Segments[i].SplitTimes.SplitTime) == 0 {
					r.State.Segments.Segments[i].SplitTimes.SplitTime = append(
						r.State.Segments.Segments[i].SplitTimes.SplitTime,
						SplitTime{Name: "Personal Best"},
					)
				}
				r.State.Segments.Segments[i].SplitTimes.SplitTime[0].RealTime = formatDurationLSS(split)
			}
		}
	}

	return SaveRun(r.State, filename)
}

// Split performs a split operation and updates relevant state
func (r *Run) Split(currentTime time.Duration) {
	if r.CurrentSplit >= len(r.State.Segments.Segments) {
		return
	}

	r.Splits[r.CurrentSplit] = currentTime

	// Compare with PB and check for gold
	pbTime := ParseTime(r.State.Segments.Segments[r.CurrentSplit].SplitTimes.SplitTime[0].RealTime)
	goldTime := ParseTime(r.State.Segments.Segments[r.CurrentSplit].BestSegmentTime.RealTime)

	// Calculate split time (time since last split)
	var splitTime time.Duration
	if r.CurrentSplit == 0 {
		splitTime = currentTime
	} else {
		splitTime = currentTime - r.Splits[r.CurrentSplit-1]
	}

	// Check if this is a gold split
	r.IsGold[r.CurrentSplit] = splitTime < goldTime || goldTime == 0

	// Calculate comparison to PB
	r.Comparison[r.CurrentSplit] = currentTime - pbTime

	r.CurrentSplit++
	if r.CurrentSplit >= len(r.State.Segments.Segments) {
		r.Started = false
		r.Completed = true
	}
}

// UndoSplit reverses the last split
func (r *Run) UndoSplit() {
	if r.CurrentSplit > 0 {
		r.CurrentSplit--
		r.Splits[r.CurrentSplit] = 0
		r.Comparison[r.CurrentSplit] = 0
		r.IsGold[r.CurrentSplit] = false
		if r.Completed {
			r.Completed = false
			r.Started = true
		}
	}
}

// Skip skips the current split
func (r *Run) SkipSplit() {
	if !r.Started || r.Completed || r.CurrentSplit >= len(r.State.Segments.Segments) {
		return
	}

	// Set current split as skipped (we'll use 0 duration to indicate a skip)
	r.Splits[r.CurrentSplit] = 0
	r.Comparison[r.CurrentSplit] = 0
	r.IsGold[r.CurrentSplit] = false

	r.CurrentSplit++
	if r.CurrentSplit >= len(r.State.Segments.Segments) {
		r.Completed = true
		r.Started = false
	}

	r.UpdateHotkeyAvailability()
}

// Reset resets the run state
func (r *Run) Reset() {
	r.Started = false
	r.Completed = false
	r.CurrentSplit = -1
	r.CurrentTime = 0
	r.StartTime = time.Time{}
	r.Splits = make([]time.Duration, len(r.State.Segments.Segments))
	r.Comparison = make([]time.Duration, len(r.State.Segments.Segments))
	r.IsGold = make([]bool, len(r.State.Segments.Segments))
	r.ResettingState = false
	r.UpdateHotkeyAvailability()
}

// Helper functions
func LoadRun(filename string) (*LiveSplitState, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var run LiveSplitState
	err = xml.Unmarshal(data, &run)
	if err != nil {
		return nil, fmt.Errorf("error parsing LSS file: %v", err)
	}

	return &run, nil
}

func SaveRun(run *LiveSplitState, filename string) error {
	data, err := xml.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// ### End of Core Splitter functions ###

// ### Utility functions ###

// FormatDuration formats a time.Duration to a human-readable
func FormatDuration(d time.Duration) string {
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

// formatDurationLSS formats a time.Duration to a LiveSplit-style string
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

// ParseTime parses a time string to a time.Duration
func ParseTime(timeStr string) time.Duration {
	if timeStr == "" {
		return 0
	}

	timeStr = strings.TrimPrefix(timeStr, "00:")

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

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(fraction*float64(time.Second))
}

// GetSumOfBest returns the sum of best segment times
func GetSumOfBest(segments []Segment) time.Duration {
	var sum time.Duration
	for _, segment := range segments {
		if segment.BestSegmentTime.RealTime != "" {
			bestSegTime := ParseTime(segment.BestSegmentTime.RealTime)
			sum += bestSegTime
		}
	}
	return sum
}

// GetSegmentTime returns the duration of a specific segment
func (r *Run) GetSegmentTime(splitIndex int) time.Duration {
	if splitIndex < 0 || splitIndex >= len(r.Splits) {
		return 0
	}

	if r.Splits[splitIndex] == 0 {
		return 0
	}

	if splitIndex == 0 {
		return r.Splits[0]
	}

	return r.Splits[splitIndex] - r.Splits[splitIndex-1]
}

// GetPBSegmentTime returns the Personal Best duration for a specific segment
func (r *Run) GetPBSegmentTime(splitIndex int) time.Duration {
	if splitIndex < 0 || splitIndex >= len(r.State.Segments.Segments) {
		return 0
	}

	segment := r.State.Segments.Segments[splitIndex]
	if len(segment.SplitTimes.SplitTime) == 0 {
		return 0
	}

	pbTime := ParseTime(segment.SplitTimes.SplitTime[0].RealTime)

	if splitIndex == 0 {
		return pbTime
	}

	prevPBTime := ParseTime(r.State.Segments.Segments[splitIndex-1].SplitTimes.SplitTime[0].RealTime)
	return pbTime - prevPBTime
}

// IsPB checks if the current run is a Personal Best
func (r *Run) IsPB() bool {
	if r.CurrentSplit != len(r.State.Segments.Segments) {
		return false
	}

	lastSplitTime := r.Splits[len(r.Splits)-1]
	currentPB := ParseTime(r.State.Segments.Segments[len(r.Splits)-1].SplitTimes.SplitTime[0].RealTime)

	return lastSplitTime < currentPB || currentPB == 0
}

// ReinitializeArrays reinitializes the run arrays after segment changes
func (r *Run) ReinitializeArrays() {
	n := len(r.State.Segments.Segments)
	r.Splits = make([]time.Duration, n)
	r.IsGold = make([]bool, n)
	r.Comparison = make([]time.Duration, n)
	r.CurrentSplit = -1
}

// ### Segment manipulation methods ###

// AddSegment adds a new segment after the specified index
func (state *LiveSplitState) AddSegment(index int, name string) {
	newSegment := Segment{
		Name: name,
		Icon: "",
		SplitTimes: SplitTimes{
			SplitTime: []SplitTime{{Name: "Personal Best", RealTime: ""}},
		},
		BestSegmentTime: BestSegmentTime{RealTime: ""},
		SegmentHistory:  SegmentHistory{Time: []Time{}},
	}

	segments := state.Segments.Segments
	// Insert after index
	if index >= len(segments)-1 {
		state.Segments.Segments = append(segments, newSegment)
	} else {
		state.Segments.Segments = append(segments[:index+1], append([]Segment{newSegment}, segments[index+1:]...)...)
	}
}

// RemoveSegment removes a segment at the specified index
func (state *LiveSplitState) RemoveSegment(index int) {
	if index < 0 || index >= len(state.Segments.Segments) {
		return
	}
	segments := state.Segments.Segments
	state.Segments.Segments = append(segments[:index], segments[index+1:]...)
}

// RenameSegment changes the name of a segment
func (state *LiveSplitState) RenameSegment(index int, name string) {
	if index >= 0 && index < len(state.Segments.Segments) {
		state.Segments.Segments[index].Name = name
	}
}

// MoveSegmentUp swaps a segment with the one above it
func (state *LiveSplitState) MoveSegmentUp(index int) {
	if index > 0 && index < len(state.Segments.Segments) {
		segments := state.Segments.Segments
		segments[index], segments[index-1] = segments[index-1], segments[index]
	}
}

// MoveSegmentDown swaps a segment with the one below it
func (state *LiveSplitState) MoveSegmentDown(index int) {
	if index >= 0 && index < len(state.Segments.Segments)-1 {
		segments := state.Segments.Segments
		segments[index], segments[index+1] = segments[index+1], segments[index]
	}
}

// CreateBlankRun creates a new empty LiveSplit state
func CreateBlankRun(gameName, categoryName string) *LiveSplitState {
	return &LiveSplitState{
		GameName:     gameName,
		CategoryName: categoryName,
		Metadata: Metadata{
			Run: MetadataRun{Version: "1.7.0"},
		},
		Offset:       "00:00:00",
		AttemptCount: 0,
		AttemptHistory: AttemptHistory{
			Attempt: []Attempt{},
		},
		Segments: Segments{
			Segments: []Segment{
				{
					Name: "Split 1",
					Icon: "",
					SplitTimes: SplitTimes{
						SplitTime: []SplitTime{{Name: "Personal Best", RealTime: ""}},
					},
					BestSegmentTime: BestSegmentTime{RealTime: ""},
					SegmentHistory:  SegmentHistory{Time: []Time{}},
				},
			},
		},
		AutoSplitterSettings: "",
	}
}
