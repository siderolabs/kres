// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lefthook

// Pre-commit stage keys: stable names for Hook.Group so multiple project blocks
// can append jobs to the same ordered group. Each value is both the lookup key
// and the group's emitted `name:`.
const (
	// PreCommitFixStage is stage 1: mutating formatters/generators, re-staged via stage_fixed.
	PreCommitFixStage = "fix"
	// PreCommitLintStage is stage 2: verification, runs after the fix stage.
	PreCommitLintStage = "lint"

	// HookGroupPreCommit is the lefthook "pre-commit" hook name.
	HookGroupPreCommit = "pre-commit"
	// HookGroupCommitMsg is the lefthook "commit-msg" hook name.
	HookGroupCommitMsg = "commit-msg"
)

// Hook represents the configuration for a single git hook (e.g. pre-commit)
// inside lefthook.yml. A hook can declare either Commands (a named map) or
// Jobs (an ordered list with nested groups) — see lefthook docs for the
// trade-off; mixing both in a single hook is generally not recommended.
type Hook struct { //nolint:govet
	// Parallel is a pointer so an explicit `parallel: false` can be emitted
	// (with a plain bool + omitempty the false zero-value would be suppressed).
	// Only meaningful for Commands-style hooks; Jobs-style hooks control
	// parallelism per-Group.
	Parallel *bool               `yaml:"parallel,omitempty"`
	Piped    bool                `yaml:"piped,omitempty"`
	Commands map[string]*Command `yaml:"commands,omitempty"`
	Jobs     []*Job              `yaml:"jobs,omitempty"`
}

// WithParallel sets the hook-level parallel flag (Commands-style hooks only).
func (h *Hook) WithParallel(parallel bool) *Hook {
	h.Parallel = &parallel

	return h
}

// WithPiped enables piped (sequential, fail-fast) execution of this hook's commands.
func (h *Hook) WithPiped(piped bool) *Hook {
	h.Piped = piped

	return h
}

// Command returns the named command on this hook, creating it on first access.
func (h *Hook) Command(name string) *Command {
	if c, ok := h.Commands[name]; ok {
		return c
	}

	if h.Commands == nil {
		h.Commands = map[string]*Command{}
	}

	c := &Command{}
	h.Commands[name] = c

	return c
}

// Job appends a new job to this hook's Jobs list and returns it for further
// configuration. Each top-level entry runs in declaration order; use AsGroup
// on the returned job to nest a parallel/sequential group of inner jobs.
func (h *Hook) Job() *Job {
	j := &Job{}

	h.Jobs = append(h.Jobs, j)

	return j
}

// Group returns the named group on this hook, wrapped in a Job entry, creating
// it on first access (mirroring makefile.Output.Target). The name is emitted as
// the wrapping job's `name:` and doubles as the lookup key, so different project
// blocks can contribute jobs to the same group regardless of compile order;
// group emission order follows first-creation order.
func (h *Hook) Group(name string) *Group {
	for _, j := range h.Jobs {
		if j.Group != nil && j.Name == name {
			return j.Group
		}
	}

	g := &Group{}

	h.Jobs = append(h.Jobs, &Job{Name: name, Group: g})

	return g
}
