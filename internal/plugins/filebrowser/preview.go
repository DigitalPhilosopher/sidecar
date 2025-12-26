package filebrowser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	maxPreviewSize  = 500 * 1024 // 500KB
	maxPreviewLines = 10000
)

// PreviewResult contains the loaded file content.
type PreviewResult struct {
	Content          string
	Lines            []string
	HighlightedLines []string // Syntax highlighted lines
	IsBinary         bool
	IsTruncated      bool
	TotalSize        int64
	Error            error
}

// PreviewLoadedMsg signals that file preview content is ready.
type PreviewLoadedMsg struct {
	Result PreviewResult
	Path   string
}

// LoadPreview creates a command to load file content.
func LoadPreview(rootDir, path string) tea.Cmd {
	return func() tea.Msg {
		fullPath := filepath.Join(rootDir, path)

		info, err := os.Stat(fullPath)
		if err != nil {
			return PreviewLoadedMsg{
				Path:   path,
				Result: PreviewResult{Error: err},
			}
		}

		result := PreviewResult{TotalSize: info.Size()}

		// Check size limit
		readSize := info.Size()
		if readSize > maxPreviewSize {
			readSize = maxPreviewSize
			result.IsTruncated = true
		}

		// Read file
		f, err := os.Open(fullPath)
		if err != nil {
			result.Error = err
			return PreviewLoadedMsg{Path: path, Result: result}
		}
		defer f.Close()

		data := make([]byte, readSize)
		n, _ := f.Read(data)
		data = data[:n]

		// Check for binary (fm pattern)
		if isBinary(data) {
			result.IsBinary = true
			return PreviewLoadedMsg{Path: path, Result: result}
		}

		result.Content = string(data)
		result.Lines = strings.Split(result.Content, "\n")

		// Apply syntax highlighting
		highlighted, err := Highlight(result.Content, filepath.Ext(path), "monokai")
		if err == nil {
			result.HighlightedLines = strings.Split(highlighted, "\n")
		} else {
			// Fallback to raw lines
			result.HighlightedLines = result.Lines
		}

		// Limit lines
		if len(result.Lines) > maxPreviewLines {
			result.Lines = result.Lines[:maxPreviewLines]
			result.HighlightedLines = result.HighlightedLines[:maxPreviewLines]
			result.IsTruncated = true
		}

		return PreviewLoadedMsg{Path: path, Result: result}
	}
}

// Highlight returns a syntax highlighted string.
// Pattern from knipferrc/fm code/code.go
func Highlight(content, extension, syntaxTheme string) (string, error) {
	buf := new(bytes.Buffer)
	if err := quick.Highlight(buf, content, extension, "terminal256", syntaxTheme); err != nil {
		return "", fmt.Errorf("highlight: %w", err)
	}
	return buf.String(), nil
}

// isBinary checks if data contains null bytes in first 512 bytes.
// Pattern from knipferrc/fm filesystem/filesystem.go
func isBinary(data []byte) bool {
	checkLen := 512
	if len(data) < checkLen {
		checkLen = len(data)
	}
	return bytes.Contains(data[:checkLen], []byte{0})
}
