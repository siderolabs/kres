// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package step defines Dockerfile steps
package step

import (
	"io"
)

// Step is an interface implemented by all Dockerfile steps.
type Step interface {
	Step()

	Generate(w io.Writer) error
}

// StageDependencies is implemented by steps which introduce dependencies to other stages.
type StageDependencies interface {
	Depends() []string
}
