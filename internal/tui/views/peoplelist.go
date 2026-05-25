package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
)

type PeopleList struct {
	People     []models.Person
	Dates      map[string][]models.SignificantDate
	Index      int
	Offset     int
	MaxVisible int
}

func NewPeopleList() PeopleList {
	return PeopleList{
		Dates: make(map[string][]models.SignificantDate),
	}
}

func (pl *PeopleList) SetData(people []models.Person, dates map[string][]models.SignificantDate) {
	pl.People = people
	pl.Dates = dates
	if pl.Index >= len(people) && len(people) > 0 {
		pl.Index = len(people) - 1
	}
	if len(people) == 0 {
		pl.Index = 0
	}
	pl.ClampScroll()
}

func (pl *PeopleList) SelectedPerson() *models.Person {
	if len(pl.People) == 0 || pl.Index >= len(pl.People) {
		return nil
	}
	return &pl.People[pl.Index]
}

func (pl *PeopleList) MoveUp() {
	if len(pl.People) == 0 {
		return
	}
	pl.Index--
	if pl.Index < 0 {
		pl.Index = len(pl.People) - 1
	}
	pl.ClampScroll()
}

func (pl *PeopleList) MoveDown() {
	if len(pl.People) == 0 {
		return
	}
	pl.Index++
	if pl.Index >= len(pl.People) {
		pl.Index = 0
	}
	pl.ClampScroll()
}

func (pl *PeopleList) ClampScroll() {
	if pl.Index < pl.Offset {
		pl.Offset = pl.Index
	}
	if pl.MaxVisible > 0 && pl.Index >= pl.Offset+pl.MaxVisible {
		pl.Offset = pl.Index - pl.MaxVisible + 1
	}
}

func (pl *PeopleList) Click(y, x int) (index int, onBump bool) {
	if y < 0 || y >= pl.MaxVisible {
		return -1, false
	}
	idx := pl.Offset + y
	if idx >= len(pl.People) {
		return -1, false
	}
	return idx, x >= 0 && x < 4
}

func (pl *PeopleList) SetSize(height int) {
	pl.MaxVisible = height - 4
	if pl.MaxVisible < 1 {
		pl.MaxVisible = 1
	}
}

func (pl *PeopleList) Render(st Styles) string {
	if len(pl.People) == 0 {
		return "  No people yet.\n\n  Press n to add your first person.\n"
	}

	var sb strings.Builder
	end := pl.Offset + pl.MaxVisible
	if end > len(pl.People) {
		end = len(pl.People)
	}

	for i := pl.Offset; i < end; i++ {
		p := pl.People[i]

		if i == pl.Index {
			sb.WriteString(st.StatusDone.Render("▸ "))
		} else {
			sb.WriteString("  ")
		}
		sb.WriteString(pl.renderRow(p, st))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (pl *PeopleList) renderRow(p models.Person, st Styles) string {
	days := steward.DaysSinceContact(p.LastContacted)

	var statusIcon string
	var daysStr string
	var daysStyle lipgloss.Style

	switch {
	case days < 0:
		statusIcon = st.StatusUndone.Render("○")
		daysStr = "never"
		daysStyle = st.PersonDaysStale
	case days == 0:
		statusIcon = st.StatusDone.Render("●")
		daysStr = "today"
		daysStyle = st.PersonDaysNew
	case days <= 7:
		statusIcon = st.StatusDone.Render("●")
		daysStr = fmt.Sprintf("%dd ago", days)
		daysStyle = st.PersonDaysRecent
	case days <= 30:
		statusIcon = st.StatusPartial.Render("◐")
		daysStr = fmt.Sprintf("%dd ago", days)
		daysStyle = st.PersonDaysOld
	default:
		statusIcon = st.StatusUndone.Render("○")
		daysStr = fmt.Sprintf("%dd ago", days)
		daysStyle = st.PersonDaysStale
	}

	pad := func(s string, w int) string {
		vis := lipgloss.Width(s)
		if vis >= w {
			return s
		}
		return s + strings.Repeat(" ", w-vis)
	}

	iconCol := " " + statusIcon + "  "
	nameCol := pad(st.PersonName.Render(p.Name), 25)
	daysCol := pad(daysStyle.Render(daysStr), 9)

	// Info icons
	var infoCol string
	if p.Email != "" || p.Phone != "" {
		var icons []string
		if p.Email != "" {
			icons = append(icons, "@")
		}
		if p.Phone != "" {
			icons = append(icons, "tel")
		}
		infoCol = pad(st.PersonInfoIcon.Render(strings.Join(icons, " ")), 8)
	}

	// Significant dates
	var datesCol string
	if dates, ok := pl.Dates[p.ID]; ok && len(dates) > 0 {
		var parts []string
		for _, d := range dates {
			parts = append(parts, d.Label+" "+d.Date)
		}
		datesCol = st.PersonInfoIcon.Render(strings.Join(parts, ", "))
	}

	return iconCol + nameCol + daysCol + infoCol + datesCol
}
