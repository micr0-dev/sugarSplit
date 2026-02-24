package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"sugarSplit/pkg/sugarSplitCore"
)

func (m model) View() string {
	if m.mode == modeEditSplits {
		return m.renderEditMode()
	}

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

		// Handle empty SplitTimes
		var pbTime = sugarSplitCore.ParseTime("")
		var pbTimeStr = "-"
		if len(segment.SplitTimes.SplitTime) > 0 && segment.SplitTimes.SplitTime[0].RealTime != "" {
			pbTime = sugarSplitCore.ParseTime(segment.SplitTimes.SplitTime[0].RealTime)
			pbTimeStr = sugarSplitCore.FormatDuration(pbTime)
		}

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
		timerStyle = timerStyle.Foreground(ColorPrimary)
	} else if m.run.CurrentSplit > 0 && m.run.Comparison[m.run.CurrentSplit-1] < 0 {
		timerStyle = timerStyle.Foreground(ColorAhead)
	} else if m.run.CurrentSplit > 0 {
		timerStyle = timerStyle.Foreground(ColorBehind)
	} else {
		timerStyle = timerStyle.Foreground(ColorPrimary)
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

func (m model) renderEditMode() string {
	var s strings.Builder
	styles := initializeStyles(m.width)

	s.WriteString("\n")
	s.WriteString(styles.title.Render("Edit Splits"))
	s.WriteString("\n")
	s.WriteString(styles.title.Render(m.run.State.GameName + " - " + m.run.State.CategoryName))
	s.WriteString("\n\n")

	// Render splits with selection
	for i, segment := range m.run.State.Segments.Segments {
		var line string
		if i == m.editIndex {
			if m.editing {
				// Show text input
				line = fmt.Sprintf("> %s█", m.editInput)
			} else {
				line = fmt.Sprintf("> %s", segment.Name)
			}
			s.WriteString(styles.currentSegment.Render(line))
		} else {
			line = fmt.Sprintf("  %s", segment.Name)
			s.WriteString(styles.segment.Render(line))
		}
		s.WriteString("\n")
	}

	// Calculate padding to push controls to bottom
	contentHeight := 5 + len(m.run.State.Segments.Segments) + 4 // header + splits + controls
	if m.height > contentHeight {
		s.WriteString(strings.Repeat("\n", m.height-contentHeight))
	}

	// Controls help
	s.WriteString("\n")
	if m.editing {
		s.WriteString(styles.controls.Render("Enter: Confirm | Esc: Cancel"))
	} else {
		s.WriteString(styles.controls.Render("j/k: Navigate | r: Rename | a: Add | d: Delete | J/K: Reorder"))
	}

	// Bottom action buttons
	s.WriteString("\n\n")
	saveBtn := styles.ahead.Render("[Enter] Save & Exit")
	cancelBtn := styles.behind.Render("[Esc] Cancel")
	s.WriteString(styles.controls.Render(saveBtn + "    " + cancelBtn))

	return s.String()
}
