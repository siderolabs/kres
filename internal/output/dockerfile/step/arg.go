// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
)

// ArgStep implements Dockerfile ARG step.
type ArgStep struct {
	arg string
}

// Arg creates new AregStep.
func Arg(arg string) *ArgStep {
	return &ArgStep{
		arg: arg,
	}
}

// Step implements Step interface.
func (step *ArgStep) Step() {}

// Generate implements Step interface.
func (step *ArgStep) Generate(w io.Writer) error {
	_, err := fmt.Fprintf(w, "ARG %s\n", step.arg)

	return err
}
