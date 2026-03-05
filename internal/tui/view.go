package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	l := m.layoutInfo()

	focusBar := m.renderFocusBar(l)
	statusBar := m.renderStatusBar(m.width)

	left := lipgloss.NewStyle().
		Background(gruvBg).
		Width(l.leftWidth).
		Height(l.panelHeight).
		Render(m.renderLeftPanel(l.leftWidth, l.panelHeight))

	dividerLines := make([]string, l.panelHeight)
	for i := range dividerLines {
		dividerLines[i] = lipgloss.NewStyle().
			Foreground(gruvDivider).
			Background(gruvBg).
			Faint(true).
			Render("│")
	}
	divider := strings.Join(dividerLines, "\n")

	center := lipgloss.NewStyle().
		Background(gruvBg).
		Width(l.centerWidth).
		Height(l.panelHeight).
		Render(m.renderCenterPanel(l.centerWidth, l.panelHeight))

	right := lipgloss.NewStyle().
		Background(gruvBg).
		Width(l.rightWidth).
		Height(l.panelHeight).
		Render(m.renderRightPanel(l.rightWidth, l.panelHeight, l.rightCoverHeight, l.rightQueueHeight))

	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, center, divider, right)

	return lipgloss.NewStyle().
		Background(gruvBg).
		Width(m.width).
		Height(m.height).
		Render(lipgloss.JoinVertical(lipgloss.Left, panels, focusBar, statusBar))
}

func (m Model) renderRightPanel(width, panelHeight, coverHeight, queueHeight int) string {
	if width <= 0 {
		return ""
	}
	coverBG := gruvBg
	queueBG := gruvBg

	if m.rightPanelMode == RightPanelLyrics {
		lyrics := strings.TrimSpace(m.lyricsText)
		if lyrics == "" {
			lyrics = "lyrics loading..."
		}
		if m.lyricsErr != "" {
			lyrics = "lyrics error\n\n" + m.lyricsErr
		}

		content := "Lyrics\n\n" + lyrics
		return lipgloss.NewStyle().
			Foreground(gruvFg).
			Background(queueBG).
			Width(width).
			Height(panelHeight).
			Align(lipgloss.Left, lipgloss.Top).
			Render(content)
	}

	coverContent := ""
	if coverHeight > 0 {
		if m.coverErr != "" {
			coverContent = lipgloss.NewStyle().
				Foreground(gruvGray).
				Background(coverBG).
				Width(width).
				Height(coverHeight).
				Render("cover error\n" + m.coverErr)
		} else if m.coverPath == "" {
			coverContent = lipgloss.NewStyle().
				Foreground(gruvGray).
				Background(coverBG).
				Width(width).
				Height(coverHeight).
				Align(lipgloss.Center).
				Render("loading cover...")
		} else {
			coverContent = lipgloss.NewStyle().
				Background(coverBG).
				Width(width).
				Height(coverHeight).
				Render("")
		}
	}

	queueContent := ""
	if queueHeight > 0 {
		visible := upNextVisibleItems(queueHeight)
		lines := make([]string, 0, visible*2+2)
		lines = append(lines, fitDisplayWidth("Up Next", width), "")

		for i := 0; i < visible; i++ {
			idx := m.upNextTop + i
			if idx >= len(m.upNext) {
				break
			}

			item := m.upNext[idx]
			line1 := fitDisplayWidth("• "+item.Title, width)
			line2 := fitDisplayWidth("  "+item.Subtitle, width)

			if idx == m.upNextSelected {
				hl := lipgloss.NewStyle().
					Foreground(gruvBg).
					Background(gruvYellow).
					Bold(true).
					Width(width)
				line1 = hl.Render(line1)
				line2 = hl.Render(line2)
			} else {
				line1 = lipgloss.NewStyle().Foreground(gruvFg).Render(line1)
				line2 = lipgloss.NewStyle().Foreground(gruvGray).Render(line2)
			}

			lines = append(lines, line1, line2)
		}

		queueText := strings.Join(lines, "\n")
		queueContent = lipgloss.NewStyle().
			Foreground(gruvFg).
			Background(queueBG).
			Width(width).
			Height(queueHeight).
			Render(queueText)
	}

	if coverContent == "" {
		return queueContent
	}
	if queueContent == "" {
		return coverContent
	}

	return lipgloss.JoinVertical(lipgloss.Left, coverContent, queueContent)
}

func (m Model) renderNowPlaying(width int) string {
	bg := gruvBg
	center := func(s string) string {
		return lipgloss.NewStyle().Background(bg).Width(width).Align(lipgloss.Center).Render(s)
	}

	track := lipgloss.NewStyle().
		Foreground(gruvFg).
		Background(bg).
		Bold(true).
		Render(m.track)

	meta := lipgloss.NewStyle().
		Foreground(gruvGray).
		Background(bg).
		Render(fmt.Sprintf("%s — %s", m.artist, m.album))

	barWidth := (width * 2) / 5
	filled := int(float64(barWidth) * m.progress)
	bar := lipgloss.NewStyle().Foreground(gruvGreen).Background(bg).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(gruvGray).Background(bg).Render(strings.Repeat("━", barWidth-filled))

	timeStr := lipgloss.NewStyle().
		Foreground(gruvGray).
		Background(bg).
		Render(fmt.Sprintf("%s  %s  %s", m.current, bar, m.total))

	return lipgloss.JoinVertical(lipgloss.Left,
		center(track),
		center(meta),
		center(timeStr),
	)
}

func (m Model) renderCenterPanel(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	playerHeight := centerPlayerHeight(height)
	if height < playerHeight {
		playerHeight = height
	}

	innerHeight := height - playerHeight
	if innerHeight < 0 {
		innerHeight = 0
	}

	player := lipgloss.NewStyle().
		Background(gruvBg).
		Width(width).
		Height(playerHeight).
		Render(m.renderNowPlaying(width))

	if innerHeight == 0 {
		return player
	}

	listRows := centerVisibleItems(height)
	if listRows < 1 {
		listRows = 1
	}

	innerLines := make([]string, 0, innerHeight)
	title := fitDisplayWidth(m.centerTitle, width)
	innerLines = append(innerLines, lipgloss.NewStyle().Foreground(gruvFg).Bold(true).Render(title))
	header := formatCenterColumns("#", "Title", "Artist", "Duration", width)
	innerLines = append(innerLines, lipgloss.NewStyle().Foreground(gruvFg).Bold(true).Render(header))

	visibleSongs := m.centerSongs
	if m.centerTop < len(m.centerSongs) {
		visibleSongs = m.centerSongs[m.centerTop:]
	} else {
		visibleSongs = nil
	}
	if len(visibleSongs) > listRows {
		visibleSongs = visibleSongs[:listRows]
	}

	for i := 0; i < len(visibleSongs); i++ {
		s := visibleSongs[i]
		dur := strings.TrimSpace(s.Duration)
		if dur == "" {
			dur = "--:--"
		}
		abs := m.centerTop + i
		idx := strconv.Itoa(abs + 1)
		playingRow := false
		if strings.TrimSpace(s.ID) != "" && strings.TrimSpace(s.ID) == strings.TrimSpace(m.trackID) {
			idx = "▶"
			playingRow = true
		}
		line := formatCenterColumns(idx, s.Title, s.Artist, dur, width)
		if playingRow {
			markerCell := padLeftDisplay("▶", 3)
			coloredCell := strings.Replace(markerCell, "▶", lipgloss.NewStyle().Foreground(gruvGreen).Bold(true).Render("▶"), 1)
			line = strings.Replace(line, markerCell, coloredCell, 1)
		}
		if m.focus == PanelCenter && abs == m.centerSelected {
			hl := lipgloss.NewStyle().
				Foreground(gruvBg).
				Background(gruvYellow).
				Bold(true).
				Width(width)
			line = hl.Render(line)
		} else {
			line = lipgloss.NewStyle().Foreground(gruvGray).Render(line)
		}
		innerLines = append(innerLines, line)
	}

	for len(innerLines) < innerHeight {
		innerLines = append(innerLines, strings.Repeat(" ", width))
	}
	inner := lipgloss.NewStyle().
		Foreground(gruvGray).
		Background(gruvBg).
		Width(width).
		Height(innerHeight).
		Render(strings.Join(innerLines, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, player, inner)
}

func formatCenterColumns(num, title, artist, duration string, width int) string {
	if width <= 0 {
		return ""
	}

	numWidth := 3
	durWidth := 7
	spacer := 2
	artistWidth := width / 4
	if artistWidth < 10 {
		artistWidth = 10
	}
	titleWidth := width - numWidth - artistWidth - durWidth - (spacer * 3)
	if titleWidth < 8 {
		titleWidth = 8
		artistWidth = width - numWidth - durWidth - titleWidth - (spacer * 3)
		if artistWidth < 6 {
			artistWidth = 6
		}
	}

	n := padLeftDisplay(truncateDisplay(num, numWidth), numWidth)
	t := padRightDisplay(truncateDisplay(title, titleWidth), titleWidth)
	a := padRightDisplay(truncateDisplay(artist, artistWidth), artistWidth)
	d := padLeftDisplay(truncateDisplay(duration, durWidth), durWidth)
	row := n + strings.Repeat(" ", spacer) + t + strings.Repeat(" ", spacer) + a + strings.Repeat(" ", spacer) + d
	return fitDisplayWidth(row, width)
}

func (m Model) renderLeftPanel(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	bg := gruvBg

	leftText := "Search..."
	rightIcon := ""
	barWidth := width - 4
	minWidth := lipgloss.Width(leftText) + lipgloss.Width(rightIcon) + 3
	if barWidth < minWidth {
		barWidth = minWidth
	}
	if barWidth > width {
		barWidth = width
	}

	gap := barWidth - lipgloss.Width(leftText) - lipgloss.Width(rightIcon) - 2
	if gap < 1 {
		gap = 1
	}
	searchLabel := " " + leftText + strings.Repeat(" ", gap) + rightIcon + " "

	searchBar := lipgloss.NewStyle().
		Foreground(gruvFg).
		Background(gruvSearchBg).
		Bold(true).
		Width(barWidth).
		Align(lipgloss.Left).
		Render(searchLabel)

	searchRow := lipgloss.PlaceHorizontal(width, lipgloss.Center, searchBar)

	topHeight := 3
	if height < topHeight {
		topHeight = height
	}
	bottomHeight := height - topHeight
	rows := m.leftVisibleRows()
	lineCount := bottomHeight
	if lineCount < 1 {
		lineCount = 1
	}
	visible := rows
	if m.leftTop < len(rows) {
		visible = rows[m.leftTop:]
	} else {
		visible = nil
	}
	if len(visible) > lineCount {
		visible = visible[:lineCount]
	}

	lines := make([]string, 0, lineCount)
	for i := 0; i < len(visible); i++ {
		r := visible[i]
		prefix := "  "
		if !r.IsModule {
			prefix = "    "
		}
		line := fitDisplayWidth(prefix+r.Text, width)
		abs := m.leftTop + i
		if m.focus == PanelLeft && abs == m.leftSelected {
			hl := lipgloss.NewStyle().
				Foreground(gruvBg).
				Background(gruvYellow).
				Bold(true).
				Width(width)
			line = hl.Render(line)
		} else {
			if r.IsModule {
				line = lipgloss.NewStyle().Foreground(gruvFg).Bold(true).Render(line)
			} else {
				line = lipgloss.NewStyle().Foreground(gruvGray).Render(line)
			}
		}
		lines = append(lines, line)
	}
	for len(lines) < lineCount {
		lines = append(lines, strings.Repeat(" ", width))
	}
	bottomContent := strings.Join(lines, "\n")

	topSection := lipgloss.NewStyle().
		Background(bg).
		Width(width).
		Height(topHeight).
		Render("\n" + searchRow)

	bottomSection := lipgloss.NewStyle().
		Background(bg).
		Width(width).
		Height(bottomHeight).
		Render(bottomContent)

	return lipgloss.NewStyle().
		Background(bg).
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, topSection, bottomSection))
}

func fitDisplayWidth(s string, width int) string {
	return padRightDisplay(truncateDisplay(s, width), width)
}

func (m Model) renderFocusBar(l layoutInfo) string {
	if l.focusBarHeight == 0 {
		return ""
	}

	seg := func(width int, focused bool) string {
		if width <= 0 {
			return ""
		}
		line := strings.Repeat(" ", width)
		if focused {
			line = strings.Repeat("━", width)
		}
		return lipgloss.NewStyle().Background(gruvBg).Foreground(gruvDivider).Width(width).Render(line)
	}

	divider := lipgloss.NewStyle().Background(gruvBg).Width(1).Render(" ")
	left := seg(l.leftWidth, m.focus == PanelLeft)
	center := seg(l.centerWidth, m.focus == PanelCenter)
	right := seg(l.rightWidth, m.focus == PanelRight)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, divider, center, divider, right)
}

func truncateDisplay(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}

	target := width - 1
	current := 0
	var b strings.Builder
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if current+rw > target {
			break
		}
		b.WriteRune(r)
		current += rw
	}
	return b.String() + "…"
}

func padRightDisplay(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func padLeftDisplay(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

func (m Model) renderStatusBar(width int) string {
	if m.state == StateCommand {
		return lipgloss.NewStyle().Foreground(gruvFg).Background(gruvBg).Width(width).Render(m.cmdInput)
	}

	state := "⏸ paused"
	if m.playing {
		state = "▶ playing"
	}

	stateStr := lipgloss.NewStyle().Foreground(gruvBg).Background(gruvYellow).Bold(true).Render(" " + state + " ")
	dimStr := lipgloss.NewStyle().Foreground(gruvBg).Background(gruvGray).Bold(true).Render(fmt.Sprintf(" %dx%d ", m.width, m.height))
	volStr := lipgloss.NewStyle().Foreground(gruvBg).Background(gruvAqua).Bold(true).Render(fmt.Sprintf(" vol %d%% ", m.volume))
	focusStr := lipgloss.NewStyle().Foreground(gruvBg).Background(gruvOrange).Bold(true).Render(" focus " + m.focusLabel() + " ")

	gap := width - lipgloss.Width(stateStr) - lipgloss.Width(dimStr) - lipgloss.Width(volStr) - lipgloss.Width(focusStr)
	if gap < 0 {
		gap = 0
	}

	return stateStr + lipgloss.NewStyle().Background(gruvBg).Render(strings.Repeat(" ", gap)) + focusStr + dimStr + volStr
}
