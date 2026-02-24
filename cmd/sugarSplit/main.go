package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"sugarSplit/pkg/sugarSplitCore"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--new" {
		// Create new LSS file
		filename := os.Args[2]
		state := sugarSplitCore.CreateBlankRun("New Game", "Any%")
		err := sugarSplitCore.SaveRun(state, filename)
		if err != nil {
			fmt.Printf("Error creating file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created %s\n", filename)

		// Open in edit mode
		m := initialModel(filename)
		m.mode = modeEditSplits
		p := tea.NewProgram(m)
		if err := p.Start(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) != 2 {
		fmt.Println("Usage: sugarSplit <filename.lss>")
		fmt.Println("       sugarSplit --new <filename.lss>")
		os.Exit(1)
	}

	m := initialModel(os.Args[1])
	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
