// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
)

// EnvStep implements Dockerfile ENV step.
type EnvStep struct {
	name  string
	value string
}

// Env creates new EnvStep.
func Env(name, value string) *EnvStep {
	return &EnvStep{
		name:  name,
		value: value,
	}
}

// Step implements Step interface.
func (step *EnvStep) Step() {}

// Generate implements Step interface.
func (step *EnvStep) Generate(w io.Writer) error {
	_, err := fmt.Fprintf(w, "ENV %s=%s\n", step.name, step.value)

	return err
}
