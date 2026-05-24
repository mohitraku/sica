package views

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type Settings struct {
	Active       bool
	effectiveDir string
	source       string
	path         textinput.Model
	saved        bool
	savedMsg     string
}

func NewSettings() Settings {
	ti := textinput.New()
	ti.Placeholder = "/path/to/data/dir"
	ti.CharLimit = 256
	ti.SetWidth(40)
	return Settings{path: ti}
}

func (s *Settings) Show(effectiveDir, source, configuredPath string) {
	s.Active = true
	s.effectiveDir = effectiveDir
	s.source = source
	s.saved = false
	s.savedMsg = ""

	initial := configuredPath
	if initial == "" {
		initial = effectiveDir
	}
	s.path.SetValue(initial)
	s.path.Focus()
}

func (s *Settings) Update(msg tea.Msg) (saved bool, newPath string) {
	if !s.Active {
		return false, ""
	}

	if s.saved {
		switch msg.(type) {
		case tea.KeyPressMsg:
			s.Active = false
			return true, s.path.Value()
		}
		return false, ""
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			s.Active = false
			return false, ""
		case "enter":
			val := strings.TrimSpace(s.path.Value())
			if val == "" {
				return false, ""
			}
			s.saved = true
			s.savedMsg = "Saved. Restart sica to apply the new data directory."
			return true, val
		}
	}
	var cmd tea.Cmd
	s.path, cmd = s.path.Update(msg)
	_ = cmd
	return false, ""
}

func (s *Settings) Render(st Styles) string {
	if !s.Active {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(st.AppName.Render("Settings"))
	sb.WriteString("\n\n")

	sb.WriteString(st.FreqLabel.Render("Data directory"))
	sb.WriteByte('\n')
	sb.WriteString(st.OverlayVal.Render("  " + s.effectiveDir))
	sb.WriteByte('\n')
	sb.WriteString(st.HelpHint.Render("  Source: " + s.source))
	sb.WriteString("\n\n")

	if s.source == "env" {
		sb.WriteString(st.HelpHint.Render("  Path is set by SICA_DATA_DIR env var.\n"))
		sb.WriteString(st.HelpHint.Render("  Unset it to use the config file instead."))
	} else if s.saved {
		sb.WriteString(st.StatusDone.Render("  " + s.savedMsg))
	} else {
		sb.WriteString(st.StatusDone.Render("▸ ") + st.FreqLabel.Render("New path: "))
		sb.WriteString(st.HabitName.Render(s.path.View()))
	}

	sb.WriteString("\n\n")
	sb.WriteString(st.HelpHint.Render("  enter: save  esc: cancel"))

	return st.Overlay.Render(sb.String())
}
