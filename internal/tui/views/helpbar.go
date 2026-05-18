package views

import (
	"strings"

	"charm.land/bubbles/v2/key"
)

type HelpBar struct {
	bindings []key.Binding
}

func NewHelpBar() HelpBar {
	return HelpBar{}
}

func (hb *HelpBar) SetBindings(b []key.Binding) {
	hb.bindings = b
}

func (hb *HelpBar) Render(st Styles) string {
	if len(hb.bindings) == 0 {
		return st.HelpBar.Render("")
	}
	var parts []string
	for _, b := range hb.bindings {
		help := b.Help()
		parts = append(parts, st.HelpHint.Render(help.Key)+" "+help.Desc)
	}
	return st.HelpBar.Render(strings.Join(parts, " │ "))
}
