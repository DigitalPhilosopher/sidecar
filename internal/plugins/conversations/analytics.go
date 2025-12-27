package conversations

import (
	"fmt"
	"strings"
	"time"

	"github.com/sst/sidecar/internal/adapter/claudecode"
	"github.com/sst/sidecar/internal/styles"
)

// renderAnalytics renders the global analytics view with scrolling support.
func (p *Plugin) renderAnalytics() string {
	// Build all content lines first
	var lines []string

	// Load stats
	stats, err := claudecode.LoadStatsCache()
	if err != nil {
		lines = append(lines, styles.PanelHeader.Render(" Usage Analytics"))
		lines = append(lines, styles.Muted.Render(strings.Repeat("━", p.width-2)))
		lines = append(lines, styles.Muted.Render(" Unable to load stats: "+err.Error()))
		p.analyticsLines = lines
		return strings.Join(lines, "\n")
	}

	// Header
	lines = append(lines, styles.PanelHeader.Render(" Usage Analytics                                    [U to close]"))
	lines = append(lines, styles.Muted.Render(strings.Repeat("━", p.width-2)))

	// Summary line
	firstDate := stats.FirstSessionDate.Format("Jan 2")
	summary := fmt.Sprintf(" Since %s  │  %d sessions  │  %s messages",
		firstDate,
		stats.TotalSessions,
		formatLargeNumber(stats.TotalMessages))
	lines = append(lines, styles.Body.Render(summary))
	lines = append(lines, "")

	// Weekly activity chart
	lines = append(lines, styles.PanelHeader.Render(" This Week's Activity"))
	lines = append(lines, styles.Muted.Render(strings.Repeat("─", p.width-2)))

	recentActivity := stats.GetRecentActivity(7)
	maxMsgs := 0
	for _, day := range recentActivity {
		if day.MessageCount > maxMsgs {
			maxMsgs = day.MessageCount
		}
	}

	for _, day := range recentActivity {
		date, _ := time.Parse("2006-01-02", day.Date)
		dayName := date.Format("Mon")
		bar := renderBar(day.MessageCount, maxMsgs, 16)
		line := fmt.Sprintf(" %s │ %s │ %5d msgs │ %2d sessions",
			dayName,
			bar,
			day.MessageCount,
			day.SessionCount)
		lines = append(lines, styles.Muted.Render(line))
	}
	lines = append(lines, "")

	// Model usage
	lines = append(lines, styles.PanelHeader.Render(" Model Usage"))
	lines = append(lines, styles.Muted.Render(strings.Repeat("─", p.width-2)))

	// Find max tokens for bar scaling
	var maxTokens int64
	for _, usage := range stats.ModelUsage {
		total := int64(usage.InputTokens) + int64(usage.OutputTokens)
		if total > maxTokens {
			maxTokens = total
		}
	}

	for model, usage := range stats.ModelUsage {
		shortName := modelShortName(model)
		if shortName == "" {
			continue
		}

		totalTokens := int64(usage.InputTokens) + int64(usage.OutputTokens)
		bar := renderBar64(totalTokens, maxTokens, 12)
		cost := claudecode.CalculateModelCost(model, usage)

		line := fmt.Sprintf(" %-6s │ %s │ %s in  %s out │ ~$%.0f",
			shortName,
			bar,
			formatLargeNumber64(int64(usage.InputTokens)),
			formatLargeNumber64(int64(usage.OutputTokens)),
			cost)
		lines = append(lines, styles.Muted.Render(line))
	}
	lines = append(lines, "")

	// Stats footer
	cacheEff := stats.CacheEfficiency()
	lines = append(lines, styles.Muted.Render(fmt.Sprintf(" Cache Efficiency: %.0f%%", cacheEff)))

	// Peak hours
	peakHours := stats.GetPeakHours(3)
	if len(peakHours) > 0 {
		peakStr := " Peak Hours:"
		for i, ph := range peakHours {
			if i > 0 {
				peakStr += ","
			}
			peakStr += fmt.Sprintf(" %s:00", ph.Hour)
		}
		lines = append(lines, styles.Muted.Render(peakStr))
	}

	// Longest session
	if stats.LongestSession.Duration > 0 {
		dur := time.Duration(stats.LongestSession.Duration) * time.Millisecond
		lines = append(lines, styles.Muted.Render(fmt.Sprintf(" Longest Session: %s", formatSessionDuration(dur))))
	}

	// Total cost
	totalCost := stats.TotalCost()
	lines = append(lines, styles.Muted.Render(fmt.Sprintf(" Total Estimated Cost: ~$%.0f", totalCost)))

	// Store lines for scroll calculation
	p.analyticsLines = lines

	// Apply scroll offset and height constraint
	contentHeight := p.height - 2 // leave room for potential padding
	if contentHeight < 1 {
		contentHeight = 1
	}

	start := p.analyticsScrollOff
	if start >= len(lines) {
		start = len(lines) - 1
		if start < 0 {
			start = 0
		}
	}
	end := start + contentHeight
	if end > len(lines) {
		end = len(lines)
	}

	visibleLines := lines[start:end]
	return strings.Join(visibleLines, "\n")
}

// renderBar renders an ASCII bar chart segment.
func renderBar(value, max, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	filled := (value * width) / max
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// renderBar64 renders an ASCII bar chart segment for int64 values.
func renderBar64(value, max int64, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	filled := int((value * int64(width)) / max)
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// formatLargeNumber formats a number with K/M suffix.
func formatLargeNumber(n int) string {
	return formatLargeNumber64(int64(n))
}

// formatLargeNumber64 formats an int64 with K/M/B suffix.
func formatLargeNumber64(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
