package flow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workflow is a saved sequence of commands. It lives in an editable YAML file.
type Workflow struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Project     string   `yaml:"project,omitempty"`
	Dir         string   `yaml:"dir,omitempty"`
	Shell       string   `yaml:"shell,omitempty"`
	Steps       []string `yaml:"steps"`
}

// ErrExists means a workflow with that name is already saved.
var ErrExists = errors.New("a workflow with that name already exists")

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// ValidName checks a workflow name. Names become file names, so they are strict.
func ValidName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid name %q: use lowercase letters, digits, - and _ (at most 40 characters, starting with a letter or digit)", name)
	}
	return nil
}

// FlowsDir is the folder that holds saved workflows. DEVROUNDS_FLOWS_DIR overrides it.
func FlowsDir() (string, error) {
	if d := os.Getenv("DEVROUNDS_FLOWS_DIR"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "devrounds", "flows"), nil
}

// WorkflowPath is the file a workflow is saved in.
func WorkflowPath(name string) (string, error) {
	if err := ValidName(name); err != nil {
		return "", err
	}
	dir, err := FlowsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".yaml"), nil
}

// Validate checks that a workflow is safe to save or run.
func (w Workflow) Validate() error {
	if err := ValidName(w.Name); err != nil {
		return err
	}
	if len(w.Steps) == 0 {
		return errors.New("a workflow needs at least one step")
	}
	for i, s := range w.Steps {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("step %d is empty", i+1)
		}
		if strings.Contains(s, redacted) {
			return fmt.Errorf("step %d contains a removed secret (%s), so it cannot be saved or run as recorded; run it with the value in an environment variable instead and record it again", i+1, redacted)
		}
	}
	return nil
}

// FromSuggestion turns a suggestion into a workflow.
func FromSuggestion(name string, s Suggestion) Workflow {
	return Workflow{
		Name:        name,
		Description: fmt.Sprintf("%d repeats of this sequence in %s", s.Count, s.Project),
		Project:     s.Project,
		Dir:         s.Dir,
		Steps:       append([]string(nil), s.Steps...),
	}
}

// SaveWorkflow writes a workflow file. It refuses to replace an existing one unless force is set.
func SaveWorkflow(w Workflow, force bool) (string, error) {
	if err := w.Validate(); err != nil {
		return "", err
	}
	path, err := WorkflowPath(w.Name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil && !force {
		return "", ErrExists
	}
	data, err := yaml.Marshal(w)
	if err != nil {
		return "", err
	}
	header := "# devrounds workflow. You can edit it; devrounds shows the steps again before it runs anything.\n"
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(header+string(data)), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return path, nil
}

// LoadWorkflow reads and checks a saved workflow.
func LoadWorkflow(name string) (Workflow, error) {
	path, err := WorkflowPath(name)
	if err != nil {
		return Workflow{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workflow{}, fmt.Errorf("no workflow named %q", name)
		}
		return Workflow{}, err
	}
	var w Workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		return Workflow{}, fmt.Errorf("%s: %w", path, err)
	}
	if w.Name == "" {
		w.Name = name
	}
	if w.Name != name {
		return Workflow{}, fmt.Errorf("%s: the name inside the file (%q) does not match the file name", path, w.Name)
	}
	if err := w.Validate(); err != nil {
		return Workflow{}, fmt.Errorf("%s: %w", path, err)
	}
	return w, nil
}

// ListWorkflows returns the valid saved workflows, sorted by name. Broken files are skipped.
func ListWorkflows() ([]Workflow, error) {
	dir, err := FlowsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Workflow
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".yaml")
		if e.IsDir() || name == e.Name() {
			continue
		}
		if w, err := LoadWorkflow(name); err == nil {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// DeleteWorkflow removes a saved workflow.
func DeleteWorkflow(name string) error {
	path, err := WorkflowPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no workflow named %q", name)
		}
		return err
	}
	return nil
}
