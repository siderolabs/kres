// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dockerfile

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/siderolabs/kres/internal/output/dockerfile/step"
)

var stripVars = regexp.MustCompile(`\$\{\w*\}`)

// Stage implements Dockerfile stage (between 'FROM ...' and next 'FROM ...'.
type Stage struct {
	name        string
	from        string
	description string

	steps []step.Step
}

// From sets FROM propery of stage.
func (stage *Stage) From(from string) *Stage {
	stage.from = from

	return stage
}

// Description sets stage comment.
func (stage *Stage) Description(description string) *Stage {
	stage.description = description

	return stage
}

// Step appends stage step.
func (stage *Stage) Step(step step.Step) *Stage {
	stage.steps = append(stage.steps, step)

	return stage
}

// Dependencies calculates dependencies of this stage on other stages.
func (stage *Stage) Dependencies() []string {
	result := []string{stage.from}

	for _, st := range stage.steps {
		if deps, ok := st.(step.StageDependencies); ok {
			result = append(result, deps.Depends()...)
		}
	}

	return result
}

// Before implements stableToposort.Node interface.
func (stage *Stage) Before(otherStage *Stage) bool {
	for _, dep := range otherStage.Dependencies() {
		sanitized := stripVars.ReplaceAllString(dep, "")
		if sanitized != dep && sanitized != "" {
			return strings.HasPrefix(stage.name, sanitized)
		}

		if dep == stage.name {
			return true
		}
	}

	return false
}

// Generate renders Dockerfile to the output.
func (stage *Stage) Generate(w io.Writer) error {
	if stage.description != "" {
		if _, err := fmt.Fprintf(w, "# %s\n", stage.description); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "FROM %s AS %s\n", stage.from, stage.name); err != nil {
		return err
	}

	for _, step := range stage.steps {
		if err := step.Generate(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	return nil
}
