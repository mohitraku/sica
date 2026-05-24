package views

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mohitraku/sica/internal/models"
)

type TaskList struct {
	Tasks      []models.Task
	Index      int
	Offset     int
	MaxVisible int
}

func NewTaskList() TaskList {
	return TaskList{}
}

func (tl *TaskList) SetData(tasks []models.Task) {
	tl.Tasks = tasks
	if tl.Index >= len(tasks) && len(tasks) > 0 {
		tl.Index = len(tasks) - 1
	}
	if len(tasks) == 0 {
		tl.Index = 0
	}
	tl.ClampScroll()
}

func (tl *TaskList) SelectedTask() *models.Task {
	if len(tl.Tasks) == 0 || tl.Index >= len(tl.Tasks) {
		return nil
	}
	return &tl.Tasks[tl.Index]
}

func (tl *TaskList) MoveUp() {
	if len(tl.Tasks) == 0 {
		return
	}
	tl.Index--
	if tl.Index < 0 {
		tl.Index = len(tl.Tasks) - 1
	}
	tl.ClampScroll()
}

func (tl *TaskList) MoveDown() {
	if len(tl.Tasks) == 0 {
		return
	}
	tl.Index++
	if tl.Index >= len(tl.Tasks) {
		tl.Index = 0
	}
	tl.ClampScroll()
}

func (tl *TaskList) ClampScroll() {
	if tl.Index < tl.Offset {
		tl.Offset = tl.Index
	}
	if tl.MaxVisible > 0 && tl.Index >= tl.Offset+tl.MaxVisible {
		tl.Offset = tl.Index - tl.MaxVisible + 1
	}
}

func (tl *TaskList) Click(y, x int) (index int, onIcon bool) {
	if y < 0 || y >= tl.MaxVisible {
		return -1, false
	}
	idx := tl.Offset + y
	if idx >= len(tl.Tasks) {
		return -1, false
	}
	return idx, x >= 0 && x < 4
}

func (tl *TaskList) SetSize(height int) {
	tl.MaxVisible = height - 3
	if tl.MaxVisible < 1 {
		tl.MaxVisible = 1
	}
}

func (tl *TaskList) Render(st Styles) string {
	if len(tl.Tasks) == 0 {
		return "  No tasks yet.\n\n  Press n to create your first task.\n"
	}

	var sb strings.Builder
	end := tl.Offset + tl.MaxVisible
	if end > len(tl.Tasks) {
		end = len(tl.Tasks)
	}

	prevDone := false
	for i := tl.Offset; i < end; i++ {
		t := tl.Tasks[i]

		// Separator between undone and done sections
		if i > tl.Offset && t.Done && !prevDone {
			sb.WriteString(st.HelpHint.Render("  ──────────────────"))
			sb.WriteByte('\n')
		}
		prevDone = t.Done

		if i == tl.Index {
			sb.WriteString(st.StatusDone.Render("▸ "))
		} else {
			sb.WriteString("  ")
		}
		sb.WriteString(tl.renderRow(t, st))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (tl *TaskList) renderRow(t models.Task, st Styles) string {
	icon := st.StatusUndone.Render("○")
	title := st.RoutineName.Render(t.Title)
	if t.Done {
		icon = st.StatusDone.Render("●")
		title = st.RoutineNameDone.Render(t.Title)
	}

	pad := func(s string, w int) string {
		vis := lipgloss.Width(s)
		if vis >= w {
			return s
		}
		return s + strings.Repeat(" ", w-vis)
	}

	iconCol := " " + icon + "  "
	titleCol := pad(title, 40)

	return iconCol + titleCol
}
