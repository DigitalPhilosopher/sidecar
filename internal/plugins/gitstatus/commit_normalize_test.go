package gitstatus

import "testing"

func TestNormalizeCommitMessage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "trims whitespace",
			in:   "  fix bug  ",
			want: "Fix bug",
		},
		{
			name: "capitalizes first letter",
			in:   "fix bug",
			want: "Fix bug",
		},
		{
			name: "already capitalized unchanged",
			in:   "Fix bug",
			want: "Fix bug",
		},
		{
			name: "removes trailing period",
			in:   "Fix bug.",
			want: "Fix bug",
		},
		{
			name: "removes multiple trailing periods",
			in:   "Fix bug...",
			want: "Fix bug",
		},
		{
			name: "strips trailing blank lines",
			in:   "Fix bug\n\n",
			want: "Fix bug",
		},
		{
			name: "strips multiple trailing blank lines",
			in:   "Fix bug\n\n\n\n",
			want: "Fix bug",
		},
		{
			name: "truncates long subject",
			in:   "This is a very long commit message that exceeds the seventy-two character limit for subject lines",
			want: "This is a very long commit message that exceeds the seventy-two charact…",
		},
		{
			name: "preserves multi-line body",
			in:   "Fix bug\n\nThis is the body of the commit message.\nIt has multiple lines.",
			want: "Fix bug\n\nThis is the body of the commit message.\nIt has multiple lines.",
		},
		{
			name: "normalizes subject preserves body",
			in:   "fix bug.\n\ndetailed description here.",
			want: "Fix bug\n\ndetailed description here.",
		},
		{
			name: "empty string unchanged",
			in:   "",
			want: "",
		},
		{
			name: "whitespace only returns empty",
			in:   "   \n  \n  ",
			want: "",
		},
		{
			name: "already conforming unchanged",
			in:   "Add feature X",
			want: "Add feature X",
		},
		{
			name: "multi-line with trailing blanks stripped",
			in:   "Fix bug\n\nBody text\n\n\n",
			want: "Fix bug\n\nBody text",
		},
		{
			name: "exactly 72 chars not truncated",
			in:   "123456789012345678901234567890123456789012345678901234567890123456789012",
			want: "123456789012345678901234567890123456789012345678901234567890123456789012",
		},
		{
			name: "73 chars truncated",
			in:   "1234567890123456789012345678901234567890123456789012345678901234567890123",
			want: "12345678901234567890123456789012345678901234567890123456789012345678901…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCommitMessage(tt.in)
			if got != tt.want {
				t.Errorf("NormalizeCommitMessage(%q)\n got: %q\nwant: %q", tt.in, got, tt.want)
			}
		})
	}
}
