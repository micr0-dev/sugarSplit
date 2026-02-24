package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"sugarSplit/pkg/sugarSplitCore"
)

type appMode int

const (
	modeNormal appMode = iota
	modeEditSplits
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
	mode          appMode
	// Edit mode fields
	editIndex int
	editInput string
	editing   bool
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
		mode:       modeNormal,
		editIndex:  0,
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
