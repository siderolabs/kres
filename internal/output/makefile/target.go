// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makefile

import (
	"fmt"
	"io"
	"strings"
)

// Target is a Makefile target.
type Target struct {
	name        string
	depends     []string
	description string
	script      []string
	phony       bool
}

// Depends appends target dependency.
func (target *Target) Depends(targets ...string) *Target {
	target.depends = append(target.depends, targets...)

	return target
}

// Description attaches target description.
func (target *Target) Description(description string) *Target {
	target.description = description

	return target
}

// Phony marks target as phony.
func (target *Target) Phony() *Target {
	target.phony = true

	return target
}

// Script appends a line to target shell script.
func (target *Target) Script(lines ...string) *Target {
	for _, line := range lines {
		target.script = append(target.script, strings.Split(strings.TrimSpace(line), "\n")...)
	}

	return target
}

// Generate output for the Makefile.
func (target *Target) Generate(w io.Writer) error {
	if target.phony {
		if _, err := fmt.Fprintf(w, ".PHONY: %s\n", target.name); err != nil {
			return err
		}
	}

	description := target.description
	if description != "" {
		description = "  ## " + description
	}

	depends := strings.Join(target.depends, " ")
	if depends != "" {
		depends = " " + depends
	}

	if _, err := fmt.Fprintf(w, "%s:%s%s\n", target.name, depends, description); err != nil {
		return err
	}

	for _, line := range target.script {
		if _, err := fmt.Fprintf(w, "\t%s\n", strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)

	return err
}
