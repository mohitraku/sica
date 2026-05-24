package tui

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Increment  key.Binding
	Decrement  key.Binding
	New        key.Binding
	Edit       key.Binding
	Delete     key.Binding
	Help       key.Binding
	Quit       key.Binding
	PrevDay    key.Binding
	NextDay    key.Binding
	Today      key.Binding
	Settings   key.Binding
	SwitchMode key.Binding
	Toggle     key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Increment: key.NewBinding(
		key.WithKeys("+", "="),
		key.WithHelp("+/=", "increment"),
	),
	Decrement: key.NewBinding(
		key.WithKeys("-"),
		key.WithHelp("-", "decrement"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	PrevDay: key.NewBinding(
		key.WithKeys("[", "left"),
		key.WithHelp("[", "prev day"),
	),
	NextDay: key.NewBinding(
		key.WithKeys("]", "right"),
		key.WithHelp("]", "next day"),
	),
	Today: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "today"),
	),
	Settings: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "settings"),
	),
	SwitchMode: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "tasks"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "toggle"),
	),
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Increment, k.Decrement, k.New, k.Edit, k.Delete},
		{k.SwitchMode, k.Toggle},
		{k.PrevDay, k.NextDay, k.Today},
		{k.Help, k.Settings, k.Quit},
	}
}
