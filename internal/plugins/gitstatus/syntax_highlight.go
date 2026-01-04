package gitstatus

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// SyntaxHighlighter provides syntax highlighting for diff content using Chroma.
type SyntaxHighlighter struct {
	lexer chroma.Lexer
	style *chroma.Style
}

// NewSyntaxHighlighter creates a highlighter for the given filename.
// Returns nil if no lexer is available for the file type.
func NewSyntaxHighlighter(filename string) *SyntaxHighlighter {
	lexer := lexers.Match(filename)
	if lexer == nil {
		// Try by extension
		ext := filepath.Ext(filename)
		if ext != "" {
			lexer = lexers.Get(ext)
		}
	}
	if lexer == nil {
		return nil
	}

	// Use monokai style - good contrast on dark backgrounds
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	return &SyntaxHighlighter{
		lexer: chroma.Coalesce(lexer),
		style: style,
	}
}

// HighlightSegment represents a segment of highlighted text.
type HighlightSegment struct {
	Text  string
	Style lipgloss.Style
}

// Highlight tokenizes and highlights a line of code.
// Returns segments with lipgloss styles applied.
func (h *SyntaxHighlighter) Highlight(line string) []HighlightSegment {
	if h == nil || h.lexer == nil {
		return []HighlightSegment{{Text: line, Style: lipgloss.NewStyle()}}
	}

	iterator, err := h.lexer.Tokenise(nil, line)
	if err != nil {
		return []HighlightSegment{{Text: line, Style: lipgloss.NewStyle()}}
	}

	var segments []HighlightSegment
	for _, token := range iterator.Tokens() {
		// Strip trailing newlines - Chroma adds them to some tokens (like comments)
		// and they cause rendering issues with lipgloss width calculations
		text := strings.TrimSuffix(token.Value, "\n")
		if text == "" {
			continue
		}
		style := h.tokenStyle(token.Type)
		segments = append(segments, HighlightSegment{
			Text:  text,
			Style: style,
		})
	}

	return segments
}

// tokenStyle converts a Chroma token type to a lipgloss style.
func (h *SyntaxHighlighter) tokenStyle(tokenType chroma.TokenType) lipgloss.Style {
	entry := h.style.Get(tokenType)
	style := lipgloss.NewStyle()

	if entry.Colour.IsSet() {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Bold == chroma.Yes {
		style = style.Bold(true)
	}
	// Note: Italic is intentionally not applied because it causes
	// width calculation issues in terminal grid layouts (side-by-side diffs).
	// The ANSI italic sequences can affect visual width in some terminals.
	if entry.Underline == chroma.Yes {
		style = style.Underline(true)
	}

	return style
}

// HighlightedLine represents a line with syntax highlighting and diff type info.
type HighlightedLine struct {
	Segments []HighlightSegment
	LineType LineType
}

// RenderHighlightedLine renders a highlighted line, blending syntax colors with diff styles.
func RenderHighlightedLine(segments []HighlightSegment, lineType LineType) string {
	if len(segments) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, seg := range segments {
		// Blend syntax highlighting with diff line type
		style := blendWithDiffStyle(seg.Style, lineType)
		sb.WriteString(style.Render(seg.Text))
	}

	return sb.String()
}

// blendWithDiffStyle blends a syntax highlight style with the diff line style.
// For add/remove lines, we keep the syntax foreground color but may adjust
// brightness to ensure readability on the conceptual "colored" background.
func blendWithDiffStyle(syntaxStyle lipgloss.Style, lineType LineType) lipgloss.Style {
	switch lineType {
	case LineAdd:
		// For added lines, ensure text is readable
		// Keep syntax color but make it slightly brighter if needed
		return syntaxStyle
	case LineRemove:
		// For removed lines, keep syntax color
		return syntaxStyle
	default:
		// Context lines - slightly dim the syntax colors
		return syntaxStyle
	}
}

// HighlightLine highlights a single line of code, returning styled segments.
func (h *SyntaxHighlighter) HighlightLine(content string) []HighlightSegment {
	return h.Highlight(content)
}
