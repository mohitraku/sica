package tui

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojitrk/sica/internal/agent"
	"github.com/mojitrk/sica/internal/chat"
	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/tasks"
)

type chatStreamChunkMsg struct {
	content string
}

type chatToolStatusMsg struct {
	status string
	tool   string
}

type chatStreamDoneMsg struct {
	err error
}

type Tab int

const (
	TabHabits Tab = iota
	TabTasks
)

func (t Tab) String() string {
	return []string{"Habits", "Tasks"}[t]
}

const numTabs = 2

type mode int

const (
	modeList mode = iota
	modeCreate
	modeEdit
	modeBackfill
)

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDelete
	confirmCreate
	confirmEditSave
	confirmMarkDone
	confirmBackfill
	confirmIncrement
	confirmDecrement
)

type NewParams struct {
	HStore   *habits.Store
	TStore   *tasks.Store
	ChStore  *chat.Store
	Ollama   *chat.Client
	DeepSeek *chat.DeepSeekClient
	Registry *agent.Registry
}

type Model struct {
	width     int
	height    int
	activeTab Tab
	viewport  viewport.Model
	ready     bool
	mode      mode
	input     textinput.Model

	createFreq         string
	createTarget       int
	createQuantityType string

	// Confirmation dialog
	confirmPending bool
	confirmAction  confirmAction
	confirmPrompt  string

	// Edit mode
	editHabitID      int64
	editFreq         string
	editTarget       int
	editQuantityType string

	hStore *habits.Store
	tStore *tasks.Store

	habits   []core.Habit
	taskList []core.Task

	selected       int
	backfillActive bool
	backfillDate   string

	// Chat
	chStore        *chat.Store
	ollama         *chat.Client
	deepseek       *chat.DeepSeekClient
	agentRegistry  *agent.Registry
	chatInput      textinput.Model
	chatHistory    []core.Message
	chatPending    bool
	chatStreamBuf  string
	chatToolStatus string
	chatBackend    string
	convID         int64
	program        *tea.Program
}

func New(p NewParams) *Model {
	ti := textinput.New()
	ti.Placeholder = "Name..."
	ti.CharLimit = 100

	ci := textinput.New()
	ci.Placeholder = "Message (enter to send, /think for DeepSeek)..."
	ci.CharLimit = 2000
	ci.Focus()

	return &Model{
		activeTab:     TabHabits,
		mode:          modeList,
		input:         ti,
		chatInput:     ci,
		createFreq:         "daily",
		createTarget:       1,
		createQuantityType: "count",
		backfillDate:  time.Now().Format("2006-01-02"),
		hStore:        p.HStore,
		tStore:        p.TStore,
		chStore:       p.ChStore,
		ollama:        p.Ollama,
		deepseek:      p.DeepSeek,
		agentRegistry: p.Registry,
		chatBackend:   "ollama",
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

func (m *Model) Init() tea.Cmd {
	m.refresh()
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case chatStreamChunkMsg:
		m.chatStreamBuf += msg.content
		return m, nil

	case chatToolStatusMsg:
		if msg.status == "start" {
			m.chatToolStatus = fmt.Sprintf("[tool: %s...]", msg.tool)
		} else {
			m.chatToolStatus = ""
		}
		return m, nil

	case chatStreamDoneMsg:
		m.chatPending = false
		m.chatStreamBuf = ""
		m.chatToolStatus = ""
		if msg.err != nil {
			log.Printf("chat error: %v", msg.err)
		}
		m.loadChatHistory()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		const chatPanelH = 5
		const helpH = 2
		contentW := m.width - 16 - 1
		contentH := m.height - chatPanelH - helpH
		if contentH < 6 {
			contentH = 6
		}
		if !m.ready {
			m.viewport = viewport.New(contentW, contentH)
			m.ready = true
		} else {
			m.viewport.Width = contentW
			m.viewport.Height = contentH
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.mode != modeList {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	// Always update chat input when in list mode (chat is always visible)
	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	return m, cmd
}

func (m *Model) targetDate() string {
	if m.backfillActive {
		return m.backfillDate
	}
	return time.Now().Format("2006-01-02")
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Confirmation dialog blocks all other input
	if m.confirmPending {
		switch msg.String() {
		case "y", "Y", "enter":
			m.executeConfirmedAction()
			m.confirmPending = false
			m.confirmAction = confirmNone
			m.refresh()
		case "n", "N", "esc":
			action := m.confirmAction
			m.confirmPending = false
			m.confirmAction = confirmNone
			switch action {
			case confirmCreate, confirmEditSave:
				// Stay in form mode, user can adjust
			case confirmBackfill:
				m.mode = modeList
				m.input.Reset()
				m.input.Placeholder = "Name..."
				m.backfillDate = time.Now().Format("2006-01-02")
			default:
				// List-mode actions: just dismiss
			}
		}
		return m, nil
	}

	// Chat input always active unless in create, edit, or backfill mode
	if m.mode == modeList {
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.chatInput.Value())
			if val == "" {
				return m, nil
			}
			if m.chatPending {
				return m, nil
			}
			m.chatPending = true
			m.chatStreamBuf = ""
			m.chatToolStatus = ""

			// Detect /think prefix for DeepSeek routing
			m.chatBackend = "ollama"
			if strings.HasPrefix(val, "/think ") {
				m.chatBackend = "deepseek"
				val = strings.TrimPrefix(val, "/think ")
			}
			if strings.HasPrefix(val, "/think") {
				m.chatBackend = "deepseek"
				val = strings.TrimSpace(strings.TrimPrefix(val, "/think"))
				if val == "" {
					m.chatPending = false
					return m, nil
				}
			}

			m.chatInput.Reset()
			go m.runChatLoop(val)
			return m, nil

		case "ctrl+c", "q":
			return m, tea.Quit

		case "1", "2":
			idx := int(msg.Runes[0] - '1')
			if int(idx) < numTabs {
				m.activeTab = Tab(idx)
				m.selected = 0
				m.mode = modeList
				m.backfillActive = false
				m.input.Reset()
				m.refresh()
			}

		case "tab":
			m.activeTab = (m.activeTab + 1) % numTabs
			m.selected = 0
			m.mode = modeList
			m.backfillActive = false
			m.refresh()

		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
			m.selected = 0
			m.mode = modeList
			m.backfillActive = false
			m.refresh()

		case "down":
			m.selected++
			m.clampSelection()

		case "up":
			m.selected--
			m.clampSelection()

		case "ctrl+n":
			if m.activeTab == TabHabits || m.activeTab == TabTasks {
				m.mode = modeCreate
				m.createFreq = "daily"
				m.createTarget = 1
				m.input.Placeholder = "Name..."
				return m, m.input.Focus()
			}

		case "+", "=":
			if m.activeTab == TabHabits && m.selected < len(m.habits) {
				h := m.habits[m.selected]
				if h.QuantityType == "binary" {
					return m, nil
				}
				current, _ := m.hStore.GetEntry(h.ID, m.targetDate())
				m.confirmPending = true
				m.confirmAction = confirmIncrement
				m.confirmPrompt = fmt.Sprintf("Increment \"%s\"?  (%d → %d)", h.Name, current, current+1)
			}

		case "-":
			if m.activeTab == TabHabits && m.selected < len(m.habits) {
				h := m.habits[m.selected]
				if h.QuantityType == "binary" {
					return m, nil
				}
				current, _ := m.hStore.GetEntry(h.ID, m.targetDate())
				newVal := current - 1
				if newVal < 0 {
					newVal = 0
				}
				m.confirmPending = true
				m.confirmAction = confirmDecrement
				m.confirmPrompt = fmt.Sprintf("Decrement \"%s\"?  (%d → %d)", h.Name, current, newVal)
			}

		case "ctrl+x":
			if m.activeTab == TabHabits && m.selected < len(m.habits) {
				h := m.habits[m.selected]
				m.confirmPending = true
				m.confirmAction = confirmMarkDone
				if h.QuantityType == "binary" {
					current, _ := m.hStore.GetEntry(h.ID, m.targetDate())
					if current > 0 {
						m.confirmPrompt = fmt.Sprintf("Mark \"%s\" as not done?", h.Name)
					} else {
						m.confirmPrompt = fmt.Sprintf("Mark \"%s\" as done?", h.Name)
					}
				} else {
					m.confirmPrompt = fmt.Sprintf("Mark \"%s\" as done?  (set to %d)", h.Name, h.TargetValue)
				}
			}
			if m.activeTab == TabTasks && m.selected < len(m.taskList) {
				m.tStore.CompleteTask(m.taskList[m.selected].ID)
				m.refresh()
			}

		case "ctrl+b":
			if m.activeTab == TabHabits {
				if m.backfillActive {
					m.backfillActive = false
					return m, nil
				}
				m.mode = modeBackfill
				m.backfillDate = time.Now().Format("2006-01-02")
				m.backfillActive = false
				m.input.SetValue(m.backfillDate)
				return m, m.input.Focus()
			}

		case "ctrl+d":
			switch m.activeTab {
			case TabHabits:
				if m.selected < len(m.habits) {
					h := m.habits[m.selected]
					m.confirmPending = true
					m.confirmAction = confirmDelete
					m.confirmPrompt = fmt.Sprintf("Delete habit \"%s\"?", h.Name)
				}
			case TabTasks:
				if m.selected < len(m.taskList) {
					m.tStore.DeleteTask(m.taskList[m.selected].ID)
					m.selected = 0
					m.refresh()
				}
			}
		case "ctrl+e":
			if m.activeTab == TabHabits && m.selected < len(m.habits) {
				h := m.habits[m.selected]
				m.mode = modeEdit
				m.editHabitID = h.ID
				m.editFreq = h.Frequency
				m.editTarget = h.TargetValue
				m.editQuantityType = h.QuantityType
				m.input.SetValue(h.Name)
				return m, m.input.Focus()
			}

		default:
			var cmd tea.Cmd
			m.chatInput, cmd = m.chatInput.Update(msg)
			return m, cmd
		}

		return m, nil
	}

	if m.mode == modeCreate {
		return m.handleCreateKey(msg)
	}
	if m.mode == modeEdit {
		return m.handleEditKey(msg)
	}
	if m.mode == modeBackfill {
		return m.handleBackfillKey(msg)
	}

	return m, nil
}

func (m *Model) handleCreateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val == "" {
			return m, nil
		}
		switch m.activeTab {
		case TabHabits:
			qt := m.createQuantityType
			if qt == "" {
				qt = "count"
			}
			m.confirmPending = true
			m.confirmAction = confirmCreate
			m.confirmPrompt = fmt.Sprintf("Create habit \"%s\"?  (%s, target %d, %s)",
				val, m.createFreq, m.createTarget, qt)
		case TabTasks:
			m.tStore.CreateTask(&core.Task{Title: val, Status: "todo", Priority: "med"})
			m.mode = modeList
			m.input.Reset()
			m.input.Placeholder = "Name..."
			m.refresh()
		}
		return m, nil

	case "tab":
		if m.activeTab == TabHabits {
			cycles := []string{"daily", "weekly", "monthly"}
			for i, f := range cycles {
				if f == m.createFreq {
					m.createFreq = cycles[(i+1)%len(cycles)]
					break
				}
			}
		}
		return m, nil

	case "ctrl+t":
		if m.activeTab == TabHabits {
			cycles := []string{"count", "binary", "duration"}
			for i, q := range cycles {
				if q == m.createQuantityType {
					m.createQuantityType = cycles[(i+1)%len(cycles)]
					break
				}
			}
		}
		return m, nil

	case "[", "{":
		if m.activeTab == TabHabits && m.createTarget > 1 && m.createQuantityType != "binary" {
			m.createTarget--
		}

	case "]", "}":
		if m.activeTab == TabHabits && m.createQuantityType != "binary" {
			m.createTarget++
		}

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleBackfillKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.backfillActive = false
		m.input.Reset()
		m.input.Placeholder = "Name..."
		m.backfillDate = time.Now().Format("2006-01-02")
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			if _, err := time.Parse("2006-01-02", val); err == nil {
				m.backfillDate = val
				m.mode = modeList
				m.input.Reset()
				m.input.Placeholder = "Name..."
				m.confirmPending = true
				m.confirmAction = confirmBackfill
				m.confirmPrompt = fmt.Sprintf("Activate backfill for %s?", val)
				return m, nil
			}
		}
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."
		return m, nil

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val == "" {
			return m, nil
		}
		qt := m.editQuantityType
		if qt == "" {
			qt = "count"
		}
		m.confirmPending = true
		m.confirmAction = confirmEditSave
		m.confirmPrompt = fmt.Sprintf("Save changes to \"%s\"?  (%s, target %d, %s)",
			val, m.editFreq, m.editTarget, qt)
		return m, nil

	case "tab":
		cycles := []string{"daily", "weekly", "monthly"}
		for i, f := range cycles {
			if f == m.editFreq {
				m.editFreq = cycles[(i+1)%len(cycles)]
				break
			}
		}
		return m, nil

	case "ctrl+t":
		cycles := []string{"count", "binary", "duration"}
		for i, q := range cycles {
			if q == m.editQuantityType {
				m.editQuantityType = cycles[(i+1)%len(cycles)]
				break
			}
		}
		return m, nil

	case "[", "{":
		if m.editTarget > 1 && m.editQuantityType != "binary" {
			m.editTarget--
		}

	case "]", "}":
		if m.editQuantityType != "binary" {
			m.editTarget++
		}

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) executeConfirmedAction() {
	switch m.confirmAction {
	case confirmDelete:
		if m.selected < len(m.habits) {
			m.hStore.Delete(m.habits[m.selected].ID)
			m.selected = 0
			m.mode = modeList
		}

	case confirmCreate:
		name := strings.TrimSpace(m.input.Value())
		qt := m.createQuantityType
		if qt == "" {
			qt = "count"
		}
		target := m.createTarget
		if qt == "binary" {
			target = 1
		}
		m.hStore.Create(&core.Habit{
			Name:         name,
			Frequency:    m.createFreq,
			TargetValue:  target,
			QuantityType: qt,
		})
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."

	case confirmEditSave:
		name := strings.TrimSpace(m.input.Value())
		qt := m.editQuantityType
		if qt == "" {
			qt = "count"
		}
		target := m.editTarget
		if qt == "binary" {
			target = 1
		}
		m.hStore.Update(&core.Habit{
			ID:           m.editHabitID,
			Name:         name,
			Frequency:    m.editFreq,
			TargetValue:  target,
			QuantityType: qt,
		})
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."

	case confirmMarkDone:
		if m.selected < len(m.habits) {
			h := m.habits[m.selected]
			date := m.targetDate()
			if h.QuantityType == "binary" {
				current, _ := m.hStore.GetEntry(h.ID, date)
				if current > 0 {
					m.hStore.RemoveEntry(h.ID, date)
				} else {
					m.hStore.SetEntry(h.ID, date, 1)
				}
			} else {
				m.hStore.SetEntry(h.ID, date, h.TargetValue)
			}
		}

	case confirmBackfill:
		m.backfillActive = true
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."

	case confirmIncrement:
		if m.selected < len(m.habits) {
			m.hStore.IncrementEntry(m.habits[m.selected].ID, m.targetDate(), 1)
		}

	case confirmDecrement:
		if m.selected < len(m.habits) {
			m.hStore.IncrementEntry(m.habits[m.selected].ID, m.targetDate(), -1)
		}
	}
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
		m.habits, _ = m.hStore.List()
	case TabTasks:
		m.taskList, _ = m.tStore.ListTasks(tasks.Filter{})
	}
	m.clampSelection()
	m.loadChatHistory()
}

func (m *Model) loadChatHistory() {
	if m.convID > 0 {
		m.chatHistory, _ = m.chStore.Messages(m.convID)
	}
}

func (m *Model) ensureConversation() {
	if m.convID == 0 {
		id, err := m.chStore.CreateConversation("Chat", m.ollama.Model)
		if err != nil {
			log.Printf("create conversation: %v", err)
			return
		}
		m.convID = id
	}
}

func (m *Model) runChatLoop(userMsg string) {
	m.ensureConversation()
	if m.convID == 0 {
		m.program.Send(chatStreamDoneMsg{err: fmt.Errorf("no conversation")})
		return
	}

	// Save user message
	if _, err := m.chStore.AddMessage(m.convID, "user", userMsg); err != nil {
		m.program.Send(chatStreamDoneMsg{err: err})
		return
	}

	// Select backend
	var backend chat.Backend = m.ollama
	modelName := m.ollama.Model
	if m.chatBackend == "deepseek" {
		if m.deepseek == nil {
			m.program.Send(chatStreamDoneMsg{err: fmt.Errorf("DeepSeek not configured (set api_key in config)")})
			return
		}
		backend = m.deepseek
		modelName = "deepseek-chat"
	}

	agt := agent.New(backend, m.chStore, m.agentRegistry, m.convID, modelName)
	_, err := agt.RunStream(agent.Callbacks{
		OnChunk: func(chunk string) {
			m.program.Send(chatStreamChunkMsg{content: chunk})
		},
		OnTool: func(tool, status string) {
			m.program.Send(chatToolStatusMsg{tool: tool, status: status})
		},
	})

	m.program.Send(chatStreamDoneMsg{err: err})
}

func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Fixed heights: chat panel (5) + help bar (2, content + top border)
	const chatPanelH = 5
	const helpH = 2

	contentH := m.height - chatPanelH - helpH
	if contentH < 6 {
		contentH = 6
	}

	sidebarW := 16
	sidebar := m.renderSidebar(sidebarW, contentH)

	contentW := m.width - sidebarW - 1
	if contentW < 20 {
		contentW = 20
	}

	m.viewport.Width = contentW
	m.viewport.Height = contentH
	m.viewport.SetContent(m.contentForTab(m.activeTab))

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, m.viewport.View())

	chatPanel := m.renderChatPanel()
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, body, chatPanel, help)
}

func (m *Model) renderSidebar(w, h int) string {
	style := lipgloss.NewStyle().
		Width(w).Height(h).
		BorderStyle(lipgloss.NormalBorder()).BorderRight(true).
		BorderForeground(lipgloss.Color("#555555"))

	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc")).Bold(true)
	sb.WriteString(title.Render("  sica") + "\n\n")

	for _, t := range []Tab{TabHabits, TabTasks} {
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

	if m.confirmPending {
		return style.Render("  y/enter: confirm  │  n/esc: cancel")
	}

	if m.mode == modeCreate {
		switch m.activeTab {
		case TabHabits:
			return style.Render(fmt.Sprintf(
				"  enter confirm  │  esc cancel  │  tab freq [%s]  │  [/] target [%d]  │  ^T type [%s]",
				m.createFreq, m.createTarget, m.createQuantityType))
		default:
			return style.Render("  enter confirm  │  esc cancel")
		}
	}

	if m.mode == modeEdit {
		return style.Render(fmt.Sprintf(
			"  enter save  │  esc cancel  │  tab freq [%s]  │  [/] target [%d]  │  ^T type [%s]",
			m.editFreq, m.editTarget, m.editQuantityType))
	}

	if m.mode == modeBackfill {
		return style.Render("  enter confirm date  │  esc cancel  │  type YYYY-MM-DD")
	}

	var actions string
	switch m.activeTab {
	case TabHabits:
		actions = "^N new  │  +/− adjust  │  ^X mark done  │  ^B backfill  │  ^E edit  │  ^D delete"
	case TabTasks:
		actions = "^N new  │  ^X complete  │  ^D delete"
	}
	return style.Render(fmt.Sprintf("  1/2 tabs  │  ↑↓ navigate  │  %s  │  q quit", actions))
}

func (m *Model) contentForTab(tab Tab) string {
	if m.confirmPending {
		return m.renderConfirmDialog()
	}
	if m.mode == modeCreate {
		return m.renderCreateForm(tab)
	}
	if m.mode == modeEdit {
		return m.renderEditForm()
	}
	if m.mode == modeBackfill {
		return m.renderBackfillForm()
	}

	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true)
	sb.WriteString(title.Render(tab.String()) + "\n\n")

	switch tab {
	case TabHabits:
		sb.WriteString(m.renderHabits())
	case TabTasks:
		sb.WriteString(m.renderTasks())
	}

	return sb.String()
}

func createLabel(tab Tab) string {
	switch tab {
	case TabHabits:
		return "Habit"
	case TabTasks:
		return "Task"
	}
	return tab.String()
}

func (m *Model) renderCreateForm(tab Tab) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("New "+createLabel(tab)) + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	switch tab {
	case TabHabits:
		acc := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
		sb.WriteString("Frequency: " + acc.Render(m.createFreq) + " (tab to cycle)\n")
		qt := m.createQuantityType
		if qt == "" {
			qt = "count"
		}
		sb.WriteString("Type:      " + acc.Render(qt) + " (^T to cycle)\n")
		targetStr := strconv.Itoa(m.createTarget)
		if qt == "binary" {
			targetStr = "1 (fixed for binary)"
		}
		sb.WriteString("Target:    " + acc.Render(targetStr) + " ([ / ] to adjust)\n")
	}

	return sb.String()
}

func (m *Model) renderEditForm() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Edit Habit") + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	acc := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	sb.WriteString("Frequency: " + acc.Render(m.editFreq) + " (tab to cycle)\n")
	qt := m.editQuantityType
	if qt == "" {
		qt = "count"
	}
	sb.WriteString("Type:      " + acc.Render(qt) + " (^T to cycle)\n")
	targetStr := strconv.Itoa(m.editTarget)
	if qt == "binary" {
		targetStr = "1 (fixed for binary)"
	}
	sb.WriteString("Target:    " + acc.Render(targetStr) + " ([ / ] to adjust)\n")

	return sb.String()
}

func (m *Model) renderConfirmDialog() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#cc7c7c")).
		Padding(1, 2).
		Width(52).
		Align(lipgloss.Center)

	return style.Render(m.confirmPrompt + "\n\n  y/enter: confirm  │  n/esc: cancel")
}

func (m *Model) renderBackfillForm() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Backfill date") + "\n\n")
	sb.WriteString("Set a date, then +/−/x will apply to that date.\n\n")
	sb.WriteString("Date: " + m.input.View() + "\n\n")
	sb.WriteString("Press enter to confirm, esc to cancel.\n")
	return sb.String()
}

func (m *Model) renderHabits() string {
	if len(m.habits) == 0 {
		return "No habits yet.\n\nPress Ctrl+N to create your first habit.\n"
	}

	var sb strings.Builder
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7ccc7c"))

	for i, h := range m.habits {
		prefix := "  "
		if i == m.selected {
			prefix = selStyle.Render("▸ ")
		}

		stats, _ := m.hStore.Stats(h.ID)
		todayVal := 0
		streak := 0
		longest := 0
		if stats != nil {
			todayVal = stats.TodayValue
			streak = stats.CurrentStreak
			longest = stats.LongestStreak
		}

		var counter string
		switch h.QuantityType {
		case "binary":
			if todayVal > 0 {
				counter = greenStyle.Render("✓  Done")
			} else {
				counter = dimStyle.Render("✗  Not done")
			}
		case "duration":
			c := fmt.Sprintf("%dm/%dm", todayVal, h.TargetValue)
			if todayVal >= h.TargetValue {
				counter = greenStyle.Render(c)
			} else {
				counter = c
			}
		default: // "count"
			c := fmt.Sprintf("%d/%d", todayVal, h.TargetValue)
			if todayVal >= h.TargetValue {
				counter = greenStyle.Render(c)
			} else {
				counter = c
			}
		}

		streakStr := ""
		if streak > 0 {
			longestStr := ""
			if longest > streak {
				longestStr = fmt.Sprintf("/%d", longest)
			}
			streakStr = dimStyle.Render(fmt.Sprintf("  [%dd%s]", streak, longestStr))
		}

		dateLabel := ""
		if m.backfillActive {
			dateLabel = dimStyle.Render("  @" + m.backfillDate)
		}

		sb.WriteString(fmt.Sprintf("%s%s  %s  %s%s%s\n",
			prefix, counter, h.Name, dimStyle.Render(h.Frequency), streakStr, dateLabel))
	}
	return sb.String()
}

func (m *Model) renderTasks() string {
	if len(m.taskList) == 0 {
		return "No tasks yet.\n\nPress Ctrl+N to create your first task.\n"
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

func (m *Model) renderChatPanel() string {
	chatH := 5
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(chatH).
		BorderStyle(lipgloss.NormalBorder()).BorderTop(true).
		BorderForeground(lipgloss.Color("#555555"))

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	aiStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7ccc7c"))
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))

	var sb strings.Builder

	// Show last 2 messages from history
	start := 0
	if len(m.chatHistory) > 2 {
		start = len(m.chatHistory) - 2
	}
	for _, msg := range m.chatHistory[start:] {
		if msg.Role == "tool" {
			continue
		}
		roleStyle := userStyle
		roleLabel := "You"
		if msg.Role == "assistant" {
			roleStyle = aiStyle
			roleLabel = "AI"
		}
		content := msg.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		content = strings.ReplaceAll(content, "\n", " ")
		sb.WriteString(roleStyle.Render(roleLabel+": ") + dimStyle.Render(content) + "\n")
	}

	// Show streaming content
	if m.chatPending {
		sb.WriteString(aiStyle.Render("AI: "))
		if m.chatStreamBuf != "" {
			streamPreview := m.chatStreamBuf
			if len(streamPreview) > 80 {
				streamPreview = streamPreview[len(streamPreview)-80:]
			}
			sb.WriteString(streamPreview)
		} else {
			sb.WriteString(dimStyle.Render("..."))
		}
		if m.chatToolStatus != "" {
			sb.WriteString("  " + dimStyle.Render(m.chatToolStatus))
		}
	}

	// Chat input
	sb.WriteString("\n" + m.chatInput.View())

	backendHint := "ollama"
	if strings.HasPrefix(m.chatInput.Value(), "/think") {
		backendHint = "deepseek"
	}
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  [enter send | /think deepseek | now: %s]", backendHint)))

	return style.Render(sb.String())
}
