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
	mainOffset  int
	colOffset   int
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
			m.mainOffset = 0
			m.colOffset = 0
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
				m.colOffset = 0
			} else {
				m.activePanel = panelMain
				m.mainCursor = 0
				m.mainOffset = 0
				m.colOffset = 0
			}

		case "up", "k":
			if m.activePanel == panelSidebar {
				if m.catCursor > 0 {
					m.catCursor--
					m.mainCursor = 0
					m.mainOffset = 0
					m.colOffset = 0
				}
			} else {
				if m.mainCursor > 0 {
					m.mainCursor--
					if m.mainCursor < m.mainOffset {
						m.mainOffset--
					}
				}
			}

		case "down", "j":
			if m.activePanel == panelSidebar {
				if m.catCursor < len(m.categories)-1 {
					m.catCursor++
					m.mainCursor = 0
					m.mainOffset = 0
					m.colOffset = 0
				}
			} else {
				rows := m.rows[m.categories[m.catCursor]]
				if m.mainCursor < len(rows)-1 {
					m.mainCursor++
					visibleRows := m.visibleMainRows()
					if m.mainCursor >= m.mainOffset+visibleRows {
						m.mainOffset++
					}
				}
			}

		case "left":
			if m.activePanel == panelMain && m.colOffset > 0 {
				m.colOffset -= 4
				if m.colOffset < 0 {
					m.colOffset = 0
				}
			}

		case "right":
			if m.activePanel == panelMain {
				m.colOffset += 4
			}
		}
	}

	return m, nil
}

func (m Model) visibleMainRows() int {
	const titleHeight = 1
	const statusHeight = 1
	const vertBorderOverhead = 2 // top + bottom border of one panel
	const headerLines = 2        // header + divider
	v := m.height - titleHeight - statusHeight - vertBorderOverhead - headerLines
	if v < 1 {
		v = 1
	}
	return v
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	const sidebarWidth = 26
	const horizBorderOverhead = 4 // 2 panels × left+right border each
	const vertBorderOverhead = 2  // top + bottom border of panel
	const titleHeight = 1
	const statusHeight = 1
	panelHeight := m.height - titleHeight - statusHeight - vertBorderOverhead
	if panelHeight < 1 {
		panelHeight = 1
	}
	mainWidth := m.width - sidebarWidth - horizBorderOverhead
	if mainWidth < 1 {
		mainWidth = 1
	}

	// ── Sidebar ──────────────────────────────────────────────────────────────
	var sidebarLines []string
	for i, cat := range m.categories {
		if i >= panelHeight {
			break
		}
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
	colW := mainWidth
	if colW < 1 {
		colW = 1
	}

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

	// clip truncates s to at most n visible characters.
	clip := func(s string, n int) string {
		if n <= 0 {
			return ""
		}
		runes := []rune(s)
		if len(runes) > n {
			return string(runes[:n])
		}
		return s
	}

	// scroll skips the first n visible characters (horizontal scroll).
	scroll := func(s string, n int) string {
		runes := []rune(s)
		if n >= len(runes) {
			return ""
		}
		return string(runes[n:])
	}

	renderRow := func(row [4]string) string {
		return fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			widths[0], row[0],
			widths[1], row[1],
			widths[2], row[2],
			row[3],
		)
	}

	// rowContentW is the usable content width inside a padded row style (Padding 0,1 = 2 chars).
	rowContentW := colW - 2
	if rowContentW < 0 {
		rowContentW = 0
	}

	header := headerStyle.Width(colW).Render(clip(scroll(renderRow(headers), m.colOffset), colW))
	divider := lipgloss.NewStyle().Foreground(colorSubtle).Render(strings.Repeat("─", colW))

	var mainLines []string
	mainLines = append(mainLines, header, divider)

	switch {
	case m.loading[category]:
		mainLines = append(mainLines, normalRowStyle.Render("fetching…"))
	case m.errors[category] != "":
		mainLines = append(mainLines, normalRowStyle.Render("error: "+m.errors[category]))
	default:
		visibleRows := m.visibleMainRows()
		// Guard against mainOffset exceeding rows (e.g. after a re-fetch with fewer results).
		maxOffset := len(m.rows[category]) - 1
		if maxOffset < 0 {
			maxOffset = 0
		}
		offset := m.mainOffset
		if offset > maxOffset {
			offset = maxOffset
		}
		end := offset + visibleRows
		if end > len(m.rows[category]) {
			end = len(m.rows[category])
		}
		for i, row := range m.rows[category][offset:end] {
			absIdx := offset + i
			line := clip(scroll(renderRow(row), m.colOffset), rowContentW)
			if m.activePanel == panelMain && absIdx == m.mainCursor {
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
			keyStyle.Render("←/→") + " scroll columns  " +
			keyStyle.Render("q") + " quit",
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, panels, statusBar)
}
