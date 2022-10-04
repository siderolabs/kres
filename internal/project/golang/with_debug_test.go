// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/golang"
)

func TestWithDebugInterfaces(t *testing.T) {
	assert.Implements(t, (*makefile.Compiler)(nil), new(golang.WithDebug))
}
