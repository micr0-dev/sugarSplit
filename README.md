# sugarSplit

a terminal-based speedrun timer. basically livesplit but in your terminal. works great on linux where livesplit can be janky over wine.

**unlike other TUI timers, sugarSplit uses actual LiveSplit `.lss` files.** your splits, golds, and PBs are fully compatible with LiveSplit. import your existing splits or share them with livesplit users.

## install

grab a binary from the [releases](../../releases) tab, or build it yourself:

```bash
go build ./cmd/sugarSplit
```

## usage

```bash
# open an existing livesplit file
./sugarSplit mysplits.lss

# create a new splits file
./sugarSplit --new game.lss
```

## controls

| key | action |
|-----|--------|
| space | start timer / split |
| r | reset |
| z | undo split |
| k | skip split |
| e | edit splits |
| q | quit |

when resetting you'll be asked to confirm:
- `y` to reset
- `s` to save and reset
- `n` or `esc` to cancel

## edit mode

press `e` to edit your splits

| key | action |
|-----|--------|
| j/k | navigate |
| r | rename split |
| a | add split |
| d | delete split |
| J/K | reorder splits |
| enter | save & exit |
| esc | cancel |

(arrow keys also work instead of j/k)

## config

config lives in `config.toml` in the same directory. you can customize hotkeys and ui layout.

example hotkey:
```toml
[[hotkey]]
key = "space"
action = "split"
description = "Start/Split"
```

available actions: `split`, `reset`, `undo`, `skip`, `quit`, `confirm`, `save_reset`, `cancel`, `edit`
