// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package step

import (
	"fmt"
	"io"
)

// CopyStep implements Dockerfile COPY step.
type CopyStep struct {
	from     string
	platform string
	src      string
	dst      string
}

// Copy creates new CopyStep.
func Copy(src, dst string) *CopyStep {
	return &CopyStep{
		src: src,
		dst: dst,
	}
}

// Step implements Step interface.
func (step *CopyStep) Step() {}

// From sets --from argument.
func (step *CopyStep) From(stage string) *CopyStep {
	step.from = stage

	return step
}

// Platform sets --platform argument.
func (step *CopyStep) Platform(platform string) *CopyStep {
	step.platform = platform

	return step
}

// Depends implements StageDependencies.
func (step *CopyStep) Depends() []string {
	if step.from == "" {
		return nil
	}

	return []string{step.from}
}

// Generate implements Step interface.
func (step *CopyStep) Generate(w io.Writer) error {
	fromClause := ""
	if step.from != "" {
		fromClause = fmt.Sprintf("--from=%s ", step.from)
	}

	if step.platform != "" {
		fromClause = fmt.Sprintf("--platform=%s %s", step.platform, fromClause)
	}

	_, err := fmt.Fprintf(w, "COPY %s%s %s\n", fromClause, step.src, step.dst)

	return err
}
