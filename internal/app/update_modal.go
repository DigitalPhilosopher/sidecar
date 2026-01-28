package app

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/markdown"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

const changelogURL = "https://raw.githubusercontent.com/marcus/sidecar/main/CHANGELOG.md"

// updateModalWidth returns the appropriate modal width based on screen size.
func (m *Model) updateModalWidth() int {
	modalW := 60
	maxW := m.width - 4
	if maxW < 20 {
		maxW = 20 // Absolute minimum for very small screens
	}
	if modalW > maxW {
		modalW = maxW
	}
	if modalW < 30 {
		modalW = 30
	}
	// Final cap: never exceed available space
	if modalW > maxW {
		modalW = maxW
	}
	return modalW
}

// renderUpdateModalOverlay renders the update modal as an overlay on top of background.
func (m *Model) renderUpdateModalOverlay(background string) string {
	// Render modal content based on state
	var modalContent string
	switch m.updateModalState {
	case UpdateModalPreview:
		modalContent = m.renderUpdatePreviewModal()
	case UpdateModalProgress:
		modalContent = m.renderUpdateProgressModal()
	case UpdateModalComplete:
		modalContent = m.renderUpdateCompleteModal()
	case UpdateModalError:
		modalContent = m.renderUpdateErrorModal()
	default:
		return background
	}

	return ui.OverlayModal(background, modalContent, m.width, m.height)
}

// renderUpdatePreviewModal renders the preview state showing release notes before update.
func (m *Model) renderUpdatePreviewModal() string {
	modalW := m.updateModalWidth()
	contentW := modalW - 4 // Account for borders and padding

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render("Sidecar Update")
	sb.WriteString(centerText(title, contentW))
	sb.WriteString("\n\n")

	// Version comparison
	if m.updateAvailable != nil {
		currentV := m.updateAvailable.CurrentVersion
		latestV := m.updateAvailable.LatestVersion
		arrow := lipgloss.NewStyle().Foreground(styles.Success).Render(" → ")
		versionLine := fmt.Sprintf("%s%s%s", currentV, arrow, latestV)
		sb.WriteString(centerText(versionLine, contentW))
		sb.WriteString("\n")
	}

	// Divider
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Release notes section
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("What's New"))
	sb.WriteString("\n\n")

	// Render release notes
	releaseNotes := m.updateReleaseNotes
	if releaseNotes == "" && m.updateAvailable != nil {
		releaseNotes = m.updateAvailable.ReleaseNotes
	}
	if releaseNotes == "" {
		releaseNotes = "No release notes available."
	}

	// Render markdown release notes
	renderedNotes := m.renderReleaseNotes(releaseNotes, contentW)

	// Limit height of release notes
	lines := strings.Split(renderedNotes, "\n")
	maxLines := 15
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.TextMuted).Render("... (truncated)"))
	}
	sb.WriteString(strings.Join(lines, "\n"))
	sb.WriteString("\n\n")

	// Changelog hint
	changelogHint := lipgloss.NewStyle().Foreground(styles.TextMuted).Render("[c] View Full Changelog")
	sb.WriteString(changelogHint)
	sb.WriteString("\n")

	// Divider
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Buttons
	updateBtn := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(styles.Primary).
		Padding(0, 2).
		Render("Update Now")

	laterBtn := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Padding(0, 2).
		Render("Later")

	buttons := fmt.Sprintf("%s    %s", updateBtn, laterBtn)
	sb.WriteString(centerText(buttons, contentW))
	sb.WriteString("\n\n")

	// Hints
	hints := lipgloss.NewStyle().Foreground(styles.TextMuted).Render("Enter: update   Esc: close")
	sb.WriteString(centerText(hints, contentW))

	// Wrap in modal box
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(1, 2).
		Width(modalW)

	return modalStyle.Render(sb.String())
}

// renderReleaseNotes renders markdown release notes.
func (m *Model) renderReleaseNotes(notes string, width int) string {
	// Try to use markdown renderer
	renderer, err := markdown.NewRenderer()
	if err != nil {
		return notes
	}

	lines := renderer.RenderContent(notes, width)
	return strings.Join(lines, "\n")
}

// centerText centers text within a given width.
func centerText(text string, width int) string {
	textWidth := lipgloss.Width(text)
	if textWidth >= width {
		return text
	}
	padding := (width - textWidth) / 2
	return strings.Repeat(" ", padding) + text
}

// renderUpdateProgressModal renders the progress state during update.
func (m *Model) renderUpdateProgressModal() string {
	modalW := m.updateModalWidth()
	contentW := modalW - 4

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.Warning).Render("Updating Sidecar")
	sb.WriteString(centerText(title, contentW))
	sb.WriteString("\n\n")

	// Version being installed
	if m.updateAvailable != nil {
		version := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(
			fmt.Sprintf("Installing %s", m.updateAvailable.LatestVersion))
		sb.WriteString(centerText(version, contentW))
		sb.WriteString("\n\n")
	}

	// Phase indicators - 3 real, observable phases
	phases := []UpdatePhase{PhaseCheckPrereqs, PhaseInstalling, PhaseVerifying}
	for _, phase := range phases {
		status := m.updatePhaseStatus[phase]
		icon := "○" // pending
		color := styles.TextMuted

		switch status {
		case "running":
			icon = "●"
			color = styles.Warning
		case "done":
			icon = "✓"
			color = styles.Success
		case "error":
			icon = "✗"
			color = styles.Error
		}

		phaseName := phase.String()
		if phase == m.updatePhase && status == "running" {
			phaseName = lipgloss.NewStyle().Bold(true).Render(phaseName)
		}

		phaseLine := fmt.Sprintf("  %s %s",
			lipgloss.NewStyle().Foreground(color).Render(icon),
			phaseName,
		)
		sb.WriteString(phaseLine)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Elapsed time
	elapsed := m.getUpdateElapsed()
	elapsedStr := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(
		fmt.Sprintf("Elapsed: %s", formatElapsed(elapsed)))
	sb.WriteString(centerText(elapsedStr, contentW))
	sb.WriteString("\n\n")

	// Divider
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Cancel hint
	cancelHint := lipgloss.NewStyle().Foreground(styles.TextMuted).Render("Esc: cancel")
	sb.WriteString(centerText(cancelHint, contentW))

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(1, 2).
		Width(modalW)

	return modalStyle.Render(sb.String())
}

// getUpdateElapsed returns the elapsed time since update started.
func (m *Model) getUpdateElapsed() time.Duration {
	if m.updateStartTime.IsZero() {
		return 0
	}
	return time.Since(m.updateStartTime)
}

// formatElapsed formats a duration as M:SS.
func formatElapsed(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// renderUpdateCompleteModal renders the completion state.
func (m *Model) renderUpdateCompleteModal() string {
	modalW := m.updateModalWidth()
	contentW := modalW - 4

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.Success).Render("Update Complete!")
	sb.WriteString(centerText(title, contentW))
	sb.WriteString("\n\n")

	// What was updated
	checkmark := lipgloss.NewStyle().Foreground(styles.Success).Render("✓")

	if m.updateAvailable != nil {
		sb.WriteString(fmt.Sprintf("  %s Sidecar updated to %s\n",
			checkmark, m.updateAvailable.LatestVersion))
	} else {
		sb.WriteString(fmt.Sprintf("  %s Sidecar updated\n", checkmark))
	}

	if m.tdVersionInfo != nil && m.tdVersionInfo.HasUpdate {
		sb.WriteString(fmt.Sprintf("  %s td updated to %s\n",
			checkmark, m.tdVersionInfo.LatestVersion))
	}

	sb.WriteString("\n")

	// Restart message
	restartMsg := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(
		"Restart sidecar to use the new version.")
	sb.WriteString(centerText(restartMsg, contentW))
	sb.WriteString("\n\n")

	// Tip
	tip := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(
		"Tip: Press q to quit, then run 'sidecar' again.")
	sb.WriteString(centerText(tip, contentW))
	sb.WriteString("\n\n")

	// Divider
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Buttons
	quitBtn := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(styles.Success).
		Padding(0, 2).
		Render("Quit & Restart")

	laterBtn := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Padding(0, 2).
		Render("Later")

	buttons := fmt.Sprintf("%s    %s", quitBtn, laterBtn)
	sb.WriteString(centerText(buttons, contentW))
	sb.WriteString("\n\n")

	// Hints
	hints := lipgloss.NewStyle().Foreground(styles.TextMuted).Render("q/Enter: quit   Esc: close")
	sb.WriteString(centerText(hints, contentW))

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(1, 2).
		Width(modalW)

	return modalStyle.Render(sb.String())
}

// renderUpdateErrorModal renders the error state.
func (m *Model) renderUpdateErrorModal() string {
	modalW := m.updateModalWidth()
	contentW := modalW - 4

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.Error).Render("Update Failed")
	sb.WriteString(centerText(title, contentW))
	sb.WriteString("\n\n")

	// Error icon and phase
	errorIcon := lipgloss.NewStyle().Foreground(styles.Error).Render("✗")
	phaseName := m.updatePhase.String()
	sb.WriteString(fmt.Sprintf("  %s Error during: %s\n\n", errorIcon, phaseName))

	// Error message
	errorMsg := m.updateError
	if errorMsg == "" {
		errorMsg = "An unknown error occurred."
	}

	// Wrap error message
	errorStyle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Width(contentW - 4)
	sb.WriteString("  ")
	sb.WriteString(errorStyle.Render(errorMsg))
	sb.WriteString("\n\n")

	// Divider
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Buttons
	retryBtn := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(styles.Warning).
		Padding(0, 2).
		Render("Retry")

	closeBtn := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Padding(0, 2).
		Render("Close")

	buttons := fmt.Sprintf("%s    %s", retryBtn, closeBtn)
	sb.WriteString(centerText(buttons, contentW))
	sb.WriteString("\n\n")

	// Hints
	hints := lipgloss.NewStyle().Foreground(styles.TextMuted).Render("r/Enter: retry   Esc: close")
	sb.WriteString(centerText(hints, contentW))

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(1, 2).
		Width(modalW)

	return modalStyle.Render(sb.String())
}

// fetchChangelog fetches the CHANGELOG.md from GitHub.
func fetchChangelog() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(changelogURL)
		if err != nil {
			return ChangelogLoadedMsg{Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return ChangelogLoadedMsg{Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ChangelogLoadedMsg{Err: err}
		}

		return ChangelogLoadedMsg{Content: string(body)}
	}
}

// renderChangelogOverlay renders the changelog as an overlay on the update preview modal.
func (m *Model) renderChangelogOverlay(background string) string {
	modalW := m.updateModalWidth() + 10 // Wider for changelog
	if modalW > m.width-4 {
		modalW = m.width - 4
	}
	contentW := modalW - 4

	// Calculate available height for changelog content
	modalMaxHeight := m.height - 6
	if modalMaxHeight < 10 {
		modalMaxHeight = 10
	}

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render("Changelog")
	sb.WriteString(centerText(title, contentW))
	sb.WriteString("\n")

	// Divider
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Changelog content
	content := m.updateChangelog
	if content == "" {
		content = "Loading changelog..."
	}

	// Render markdown
	renderedContent := m.renderReleaseNotes(content, contentW)
	lines := strings.Split(renderedContent, "\n")

	// Apply scroll offset and limit lines
	maxContentLines := modalMaxHeight - 8 // Leave room for title, hints, borders
	if maxContentLines < 5 {
		maxContentLines = 5
	}

	startLine := m.changelogScrollOffset
	if startLine > len(lines)-maxContentLines {
		startLine = len(lines) - maxContentLines
	}
	if startLine < 0 {
		startLine = 0
	}

	endLine := startLine + maxContentLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleLines := lines[startLine:endLine]
	sb.WriteString(strings.Join(visibleLines, "\n"))
	sb.WriteString("\n\n")

	// Scroll indicator
	if len(lines) > maxContentLines {
		scrollInfo := fmt.Sprintf("Lines %d-%d of %d", startLine+1, endLine, len(lines))
		sb.WriteString(centerText(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(scrollInfo), contentW))
		sb.WriteString("\n")
	}

	// Divider
	sb.WriteString(lipgloss.NewStyle().Foreground(styles.TextMuted).Render(strings.Repeat("─", contentW)))
	sb.WriteString("\n\n")

	// Hints
	changelogHints := lipgloss.NewStyle().Foreground(styles.TextMuted).Render("j/k scroll   Esc: close")
	sb.WriteString(centerText(changelogHints, contentW))

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(1, 2).
		Width(modalW).
		MaxHeight(modalMaxHeight)

	modalContent := modalStyle.Render(sb.String())
	return ui.OverlayModal(background, modalContent, m.width, m.height)
}
