package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterWorktrees(t *testing.T) {
	worktrees := []WorktreeInfo{
		{Path: "/main/repo", Branch: "main", IsMain: true},
		{Path: "/worktrees/feature-auth", Branch: "feature-auth", IsMain: false},
		{Path: "/worktrees/feature-billing", Branch: "feature-billing", IsMain: false},
		{Path: "/worktrees/bugfix-login", Branch: "bugfix-login", IsMain: false},
	}

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"empty query returns all", "", 4},
		{"filter by branch name", "feature", 2},
		{"filter by auth", "auth", 1},
		{"filter by billing", "billing", 1},
		{"filter by bugfix", "bugfix", 1},
		{"filter by main", "main", 1},
		{"case insensitive", "FEATURE", 2},
		{"no matches", "nonexistent", 0},
		{"partial match", "log", 1}, // matches "bugfix-login"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterWorktrees(worktrees, tt.query)
			if len(result) != tt.expected {
				t.Errorf("filterWorktrees(%q) returned %d results, want %d", tt.query, len(result), tt.expected)
			}
		})
	}
}

func TestWorktreeSwitcherEnsureCursorVisible(t *testing.T) {
	tests := []struct {
		name       string
		cursor     int
		scroll     int
		maxVisible int
		expected   int
	}{
		{"cursor in view", 3, 0, 8, 0},
		{"cursor at top, need to scroll up", 2, 5, 8, 2},
		{"cursor at bottom, need to scroll down", 10, 0, 8, 3},
		{"cursor at edge", 7, 0, 8, 0},
		{"cursor just past edge", 8, 0, 8, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := worktreeSwitcherEnsureCursorVisible(tt.cursor, tt.scroll, tt.maxVisible)
			if result != tt.expected {
				t.Errorf("worktreeSwitcherEnsureCursorVisible(%d, %d, %d) = %d, want %d",
					tt.cursor, tt.scroll, tt.maxVisible, result, tt.expected)
			}
		})
	}
}

func TestWorktreeExists(t *testing.T) {
	// Create a temp directory to test with
	tempDir := t.TempDir()

	// Create a mock .git file
	gitPath := filepath.Join(tempDir, ".git")
	if err := os.WriteFile(gitPath, []byte("gitdir: /path/to/main/.git/worktrees/test"), 0644); err != nil {
		t.Fatalf("failed to create .git file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"valid directory with .git", tempDir, true},
		{"non-existent directory", "/nonexistent/path/12345", false},
		{"file instead of directory", gitPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WorktreeExists(tt.path)
			if result != tt.expected {
				t.Errorf("WorktreeExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestWorktreeSwitcherItemID(t *testing.T) {
	tests := []struct {
		idx      int
		expected string
	}{
		{0, "worktree-switcher-item-0"},
		{1, "worktree-switcher-item-1"},
		{10, "worktree-switcher-item-10"},
		{99, "worktree-switcher-item-99"},
	}

	for _, tt := range tests {
		result := worktreeSwitcherItemID(tt.idx)
		if result != tt.expected {
			t.Errorf("worktreeSwitcherItemID(%d) = %q, want %q", tt.idx, result, tt.expected)
		}
	}
}

func TestCheckCurrentWorktree(t *testing.T) {
	// Test with non-existent path
	exists, mainPath := CheckCurrentWorktree("/nonexistent/path/that/does/not/exist")
	if exists {
		t.Error("CheckCurrentWorktree should return false for non-existent path")
	}
	// mainPath may or may not be found depending on the test environment
	_ = mainPath

	// Test with existing path (use temp dir as a valid directory)
	tempDir := t.TempDir()
	gitPath := filepath.Join(tempDir, ".git")
	if err := os.WriteFile(gitPath, []byte("gitdir: /path/to/main/.git/worktrees/test"), 0644); err != nil {
		t.Fatalf("failed to create .git file: %v", err)
	}

	exists, _ = CheckCurrentWorktree(tempDir)
	if !exists {
		t.Error("CheckCurrentWorktree should return true for existing directory with .git")
	}
}
