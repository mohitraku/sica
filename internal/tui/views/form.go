package views

import (
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

type HabitForm struct {
	Mode FormMode

	// For create/edit
	Name         textinput.Model
	FreqIdx      int
	QtyIdx       int
	TargetValue  int
	FocusField   int // 0=name, 1=freq, 2=qty, 3=target
	EditHabitID  string
	EditName     string
	Error        string

}

func NewHabitForm() HabitForm {
	ti := textinput.New()
	ti.Placeholder = "Habit name"
	ti.CharLimit = 128
	ti.SetWidth(30)

	return HabitForm{
		Name: ti,
	}
}

func (hf *HabitForm) Active() bool {
	return hf.Mode != FormNone
}

func (hf *HabitForm) StartNew() {
	hf.Name.Reset()
	hf.Name.Placeholder = "Habit name"
	hf.Name.Focus()
	hf.FreqIdx = 0
	hf.QtyIdx = 0
	hf.TargetValue = 1
	hf.FocusField = 0
	hf.EditHabitID = ""
	hf.Error = ""
	hf.Mode = FormNew
}

func (hf *HabitForm) StartEdit(id, name, freq, qty string, target int) {
	hf.Name.SetValue(name)
	hf.Name.Focus()
	hf.FreqIdx = indexOf(freqOptions, freq)
	hf.QtyIdx = indexOf(qtyOptions, qty)
	hf.TargetValue = target
	hf.FocusField = 0
	hf.EditHabitID = id
	hf.EditName = name
	hf.Error = ""
	hf.Mode = FormEdit
}

func indexOf(slice []string, val string) int {
	for i, s := range slice {
		if s == val {
			return i
		}
	}
	return 0
}

func (hf *HabitForm) Cancel() {
	hf.Mode = FormNone
	hf.Name.Blur()
	hf.Error = ""
}

func (hf *HabitForm) SetError(msg string) {
	hf.Error = msg
}

func (hf *HabitForm) GetHabitFields() (name, freq, qty string, target int) {
	name = strings.TrimSpace(hf.Name.Value())
	freq = freqOptions[hf.FreqIdx]
	qty = qtyOptions[hf.QtyIdx]
	target = hf.TargetValue
	return
}

func (hf *HabitForm) Update(msg tea.Msg) tea.Cmd {
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

func (hf *HabitForm) numFields() int {
	if hf.QtyIdx == 1 { // "count"
		return 4
	}
	return 3
}

func (hf *HabitForm) cycleField(delta int) {
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

func (hf *HabitForm) Focus() string {
	switch hf.FocusField {
	case 1:
		return freqOptions[hf.FreqIdx]
	case 2:
		return qtyOptions[hf.QtyIdx]
	case 3:
		return strconv.Itoa(hf.TargetValue)
	}
	return ""
}

func (hf *HabitForm) Render(st Styles) string {
	switch hf.Mode {
	case FormNew, FormEdit:
		return hf.renderFullForm(st)
	}
	return ""
}

func (hf *HabitForm) renderFullForm(st Styles) string {
	var title string
	if hf.Mode == FormNew {
		title = st.AppName.Render("New Habit")
	} else {
		title = st.AppName.Render("Edit: " + hf.EditName)
	}

	fields := []struct {
		label   string
		content string
		active  bool
	}{
		{"Name", hf.Name.View(), hf.FocusField == 0},
		{"Frequency", hf.freqDisplay(st), hf.FocusField == 1},
		{"Type", hf.qtyDisplay(st), hf.FocusField == 2},
	}
	if hf.QtyIdx == 1 { // "count"
		fields = append(fields, struct {
			label   string
			content string
			active  bool
		}{"Target", hf.targetDisplay(st), hf.FocusField == 3})
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

func (hf *HabitForm) freqDisplay(st Styles) string {
	active := hf.FocusField == 1
	s := st.HabitName
	if !active {
		s = st.FreqLabel
	}
	arrow := st.HelpHint.Render(" ←→ ")
	return arrow + s.Render(hf.FreqVal())
}

func (hf *HabitForm) qtyDisplay(st Styles) string {
	active := hf.FocusField == 2
	s := st.HabitName
	if !active {
		s = st.FreqLabel
	}
	arrow := st.HelpHint.Render(" ←→ ")
	return arrow + s.Render(hf.QtyVal())
}

func (hf *HabitForm) targetDisplay(st Styles) string {
	active := hf.FocusField == 3
	s := st.HabitName
	if !active {
		s = st.FreqLabel
	}
	arrow := st.HelpHint.Render(" ←→ ")
	return arrow + s.Render(strconv.Itoa(hf.TargetValue))
}

func (hf *HabitForm) FreqVal() string  { return freqOptions[hf.FreqIdx] }
func (hf *HabitForm) QtyVal() string   { return qtyOptions[hf.QtyIdx] }
