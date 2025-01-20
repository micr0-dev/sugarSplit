package sugarSplitCore

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type UIComponent string

const (
	UIHeader          UIComponent = "header"
	UISplits          UIComponent = "splits"
	UITimer           UIComponent = "timer"
	UIPreviousSegment UIComponent = "previous_segment"
	UIControls        UIComponent = "controls"
)

type UISection string

const (
	SectionTop    UISection = "top"
	SectionMiddle UISection = "middle"
	SectionBottom UISection = "bottom"
)

type UIComponentConfig struct {
	Component UIComponent `toml:"component"`
	Section   UISection   `toml:"section"`
}

type UIConfig struct {
	Layout   []UIComponent       `toml:"layout"`
	Sections []UIComponentConfig `toml:"sections"`
}

var defaultUIConfig = UIConfig{
	Layout: []UIComponent{
		UIHeader,
		UISplits,
		UITimer,
		UIPreviousSegment,
		UIControls,
	},
}

func LoadUIConfig(configPath string) (*UIConfig, error) {
	var config struct {
		UI UIConfig `toml:"ui"`
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &defaultUIConfig, nil
	}

	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return nil, fmt.Errorf("error loading UI config: %v", err)
	}

	// If layout is empty, use default
	if len(config.UI.Layout) == 0 {
		return &defaultUIConfig, nil
	}

	return &config.UI, nil
}
