package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type Model struct {
	width     int
	height    int
	activeTab Tab
	viewport  viewport.Model
	ready     bool
}

func New() *Model {
	return &Model{
		activeTab: TabHabits,
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
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.activeTab = TabHabits
		case "2":
			m.activeTab = TabTasks
		case "3":
			m.activeTab = TabFinance
		case "4":
			m.activeTab = TabCalendar
		case "5":
			m.activeTab = TabKnowledge
		case "6":
			m.activeTab = TabChat
		case "tab":
			m.activeTab = (m.activeTab + 1) % Tab(len(tabList))
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + Tab(len(tabList))) % Tab(len(tabList))
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		}
		m.viewport.SetContent(m.contentForTab(m.activeTab))
		return m, nil
	}

	return m, nil
}

func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	sidebarW := 16
	contentW := m.width - sidebarW - 1

	sidebar := m.renderSidebar(sidebarW, m.height-2)
	m.viewport.Width = contentW
	m.viewport.Height = m.height - 3
	m.viewport.SetContent(m.contentForTab(m.activeTab))

	content := m.viewport.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, body, help)
}

func (m *Model) renderSidebar(w, h int) string {
	style := lipgloss.NewStyle().
		Width(w).
		Height(h).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(lipgloss.Color("#555555"))

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Bold(true).Render("  sica") + "\n\n")

	for _, t := range tabList {
		prefix := "  "
		suffix := ""
		if t == m.activeTab {
			prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Render("▸ ")
			suffix = lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Render(" ◂")
		}
		sb.WriteString(prefix + t.String() + suffix + "\n")
	}

	return style.Render(sb.String())
}

func (m *Model) renderHelp() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("#555555")).
		Foreground(lipgloss.Color("#888888"))

	help := fmt.Sprintf("  1-6 tabs  │  tab/shift+tab cycle  │  j/k scroll  │  g/G top/bottom  │  q quit")
	return style.Render(help)
}

func (m *Model) contentForTab(tab Tab) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(tab.String()) + "\n\n")

	switch tab {
	case TabHabits:
		sb.WriteString("No habits yet.\n\nPress 'n' to create your first habit.\n")
	case TabTasks:
		sb.WriteString("No tasks yet.\n\nPress 'n' to create your first task.\n")
	case TabFinance:
		sb.WriteString("No transactions yet.\n\nPress 'n' to add a transaction.\n")
	case TabCalendar:
		sb.WriteString("No events yet.\n\nPress 'n' to add an event.\n")
	case TabKnowledge:
		sb.WriteString("No documents yet.\n\nUse the API or web UI to ingest content.\n")
	case TabChat:
		sb.WriteString("AI Chat\n\nChat interface coming in Phase 5.\n")
	}

	return sb.String()
}
