package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/server"
	"github.com/mojitrk/sica/internal/tasks"
)

type Tab int

const (
	TabHabits Tab = iota
	TabTasks
	TabFinance
	TabCalendar
	TabKnowledge
	TabChat
)

func (t Tab) String() string {
	return []string{"Habits", "Tasks", "Finance", "Calendar", "Knowledge", "Chat"}[t]
}

var tabList = []Tab{TabHabits, TabTasks, TabFinance, TabCalendar, TabKnowledge, TabChat}

type mode int

const (
	modeList mode = iota
	modeCreate
)

type Model struct {
	width      int
	height     int
	activeTab  Tab
	viewport   viewport.Model
	ready      bool
	mode       mode
	input      textinput.Model
	createFreq string

	hStore *habits.Store
	tStore *tasks.Store

	habits     []core.Habit
	taskList   []core.Task
	selected   int
}

func New(deps server.Deps) *Model {
	ti := textinput.New()
	ti.Placeholder = "Name..."
	ti.CharLimit = 100

	return &Model{
		activeTab:  TabHabits,
		mode:       modeList,
		input:      ti,
		createFreq: "daily",
		hStore:     deps.HabitsStore,
		tStore:     deps.TasksStore,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		sidebarW := 16
		contentW := m.width - sidebarW - 1
		if !m.ready {
			m.viewport = viewport.New(contentW, m.height-3)
			m.ready = true
		} else {
			m.viewport.Width = contentW
			m.viewport.Height = m.height - 3
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.mode == modeCreate {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeCreate {
		switch msg.String() {
		case "esc":
			m.mode = modeList
			m.input.Reset()
			return m, nil
		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				switch m.activeTab {
				case TabHabits:
					m.hStore.Create(&core.Habit{Name: val, Frequency: m.createFreq})
				case TabTasks:
					m.tStore.CreateTask(&core.Task{Title: val, Status: "todo", Priority: "med"})
				}
			}
			m.mode = modeList
			m.input.Reset()
			m.refresh()
			return m, nil
		case "tab":
			if m.activeTab == TabHabits {
				if m.createFreq == "daily" {
					m.createFreq = "weekly"
				} else {
					m.createFreq = "daily"
				}
			}
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "1", "2", "3", "4", "5", "6":
		idx := int(msg.Runes[0] - '1')
		m.activeTab = Tab(idx)
		m.selected = 0
		m.mode = modeList
		m.input.Reset()
		m.refresh()

	case "tab":
		m.activeTab = (m.activeTab + 1) % Tab(len(tabList))
		m.selected = 0
		m.mode = modeList
		m.refresh()

	case "shift+tab":
		m.activeTab = (m.activeTab - 1 + Tab(len(tabList))) % Tab(len(tabList))
		m.selected = 0
		m.mode = modeList
		m.refresh()

	case "j", "down":
		m.selected++
		m.clampSelection()
		m.viewport.SetContent(m.contentForTab(m.activeTab))

	case "k", "up":
		m.selected--
		m.clampSelection()
		m.viewport.SetContent(m.contentForTab(m.activeTab))

	case "g":
		m.selected = 0
		m.viewport.GotoTop()

	case "G":
		switch m.activeTab {
		case TabHabits:
			m.selected = len(m.habits) - 1
		case TabTasks:
			m.selected = len(m.taskList) - 1
		}
		m.clampSelection()
		m.viewport.GotoBottom()

	case "n":
		m.mode = modeCreate
		m.input.Focus()
		return m, nil

	case "x":
		switch m.activeTab {
		case TabHabits:
			if m.selected < len(m.habits) {
				today := time.Now().Format("2006-01-02")
				m.hStore.AddEntry(m.habits[m.selected].ID, today, 1, "")
				m.refresh()
			}
		case TabTasks:
			if m.selected < len(m.taskList) {
				m.tStore.CompleteTask(m.taskList[m.selected].ID)
				m.refresh()
			}
		}

	case "d":
		switch m.activeTab {
		case TabHabits:
			if m.selected < len(m.habits) {
				m.hStore.Delete(m.habits[m.selected].ID)
				m.selected = 0
				m.refresh()
			}
		case TabTasks:
			if m.selected < len(m.taskList) {
				m.tStore.DeleteTask(m.taskList[m.selected].ID)
				m.selected = 0
				m.refresh()
			}
		}
	}

	return m, nil
}

func (m *Model) clampSelection() {
	var max int
	switch m.activeTab {
	case TabHabits:
		max = len(m.habits) - 1
	case TabTasks:
		max = len(m.taskList) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected > max {
		m.selected = max
	}
}

func (m *Model) refresh() {
	switch m.activeTab {
	case TabHabits:
		m.habits, _ = m.hStore.List(false)
	case TabTasks:
		m.taskList, _ = m.tStore.ListTasks(tasks.Filter{})
	}
	m.clampSelection()
}

func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	m.refresh()

	sidebarW := 16
	sidebar := m.renderSidebar(sidebarW, m.height-2)

	m.viewport.Width = m.width - sidebarW - 1
	m.viewport.Height = m.height - 3
	m.viewport.SetContent(m.contentForTab(m.activeTab))

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, m.viewport.View())
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, body, help)
}

func (m *Model) renderSidebar(w, h int) string {
	style := lipgloss.NewStyle().
		Width(w).Height(h).
		BorderStyle(lipgloss.NormalBorder()).BorderRight(true).
		BorderForeground(lipgloss.Color("#555555"))

	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Bold(true)
	sb.WriteString(title.Render("  sica") + "\n\n")

	for _, t := range tabList {
		prefix := "  "
		suffix := ""
		if t == m.activeTab {
			prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Render("▸ ")
			suffix = lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Render(" ◂")
		}
		sb.WriteString(prefix + t.String() + suffix + "\n")
	}

	switch m.activeTab {
	case TabHabits:
		sb.WriteString(fmt.Sprintf("\n  %d items", len(m.habits)))
	case TabTasks:
		sb.WriteString(fmt.Sprintf("\n  %d items", len(m.taskList)))
	}

	return style.Render(sb.String())
}

func (m *Model) renderHelp() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).BorderTop(true).
		BorderForeground(lipgloss.Color("#555555")).
		Foreground(lipgloss.Color("#888888"))

	if m.mode == modeCreate {
		return style.Render("  enter confirm  │  esc cancel  │  tab toggle frequency")
	}

	var actions string
	switch m.activeTab {
	case TabHabits:
		actions = "n new  │  x complete  │  d delete"
	case TabTasks:
		actions = "n new  │  x complete  │  d delete"
	default:
		actions = ""
	}
	return style.Render(fmt.Sprintf("  1-6 tabs  │  j/k navigate  │  %s  │  q quit", actions))
}

func (m *Model) contentForTab(tab Tab) string {
	if m.mode == modeCreate {
		return m.renderCreateForm(tab)
	}

	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true)
	sb.WriteString(title.Render(tab.String()) + "\n\n")

	switch tab {
	case TabHabits:
		sb.WriteString(m.renderHabits())
	case TabTasks:
		sb.WriteString(m.renderTasks())
	case TabFinance, TabCalendar, TabKnowledge, TabChat:
		sb.WriteString("Coming soon.\n")
	}

	return sb.String()
}

func (m *Model) renderCreateForm(tab Tab) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("New "+tab.String()[:len(tab.String())-1]) + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	if tab == TabHabits {
		freqStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
		sb.WriteString("Frequency: " + freqStyle.Render(m.createFreq) + " (tab to toggle)\n")
	}

	return sb.String()
}

func (m *Model) renderHabits() string {
	if len(m.habits) == 0 {
		return "No habits yet.\n\nPress 'n' to create your first habit.\n"
	}

	var sb strings.Builder
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	for i, h := range m.habits {
		prefix := "  "
		if i == m.selected {
			prefix = selStyle.Render("▸ ")
		}

		stats, _ := m.hStore.Stats(h.ID)
		streak := ""
		if stats != nil && stats.CurrentStreak > 0 {
			streak = dimStyle.Render(fmt.Sprintf("  [%dd streak]", stats.CurrentStreak))
		}

		done := " "
		if stats != nil && stats.TodayDone {
			done = selStyle.Render("✓")
		}

		sb.WriteString(fmt.Sprintf("%s[%s] %s%s%s\n",
			prefix, done, h.Name, streak,
			dimStyle.Render("  "+h.Frequency)))
	}
	return sb.String()
}

func (m *Model) renderTasks() string {
	if len(m.taskList) == 0 {
		return "No tasks yet.\n\nPress 'n' to create your first task.\n"
	}

	var sb strings.Builder
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	prioColors := map[string]string{
		"high": lipgloss.NewStyle().Foreground(lipgloss.Color("#cc7c7c")).Render("high"),
		"med":  lipgloss.NewStyle().Foreground(lipgloss.Color("#cccc7c")).Render("med "),
		"low":  dimStyle.Render("low "),
	}

	for i, t := range m.taskList {
		prefix := "  "
		if i == m.selected {
			prefix = selStyle.Render("▸ ")
		}

		status := " "
		if t.Status == "done" {
			status = selStyle.Render("✓")
			sb.WriteString(fmt.Sprintf("%s[%s] %s %s%s\n",
				prefix, status, dimStyle.Render(t.Title),
				prioColors[t.Priority],
				dimStyle.Render("  done")))
		} else {
			sb.WriteString(fmt.Sprintf("%s[%s] %s %s\n",
				prefix, status, t.Title, prioColors[t.Priority]))
		}
	}
	return sb.String()
}
