package tui

import (
	"github.com/charmbracelet/lipgloss"
	"lazycider/internal/term/kitty"
)

type layoutInfo struct {
	leftWidth      int
	rightWidth     int
	centerWidth    int
	panelHeight    int
	focusBarHeight int
	rightX         int
	rightY         int

	rightCoverHeight int
	rightQueueHeight int
	rightCoverY      int
	rightQueueY      int
}

func (m Model) layoutInfo() layoutInfo {
	if m.width <= 0 || m.height <= 0 {
		return layoutInfo{}
	}

	leftWidth := (m.width * 20) / 100
	rightWidth := (m.width * 20) / 100

	contentWidth := m.width - 2
	if contentWidth < 0 {
		contentWidth = 0
	}

	if leftWidth+rightWidth > contentWidth {
		overflow := leftWidth + rightWidth - contentWidth
		reduceLeft := overflow / 2
		reduceRight := overflow - reduceLeft
		leftWidth -= reduceLeft
		rightWidth -= reduceRight
		if leftWidth < 0 {
			leftWidth = 0
		}
		if rightWidth < 0 {
			rightWidth = 0
		}
	}

	centerWidth := contentWidth - leftWidth - rightWidth
	if centerWidth < 0 {
		centerWidth = 0
	}

	statusHeight := lipgloss.Height(m.renderStatusBar(m.width))
	focusBarHeight := 1
	if m.height <= statusHeight+1 {
		focusBarHeight = 0
	}
	panelHeight := m.height - statusHeight - focusBarHeight
	if panelHeight < 0 {
		panelHeight = 0
	}

	rightX := leftWidth + 1 + centerWidth + 1
	rightCoverHeight, rightQueueHeight := splitRightPanel(m.width, rightWidth, panelHeight, m.coverW, m.coverH)

	return layoutInfo{
		leftWidth:      leftWidth,
		rightWidth:     rightWidth,
		centerWidth:    centerWidth,
		panelHeight:    panelHeight,
		focusBarHeight: focusBarHeight,
		rightX:         rightX,
		rightY:         0,

		rightCoverHeight: rightCoverHeight,
		rightQueueHeight: rightQueueHeight,
		rightCoverY:      0,
		rightQueueY:      rightCoverHeight,
	}
}

func splitRightPanel(termWidth, rightWidth, panelHeight, coverImgW, coverImgH int) (coverHeight, queueHeight int) {
	if panelHeight <= 0 || rightWidth <= 0 {
		return 0, 0
	}

	const bottomMin = 6
	_, renderedCoverH := kitty.ComputeCoverPlacementSize(termWidth, rightWidth, panelHeight, coverImgW, coverImgH)
	if renderedCoverH <= 0 {
		renderedCoverH = rightWidth
	}

	coverHeight = renderedCoverH
	if coverHeight < 3 {
		coverHeight = 3
	}

	if panelHeight-coverHeight < bottomMin {
		coverHeight = panelHeight - bottomMin
	}

	if coverHeight < 1 {
		coverHeight = panelHeight
	}

	queueHeight = panelHeight - coverHeight
	if queueHeight < 0 {
		queueHeight = 0
	}

	return coverHeight, queueHeight
}
