// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
)

// AddStep implements Dockerfile COPY step.
type AddStep struct {
	src string
	dst string
}

// Add creates new AddStep.
func Add(src, dst string) *AddStep {
	return &AddStep{
		src: src,
		dst: dst,
	}
}

// Step implements Step interface.
func (step *AddStep) Step() {}

// Generate implements Step interface.
func (step *AddStep) Generate(w io.Writer) error {
	_, err := fmt.Fprintf(w, "ADD %s %s\n", step.src, step.dst)

	return err
}
