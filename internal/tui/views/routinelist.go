package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
)

type RoutineList struct {
	Routines   []models.Routine
	Entries    map[string][]models.RoutineEntry // routineID -> all entries
	DateVals   map[string]int                 // routineID -> value for selected date
	Date       string                         // selected date (YYYY-MM-DD)
	Index      int
	Width      int
	Offset     int // scroll offset
	MaxVisible int
}

func NewRoutineList() RoutineList {
	return RoutineList{
		Entries:  make(map[string][]models.RoutineEntry),
		DateVals: make(map[string]int),
	}
}

func (hl *RoutineList) SetData(routines []models.Routine, entries map[string][]models.RoutineEntry, vals map[string]int, date string) {
	hl.Routines = routines
	hl.Entries = entries
	hl.DateVals = vals
	hl.Date = date
	if hl.Index >= len(routines) && len(routines) > 0 {
		hl.Index = len(routines) - 1
	}
	if len(routines) == 0 {
		hl.Index = 0
	}
	hl.ClampScroll()
}

func (hl *RoutineList) SelectedRoutine() *models.Routine {
	if len(hl.Routines) == 0 || hl.Index >= len(hl.Routines) {
		return nil
	}
	return &hl.Routines[hl.Index]
}

func (hl *RoutineList) MoveUp() {
	if len(hl.Routines) == 0 {
		return
	}
	hl.Index--
	if hl.Index < 0 {
		hl.Index = len(hl.Routines) - 1
	}
	hl.ClampScroll()
}

func (hl *RoutineList) MoveDown() {
	if len(hl.Routines) == 0 {
		return
	}
	hl.Index++
	if hl.Index >= len(hl.Routines) {
		hl.Index = 0
	}
	hl.ClampScroll()
}

func (hl *RoutineList) ClampScroll() {
	if hl.Index < hl.Offset {
		hl.Offset = hl.Index
	}
	if hl.MaxVisible > 0 && hl.Index >= hl.Offset+hl.MaxVisible {
		hl.Offset = hl.Index - hl.MaxVisible + 1
	}
}

// Click converts a mouse y position (relative to the list's first line) to a
// routine index. Returns -1 if the click is outside the visible rows. The x
// position determines whether the click is on the status icon (first 4 columns).
func (hl *RoutineList) Click(y, x int) (index int, onIcon bool) {
	if y < 0 || y >= hl.MaxVisible {
		return -1, false
	}
	idx := hl.Offset + y
	if idx >= len(hl.Routines) {
		return -1, false
	}
	return idx, x >= 0 && x < 4
}

func (hl *RoutineList) SetSize(height int) {
	// Reserve space for title bar (1), help bar (1), scroll hint (1)
	hl.MaxVisible = height - 3
	if hl.MaxVisible < 1 {
		hl.MaxVisible = 1
	}
}

func (hl *RoutineList) Render(st Styles) string {
	if len(hl.Routines) == 0 {
		return "  No routines yet.\n\n  Press n to create your first routine.\n"
	}

	var sb strings.Builder
	end := hl.Offset + hl.MaxVisible
	if end > len(hl.Routines) {
		end = len(hl.Routines)
	}

	for i := hl.Offset; i < end; i++ {
		h := hl.Routines[i]
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

func (hl *RoutineList) renderRow(h models.Routine, st Styles) string {
	entries := hl.Entries[h.ID]
	periodVal := steward.PeriodValue(entries, h, hl.Date)
	done := steward.IsPeriodDone(entries, h, hl.Date)
	partial := periodVal > 0 && !done

	icon := st.StatusUndone.Render("○")
	name := st.RoutineName.Render(h.Name)
	if done {
		icon = st.StatusDone.Render("●")
		name = st.RoutineNameDone.Render(h.Name)
	} else if partial {
		icon = st.StatusPartial.Render("◐")
	}

	prog := "—"
	if h.QuantityType == "count" {
		prog = fmt.Sprintf("%d/%d", periodVal, h.TargetValue)
	} else if done {
		prog = "✓"
	}
	prog = st.Progress.Render(prog)

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
