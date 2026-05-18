package tui

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/mojitrk/sica/internal/models"
	"github.com/mojitrk/sica/internal/steward"
	"github.com/mojitrk/sica/internal/storage"
	"github.com/mojitrk/sica/internal/tui/views"
)

type Model struct {
	hStore *storage.HabitStore
	eStore *storage.EntryStore

	styles  views.Styles
	keys    keyMap
	hl      views.HabitList
	helpBar views.HelpBar
	form    views.HabitForm
	confirm views.Confirm

	showHelp     bool
	selectedDate string

	width  int
	height int
	ready  bool
}

func New(hStore *storage.HabitStore, eStore *storage.EntryStore) *Model {
	return &Model{
		hStore:       hStore,
		eStore:       eStore,
		styles:       views.BuildStyles(false),
		keys:         keys,
		hl:           views.NewHabitList(),
		helpBar:      views.NewHelpBar(),
		form:         views.NewHabitForm(),
		confirm:      views.NewConfirm(),
		selectedDate: models.Today(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m *Model) loadData() {
	habits, _ := m.hStore.List()

	entries := make(map[string][]models.HabitEntry)
	dateVals := make(map[string]int)

	for _, h := range habits {
		e, _ := m.eStore.GetForHabit(h.ID)
		entries[h.ID] = e
		dateVals[h.ID], _ = m.eStore.GetValue(h.ID, m.selectedDate)
	}

	m.hl.SetData(habits, entries, dateVals, m.selectedDate)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog intercepts all keys
	if m.confirm.Active {
		confirmed, _ := m.confirm.Update(msg)
		if confirmed {
			h := m.hl.SelectedHabit()
			if h != nil {
				m.hStore.Delete(h.ID)
				m.loadData()
			}
		}
		return m, nil
	}

	// Form intercepts keys
	if m.form.Active() {
		return m.handleFormMsg(msg)
	}

	// Help overlay intercepts keys
	if m.showHelp {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			if msg.String() == "?" || msg.String() == "esc" {
				m.showHelp = false
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.styles = views.BuildStyles(msg.IsDark())
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.hl.SetSize(m.height)
		if !m.ready {
			m.loadData()
			m.ready = true
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleFormMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.form.Cancel()
			return m, nil
		case "enter":
			// New/Edit form submission
			name, freq, qty, target := m.form.GetHabitFields()
			if err := steward.ValidateName(name); err != nil {
				m.form.SetError(err.Error())
				return m, nil
			}
			now := models.NowUTC()
			if m.form.Mode == views.FormNew {
				h := &models.Habit{
					ID:           newID(),
					Name:         name,
					Frequency:    freq,
					QuantityType: qty,
					TargetValue:  target,
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				steward.NormalizeHabit(h)
				m.hStore.Create(h)
			} else {
				existing, _ := m.hStore.GetByID(m.form.EditHabitID)
				if existing != nil {
					existing.Name = name
					existing.Frequency = freq
					existing.QuantityType = qty
					existing.TargetValue = target
					existing.UpdatedAt = now
					steward.NormalizeHabit(existing)
					m.hStore.Update(existing)
				}
			}
			m.form.Cancel()
			m.loadData()
			return m, nil
		}
	}
	cmd := m.form.Update(msg)
	return m, cmd
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		idx, onIcon := m.hl.Click(msg.Y-1, msg.X)
		if idx < 0 {
			return m, nil
		}
		if onIcon && idx < len(m.hl.Habits) {
			h := m.hl.Habits[idx]
			cur, _ := m.eStore.GetValue(h.ID, m.selectedDate)
			next := steward.IncrementValue(cur, h.TargetValue)
			m.eStore.SetValue(h.ID, m.selectedDate, next)
			m.loadData()
		} else {
			m.hl.Index = idx
			m.hl.ClampScroll()
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.hl.MoveUp()
		case tea.MouseWheelDown:
			m.hl.MoveDown()
		}
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Up):
		m.hl.MoveUp()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.hl.MoveDown()
		return m, nil

	case key.Matches(msg, m.keys.Increment):
		h := m.hl.SelectedHabit()
		if h == nil {
			return m, nil
		}
		cur, _ := m.eStore.GetValue(h.ID, m.selectedDate)
		next := steward.IncrementValue(cur, h.TargetValue)
		m.eStore.SetValue(h.ID, m.selectedDate, next)
		m.loadData()
		return m, nil

	case key.Matches(msg, m.keys.Decrement):
		h := m.hl.SelectedHabit()
		if h == nil {
			return m, nil
		}
		cur, _ := m.eStore.GetValue(h.ID, m.selectedDate)
		next := steward.DecrementValue(cur)
		m.eStore.SetValue(h.ID, m.selectedDate, next)
		m.loadData()
		return m, nil

	case key.Matches(msg, m.keys.New):
		m.form.StartNew()
		return m, nil

	case key.Matches(msg, m.keys.Edit):
		h := m.hl.SelectedHabit()
		if h == nil {
			return m, nil
		}
		m.form.StartEdit(h.ID, h.Name, h.Frequency, h.QuantityType, h.TargetValue)
		return m, nil

	case key.Matches(msg, m.keys.Delete):
		h := m.hl.SelectedHabit()
		if h == nil {
			return m, nil
		}
		m.confirm.Show("Delete \"" + h.Name + "\"?")
		return m, nil

	case key.Matches(msg, m.keys.PrevDay):
		t, err := time.Parse(models.DateLayout, m.selectedDate)
		if err == nil {
			m.selectedDate = t.AddDate(0, 0, -1).Format(models.DateLayout)
			m.loadData()
		}
		return m, nil

	case key.Matches(msg, m.keys.NextDay):
		t, err := time.Parse(models.DateLayout, m.selectedDate)
		if err == nil {
			next := t.AddDate(0, 0, 1).Format(models.DateLayout)
			if next <= models.Today() {
				m.selectedDate = next
				m.loadData()
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Today):
		if !models.IsToday(m.selectedDate) {
			m.selectedDate = models.Today()
			m.loadData()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Loading...")
	}

	// Full help overlay
	if m.showHelp {
		return m.helpOverlay()
	}

	// Confirm dialog overlay
	if m.confirm.Active {
		confirmView := m.confirm.Render(m.styles)
		v := tea.NewView(confirmView)
		v.AltScreen = true
		return v
	}

	var sb strings.Builder

	// Title bar
	var titleLeft string
	if models.IsToday(m.selectedDate) {
		titleLeft = m.styles.AppName.Render("Sica ") +
			m.styles.DateLabel.Render(m.selectedDate)
	} else {
		arrow := m.styles.StatusPartial.Render("◀ ")
		titleLeft = m.styles.AppName.Render("Sica ") +
			arrow + m.styles.DateLabel.Render(m.selectedDate)
	}
	titleRight := m.styles.HelpHint.Render("↑↓ nav  ? help  q quit")
	titleGap := m.width - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight) - 4
	if titleGap < 1 {
		titleGap = 1
	}
	sb.WriteString(m.styles.TitleBar.Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			titleLeft,
			strings.Repeat(" ", titleGap),
			titleRight,
		)))
	sb.WriteByte('\n')

	// Habit list or empty state
	sb.WriteString(m.hl.Render(m.styles))

	// Form when active
	if m.form.Active() {
		sb.WriteByte('\n')
		sb.WriteString(m.form.Render(m.styles))
	}

	// Scroll hint when list overflows
	if len(m.hl.Habits) > m.hl.MaxVisible {
		sb.WriteByte('\n')
		sb.WriteString(m.styles.HelpHint.Render("  … more below"))
	}

	// Help bar
	var helpBindings []key.Binding
	if m.form.Active() {
		helpBindings = []key.Binding{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	} else {
		helpBindings = []key.Binding{
			m.keys.New, m.keys.Edit,
			m.keys.Increment, m.keys.Decrement,
			m.keys.Delete, m.keys.Help,
			m.keys.PrevDay, m.keys.NextDay, m.keys.Today,
		}
	}
	m.helpBar.SetBindings(helpBindings)
	sb.WriteByte('\n')
	sb.WriteString(m.helpBar.Render(m.styles))

	v := tea.NewView(sb.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *Model) helpOverlay() tea.View {
	groups := m.keys.FullHelp()
	var sb strings.Builder
	sb.WriteString(m.styles.AppName.Render("Keybindings"))
	sb.WriteString("\n\n")

	for _, group := range groups {
		for _, b := range group {
			h := b.Help()
			key := m.styles.OverlayKey.Render("  " + h.Key)
			val := m.styles.OverlayVal.Render(h.Desc)
			sb.WriteString(key + "  " + val)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(m.styles.HelpHint.Render("  Press ? or esc to close"))

	content := m.styles.Overlay.Render(sb.String())
	placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}

func newID() string {
	var b [8]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
