package views

import (
	tea "charm.land/bubbletea/v2"
)

type Confirm struct {
	Active  bool
	Message string
}

func NewConfirm() Confirm {
	return Confirm{}
}

func (c *Confirm) Show(msg string) {
	c.Active = true
	c.Message = msg
}

func (c *Confirm) Hide() {
	c.Active = false
	c.Message = ""
}

func (c *Confirm) Update(msg tea.Msg) (confirmed bool, dismissed bool) {
	if !c.Active {
		return false, false
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y", "Y":
			c.Hide()
			return true, false
		default:
			c.Hide()
			return false, true
		}
	}
	return false, false
}

func (c *Confirm) Render(st Styles) string {
	if !c.Active {
		return ""
	}
	content := st.OverlayVal.Render(c.Message) + "\n\n" +
		st.HelpHint.Render("  y:yes │ any other key:no  ")
	return st.Overlay.Render(content)
}
