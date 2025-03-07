// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/kballard/go-shellquote"
)

// RunStep implements Dockerfile RUN step.
type RunStep struct {
	command string
	args    []string

	script string

	security string
	env      []string
	mounts   []string
}

// Run creates new RunStep from command + args, properly escaping them.
func Run(command string, args ...string) *RunStep {
	return &RunStep{
		command: command,
		args:    args,
	}
}

// Script creates new Run step from shell script, it prints it verbatim.
func Script(script string) *RunStep {
	return &RunStep{
		script: script,
	}
}

// SecurityInsecure enables --security=insecure.
func (step *RunStep) SecurityInsecure() *RunStep {
	step.security = "insecure"

	return step
}

// Env sets up environment variables for the step.
func (step *RunStep) Env(name, value string) *RunStep {
	step.env = append(step.env, name+"="+value)

	return step
}

// CacheOption is a function that modifies cache mount.
type CacheOption func(*string)

// CacheLocked modifies cache mount to be locked.
func CacheLocked(mount *string) {
	*mount += ",sharing=locked"
}

// MountCache mounts cache at specified target path.
func (step *RunStep) MountCache(target, idPrefix string, opts ...CacheOption) *RunStep {
	mount := "type=cache,target=" + target + ",id=" + idPrefix + target

	for _, opt := range opts {
		opt(&mount)
	}

	step.mounts = append(step.mounts, mount)

	return step
}

// Step implements Step interface.
func (step *RunStep) Step() {}

// Generate implements Step interface.
func (step *RunStep) Generate(w io.Writer) error {
	security := ""
	if step.security != "" {
		security = fmt.Sprintf("--security=%s ", step.security)
	}

	env := strings.Join(step.env, " ")
	if env != "" {
		env += " "
	}

	mounts := slices.Clone(step.mounts)
	for i := range mounts {
		mounts[i] = "--mount=" + mounts[i]
	}

	mount := strings.Join(mounts, " ")
	if mount != "" {
		mount += " "
	}

	script := step.script
	if script == "" {
		script = step.command + " " + shellquote.Join(step.args...)
	}

	_, err := fmt.Fprintf(w, "RUN %s%s%s%s\n", security, mount, env, script)

	return err
}
