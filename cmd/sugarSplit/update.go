package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"sugarSplit/pkg/sugarSplitCore"
)

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

	case sugarSplitCore.ActionEdit:
		if !m.run.Started && !m.run.Completed {
			m.mode = modeEditSplits
			m.editIndex = 0
			return m, nil
		}
	}

	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle edit mode separately
	if m.mode == modeEditSplits {
		return m.updateEditMode(msg)
	}

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

func (m model) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// If actively editing a field
		if m.editing {
			switch key {
			case "enter":
				// Commit the edit
				if m.editIndex < len(m.run.State.Segments.Segments) {
					m.run.State.RenameSegment(m.editIndex, m.editInput)
				}
				m.editing = false
				m.editInput = ""
				return m, nil
			case "esc":
				m.editing = false
				m.editInput = ""
				return m, nil
			case "backspace":
				if len(m.editInput) > 0 {
					m.editInput = m.editInput[:len(m.editInput)-1]
				}
				return m, nil
			default:
				// Only add printable characters
				if len(key) == 1 {
					m.editInput += key
				}
				return m, nil
			}
		}

		// Navigation and actions when not editing
		switch key {
		case "esc":
			// Cancel - reload from file to discard changes
			state, err := sugarSplitCore.LoadRun(m.filename)
			if err == nil {
				m.run.State = state
				m.run.ReinitializeArrays()
			}
			m.mode = modeNormal
			return m, nil
		case "enter":
			// Save and exit
			err := sugarSplitCore.SaveRun(m.run.State, m.filename)
			if err == nil {
				m.mode = modeNormal
			}
			return m, nil
		case "up", "k":
			if m.editIndex > 0 {
				m.editIndex--
			}
		case "down", "j":
			if m.editIndex < len(m.run.State.Segments.Segments)-1 {
				m.editIndex++
			}
		case "r":
			// Rename current segment
			if m.editIndex < len(m.run.State.Segments.Segments) {
				m.editing = true
				m.editInput = m.run.State.Segments.Segments[m.editIndex].Name
			}
		case "a":
			// Add new split after current
			m.run.State.AddSegment(m.editIndex, "New Split")
			m.run.ReinitializeArrays()
			m.editIndex++
		case "d":
			// Delete current split (but keep at least one)
			if len(m.run.State.Segments.Segments) > 1 {
				m.run.State.RemoveSegment(m.editIndex)
				m.run.ReinitializeArrays()
				if m.editIndex >= len(m.run.State.Segments.Segments) {
					m.editIndex = len(m.run.State.Segments.Segments) - 1
				}
			}
		case "K", "shift+up":
			// Move split up
			if m.editIndex > 0 {
				m.run.State.MoveSegmentUp(m.editIndex)
				m.editIndex--
			}
		case "J", "shift+down":
			// Move split down
			if m.editIndex < len(m.run.State.Segments.Segments)-1 {
				m.run.State.MoveSegmentDown(m.editIndex)
				m.editIndex++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}
