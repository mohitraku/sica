package views

import "charm.land/lipgloss/v2"

type Styles struct {
	TitleBar lipgloss.Style
	HelpBar  lipgloss.Style
	HelpHint lipgloss.Style
	AppName  lipgloss.Style

	ListItem      lipgloss.Style
	StatusDone    lipgloss.Style
	StatusPartial  lipgloss.Style
	StatusUndone   lipgloss.Style
	HabitName      lipgloss.Style
	HabitNameDone  lipgloss.Style
	Progress       lipgloss.Style
	StreakCurrent  lipgloss.Style
	StreakLongest  lipgloss.Style
	FreqLabel      lipgloss.Style
	DateLabel      lipgloss.Style

	Overlay    lipgloss.Style
	OverlayKey lipgloss.Style
	OverlayVal lipgloss.Style
}

func BuildStyles(dark bool) Styles {
	ld := lipgloss.LightDark(dark)
	highlight := lipgloss.Color("#04B575")
	warn := lipgloss.Color("#D4A017")
	dim := ld(lipgloss.Color("#888888"), lipgloss.Color("#666666"))
	text := ld(lipgloss.Color("#1A1A1A"), lipgloss.Color("#E0E0E0"))
	borderDim := ld(lipgloss.Color("#CCCCCC"), lipgloss.Color("#444444"))
	base := lipgloss.NewStyle()

	return Styles{
		TitleBar: base.Copy().
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(borderDim).
			Padding(0, 1),

		HelpBar: base.Copy().
			Foreground(dim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(borderDim).
			Padding(0, 1),

		HelpHint: base.Copy().Foreground(dim),

		AppName: base.Copy().Bold(true).Foreground(highlight),

		ListItem: base.Copy(),

		StatusDone:    base.Copy().Foreground(highlight),
		StatusPartial: base.Copy().Foreground(warn),
		StatusUndone:  base.Copy().Foreground(dim),

		HabitName:     base.Copy().Foreground(text),
		HabitNameDone: base.Copy().Bold(true).Foreground(highlight),

		Progress: base.Copy().Foreground(dim),

		StreakCurrent: base.Copy().Foreground(warn),
		StreakLongest: base.Copy().Foreground(highlight),

		FreqLabel: base.Copy().Foreground(dim),
		DateLabel: base.Copy().Foreground(text),

		Overlay: base.Copy().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(1, 2),

		OverlayKey: base.Copy().Foreground(dim),
		OverlayVal: base.Copy().Foreground(text),
	}
}
