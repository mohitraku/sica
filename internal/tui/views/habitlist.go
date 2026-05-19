package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
)

type HabitList struct {
	Habits     []models.Habit
	Entries    map[string][]models.HabitEntry // habitID -> all entries
	DateVals   map[string]int                 // habitID -> value for selected date
	Date       string                         // selected date (YYYY-MM-DD)
	Index      int
	Width      int
	Offset     int // scroll offset
	MaxVisible int
}

func NewHabitList() HabitList {
	return HabitList{
		Entries:  make(map[string][]models.HabitEntry),
		DateVals: make(map[string]int),
	}
}

func (hl *HabitList) SetData(habits []models.Habit, entries map[string][]models.HabitEntry, vals map[string]int, date string) {
	hl.Habits = habits
	hl.Entries = entries
	hl.DateVals = vals
	hl.Date = date
	if hl.Index >= len(habits) && len(habits) > 0 {
		hl.Index = len(habits) - 1
	}
	if len(habits) == 0 {
		hl.Index = 0
	}
	hl.clampScroll()
}

func (hl *HabitList) SelectedIndex() int {
	return hl.Index
}

func (hl *HabitList) SelectedHabit() *models.Habit {
	if len(hl.Habits) == 0 || hl.Index >= len(hl.Habits) {
		return nil
	}
	return &hl.Habits[hl.Index]
}

func (hl *HabitList) MoveUp() {
	if len(hl.Habits) == 0 {
		return
	}
	hl.Index--
	if hl.Index < 0 {
		hl.Index = len(hl.Habits) - 1
	}
	hl.clampScroll()
}

func (hl *HabitList) MoveDown() {
	if len(hl.Habits) == 0 {
		return
	}
	hl.Index++
	if hl.Index >= len(hl.Habits) {
		hl.Index = 0
	}
	hl.clampScroll()
}

func (hl *HabitList) ClampScroll() {
	if hl.Index < hl.Offset {
		hl.Offset = hl.Index
	}
	if hl.MaxVisible > 0 && hl.Index >= hl.Offset+hl.MaxVisible {
		hl.Offset = hl.Index - hl.MaxVisible + 1
	}
}

func (hl *HabitList) clampScroll() {
	hl.ClampScroll()
}

// Click converts a mouse y position (relative to the list's first line) to a
// habit index. Returns -1 if the click is outside the visible rows. The x
// position determines whether the click is on the status icon (first 4 columns).
func (hl *HabitList) Click(y, x int) (index int, onIcon bool) {
	if y < 0 || y >= hl.MaxVisible {
		return -1, false
	}
	idx := hl.Offset + y
	if idx >= len(hl.Habits) {
		return -1, false
	}
	return idx, x >= 0 && x < 4
}

func (hl *HabitList) SetSize(height int) {
	// Reserve space for title bar (1), help bar (1), scroll hint (1)
	hl.MaxVisible = height - 3
	if hl.MaxVisible < 1 {
		hl.MaxVisible = 1
	}
}

func (hl *HabitList) Render(st Styles) string {
	if len(hl.Habits) == 0 {
		return "  No habits yet.\n\n  Press n to create your first habit.\n"
	}

	var sb strings.Builder
	end := hl.Offset + hl.MaxVisible
	if end > len(hl.Habits) {
		end = len(hl.Habits)
	}

	for i := hl.Offset; i < end; i++ {
		h := hl.Habits[i]
		if i == hl.Index {
			sb.WriteString(st.StatusDone.Render("▸ "))
		} else {
			sb.WriteString("  ")
		}
		sb.WriteString(hl.renderRow(h, st))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (hl *HabitList) renderRow(h models.Habit, st Styles) string {
	val := hl.DateVals[h.ID]
	done := steward.IsDone(val, h.TargetValue)
	partial := val > 0 && !done

	icon := st.StatusUndone.Render("○")
	name := st.HabitName.Render(h.Name)
	if done {
		icon = st.StatusDone.Render("●")
		name = st.HabitNameDone.Render(h.Name)
	} else if partial {
		icon = st.StatusPartial.Render("◐")
	}

	prog := "—"
	if h.QuantityType == "count" {
		prog = fmt.Sprintf("%d/%d", val, h.TargetValue)
	} else if done {
		prog = "✓"
	}
	prog = st.Progress.Render(prog)

	entries := hl.Entries[h.ID]
	cur := steward.CurrentStreak(entries, h, hl.Date)
	longest := steward.LongestStreak(entries, h)
	streakCur := st.StreakCurrent.Render(fmt.Sprintf("↑%2dd", cur))
	streakLong := st.StreakLongest.Render(fmt.Sprintf("⇈%2dd", longest))

	freq := st.FreqLabel.Render(h.Frequency)

	// Width-aware padding — lipgloss.Width ignores ANSI escapes.
	pad := func(s string, w int) string {
		vis := lipgloss.Width(s)
		if vis >= w {
			return s
		}
		return s + strings.Repeat(" ", w-vis)
	}

	iconCol := " " + icon + "  "
	nameCol := pad(name, 22)
	progCol := pad(prog, 8)
	streakCol := pad(streakCur+"  "+streakLong, 16)
	freqCol := pad(freq, 8)

	return iconCol + nameCol + progCol + streakCol + freqCol
}
