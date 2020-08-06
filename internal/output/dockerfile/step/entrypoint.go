// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"encoding/json"
	"fmt"
	"io"
)

// EntrypointStep implements Dockerfile ENTRYPOINT step.
type EntrypointStep struct {
	args []string
}

// Entrypoint creates new EntrypointStep.
func Entrypoint(command string, args ...string) *EntrypointStep {
	return &EntrypointStep{
		args: append([]string{command}, args...),
	}
}

// Step implements Step interface.
func (step *EntrypointStep) Step() {}

// Generate implements Step interface.
func (step *EntrypointStep) Generate(w io.Writer) error {
	res, err := json.Marshal(step.args)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "ENTRYPOINT %s\n", string(res))

	return err
}
