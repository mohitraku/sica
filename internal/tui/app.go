package tui

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/knowledge"
	"github.com/mojitrk/sica/internal/tasks"
	"github.com/mojitrk/sica/internal/calendar"
	"github.com/mojitrk/sica/internal/budgets"
	"github.com/mojitrk/sica/internal/chat"
	"github.com/mojitrk/sica/internal/transactions"
)

type ingestMsg struct {
	err error
	doc *core.KnowledgeDoc
}

func ingestURL(kStore *knowledge.Store, rawURL string) tea.Cmd {
	return func() tea.Msg {
		result, err := knowledge.IngestURL(rawURL)
		if err != nil {
			return ingestMsg{err: err}
		}
		doc, err := kStore.Save(result.Title, result.Body, result.SourceURL, nil)
		if err != nil {
			return ingestMsg{err: err}
		}
		return ingestMsg{doc: doc}
	}
}

type chatResponseMsg struct {
	err    error
	convID int64
}

func sendChat(chStore *chat.Store, ollama *chat.Client, convID int64, history []core.Message, userMsg string) tea.Cmd {
	return func() tea.Msg {
		chStore.AddMessage(convID, "user", userMsg)
		msgs := make([]chat.Message, 0, len(history)+1)
		for _, m := range history {
			msgs = append(msgs, chat.Message{Role: m.Role, Content: m.Content})
		}
		msgs = append(msgs, chat.Message{Role: "user", Content: userMsg})
		response, err := ollama.Chat(msgs)
		if err != nil {
			return chatResponseMsg{err: err, convID: convID}
		}
		chStore.AddMessage(convID, "assistant", response)
		return chatResponseMsg{convID: convID}
	}
}

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
	modeList     mode = iota
	modeCreate        // creating a new item
	modeBackfill      // selecting backfill date, then increment/decrement applies to that date
)

var freqCycle = []string{"daily", "weekly", "monthly"}

type Model struct {
	width        int
	height       int
	activeTab    Tab
	viewport     viewport.Model
	ready        bool
	mode         mode
	input        textinput.Model
	createFreq   string
	createTarget int

	hStore    *habits.Store
	tStore    *tasks.Store
	kStore    *knowledge.Store
	txnStore  *transactions.Store
	bStore    *budgets.Store
	cStore    *calendar.Store
	chStore    *chat.Store
	ollama     *chat.Client

	habits       []core.Habit
	taskList     []core.Task
	docs         []core.KnowledgeDoc
	transactions []core.Transaction
	budgets      []core.Budget
	events       []core.CalendarEvent
	chatConvs    []core.Conversation
	chatMsgs     []core.Message
	chatConvID   int64
	chatPending  bool
	selected     int
	backfillDate string
}

func New(hStore *habits.Store, tStore *tasks.Store, kStore *knowledge.Store, txnStore *transactions.Store, bStore *budgets.Store, cStore *calendar.Store, chStore *chat.Store, ollama *chat.Client) *Model {
	ti := textinput.New()
	ti.Placeholder = "Name..."
	ti.CharLimit = 100

	return &Model{
		activeTab:    TabHabits,
		mode:         modeList,
		input:        ti,
		createFreq:   "daily",
		createTarget: 1,
		backfillDate: time.Now().Format("2006-01-02"),
		hStore:       hStore,
		tStore:       tStore,
		kStore:       kStore,
		txnStore:     txnStore,
		bStore:       bStore,
		cStore:       cStore,
		chStore:      chStore,
		ollama:       ollama,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ingestMsg:
		if msg.err != nil {
			log.Printf("ingest error: %v", msg.err)
		} else {
			log.Printf("ingested: %s", msg.doc.Title)
		}
		m.mode = modeList
		m.input.Reset()
		m.input.Placeholder = "Name..."
		m.refresh()
		return m, nil

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

	if m.mode != modeList {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) targetDate() string {
	if m.mode == modeBackfill {
		return m.backfillDate
	}
	return time.Now().Format("2006-01-02")
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeCreate {
		return m.handleCreateKey(msg)
	}
	if m.mode == modeBackfill {
		return m.handleBackfillKey(msg)
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

	case "k", "up":
		m.selected--
		m.clampSelection()

	case "g":
		m.selected = 0
		m.viewport.GotoTop()
	case "G":
		switch m.activeTab {
		case TabHabits:
			m.selected = len(m.habits) - 1
		case TabTasks:
			m.selected = len(m.taskList) - 1
		case TabFinance:
			m.selected = len(m.transactions) - 1
		case TabCalendar:
			m.selected = len(m.events) - 1
		case TabKnowledge:
			m.selected = len(m.docs) - 1
		}
		m.clampSelection()
		m.viewport.GotoBottom()

	case "n":
		if m.activeTab == TabHabits || m.activeTab == TabTasks || m.activeTab == TabKnowledge || m.activeTab == TabFinance || m.activeTab == TabCalendar || m.activeTab == TabChat {
			m.mode = modeCreate
			m.createFreq = "daily"
			m.createTarget = 1
			m.input.Focus()
			switch m.activeTab {
			case TabKnowledge:
				m.input.Placeholder = "URL..."
			case TabFinance:
				m.input.Placeholder = "+/-amt category..."
			case TabCalendar:
				m.input.Placeholder = "title YYYY-MM-DD..."
			default:
				m.input.Placeholder = "Name..."
			}
		}


		case "B":
			if m.activeTab == TabFinance {
				m.mode = modeCreate
				m.input.Placeholder = "category amount..."
				m.input.Focus()
			}
	case "+", "=":
		if m.activeTab == TabHabits && m.selected < len(m.habits) {
			m.hStore.IncrementEntry(m.habits[m.selected].ID, m.targetDate(), 1)
			m.refresh()
		}

	case "-":
		if m.activeTab == TabHabits && m.selected < len(m.habits) {
			m.hStore.IncrementEntry(m.habits[m.selected].ID, m.targetDate(), -1)
			m.refresh()
		}

	case "x":
		if m.activeTab == TabHabits && m.selected < len(m.habits) {
			h := m.habits[m.selected]
			m.hStore.SetEntry(h.ID, m.targetDate(), h.TargetValue, "")
			m.refresh()
		}
		if m.activeTab == TabTasks && m.selected < len(m.taskList) {
			m.tStore.CompleteTask(m.taskList[m.selected].ID)
			m.refresh()
		}

	case "b":
		if m.activeTab == TabHabits {
			m.mode = modeBackfill
			m.backfillDate = time.Now().Format("2006-01-02")
			m.input.SetValue(m.backfillDate)
			m.input.Focus()
		}

	case "enter":
		if m.activeTab == TabKnowledge && m.selected < len(m.docs) {
			m.viewDoc(m.docs[m.selected].ID)
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
		case TabFinance:
			if m.selected < len(m.transactions) {
				m.txnStore.Delete(m.transactions[m.selected].ID)
				m.selected = 0
				m.refresh()
			}
		case TabKnowledge:
			if m.selected < len(m.docs) {
				m.kStore.Delete(m.docs[m.selected].ID)
				m.selected = 0
				m.refresh()
			}
		case TabCalendar:
			if m.selected < len(m.events) {
				m.cStore.Delete(m.events[m.selected].ID)
				m.selected = 0
				m.refresh()
			}
		}
	}

	return m, nil
}

func (m *Model) viewDoc(id int64) {
	content, err := m.kStore.ReadContent(id)
	if err != nil {
		return
	}
	_ = content
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
			m.hStore.Create(&core.Habit{Name: val, Frequency: m.createFreq, TargetValue: m.createTarget})
			m.mode = modeList
			m.input.Reset()
			m.input.Placeholder = "Name..."
			m.refresh()
			return m, nil
		case TabTasks:
			m.tStore.CreateTask(&core.Task{Title: val, Status: "todo", Priority: "med"})
			m.mode = modeList
			m.input.Reset()
			m.input.Placeholder = "Name..."
			m.refresh()
			return m, nil
		case TabKnowledge:
			return m, ingestURL(m.kStore, val)
		case TabFinance:
			amount, cat, txType := parseTransaction(val)
			m.txnStore.Add(&core.Transaction{
				Amount:   amount,
				Type:     txType,
				Category: cat,
			})
			m.mode = modeList
			m.input.Reset()
			m.input.Placeholder = "Name..."
			m.refresh()
			return m, nil
		}

		return m, nil

	case "tab":
		if m.activeTab == TabHabits {
			for i, f := range freqCycle {
				if f == m.createFreq {
					m.createFreq = freqCycle[(i+1)%len(freqCycle)]
					break
				}
			}
		}
		return m, nil

	case "[", "{":
		if m.activeTab == TabHabits && m.createTarget > 1 {
			m.createTarget--
		}

	case "]", "}":
		if m.activeTab == TabHabits {
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
		m.input.Reset()
		m.input.Placeholder = "Name..."
		m.backfillDate = time.Now().Format("2006-01-02")
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			if _, err := time.Parse("2006-01-02", val); err == nil {
				m.backfillDate = val
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

func (m *Model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.chatConvID = 0
		m.chatMsgs = nil
		m.selected = 0
		m.input.Reset()
		m.input.Placeholder = "Name..."
		m.refresh()
		return m, nil

	case "enter":
		if m.chatPending {
			return m, nil
		}
		val := strings.TrimSpace(m.input.Value())
		if val == "" {
			return m, nil
		}
		m.chatPending = true
		m.input.Reset()
		m.refresh()
		return m, sendChat(m.chStore, m.ollama, m.chatConvID, m.chatMsgs, val)

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *Model) clampSelection() {
	var max int
	switch m.activeTab {
	case TabHabits:
		max = len(m.habits) - 1
	case TabTasks:
		max = len(m.taskList) - 1
	case TabFinance:
		max = len(m.transactions) - 1
	case TabKnowledge:
		max = len(m.docs) - 1
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
	case TabFinance:
		now := time.Now()
		m.budgets, _ = m.bStore.List()
		m.transactions, _ = m.txnStore.List(transactions.Filter{
			Year: now.Year(), Month: int(now.Month()),
		})
	case TabKnowledge:
		m.docs, _ = m.kStore.List()
	case TabCalendar:
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, 0)
		m.events, _ = m.cStore.List(start, end)
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
	case TabFinance:
		sb.WriteString(fmt.Sprintf("\n  %d items", len(m.transactions)))
	case TabKnowledge:
		sb.WriteString(fmt.Sprintf("\n  %d items", len(m.docs)))
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
		switch m.activeTab {
		case TabHabits:
			return style.Render(fmt.Sprintf(
				"  enter confirm  │  esc cancel  │  tab freq [%s]  │  [/] target [%d]",
				m.createFreq, m.createTarget))
		case TabFinance:
			return style.Render("  enter confirm  │  esc cancel  │  type +/−amt category")
		case TabKnowledge:
			return style.Render("  enter ingest URL  │  esc cancel")
		default:
			return style.Render("  enter confirm  │  esc cancel")
		}
	}

	if m.mode == modeBackfill {
		return style.Render("  enter confirm date  │  esc cancel  │  type YYYY-MM-DD")
	}

	var actions string
	switch m.activeTab {
	case TabHabits:
		actions = "n new  │  +/− adjust  │  x meet target  │  b backfill  │  d delete"
	case TabTasks:
		actions = "n new  │  x complete  │  d delete"
	case TabFinance:
		actions = "n add  │  d delete"
	case TabKnowledge:
		actions = "n ingest URL  │  enter view  │  d delete"
	default:
		actions = ""
	}
	return style.Render(fmt.Sprintf("  1-6 tabs  │  j/k navigate  │  %s  │  q quit", actions))
}

func (m *Model) contentForTab(tab Tab) string {
	if m.mode == modeCreate {
		return m.renderCreateForm(tab)
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
	case TabFinance:
		sb.WriteString(m.renderFinance())
	case TabKnowledge:
		sb.WriteString(m.renderDocs())
	case TabCalendar:
		sb.WriteString(m.renderCalendar())
	case TabChat:
		sb.WriteString("Coming soon.\n")
	}

	return sb.String()
}

func (m *Model) renderCreateForm(tab Tab) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("New "+tab.String()[:len(tab.String())-1]) + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	switch tab {
	case TabHabits:
		acc := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
		sb.WriteString("Frequency: " + acc.Render(m.createFreq) + " (tab to cycle)\n")
		sb.WriteString("Target:    " + acc.Render(strconv.Itoa(m.createTarget)) + " ([ / ] to adjust)\n")
	case TabFinance:
		sb.WriteString("Format: +/−amount category   e.g. -12.50 lunch\n")
	case TabKnowledge:
		sb.WriteString("Paste a URL to ingest.\n")
	}

	return sb.String()
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
		return "No habits yet.\n\nPress 'n' to create your first habit.\n"
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
		if stats != nil {
			todayVal = stats.TodayValue
			streak = stats.CurrentStreak
		}

		counter := fmt.Sprintf("%d/%d", todayVal, h.TargetValue)
		if todayVal >= h.TargetValue {
			counter = greenStyle.Render(counter)
		}

		streakStr := ""
		if streak > 0 {
			streakStr = dimStyle.Render(fmt.Sprintf("  [%dd streak]", streak))
		}

		dateLabel := ""
		if m.mode == modeBackfill {
			dateLabel = dimStyle.Render("  @" + m.backfillDate)
		}

		sb.WriteString(fmt.Sprintf("%s%s  %s  %s%s%s\n",
			prefix, counter, h.Name, dimStyle.Render(h.Frequency), streakStr, dateLabel))
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

func (m *Model) renderFinance() string {
	var sb strings.Builder
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	incomeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7ccc7c"))
	expenseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc7c7c"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc7c7c"))

	now := time.Now()
	summary, _ := m.txnStore.MonthSummary(now.Year(), int(now.Month()))
	if summary != nil {
		balance := summary.Income - summary.Expense
		sb.WriteString(fmt.Sprintf("  Income:   %s\n", incomeStyle.Render("$"+formatCents(summary.Income))))
		sb.WriteString(fmt.Sprintf("  Expenses: %s\n", expenseStyle.Render("$"+formatCents(summary.Expense))))
		balStyle := incomeStyle
		if balance < 0 {
			balStyle = expenseStyle
		}
		sb.WriteString(fmt.Sprintf("  Balance:  %s\n", balStyle.Render("$"+formatCents(balance))))

		if len(m.budgets) > 0 {
			sb.WriteString("\n  ── Budgets ──\n")
			for _, b := range m.budgets {
				spent := summary.ByCategory[b.Category]
				pct := int64(0)
				if b.AmountCents > 0 {
					pct = spent * 100 / b.AmountCents
				}
				pctStr := dimStyle.Render(fmt.Sprintf("[%d%%]", pct))
				if pct > 100 {
					pctStr = warnStyle.Render(fmt.Sprintf("[%d%%!]", pct))
				}
				spentStr := expenseStyle.Render("$" + formatCents(spent))
				budgetStr := dimStyle.Render("$" + formatCents(b.AmountCents))
				sb.WriteString(fmt.Sprintf("    %s  %s / %s  %s\n", b.Category, spentStr, budgetStr, pctStr))
			}
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
	}

	if len(m.transactions) == 0 {
		sb.WriteString("No transactions this month.\n\nPress 'n' to add one:  +/−amt category\n")
		sb.WriteString("Press 'B' to set a budget:     category amount\n")
		return sb.String()
	}

	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))

	for i, tx := range m.transactions {
		prefix := "  "
		if i == m.selected {
			prefix = selStyle.Render("▸ ")
		}

		amt := "$" + formatCents(tx.Amount)
		amtStyle := expenseStyle
		if tx.Type == "income" {
			amtStyle = incomeStyle
			amt = "+" + amt
		} else {
			amt = "-" + amt
		}

		sb.WriteString(fmt.Sprintf("%s%s %s  %s  %s\n",
			prefix, dimStyle.Render(tx.Date), amtStyle.Render(amt), tx.Category, dimStyle.Render(tx.Description)))
	}
	return sb.String()
}


func (m *Model) renderCalendar() string {
	if len(m.events) == 0 {
		return "No events this month.\n\nPress 'n' to add one:  title YYYY-MM-DD\n"
	}

	var sb strings.Builder
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	dateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7ccc7c"))

	var lastDate string
	for i, e := range m.events {
		dateStr := e.StartTime.Format("Mon 2006-01-02")
		if dateStr != lastDate {
			lastDate = dateStr
			sb.WriteString("\n  " + dateStyle.Render(dateStr) + "\n")
		}

		prefix := "    "
		if i == m.selected {
			prefix = "  " + selStyle.Render("▸ ")
		}

		timeStr := e.StartTime.Format("15:04")
		if timeStr == "00:00" {
			timeStr = "all day"
		}
		sb.WriteString(fmt.Sprintf("%s%s  %s\n", prefix, dimStyle.Render(timeStr), e.Title))
	}
	return sb.String()
}

func (m *Model) renderChat() string {
	if m.chatConvID > 0 {
		return m.renderChatMessages()
	}

	if len(m.chatConvs) == 0 {
		return "No conversations yet.\n\nPress 'n' to start a new conversation.\n"
	}

	var sb strings.Builder
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	for i, c := range m.chatConvs {
		prefix := "  "
		if i == m.selected {
			prefix = selStyle.Render("▸ ")
		}
		modelStr := dimStyle.Render("  [" + c.Model + "]")
		sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, c.Title, modelStr))
	}
	sb.WriteString("\n" + dimStyle.Render("enter to open  │  d to delete"))
	return sb.String()
}

func (m *Model) renderChatMessages() string {
	var sb strings.Builder
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	aiStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7ccc7c"))

	// Find conversation title
	title := "Chat"
	for _, c := range m.chatConvs {
		if c.ID == m.chatConvID {
			title = c.Title
			break
		}
	}
	sb.WriteString(dimStyle.Render("── " + title + " ──") + "\n\n")

	for _, msg := range m.chatMsgs {
		roleStyle := userStyle
		roleLabel := "You"
		if msg.Role == "assistant" {
			roleStyle = aiStyle
			roleLabel = "AI"
		}
		sb.WriteString(roleStyle.Render(roleLabel + ":") + "\n")
		sb.WriteString(msg.Content + "\n\n")
	}

	if m.chatPending {
		sb.WriteString(aiStyle.Render("AI:") + "\n")
		sb.WriteString(dimStyle.Render("...") + "\n")
	}

	sb.WriteString("\n" + dimStyle.Render("───") + "\n")
	sb.WriteString(m.input.View() + "\n")
	sb.WriteString(dimStyle.Render("enter send  │  esc back"))
	return sb.String()
}
func (m *Model) renderDocs() string {
	if len(m.docs) == 0 {
		return "No documents yet.\n\nPress 'n' to ingest a URL.\n"
	}

	var sb strings.Builder
	selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9acc"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	for i, d := range m.docs {
		prefix := "  "
		if i == m.selected {
			prefix = selStyle.Render("▸ ")
		}
		tags := ""
		if len(d.Tags) > 0 {
			tags = dimStyle.Render("  [" + strings.Join(d.Tags, ", ") + "]")
		}
		sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, d.Title, tags))
	}
	sb.WriteString("\n" + dimStyle.Render("enter to read  │  d to delete"))
	return sb.String()
}

func formatCents(cents int64) string {
	dollars := float64(cents) / 100.0
	whole := int64(math.Abs(dollars))
	frac := int64(math.Abs(dollars)*100) % 100
	if cents < 0 {
		return fmt.Sprintf("-%d.%02d", whole, frac)
	}
	return fmt.Sprintf("%d.%02d", whole, frac)
}

func parseTransaction(input string) (amount int64, category string, txType string) {
	input = strings.TrimSpace(input)
	txType = "expense"
	if input == "" {
		return 0, "other", txType
	}

	if input[0] == '+' {
		txType = "income"
		input = input[1:]
	} else if input[0] == '-' {
		input = input[1:]
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return 0, "other", txType
	}

	amountStr := parts[0]
	category = "other"
	if len(parts) > 1 {
		category = strings.Join(parts[1:], " ")
	}

	dollars, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, category, txType
	}

	cents := int64(math.Round(math.Abs(dollars) * 100))
	return cents, category, txType
}

func parseBudget(input string) (category string, amountCents int64) {
	input = strings.TrimSpace(input)
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "other", 0
	}
	category = parts[0]
	if len(parts) > 1 {
		dollars, err := strconv.ParseFloat(parts[1], 64)
		if err == nil {
			amountCents = int64(math.Round(math.Abs(dollars) * 100))
		}
	}
	return category, amountCents
}

func parseCalendarEvent(input string) (title string, start, end time.Time) {
	input = strings.TrimSpace(input)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Default: all-day event today
	if input == "" {
		return "Untitled", today, today.Add(24 * time.Hour)
	}

	// Find the date pattern YYYY-MM-DD at the end
	parts := strings.Fields(input)
	dateStr := parts[len(parts)-1]
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err == nil {
		title = strings.Join(parts[:len(parts)-1], " ")
		start = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local)
	} else {
		title = input
		start = today
	}
	if title == "" {
		title = "Untitled"
	}
	end = start.Add(24 * time.Hour)
	return title, start, end
}
