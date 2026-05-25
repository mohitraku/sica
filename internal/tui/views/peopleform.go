package views

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type PeopleForm struct {
	Mode       FormMode
	Name       textinput.Model
	Email      textinput.Model
	Phone      textinput.Model
	FocusField int // 0=name, 1=email, 2=phone
	EditID     string
	EditName   string
	Error      string
}

func NewPeopleForm() PeopleForm {
	ni := textinput.New()
	ni.Placeholder = "Name"
	ni.CharLimit = 128
	ni.SetWidth(30)

	ei := textinput.New()
	ei.Placeholder = "Email (optional)"
	ei.CharLimit = 254
	ei.SetWidth(30)

	pi := textinput.New()
	pi.Placeholder = "Phone (optional)"
	pi.CharLimit = 64
	pi.SetWidth(30)

	return PeopleForm{
		Name:  ni,
		Email: ei,
		Phone: pi,
	}
}

func (pf *PeopleForm) Active() bool {
	return pf.Mode != FormNone
}

func (pf *PeopleForm) StartNew() {
	pf.Name.Reset()
	pf.Name.Placeholder = "Name"
	pf.Name.Focus()
	pf.Email.Reset()
	pf.Email.Placeholder = "Email (optional)"
	pf.Email.Blur()
	pf.Phone.Reset()
	pf.Phone.Placeholder = "Phone (optional)"
	pf.Phone.Blur()
	pf.FocusField = 0
	pf.EditID = ""
	pf.Error = ""
	pf.Mode = FormNew
}

func (pf *PeopleForm) StartEdit(id, name, email, phone string) {
	pf.Name.SetValue(name)
	pf.Name.Focus()
	pf.Email.SetValue(email)
	pf.Email.Blur()
	pf.Phone.SetValue(phone)
	pf.Phone.Blur()
	pf.FocusField = 0
	pf.EditID = id
	pf.EditName = name
	pf.Error = ""
	pf.Mode = FormEdit
}

func (pf *PeopleForm) Cancel() {
	pf.Mode = FormNone
	pf.Name.Blur()
	pf.Email.Blur()
	pf.Phone.Blur()
	pf.Error = ""
}

func (pf *PeopleForm) SetError(msg string) {
	pf.Error = msg
}

func (pf *PeopleForm) GetFields() (name, email, phone string) {
	return strings.TrimSpace(pf.Name.Value()),
		strings.TrimSpace(pf.Email.Value()),
		strings.TrimSpace(pf.Phone.Value())
}

func (pf *PeopleForm) Update(msg tea.Msg) tea.Cmd {
	if pf.Mode == FormNone {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			pf.FocusField++
			if pf.FocusField > 2 {
				pf.FocusField = 0
			}
			pf.updateFocus()
		case "shift+tab":
			pf.FocusField--
			if pf.FocusField < 0 {
				pf.FocusField = 2
			}
			pf.updateFocus()
		case "left", "right":
			// No-op for single-value text fields
		}
	}

	switch pf.FocusField {
	case 0:
		var cmd tea.Cmd
		pf.Name, cmd = pf.Name.Update(msg)
		return cmd
	case 1:
		var cmd tea.Cmd
		pf.Email, cmd = pf.Email.Update(msg)
		return cmd
	case 2:
		var cmd tea.Cmd
		pf.Phone, cmd = pf.Phone.Update(msg)
		return cmd
	}
	return nil
}

func (pf *PeopleForm) updateFocus() {
	pf.Name.Blur()
	pf.Email.Blur()
	pf.Phone.Blur()
	switch pf.FocusField {
	case 0:
		pf.Name.Focus()
	case 1:
		pf.Email.Focus()
	case 2:
		pf.Phone.Focus()
	}
}

func (pf *PeopleForm) Render(st Styles) string {
	switch pf.Mode {
	case FormNew, FormEdit:
		return pf.renderFullForm(st)
	}
	return ""
}

func (pf *PeopleForm) renderFullForm(st Styles) string {
	var title string
	if pf.Mode == FormNew {
		title = st.AppName.Render("New Person")
	} else {
		title = st.AppName.Render("Edit: " + pf.EditName)
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteByte('\n')

	if pf.Error != "" {
		sb.WriteString(st.StatusPartial.Render("  " + pf.Error))
		sb.WriteByte('\n')
	}

	renderField := func(label, val string, focused bool) string {
		prefix := "  "
		if focused {
			prefix = st.StatusDone.Render("▸ ")
		}
		return prefix + st.FreqLabel.Render(label+": ") + st.RoutineName.Render(val)
	}

	sb.WriteString(renderField("Name", pf.Name.View(), pf.FocusField == 0))
	sb.WriteByte('\n')
	sb.WriteString(renderField("Email", pf.Email.View(), pf.FocusField == 1))
	sb.WriteByte('\n')
	sb.WriteString(renderField("Phone", pf.Phone.View(), pf.FocusField == 2))

	return st.ListItem.Render(sb.String())
}
