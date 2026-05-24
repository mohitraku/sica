package views

import (
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

)

type FormMode int

const (
	FormNone FormMode = iota
	FormNew
	FormEdit
)

var freqOptions = []string{"daily", "weekly", "monthly"}
var qtyOptions = []string{"binary", "count"}

type RoutineForm struct {
	Mode FormMode

	// For create/edit
	Name         textinput.Model
	FreqIdx      int
	QtyIdx       int
	TargetValue  int
	FocusField   int // 0=name, 1=freq, 2=qty, 3=target
	EditRoutineID  string
	EditName     string
	Error        string

}

func NewRoutineForm() RoutineForm {
	ti := textinput.New()
	ti.Placeholder = "Routine name"
	ti.CharLimit = 128
	ti.SetWidth(30)

	return RoutineForm{
		Name: ti,
	}
}

func (hf *RoutineForm) Active() bool {
	return hf.Mode != FormNone
}

func (hf *RoutineForm) StartNew() {
	hf.Name.Reset()
	hf.Name.Placeholder = "Routine name"
	hf.Name.Focus()
	hf.FreqIdx = 0
	hf.QtyIdx = 0
	hf.TargetValue = 1
	hf.FocusField = 0
	hf.EditRoutineID = ""
	hf.Error = ""
	hf.Mode = FormNew
}

func (hf *RoutineForm) StartEdit(id, name, freq, qty string, target int) {
	hf.Name.SetValue(name)
	hf.Name.Focus()
	hf.FreqIdx = slices.Index(freqOptions, freq)
	hf.QtyIdx = slices.Index(qtyOptions, qty)
	hf.TargetValue = target
	hf.FocusField = 0
	hf.EditRoutineID = id
	hf.EditName = name
	hf.Error = ""
	hf.Mode = FormEdit
}

func (hf *RoutineForm) Cancel() {
	hf.Mode = FormNone
	hf.Name.Blur()
	hf.Error = ""
}

func (hf *RoutineForm) SetError(msg string) {
	hf.Error = msg
}

func (hf *RoutineForm) GetRoutineFields() (name, freq, qty string, target int) {
	name = strings.TrimSpace(hf.Name.Value())
	freq = freqOptions[hf.FreqIdx]
	qty = qtyOptions[hf.QtyIdx]
	target = hf.TargetValue
	return
}

func (hf *RoutineForm) Update(msg tea.Msg) tea.Cmd {
	if hf.Mode == FormNone {
		return nil
	}

	switch hf.Mode {
	case FormNew, FormEdit:
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "tab":
				hf.FocusField = (hf.FocusField + 1) % hf.numFields()
				hf.Error = ""
			case "shift+tab":
				hf.FocusField = (hf.FocusField + hf.numFields() - 1) % hf.numFields()
				hf.Error = ""
			case "left":
				hf.cycleField(-1)
				hf.Error = ""
			case "right":
				hf.cycleField(1)
				hf.Error = ""
			}
		}
		var cmd tea.Cmd
		hf.Name, cmd = hf.Name.Update(msg)
		return cmd
	}
	return nil
}

func (hf *RoutineForm) numFields() int {
	if hf.QtyIdx == 1 { // "count"
		return 4
	}
	return 3
}

func (hf *RoutineForm) cycleField(delta int) {
	switch hf.FocusField {
	case 1:
		hf.FreqIdx = (hf.FreqIdx + delta + len(freqOptions)) % len(freqOptions)
	case 2:
		oldQty := hf.QtyIdx
		hf.QtyIdx = (hf.QtyIdx + delta + len(qtyOptions)) % len(qtyOptions)
		// If switching from count to binary, reset target to 1 and pull focus back.
		if oldQty == 1 && hf.QtyIdx == 0 {
			hf.TargetValue = 1
			if hf.FocusField == 3 {
				hf.FocusField = 2
			}
		}
	case 3:
		hf.TargetValue += delta
		if hf.TargetValue < 1 {
			hf.TargetValue = 99
		} else if hf.TargetValue > 99 {
			hf.TargetValue = 1
		}
	}
}

func (hf *RoutineForm) Render(st Styles) string {
	switch hf.Mode {
	case FormNew, FormEdit:
		return hf.renderFullForm(st)
	}
	return ""
}

func (hf *RoutineForm) renderFullForm(st Styles) string {
	var title string
	if hf.Mode == FormNew {
		title = st.AppName.Render("New Routine")
	} else {
		title = st.AppName.Render("Edit: " + hf.EditName)
	}

	fields := []struct {
		label   string
		content string
		active  bool
	}{
		{"Name", hf.Name.View(), hf.FocusField == 0},
		{"Frequency", hf.fieldDisplay(st, 1, hf.FreqVal()), hf.FocusField == 1},
		{"Type", hf.fieldDisplay(st, 2, hf.QtyVal()), hf.FocusField == 2},
	}
	if hf.QtyIdx == 1 { // "count"
		fields = append(fields, struct {
			label   string
			content string
			active  bool
		}{"Target", hf.fieldDisplay(st, 3, strconv.Itoa(hf.TargetValue)), hf.FocusField == 3})
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteByte('\n')

	if hf.Error != "" {
		sb.WriteString(st.StatusPartial.Render("  " + hf.Error))
		sb.WriteByte('\n')
	}

	for _, f := range fields {
		prefix := "  "
		if f.active {
			prefix = st.StatusDone.Render("▸ ")
		}
		label := st.FreqLabel.Render(f.label + ": ")
		sb.WriteString(prefix + label + f.content)
		sb.WriteByte('\n')
	}

	return st.ListItem.Render(sb.String())
}

func (hf *RoutineForm) fieldDisplay(st Styles, focusField int, value string) string {
	active := hf.FocusField == focusField
	s := st.RoutineName
	if !active {
		s = st.FreqLabel
	}
	arrow := st.HelpHint.Render(" ←→ ")
	return arrow + s.Render(value)
}

func (hf *RoutineForm) FreqVal() string  { return freqOptions[hf.FreqIdx] }
func (hf *RoutineForm) QtyVal() string   { return qtyOptions[hf.QtyIdx] }
