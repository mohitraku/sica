package tui

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/mohitraku/sica/internal/config"
	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
	"github.com/mohitraku/sica/internal/storage"
	"github.com/mohitraku/sica/internal/tui/views"
)

type AppMode int

const (
	ModeRoutines AppMode = iota
	ModeTasks
)

type Model struct {
	rStore *storage.RoutineStore
	eStore *storage.EntryStore
	tStore *storage.TaskStore

	styles        views.Styles
	keys          keyMap
	rl            views.RoutineList
	tl            views.TaskList
	helpBar       views.HelpBar
	form          views.RoutineForm
	taskForm      views.TaskForm
	confirm       views.Confirm
	confirmAction func()
	settings      views.Settings

	showHelp      bool
	selectedDate  string
	dataDir       string
	dataDirSource string
	mode          AppMode

	width  int
	height int
	ready  bool
}

func New(rStore *storage.RoutineStore, eStore *storage.EntryStore, tStore *storage.TaskStore, dataDir, dataDirSource string) *Model {
	return &Model{
		rStore:        rStore,
		eStore:        eStore,
		tStore:        tStore,
		styles:        views.BuildStyles(false),
		keys:          keys,
		rl:            views.NewRoutineList(),
		tl:            views.NewTaskList(),
		helpBar:       views.NewHelpBar(),
		form:          views.NewRoutineForm(),
		taskForm:      views.NewTaskForm(),
		confirm:       views.NewConfirm(),
		settings:      views.NewSettings(),
		selectedDate:  models.Today(),
		dataDir:       dataDir,
		dataDirSource: dataDirSource,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m *Model) loadData() {
	routines, _ := m.rStore.List()

	entries := make(map[string][]models.RoutineEntry)
	dateVals := make(map[string]int)

	for _, r := range routines {
		e, _ := m.eStore.GetForRoutine(r.ID)
		entries[r.ID] = e
		dateVals[r.ID], _ = m.eStore.GetValue(r.ID, m.selectedDate)
	}

	m.rl.SetData(routines, entries, dateVals, m.selectedDate)

	tasks, _ := m.tStore.List()
	m.tl.SetData(tasks)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog intercepts all keys
	if m.confirm.Active {
		confirmed, _ := m.confirm.Update(msg)
		if confirmed && m.confirmAction != nil {
			m.confirmAction()
			m.confirmAction = nil
		}
		return m, nil
	}

	// Form intercepts keys
	if m.form.Active() {
		return m.handleFormMsg(msg)
	}
	if m.taskForm.Active() {
		return m.handleTaskFormMsg(msg)
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

	// Settings overlay intercepts keys
	if m.settings.Active {
		saved, newPath := m.settings.Update(msg)
		if saved {
			config.Save(config.Config{DataDir: newPath})
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
		m.rl.SetSize(m.height)
		m.tl.SetSize(m.height)
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
			name, freq, qty, target := m.form.GetRoutineFields()
			if err := steward.ValidateName(name); err != nil {
				m.form.SetError(err.Error())
				return m, nil
			}
			now := models.NowUTC()
			if m.form.Mode == views.FormNew {
				h := &models.Routine{
					ID:           newID(),
					Name:         name,
					Frequency:    freq,
					QuantityType: qty,
					TargetValue:  target,
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				steward.NormalizeRoutine(h)
				m.rStore.Create(h)
			} else {
				existing, _ := m.rStore.GetByID(m.form.EditRoutineID)
				if existing != nil {
					existing.Name = name
					existing.Frequency = freq
					existing.QuantityType = qty
					existing.TargetValue = target
					existing.UpdatedAt = now
					steward.NormalizeRoutine(existing)
					m.rStore.Update(existing)
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

func (m *Model) handleTaskFormMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.taskForm.Cancel()
			return m, nil
		case "enter":
			title := m.taskForm.GetTitle()
			if err := steward.ValidateTaskTitle(title); err != nil {
				m.taskForm.SetError(err.Error())
				return m, nil
			}
			now := models.NowUTC()
			if m.taskForm.Mode == views.FormNew {
				t := &models.Task{
					ID:        newID(),
					Title:     title,
					Done:      false,
					CreatedAt: now,
					UpdatedAt: now,
				}
				m.tStore.Create(t)
			} else {
				existing, _ := m.tStore.GetByID(m.taskForm.EditID)
				if existing != nil {
					existing.Title = title
					existing.UpdatedAt = now
					m.tStore.Update(existing)
				}
			}
			m.taskForm.Cancel()
			m.loadData()
			return m, nil
		}
	}
	cmd := m.taskForm.Update(msg)
	return m, cmd
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}

		if m.mode == ModeTasks {
			idx, onIcon := m.tl.Click(msg.Y-1, msg.X)
			if idx < 0 {
				return m, nil
			}
			if onIcon && idx < len(m.tl.Tasks) {
				t := m.tl.Tasks[idx]
				t.Done = !t.Done
				t.UpdatedAt = models.NowUTC()
				m.tStore.Update(&t)
				m.loadData()
			} else {
				m.tl.Index = idx
				m.tl.ClampScroll()
			}
			return m, nil
		}

		idx, onIcon := m.rl.Click(msg.Y-1, msg.X)
		if idx < 0 {
			return m, nil
		}
		if onIcon && idx < len(m.rl.Routines) {
			h := m.rl.Routines[idx]
			cur, _ := m.eStore.GetValue(h.ID, m.selectedDate)
			next := steward.IncrementValue(cur, h.TargetValue)
			m.eStore.SetValue(h.ID, m.selectedDate, next)
			m.loadData()
		} else {
			m.rl.Index = idx
			m.rl.ClampScroll()
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.mode == ModeTasks {
				m.tl.MoveUp()
			} else {
				m.rl.MoveUp()
			}
		case tea.MouseWheelDown:
			if m.mode == ModeTasks {
				m.tl.MoveDown()
			} else {
				m.rl.MoveDown()
			}
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

	case key.Matches(msg, m.keys.Settings):
		configuredPath := ""
		cfg, _ := config.Load()
		configuredPath = cfg.DataDir
		m.settings.Show(m.dataDir, m.dataDirSource, configuredPath)
		return m, nil

	case key.Matches(msg, m.keys.SwitchMode):
		if m.mode == ModeRoutines {
			m.mode = ModeTasks
		} else {
			m.mode = ModeRoutines
		}
		return m, nil
	}

	if m.mode == ModeTasks {
		return m.handleTaskKey(msg)
	}

	// Routine mode
	switch {
	case key.Matches(msg, m.keys.Up):
		m.rl.MoveUp()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.rl.MoveDown()
		return m, nil

	case key.Matches(msg, m.keys.Increment):
		h := m.rl.SelectedRoutine()
		if h == nil {
			return m, nil
		}
		cur, _ := m.eStore.GetValue(h.ID, m.selectedDate)
		next := steward.IncrementValue(cur, h.TargetValue)
		m.eStore.SetValue(h.ID, m.selectedDate, next)
		m.loadData()
		return m, nil

	case key.Matches(msg, m.keys.Decrement):
		h := m.rl.SelectedRoutine()
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
		h := m.rl.SelectedRoutine()
		if h == nil {
			return m, nil
		}
		m.form.StartEdit(h.ID, h.Name, h.Frequency, h.QuantityType, h.TargetValue)
		return m, nil

	case key.Matches(msg, m.keys.Delete):
		h := m.rl.SelectedRoutine()
		if h == nil {
			return m, nil
		}
		m.confirmAction = func() {
			m.rStore.Delete(h.ID)
			m.loadData()
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

func (m *Model) handleTaskKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.tl.MoveUp()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.tl.MoveDown()
		return m, nil

	case key.Matches(msg, m.keys.New):
		m.taskForm.StartNew()
		return m, nil

	case key.Matches(msg, m.keys.Edit):
		t := m.tl.SelectedTask()
		if t == nil {
			return m, nil
		}
		m.taskForm.StartEdit(t.ID, t.Title)
		return m, nil

	case key.Matches(msg, m.keys.Delete):
		t := m.tl.SelectedTask()
		if t == nil {
			return m, nil
		}
		m.confirmAction = func() {
			m.tStore.Delete(t.ID)
			m.loadData()
		}
		m.confirm.Show("Delete \"" + t.Title + "\"?")
		return m, nil

	case key.Matches(msg, m.keys.Toggle):
		t := m.tl.SelectedTask()
		if t == nil {
			return m, nil
		}
		t.Done = !t.Done
		t.UpdatedAt = models.NowUTC()
		m.tStore.Update(t)
		m.loadData()
		return m, nil

	case key.Matches(msg, m.keys.ClearDone):
		m.confirmAction = func() {
			m.tStore.DeleteCompleted()
			m.loadData()
		}
		m.confirm.Show("Clear all completed tasks?")
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

	// Settings overlay
	if m.settings.Active {
		settingsView := m.settings.Render(m.styles)
		v := tea.NewView(settingsView)
		v.AltScreen = true
		return v
	}

	var sb strings.Builder

	// Title bar
	var titleLeft string
	if m.mode == ModeTasks {
		titleLeft = m.styles.AppName.Render("Sica ") +
			m.styles.DateLabel.Render("Tasks")
	} else if models.IsToday(m.selectedDate) {
		titleLeft = m.styles.AppName.Render("Sica ") +
			m.styles.DateLabel.Render(m.selectedDate)
	} else {
		arrow := m.styles.StatusPartial.Render("◀ ")
		titleLeft = m.styles.AppName.Render("Sica ") +
			arrow + m.styles.DateLabel.Render(m.selectedDate)
	}
	titleRight := m.styles.HelpHint.Render("S settings  ? help  q quit")
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

	// Main list
	if m.mode == ModeTasks {
		sb.WriteString(m.tl.Render(m.styles))

		if m.taskForm.Active() {
			sb.WriteByte('\n')
			sb.WriteString(m.taskForm.Render(m.styles))
		}

		if len(m.tl.Tasks) > m.tl.MaxVisible {
			sb.WriteByte('\n')
			sb.WriteString(m.styles.HelpHint.Render("  … more below"))
		}
	} else {
		sb.WriteString(m.rl.Render(m.styles))

		if m.form.Active() {
			sb.WriteByte('\n')
			sb.WriteString(m.form.Render(m.styles))
		}

		if len(m.rl.Routines) > m.rl.MaxVisible {
			sb.WriteByte('\n')
			sb.WriteString(m.styles.HelpHint.Render("  … more below"))
		}
	}

	// Help bar
	var helpBindings []key.Binding
	if m.form.Active() || m.taskForm.Active() {
		helpBindings = []key.Binding{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	} else if m.mode == ModeTasks {
		helpBindings = []key.Binding{
			m.keys.New, m.keys.Edit,
			m.keys.Toggle, m.keys.ClearDone,
			m.keys.Delete, m.keys.SwitchMode, m.keys.Help,
		}
	} else {
		helpBindings = []key.Binding{
			m.keys.New, m.keys.Edit,
			m.keys.Increment, m.keys.Decrement,
			m.keys.Delete, m.keys.Settings, m.keys.Help,
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
