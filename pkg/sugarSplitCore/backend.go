package sugarSplitCore

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
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
}

// NewRun creates a new Run instance from a LiveSplitState
func NewRun(state *LiveSplitState, configPath string) (*Run, error) {
	hotkeys, err := LoadHotkeys(configPath)
	if err != nil {
		return nil, err
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
	}

	run.UpdateHotkeyAvailability()
	return run, nil
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
				RealTime: FormatDurationLSS(split),
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
				r.State.Segments.Segments[i].BestSegmentTime.RealTime = FormatDurationLSS(splitTime)
			}

			// Update PB split time if this is a PB run
			if isPB {
				if len(r.State.Segments.Segments[i].SplitTimes.SplitTime) == 0 {
					r.State.Segments.Segments[i].SplitTimes.SplitTime = append(
						r.State.Segments.Segments[i].SplitTimes.SplitTime,
						SplitTime{Name: "Personal Best"},
					)
				}
				r.State.Segments.Segments[i].SplitTimes.SplitTime[0].RealTime = FormatDurationLSS(split)
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
	data, err := ioutil.ReadFile(filename)
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
	return ioutil.WriteFile(filename, data, 0644)
}

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

func FormatDurationLSS(d time.Duration) string {
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
