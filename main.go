package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	currentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	counterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	copiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")).
			Bold(true)
)

type model struct {
	lines      []string
	cursor     int
	topLine    int
	copiedLine int
	copyErr    string
	quitted    bool
	width      int
	height     int
}

func initialModel(lines []string) model {
	return model{lines: lines, cursor: 0, topLine: 0, copiedLine: -1}
}

// maxContent returns the number of rows available for line content.
func (m *model) maxContent() int {
	// header 1 + blank 1 + help separator 1 + help 1 + margin 2 = 6
	r := m.height - 6
	if r < 5 {
		r = 5
	}
	return r
}

// contentWidth returns the available width for line text.
func (m *model) contentWidth() int {
	w := m.width
	if w <= 0 {
		w = 80
	}
	gutterWidth := len(fmt.Sprintf("%d", len(m.lines)))
	prefixWidth := 2 + gutterWidth + 1
	cw := w - prefixWidth
	if cw < 1 {
		cw = 1
	}
	return cw
}

// lineRows returns how many display rows a given line occupies.
func (m *model) lineRows(i int) int {
	line := m.lines[i]
	if line == "" {
		return 1
	}
	cw := m.contentWidth()
	if cw <= 0 {
		cw = 1
	}
	rows := (runewidth.StringWidth(line) + cw - 1) / cw
	if rows < 1 {
		rows = 1
	}
	return rows
}

// visibleEnd calculates the last visible line index (exclusive) from a given start,
// accounting for ellipsis lines.
func (m *model) visibleEnd(from int, budget int) int {
	hasAbove := from > 0
	available := budget
	if hasAbove {
		available--
	}

	used := 0
	end := from
	for end < len(m.lines) {
		rows := m.lineRows(end)
		// Reserve 1 row for "··· below" if there will be more lines after this one
		reserve := 0
		if end+1 < len(m.lines) {
			reserve = 1
		}
		if used+rows > available-reserve {
			break
		}
		used += rows
		end++
	}

	// If we reached the bottom, recalculate without the below reserve
	if end == len(m.lines) {
		used = 0
		end = from
		for end < len(m.lines) {
			rows := m.lineRows(end)
			if used+rows > available {
				break
			}
			used += rows
			end++
		}
	}
	return end
}

// adjustViewport ensures cursor is visible, vim-style:
// - Moving up: viewport scrolls only when cursor goes above the top
// - Moving down: viewport scrolls only when cursor goes past the bottom
func (m *model) adjustViewport() {
	budget := m.maxContent()

	// Cursor above viewport: snap topLine to cursor
	if m.cursor < m.topLine {
		m.topLine = m.cursor
		return
	}

	// Cursor below viewport: scroll down minimally
	for {
		end := m.visibleEnd(m.topLine, budget)
		if m.cursor < end {
			break
		}
		m.topLine++
		if m.topLine >= len(m.lines) {
			m.topLine = len(m.lines) - 1
			break
		}
	}
}

func (m *model) copyCurrentLine() {
	if m.cursor >= len(m.lines) {
		return
	}
	line := m.lines[m.cursor]
	if err := toClipboard(line); err != nil {
		m.copiedLine = -1
		m.copyErr = err.Error()
	} else {
		m.copiedLine = m.cursor
		m.copyErr = ""
	}
}

func toClipboard(s string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard command found")
		}
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if _, err := pipe.Write([]byte(s)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := pipe.Close(); err != nil {
		return fmt.Errorf("close pipe: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait: %w", err)
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitted = true
			return m, tea.Quit

		case "enter", "c":
			m.copyCurrentLine()

		case "j", "down", " ", "n":
			if m.cursor < len(m.lines)-1 {
				m.cursor++
				m.copiedLine = -1
			}

		case "k", "up", "p":
			if m.cursor > 0 {
				m.cursor--
				m.copiedLine = -1
			}

		case "g", "home":
			m.cursor = 0
			m.topLine = 0
			m.copiedLine = -1

		case "G", "end":
			m.cursor = len(m.lines) - 1
			m.copiedLine = -1
		}
		m.adjustViewport()
	}
	return m, nil
}

// wrapText splits s into lines whose display width does not exceed width.
// Width is measured with runewidth so that CJK/double-width characters are
// handled correctly and multi-byte runes are never split mid-character.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	if runewidth.StringWidth(s) <= width {
		return []string{s}
	}
	var lines []string
	var cur strings.Builder
	curW := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			cur.WriteRune(r)
			continue
		}
		if curW+rw > width {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteRune(r)
		curW += rw
	}
	if cur.Len() > 0 || len(lines) == 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

func (m model) View() string {
	if m.quitted {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.lines))
	b.WriteString(counterStyle.Render(header))
	if m.copiedLine == m.cursor {
		b.WriteString("  " + copiedStyle.Render("✓ copied"))
	} else if m.copyErr != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		b.WriteString("  " + errStyle.Render("✗ "+m.copyErr))
	}
	b.WriteString("\n\n")

	budget := m.maxContent()
	endIdx := m.visibleEnd(m.topLine, budget)

	gutterWidth := len(fmt.Sprintf("%d", len(m.lines)))
	prefixWidth := 2 + gutterWidth + 1
	contentWidth := m.contentWidth()

	ellipsisAbove := m.topLine > 0
	ellipsisBelow := endIdx < len(m.lines)

	if ellipsisAbove {
		b.WriteString(normalStyle.Render(fmt.Sprintf("··· (%d lines above)", m.topLine)))
		b.WriteString("\n")
	}

	for i := m.topLine; i < endIdx; i++ {
		line := m.lines[i]
		lineNum := fmt.Sprintf("%*d ", gutterWidth, i+1)
		indent := strings.Repeat(" ", prefixWidth)

		style := normalStyle
		marker := "  " + lineNum
		if i == m.cursor {
			style = currentStyle
			marker = "▶ " + lineNum
		}

		wrapped := wrapText(line, contentWidth)
		for li, l := range wrapped {
			if li == 0 {
				b.WriteString(style.Render(marker + l))
			} else {
				b.WriteString(style.Render(indent + l))
			}
			b.WriteString("\n")
		}
	}

	if ellipsisBelow {
		b.WriteString(normalStyle.Render(fmt.Sprintf("··· (%d lines below)", len(m.lines)-endIdx)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := "Enter/c:copy  j/↓/Space:next  k/↑:prev  g/G:top/end  q:quit"
	b.WriteString(helpStyle.Render(help))
	b.WriteString("\n")

	return b.String()
}

func main() {
	var scanner *bufio.Scanner

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner = bufio.NewScanner(os.Stdin)
	} else if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	} else {
		fmt.Fprintln(os.Stderr, "Usage: linepaste <file>  or  cat file | linepaste")
		os.Exit(1)
	}

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "No input lines.")
		os.Exit(1)
	}

	tty, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open /dev/tty: %v\n", err)
		os.Exit(1)
	}
	defer tty.Close()

	p := tea.NewProgram(
		initialModel(lines),
		tea.WithInput(tty),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
