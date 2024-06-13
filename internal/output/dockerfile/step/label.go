// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
)

// LabelStep implements Dockerfile LABEL step.
type LabelStep struct {
	key   string
	value string
}

// Label creates new LabelStep.
func Label(key, value string) *LabelStep {
	return &LabelStep{
		key:   key,
		value: value,
	}
}

// Step implements Step interface.
func (step *LabelStep) Step() {}

// Generate implements Step interface.
func (step *LabelStep) Generate(w io.Writer) error {
	_, err := fmt.Fprintf(w, "LABEL %s=%s\n", step.key, step.value)

	return err
}
