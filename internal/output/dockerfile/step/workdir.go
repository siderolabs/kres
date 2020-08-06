// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
)

// WorkDirStep implements Dockerfile WORKDIR step.
type WorkDirStep struct {
	dir string
}

// WorkDir creates new WorkDirStep.
func WorkDir(dir string) *WorkDirStep {
	return &WorkDirStep{
		dir: dir,
	}
}

// Step implements Step interface.
func (step *WorkDirStep) Step() {}

// Generate implements Step interface.
func (step *WorkDirStep) Generate(w io.Writer) error {
	_, err := fmt.Fprintf(w, "WORKDIR %s\n", step.dir)

	return err
}
