package tui

import (
	"fmt"
	"log"
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

type NewParams struct {
	HStore   *habits.Store
	ChStore  *chat.Store
	Ollama   *chat.Client
	DeepSeek *chat.DeepSeekClient
	Registry *agent.Registry
}

type Model struct {
	width    int
	height   int
	viewport viewport.Model
	ready    bool

	hStore *habits.Store

	habits   []core.Habit
	selected int

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
	ci := textinput.New()
	ci.Placeholder = "Message (enter to send)..."
	ci.CharLimit = 2000
	ci.Focus()

	return &Model{
		chatInput:     ci,
		hStore:        p.HStore,
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

	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.chatInput.Value())
		if val == "" {
			if m.selected < len(m.habits) {
				m.toggleHabit(m.habits[m.selected])
			}
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

	case "down":
		m.selected++
		m.clampSelection()

	case "up":
		m.selected--
		m.clampSelection()

	default:
		var cmd tea.Cmd
		m.chatInput, cmd = m.chatInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) toggleHabit(h core.Habit) {
	date := time.Now().Format("2006-01-02")
	if h.QuantityType == "binary" {
		current, _ := m.hStore.GetEntry(h.ID, date)
		if current > 0 {
			m.hStore.RemoveEntry(h.ID, date)
		} else {
			m.hStore.SetEntry(h.ID, date, 1)
		}
	} else {
		current, _ := m.hStore.GetEntry(h.ID, date)
		if current >= h.TargetValue {
			m.hStore.RemoveEntry(h.ID, date)
		} else {
			m.hStore.SetEntry(h.ID, date, h.TargetValue)
		}
	}
	m.refresh()
}

func (m *Model) clampSelection() {
	max := len(m.habits) - 1
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected > max {
		m.selected = max
	}
}

func (m *Model) refresh() {
	m.habits, _ = m.hStore.List()
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

	if _, err := m.chStore.AddMessage(m.convID, "user", userMsg); err != nil {
		m.program.Send(chatStreamDoneMsg{err: err})
		return
	}

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
	m.viewport.SetContent(m.renderHabitsView())

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

	active := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	sb.WriteString(active.Render("▸ Habits ◂") + "\n")
	sb.WriteString(fmt.Sprintf("\n  %d items", len(m.habits)))

	return style.Render(sb.String())
}

func (m *Model) renderHelp() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).BorderTop(true).
		BorderForeground(lipgloss.Color("#555555")).
		Foreground(lipgloss.Color("#888888"))

	return style.Render("  ↑↓ navigate  │  enter toggle habit / send chat  │  q quit")
}

func (m *Model) renderHabitsView() string {
	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true)
	sb.WriteString(title.Render("Habits") + "\n\n")
	sb.WriteString(m.renderHabits())
	return sb.String()
}

func (m *Model) renderHabits() string {
	if len(m.habits) == 0 {
		return "No habits yet.\n\nUse the chat to create your first habit.\n"
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

		sb.WriteString(fmt.Sprintf("%s%s  %s  %s%s\n",
			prefix, counter, h.Name, dimStyle.Render(h.Frequency), streakStr))
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

	sb.WriteString("\n" + m.chatInput.View())

	sb.WriteString(dimStyle.Render("  [enter send | /think deepseek]"))

	return style.Render(sb.String())
}
