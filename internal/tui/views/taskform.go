package views

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type TaskForm struct {
	Mode     FormMode
	Title    textinput.Model
	EditID   string
	EditName string
	Error    string
}

func NewTaskForm() TaskForm {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.CharLimit = 128
	ti.SetWidth(30)
	return TaskForm{Title: ti}
}

func (tf *TaskForm) Active() bool {
	return tf.Mode != FormNone
}

func (tf *TaskForm) StartNew() {
	tf.Title.Reset()
	tf.Title.Placeholder = "Task title"
	tf.Title.Focus()
	tf.EditID = ""
	tf.Error = ""
	tf.Mode = FormNew
}

func (tf *TaskForm) StartEdit(id, title string) {
	tf.Title.SetValue(title)
	tf.Title.Focus()
	tf.EditID = id
	tf.EditName = title
	tf.Error = ""
	tf.Mode = FormEdit
}

func (tf *TaskForm) Cancel() {
	tf.Mode = FormNone
	tf.Title.Blur()
	tf.Error = ""
}

func (tf *TaskForm) SetError(msg string) {
	tf.Error = msg
}

func (tf *TaskForm) GetTitle() string {
	return strings.TrimSpace(tf.Title.Value())
}

func (tf *TaskForm) Update(msg tea.Msg) tea.Cmd {
	if tf.Mode == FormNone {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "shift+tab", "left", "right":
			// Single field — navigation keys are no-op
		}
	}
	var cmd tea.Cmd
	tf.Title, cmd = tf.Title.Update(msg)
	return cmd
}

func (tf *TaskForm) Render(st Styles) string {
	switch tf.Mode {
	case FormNew, FormEdit:
		return tf.renderFullForm(st)
	}
	return ""
}

func (tf *TaskForm) renderFullForm(st Styles) string {
	var title string
	if tf.Mode == FormNew {
		title = st.AppName.Render("New Task")
	} else {
		title = st.AppName.Render("Edit: " + tf.EditName)
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteByte('\n')

	if tf.Error != "" {
		sb.WriteString(st.StatusPartial.Render("  " + tf.Error))
		sb.WriteByte('\n')
	}

	sb.WriteString(st.StatusDone.Render("▸ ") + st.FreqLabel.Render("Title: "))
	sb.WriteString(st.RoutineName.Render(tf.Title.View()))

	return st.ListItem.Render(sb.String())
}
