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
)

var (
	currentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	upcomingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	doneStyle = lipgloss.NewStyle().
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
	copiedLine int // -1 = not copied yet
	copyErr    string
	quitted    bool
	width      int
	height     int
}

func initialModel(lines []string) model {
	return model{lines: lines, cursor: 0, copiedLine: -1}
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
			m.copiedLine = -1

		case "G", "end":
			m.cursor = len(m.lines) - 1
			m.copiedLine = -1
		}
	}
	return m, nil
}

func wrapText(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	var b strings.Builder
	for len(s) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		end := width
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[:end])
		s = s[end:]
	}
	return b.String()
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

	// Reserve rows: header 1 + blank 1 + help separator 1 + help 1 + margin 2 = 6
	maxContentRows := m.height - 6
	if maxContentRows < 5 {
		maxContentRows = 5
	}

	// Line number gutter width (based on total lines)
	gutterWidth := len(fmt.Sprintf("%d", len(m.lines)))
	// Layout: "▶ " or "  " (2 chars) + lineNum (gutterWidth) + " " + content
	prefixWidth := 2 + gutterWidth + 1
	contentWidth := w - prefixWidth - 3 // padding margin
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Pre-render all lines and count their display rows
	type renderedLine struct {
		text string
		rows int
	}

	renderLine := func(i int) renderedLine {
		line := m.lines[i]
		if line == "" {
			line = "⏎"
		}
		lineNum := fmt.Sprintf("%*d ", gutterWidth, i+1)
		indent := strings.Repeat(" ", prefixWidth)

		var sb strings.Builder
		if i == m.cursor {
			marker := "▶ " + lineNum
			wrapped := wrapText(line, contentWidth)
			parts := strings.Split(wrapped, "\n")
			for li, l := range parts {
				if li == 0 {
					sb.WriteString(currentStyle.Render(marker + l))
				} else {
					sb.WriteString(currentStyle.Render(indent + l))
				}
				sb.WriteString("\n")
			}
			return renderedLine{sb.String(), len(parts)}
		}
		marker := "  " + lineNum
		style := upcomingStyle
		if i < m.cursor {
			style = doneStyle
		}
		wrapped := wrapText(line, contentWidth)
		parts := strings.Split(wrapped, "\n")
		for li, l := range parts {
			if li == 0 {
				sb.WriteString(style.Render(marker + l))
			} else {
				sb.WriteString(style.Render(indent + l))
			}
			sb.WriteString("\n")
		}
		return renderedLine{sb.String(), len(parts)}
	}

	// Build visible lines: start from cursor, expand outward
	// Always include the current line first
	curRendered := renderLine(m.cursor)
	usedRows := curRendered.rows

	// Expand before cursor
	startIdx := m.cursor
	for s := m.cursor - 1; s >= 0; s-- {
		r := renderLine(s)
		if usedRows+r.rows > maxContentRows {
			break
		}
		usedRows += r.rows
		startIdx = s
	}

	// Expand after cursor
	endIdx := m.cursor + 1
	for e := m.cursor + 1; e < len(m.lines); e++ {
		r := renderLine(e)
		if usedRows+r.rows > maxContentRows {
			break
		}
		usedRows += r.rows
		endIdx = e + 1
	}

	// Trim back to make room for "···" indicator lines
	ellipsisAbove := startIdx > 0
	ellipsisBelow := endIdx < len(m.lines)
	needed := 0
	if ellipsisAbove {
		needed++
	}
	if ellipsisBelow {
		needed++
	}
	// Remove lines from the edges until we have room for ellipsis
	for usedRows+needed > maxContentRows && endIdx > m.cursor+1 {
		endIdx--
		usedRows -= renderLine(endIdx).rows
		if endIdx < len(m.lines) {
			ellipsisBelow = true
		}
	}
	for usedRows+needed > maxContentRows && startIdx < m.cursor {
		usedRows -= renderLine(startIdx).rows
		startIdx++
		if startIdx > 0 {
			ellipsisAbove = true
		}
	}
	// Recalculate needed after trimming
	needed = 0
	if ellipsisAbove {
		needed++
	}
	if ellipsisBelow {
		needed++
	}

	// Render
	if ellipsisAbove {
		b.WriteString(upcomingStyle.Render(fmt.Sprintf("··· (%d lines above)", startIdx)))
		b.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		r := renderLine(i)
		b.WriteString(r.text)
	}

	if ellipsisBelow {
		b.WriteString(upcomingStyle.Render(fmt.Sprintf("··· (%d lines below)", len(m.lines)-endIdx)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := "Enter/c:copy  j/↓/Space:next  k/↑:prev  g/G:top/end  q:quit"
	b.WriteString(helpStyle.Render(help))
	b.WriteString("\n")

	return b.String()
}

func main() {
	// Read lines from stdin or file
	var scanner *bufio.Scanner

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input
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

	// Re-open /dev/tty for bubbletea input
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
