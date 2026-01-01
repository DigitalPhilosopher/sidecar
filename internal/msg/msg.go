package msg

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ToastMsg displays a temporary message.
type ToastMsg struct {
	Message  string
	Duration time.Duration
}

// ShowToast returns a command to show a toast message.
func ShowToast(message string, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ToastMsg{
			Message:  message,
			Duration: duration,
		}
	}
}
