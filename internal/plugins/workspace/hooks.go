package workspace

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

// Hook represents a post-create hook command.
type Hook struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Source  string `json:"-"` // "global" or "project" (set at load time)
}

// configWithHooks is the config structure for loading hooks.
type configWithHooks struct {
	Hooks *HooksConfig `json:"hooks"`
}

// HooksConfig holds hook configuration by lifecycle event.
type HooksConfig struct {
	PostCreate []Hook `json:"post-create"`
}

// LoadHooks loads and merges hooks from global and project config directories.
// Project hooks override global hooks with the same name.
// Returns sorted list by name.
func LoadHooks(globalConfigDir, projectDir string) []Hook {
	// Load from global config
	globalHooks := loadHooksFromDir(globalConfigDir, "global")

	// Load from project config (.sidecar/ directory)
	projectConfigDir := filepath.Join(projectDir, ".sidecar")
	projectHooks := loadHooksFromDir(projectConfigDir, "project")

	// Merge: project overrides global by name
	merged := make(map[string]Hook)
	for _, h := range globalHooks {
		merged[h.Name] = h
	}
	for _, h := range projectHooks {
		merged[h.Name] = h
	}

	// Convert to sorted slice
	result := make([]Hook, 0, len(merged))
	for _, h := range merged {
		result = append(result, h)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// loadHooksFromDir loads hooks from a config.json file in the given directory.
func loadHooksFromDir(dir, source string) []Hook {
	path := filepath.Join(dir, "config.json")
	hooks, err := loadHooksFromFile(path, source)
	if err == nil && len(hooks) > 0 {
		return hooks
	}
	return nil
}

// loadHooksFromFile loads hooks from a JSON config file.
func loadHooksFromFile(path, source string) ([]Hook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg configWithHooks
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Hooks == nil {
		return nil, nil
	}

	// Set source on all hooks
	for i := range cfg.Hooks.PostCreate {
		cfg.Hooks.PostCreate[i].Source = source
	}

	return cfg.Hooks.PostCreate, nil
}

// HookResultMsg signals that post-create hooks have completed.
type HookResultMsg struct {
	WorktreeName string
	Results      []HookExecResult
}

// HookExecResult holds the result of a single hook execution.
type HookExecResult struct {
	Name    string
	Command string
	Output  string
	Err     error
}

// runPostCreateHooks executes selected hooks in the new worktree directory.
func (p *Plugin) runPostCreateHooks(wt *Worktree, hooks []Hook) tea.Cmd {
	wtPath := wt.Path
	wtName := wt.Name
	mainWorkDir := p.ctx.WorkDir
	return func() tea.Msg {
		var results []HookExecResult
		isolatedEnv := ApplyEnvOverrides(os.Environ(), BuildEnvOverrides(mainWorkDir))
		for _, hook := range hooks {
			cmd := exec.Command("bash", "-c", hook.Command)
			cmd.Dir = wtPath
			cmd.Env = append(isolatedEnv,
				"MAIN_WORKTREE="+mainWorkDir,
				"WORKTREE_BRANCH="+wt.Branch,
				"WORKTREE_PATH="+wtPath,
			)
			output, err := cmd.CombinedOutput()
			results = append(results, HookExecResult{
				Name:    hook.Name,
				Command: hook.Command,
				Output:  string(output),
				Err:     err,
			})
		}
		return HookResultMsg{WorktreeName: wtName, Results: results}
	}
}
