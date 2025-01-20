package sugarSplitCore

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Action string

const (
	ActionSplit     Action = "split"
	ActionReset     Action = "reset"
	ActionUndo      Action = "undo"
	ActionQuit      Action = "quit"
	ActionConfirm   Action = "confirm"
	ActionSaveReset Action = "save_reset"
	ActionCancel    Action = "cancel"
	ActionSkip      Action = "skip"
)

type Hotkey struct {
	Key         string `toml:"key"`
	Action      Action `toml:"action"`
	Description string `toml:"description"`
	Available   bool
}

type HotkeyConfig struct {
	Hotkeys []Hotkey `toml:"hotkey"`
}

// Default configuration if no config file is found
var defaultHotkeys = []Hotkey{
	{Key: "space", Action: ActionSplit, Description: "Start/Split"},
	{Key: "r", Action: ActionReset, Description: "Reset"},
	{Key: "z", Action: ActionUndo, Description: "Undo Split"},
	{Key: "q", Action: ActionQuit, Description: "Quit"},
	{Key: "y", Action: ActionConfirm, Description: "Confirm"},
	{Key: "s", Action: ActionSaveReset, Description: "Save and Reset"},
	{Key: "n", Action: ActionCancel, Description: "Cancel"},
	{Key: "esc", Action: ActionCancel, Description: "Cancel"},
	{Key: "k", Action: ActionSkip, Description: "Skip Split"},
}

// LoadHotkeys loads hotkeys from a TOML file
func LoadHotkeys(configPath string) ([]Hotkey, error) {
	var config HotkeyConfig

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return defaultHotkeys, nil
	}

	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return nil, fmt.Errorf("error loading hotkey config: %v", err)
	}

	return config.Hotkeys, nil
}

// UpdateHotkeyAvailability updates which hotkeys are currently available based on run state
func (r *Run) UpdateHotkeyAvailability() {
	for i := range r.Hotkeys {
		if r.ResettingState {
			switch r.Hotkeys[i].Action {
			case ActionConfirm, ActionSaveReset, ActionCancel:
				r.Hotkeys[i].Available = true
			default:
				r.Hotkeys[i].Available = false
			}
			continue
		}

		switch r.Hotkeys[i].Action {
		case ActionSplit:
			if !r.Started {
				r.Hotkeys[i].Available = true
			} else {
				r.Hotkeys[i].Available = !r.Completed
			}
		case ActionReset:
			r.Hotkeys[i].Available = r.Started || r.Completed
		case ActionUndo:
			r.Hotkeys[i].Available = r.Started && r.CurrentSplit > 0 && !r.Completed
		case ActionSkip:
			r.Hotkeys[i].Available = r.Started && !r.Completed && r.CurrentSplit < len(r.State.Segments.Segments)
		case ActionQuit:
			r.Hotkeys[i].Available = true
		case ActionConfirm, ActionSaveReset, ActionCancel:
			r.Hotkeys[i].Available = false
		}
	}
}

// GetAvailableHotkeys returns a formatted string of currently available hotkeys
func (r *Run) GetAvailableHotkeys() string {
	actionMap := make(map[Action][]string)

	var actions []Action
	seenActions := make(map[Action]bool)

	for _, hk := range r.Hotkeys {
		if hk.Available {
			actionMap[hk.Action] = append(actionMap[hk.Action], hk.Key)
			if !seenActions[hk.Action] {
				actions = append(actions, hk.Action)
				seenActions[hk.Action] = true
			}
		}
	}

	descMap := make(map[Action]string)
	for _, hk := range r.Hotkeys {
		descMap[hk.Action] = hk.Description
	}

	var result string
	first := true

	for _, action := range actions {
		if !first {
			result += "  "
		}
		// Sort the keys for consistent ordering
		keys := actionMap[action]
		sort.Strings(keys)
		keyStr := strings.Join(keys, "/")
		result += fmt.Sprintf("%s: %s", keyStr, descMap[action])
		first = false
	}

	return result
}

// GetAction returns the action associated with a key press, if any
func (r *Run) GetAction(key string) (Action, bool) {
	if key == " " {
		key = "space"
	}
	for _, hk := range r.Hotkeys {
		if hk.Key == key && hk.Available {
			return hk.Action, true
		}
	}
	return "", false
}
