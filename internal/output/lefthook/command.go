// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lefthook

// Command represents a single named command under a hook's commands: map.
type Command struct { //nolint:govet
	Run         string            `yaml:"run"`
	Tags        []string          `yaml:"tags,omitempty"`
	Glob        string            `yaml:"glob,omitempty"`
	Files       string            `yaml:"files,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Skip        []string          `yaml:"skip,omitempty"`
	Only        []string          `yaml:"only,omitempty"`
	Interactive bool              `yaml:"interactive,omitempty"`
	StageFixed  bool              `yaml:"stage_fixed,omitempty"`
	Priority    int               `yaml:"priority,omitempty"`
}

// WithRun sets the shell command lefthook executes for this command.
func (c *Command) WithRun(run string) *Command {
	c.Run = run

	return c
}

// WithTags attaches tags used for selective hook execution (lefthook --tags ...).
func (c *Command) WithTags(tags ...string) *Command {
	c.Tags = tags

	return c
}

// WithGlob restricts the command to files matching the given glob.
func (c *Command) WithGlob(glob string) *Command {
	c.Glob = glob

	return c
}

// WithFiles overrides the default file source (e.g. "git diff --name-only ...").
func (c *Command) WithFiles(files string) *Command {
	c.Files = files

	return c
}

// WithEnv sets an environment variable on the command; safe to call multiple times.
func (c *Command) WithEnv(name, value string) *Command {
	if c.Env == nil {
		c.Env = map[string]string{}
	}

	c.Env[name] = value

	return c
}

// WithSkip lists git states or refs where the command should be skipped (e.g. "merge", "rebase").
func (c *Command) WithSkip(skip ...string) *Command {
	c.Skip = skip

	return c
}

// WithOnly is the inverse of WithSkip: command runs only in the listed states.
func (c *Command) WithOnly(only ...string) *Command {
	c.Only = only

	return c
}

// WithInteractive marks the command as needing a TTY (stdin/stdout passthrough).
func (c *Command) WithInteractive() *Command {
	c.Interactive = true

	return c
}

// WithStageFixed re-stages files modified by the command (useful for formatters).
func (c *Command) WithStageFixed() *Command {
	c.StageFixed = true

	return c
}

// WithPriority sets the command's run order within its hook (lower runs first).
func (c *Command) WithPriority(priority int) *Command {
	c.Priority = priority

	return c
}
