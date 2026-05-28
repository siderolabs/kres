// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lefthook

// Job is an entry under a hook's or group's jobs: list. Each Job either runs
// a command directly (via Run or Script) or wraps a nested Group.
type Job struct { //nolint:govet
	Name        string            `yaml:"name,omitempty"`
	Run         string            `yaml:"run,omitempty"`
	Script      string            `yaml:"script,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Glob        []string          `yaml:"glob,omitempty"`
	Exclude     []string          `yaml:"exclude,omitempty"`
	Root        string            `yaml:"root,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Skip        []string          `yaml:"skip,omitempty"`
	Only        []string          `yaml:"only,omitempty"`
	Interactive bool              `yaml:"interactive,omitempty"`
	StageFixed  bool              `yaml:"stage_fixed,omitempty"`
	Priority    int               `yaml:"priority,omitempty"`
	Group       *Group            `yaml:"group,omitempty"`
}

// WithName sets the job's display name.
func (j *Job) WithName(name string) *Job {
	j.Name = name

	return j
}

// WithRun sets the shell command to execute for this job.
func (j *Job) WithRun(run string) *Job {
	j.Run = run

	return j
}

// WithScript sets a script file to execute (relative to lefthook source_dir).
func (j *Job) WithScript(script string) *Job {
	j.Script = script

	return j
}

// WithTags attaches selectable tags (lefthook --tags ...).
func (j *Job) WithTags(tags ...string) *Job {
	j.Tags = tags

	return j
}

// WithGlob restricts the job to files matching the given glob(s).
func (j *Job) WithGlob(glob ...string) *Job {
	j.Glob = glob

	return j
}

// WithExclude is the inverse of WithGlob: skip files matching these patterns.
func (j *Job) WithExclude(exclude ...string) *Job {
	j.Exclude = exclude

	return j
}

// WithRoot changes the working directory for the job.
func (j *Job) WithRoot(root string) *Job {
	j.Root = root

	return j
}

// WithEnv sets an environment variable on the job; safe to call multiple times.
func (j *Job) WithEnv(name, value string) *Job {
	if j.Env == nil {
		j.Env = map[string]string{}
	}

	j.Env[name] = value

	return j
}

// WithSkip lists git states or refs where the job should be skipped (e.g. "merge", "rebase").
func (j *Job) WithSkip(skip ...string) *Job {
	j.Skip = skip

	return j
}

// WithOnly is the inverse of WithSkip: job runs only in the listed states.
func (j *Job) WithOnly(only ...string) *Job {
	j.Only = only

	return j
}

// WithInteractive marks the job as needing a TTY (stdin/stdout passthrough).
func (j *Job) WithInteractive() *Job {
	j.Interactive = true

	return j
}

// WithStageFixed re-stages files modified by the job (useful for formatters).
func (j *Job) WithStageFixed() *Job {
	j.StageFixed = true

	return j
}

// WithPriority sets the job's run order within its container (lower runs first).
func (j *Job) WithPriority(priority int) *Job {
	j.Priority = priority

	return j
}

// AsGroup turns this job into a container for a nested Group, creating the
// group on first call and returning it for chained configuration. The job's
// Run/Script fields are typically left empty when AsGroup is used.
func (j *Job) AsGroup() *Group {
	if j.Group != nil {
		return j.Group
	}

	j.Group = &Group{}

	return j.Group
}
