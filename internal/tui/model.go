package tui

import (
	"fmt"
	"strings"

	"github.com/LotadTech/lazy-fluxcd/internal/k8"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/client-go/dynamic"
)

type panel int

const (
	panelSidebar panel = iota
	panelMain
)

var (
	colorPurple = lipgloss.Color("#7D56F4")
	colorSubtle = lipgloss.Color("#383838")
	colorGray   = lipgloss.Color("#626262")
	colorWhite  = lipgloss.Color("#FAFAFA")

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPurple)

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSubtle)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorPurple).
				Padding(0, 1)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGray)

	keyStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorGray)
)

// fetchResultMsg carries the result of a background cluster fetch.
type fetchResultMsg struct {
	category string
	rows     []k8.Row
	err      error
}

func fetchCategory(client dynamic.Interface, category string) tea.Cmd {
	return func() tea.Msg {
		rows, err := k8.FetchFluxResources(client, category)
		return fetchResultMsg{category: category, rows: rows, err: err}
	}
}

// Model is the root Bubbletea model.
type Model struct {
	client      dynamic.Interface
	categories  []string
	rows        map[string][]k8.Row
	loading     map[string]bool
	errors      map[string]string
	catCursor   int
	mainCursor  int
	activePanel panel
	width       int
	height      int
}

// New returns an initialised Model.
func New(client dynamic.Interface) Model {
	categories := []string{
		"Kustomizations",
		"Helm Releases",
		"Git Repositories",
		"OCI Repositories",
		"Helm Repositories",
		"Helm Charts",
		"Buckets",
		"Alerts",
		"Providers",
		"Receivers",
	}
	loading := make(map[string]bool, len(categories))
	for _, c := range categories {
		loading[c] = true
	}
	return Model{
		client:      client,
		categories:  categories,
		rows:        make(map[string][]k8.Row),
		loading:     loading,
		errors:      make(map[string]string),
		activePanel: panelSidebar,
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, cat := range m.categories {
		cmds = append(cmds, fetchCategory(m.client, cat))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchResultMsg:
		m.loading[msg.category] = false
		if msg.err != nil {
			m.errors[msg.category] = msg.err.Error()
		} else {
			m.rows[msg.category] = msg.rows
		}
		if msg.category == m.categories[m.catCursor] {
			m.mainCursor = 0
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab", "l", "h":
			if msg.String() == "h" || (msg.String() == "tab" && m.activePanel == panelMain) {
				m.activePanel = panelSidebar
			} else {
				m.activePanel = panelMain
				m.mainCursor = 0
			}

		case "up", "k":
			if m.activePanel == panelSidebar {
				if m.catCursor > 0 {
					m.catCursor--
					m.mainCursor = 0
				}
			} else {
				if m.mainCursor > 0 {
					m.mainCursor--
				}
			}

		case "down", "j":
			if m.activePanel == panelSidebar {
				if m.catCursor < len(m.categories)-1 {
					m.catCursor++
					m.mainCursor = 0
				}
			} else {
				rows := m.rows[m.categories[m.catCursor]]
				if m.mainCursor < len(rows)-1 {
					m.mainCursor++
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	const sidebarWidth = 26
	const borderOverhead = 4 // 2 borders × 2 sides
	const titleHeight = 2
	const statusHeight = 1
	panelHeight := m.height - titleHeight - statusHeight - borderOverhead
	if panelHeight < 1 {
		panelHeight = 1
	}
	mainWidth := m.width - sidebarWidth - borderOverhead

	// ── Sidebar ──────────────────────────────────────────────────────────────
	var sidebarLines []string
	for i, cat := range m.categories {
		label := fmt.Sprintf("%-*s", sidebarWidth-2, cat)
		if i == m.catCursor {
			sidebarLines = append(sidebarLines, selectedRowStyle.Width(sidebarWidth-2).Render(label))
		} else {
			sidebarLines = append(sidebarLines, normalRowStyle.Width(sidebarWidth-2).Render(label))
		}
	}
	sidebarContent := strings.Join(sidebarLines, "\n")

	// ── Main panel ───────────────────────────────────────────────────────────
	category := m.categories[m.catCursor]
	colW := mainWidth - borderOverhead

	// Compute column widths from content so nothing overlaps.
	headers := [4]string{"NAME", "READY", "STATUS", "LAST RECONCILED"}
	widths := [4]int{}
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range m.rows[category] {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	renderRow := func(row [4]string) string {
		return fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			widths[0], row[0],
			widths[1], row[1],
			widths[2], row[2],
			row[3],
		)
	}

	header := headerStyle.Width(colW).Render(renderRow(headers))
	divider := lipgloss.NewStyle().Foreground(colorSubtle).Render(strings.Repeat("─", colW))

	var mainLines []string
	mainLines = append(mainLines, header, divider)

	switch {
	case m.loading[category]:
		mainLines = append(mainLines, normalRowStyle.Render("fetching…"))
	case m.errors[category] != "":
		mainLines = append(mainLines, normalRowStyle.Render("error: "+m.errors[category]))
	default:
		for i, row := range m.rows[category] {
			line := renderRow(row)
			if m.activePanel == panelMain && i == m.mainCursor {
				mainLines = append(mainLines, selectedRowStyle.Width(colW).Render(line))
			} else {
				mainLines = append(mainLines, normalRowStyle.Width(colW).Render(line))
			}
		}
	}
	mainContent := strings.Join(mainLines, "\n")

	// ── Apply borders ─────────────────────────────────────────────────────────
	sidebarBorder := inactiveBorderStyle
	mainBorder := inactiveBorderStyle
	if m.activePanel == panelSidebar {
		sidebarBorder = activeBorderStyle
	} else {
		mainBorder = activeBorderStyle
	}

	sidebarPanel := sidebarBorder.Width(sidebarWidth).Height(panelHeight).Render(sidebarContent)
	mainPanel := mainBorder.Width(mainWidth).Height(panelHeight).Render(mainContent)

	// ── Compose ───────────────────────────────────────────────────────────────
	title := titleStyle.Render("⚡ lazy-fluxcd")
	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPanel, mainPanel)

	statusBar := statusStyle.Render(
		keyStyle.Render("tab/l/h") + " switch panel  " +
			keyStyle.Render("↑/↓") + " navigate  " +
			keyStyle.Render("q") + " quit",
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, panels, statusBar)
}
